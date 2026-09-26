// Package oci verifies the offline linux/arm64 closure of Docker OCI exports.
package oci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type descriptor struct {
	Digest   string
	Size     int64
	Platform struct{ OS, Architecture string }
}
type document struct {
	Manifests        []descriptor
	Config           descriptor
	Layers           []descriptor
	OS, Architecture string
}
type blob struct {
	size int64
	data []byte
}

// Verify streams every blob through SHA-256, then follows the locked index to
// the supported manifest, config and all layers. Foreign platforms may be absent.
// No archive paths are extracted and untrusted descriptors never address files.
func Verify(path, image string) error {
	_, locked, ok := strings.Cut(image, "@sha256:")
	if !ok || len(locked) != 64 {
		return errors.New("expected digest-pinned image")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	tr := tar.NewReader(f)
	blobs := map[string]blob{}
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		if h.Typeflag == tar.TypeDir || !strings.HasPrefix(h.Name, "blobs/sha256/") {
			continue
		}
		name := strings.TrimPrefix(h.Name, "blobs/sha256/")
		if len(name) != 64 || h.Typeflag != tar.TypeReg {
			return errors.New("invalid OCI blob entry")
		}
		if _, exists := blobs[name]; exists {
			return errors.New("duplicate OCI blob")
		}
		hash := sha256.New()
		var data bytes.Buffer
		var out io.Writer = hash
		if h.Size <= 8<<20 {
			out = io.MultiWriter(hash, &data)
		}
		n, e := io.Copy(out, tr)
		if e != nil {
			return e
		}
		if n != h.Size || hex.EncodeToString(hash.Sum(nil)) != name {
			return errors.New("OCI blob checksum mismatch")
		}
		blobs[name] = blob{n, data.Bytes()}
	}
	read := func(d descriptor) (blob, error) {
		key, ok := strings.CutPrefix(d.Digest, "sha256:")
		if !ok {
			return blob{}, errors.New("unsupported OCI digest")
		}
		b, ok := blobs[key]
		if !ok || b.size != d.Size {
			return blob{}, fmt.Errorf("missing or truncated OCI blob: %s", d.Digest)
		}
		return b, nil
	}
	var visit func(descriptor, int) error
	visit = func(d descriptor, depth int) error {
		if depth > 8 {
			return errors.New("OCI index nesting limit")
		}
		b, e := read(d)
		if e != nil {
			return e
		}
		var doc document
		if e = json.Unmarshal(b.data, &doc); e != nil {
			return e
		}
		if len(doc.Manifests) > 0 {
			var last error
			for _, child := range doc.Manifests {
				if child.Platform.OS != "" && (child.Platform.OS != "linux" || child.Platform.Architecture != "arm64") {
					continue
				}
				if e = visit(child, depth+1); e == nil {
					return nil
				}
				last = e
			}
			return fmt.Errorf("linux/arm64 closure unavailable: %v", last)
		}
		c, e := read(doc.Config)
		if e != nil {
			return e
		}
		var cfg document
		if e = json.Unmarshal(c.data, &cfg); e != nil {
			return e
		}
		if cfg.OS != "linux" || cfg.Architecture != "arm64" {
			return errors.New("OCI config platform mismatch")
		}
		if len(doc.Layers) == 0 {
			return errors.New("empty runtime image")
		}
		for _, layer := range doc.Layers {
			if _, e = read(layer); e != nil {
				return e
			}
		}
		return nil
	}
	top, ok := blobs[locked]
	if !ok {
		return errors.New("locked OCI digest missing from export")
	}
	return visit(descriptor{Digest: "sha256:" + locked, Size: top.size}, 0)
}
