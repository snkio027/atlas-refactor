package installation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Tool struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	Member       string `json:"member,omitempty"`
	BinarySHA256 string `json:"binarySHA256"`
}

func ValidateTools(tools []Tool) error {
	need := map[string]bool{"helm": false, "kind": false, "kubectl": false, "kubeseal": false}
	for _, t := range tools {
		if used, ok := need[t.Name]; !ok || used {
			return errors.New("invalid tool inventory")
		}
		need[t.Name] = true
		u, e := url.Parse(t.URL)
		if e != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !(u.Host == "github.com" || u.Host == "get.helm.sh" || u.Host == "dl.k8s.io") {
			return errors.New("untrusted tool source")
		}
		if !sha.MatchString(t.SHA256) || !sha.MatchString(t.BinarySHA256) || t.Version == "" {
			return errors.New("unpinned tool")
		}
		if t.Member != "" && (filepath.IsAbs(t.Member) || filepath.Clean(t.Member) != t.Member || strings.Contains(t.Member, "..")) {
			return errors.New("unsafe archive member")
		}
	}
	for _, found := range need {
		if !found {
			return errors.New("missing runtime tool")
		}
	}
	return nil
}

// Tools are prepared into a content-addressed, installation-private directory.
// Only PrepareTools downloads; verification never replaces a corrupt binary.
func ToolDirectory(state string, tools []Tool) string {
	return filepath.Join(state, "tools", "darwin-arm64", Digest(JSON(tools)), "bin")
}
func VerifyTools(state string, tools []Tool) error {
	if e := ValidateTools(tools); e != nil {
		return e
	}
	for _, t := range tools {
		b, e := privateRead(filepath.Join(ToolDirectory(state, tools), t.Name))
		if e != nil {
			return e
		}
		if Digest(b) != t.BinarySHA256 {
			return errors.New("runtime tool digest mismatch: " + t.Name)
		}
		st, e := os.Stat(filepath.Join(ToolDirectory(state, tools), t.Name))
		if e != nil || st.Mode().Perm()&0100 == 0 {
			return errors.New("runtime tool is not executable")
		}
	}
	return nil
}
func PrepareTools(ctx context.Context, state string, tools []Tool) error {
	if e := privateDir(state); e != nil {
		return e
	}
	if e := ValidateTools(tools); e != nil {
		return e
	}
	dir := ToolDirectory(state, tools)
	if e := privateDir(dir); e != nil {
		return e
	}
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || req.URL.Scheme != "https" {
			return errors.New("unsafe tool redirect")
		}
		return nil
	}}
	for _, t := range tools {
		path := filepath.Join(dir, t.Name)
		if b, e := privateRead(path); e == nil {
			if Digest(b) != t.BinarySHA256 {
				return errors.New("existing tool was modified: " + t.Name)
			}
			if e = os.Chmod(path, 0700); e != nil {
				return e
			}
			continue
		} else if !os.IsNotExist(e) {
			return e
		}
		data, e := downloadTool(ctx, client, filepath.Join(state, "downloads"), t)
		if e != nil {
			return e
		}
		binary, e := toolBinary(t, data)
		if e != nil {
			return e
		}
		if Digest(binary) != t.BinarySHA256 {
			return errors.New("binary digest mismatch: " + t.Name)
		}
		if e = save(path, binary, true); e != nil {
			return e
		}
		if e = os.Chmod(path, 0700); e != nil {
			return e
		}
	}
	return VerifyTools(state, tools)
}
func toolBinary(t Tool, data []byte) ([]byte, error) {
	if t.Member == "" {
		return data, nil
	}
	gz, e := gzip.NewReader(bytes.NewReader(data))
	if e != nil {
		return nil, e
	}
	defer gz.Close()
	tr := tar.NewReader(io.LimitReader(gz, 512<<20))
	var result []byte
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if filepath.IsAbs(h.Name) || filepath.Clean(h.Name) != strings.TrimSuffix(h.Name, "/") || strings.Contains(h.Name, "..") {
			return nil, errors.New("unsafe tool archive path")
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
			return nil, errors.New("tool archive contains a link or special file")
		}
		if h.Name == t.Member {
			if result != nil || h.Typeflag != tar.TypeReg || h.Size > 256<<20 {
				return nil, errors.New("invalid tool archive member")
			}
			result, e = io.ReadAll(tr)
			if e != nil {
				return nil, e
			}
		}
	}
	if len(result) == 0 {
		return nil, errors.New("tool archive lacks executable")
	}
	return result, nil
}
