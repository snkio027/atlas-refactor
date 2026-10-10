package oci

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func staticTestBinary(t *testing.T) []byte {
	t.Helper()
	d := t.TempDir()
	src := filepath.Join(d, "main.go")
	bin := filepath.Join(d, "app")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := exec.Command("go", "build", "-trimpath", "-o", bin, src)
	c.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if b, err := c.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestApplicationArchiveDeterministicAndBounded(t *testing.T) {
	binary := staticTestBinary(t)
	a, ref, err := BuildStatic(binary, "atlas.local/experiment-archive:v1")
	if err != nil {
		t.Fatal(err)
	}
	b, ref2, err := BuildStatic(binary, "atlas.local/experiment-archive:v1")
	if err != nil || ref != ref2 || !bytes.Equal(a, b) {
		t.Fatal("nondeterministic archive", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "image.tar")
	if err = os.WriteFile(path, a, 0600); err != nil {
		t.Fatal(err)
	}
	if err = VerifyApplication(path, ref); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	tr := tar.NewReader(bytes.NewReader(a))
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		raw, e := io.ReadAll(tr)
		if e != nil {
			t.Fatal(e)
		}
		files[h.Name] = raw
	}
	original := append([]byte{}, files["index.json"]...)
	for _, failure := range []string{"extra-image", "tag-alias", "wrong-platform", "damaged-layer"} {
		t.Run(failure, func(t *testing.T) {
			copyFiles := map[string][]byte{}
			for n, v := range files {
				copyFiles[n] = v
			}
			var index map[string]any
			if e := json.Unmarshal(original, &index); e != nil {
				t.Fatal(e)
			}
			entries := index["manifests"].([]any)
			d := entries[0].(map[string]any)
			switch failure {
			case "extra-image":
				index["manifests"] = append(entries, d)
			case "tag-alias":
				d["annotations"].(map[string]any)["io.containerd.image.name"] = "foreign.local/app:v1"
			case "wrong-platform":
				d["mediaType"] = "application/vnd.oci.image.index.v1+json"
			case "damaged-layer":
				for n := range copyFiles {
					if strings.HasPrefix(n, "blobs/sha256/") {
						copyFiles[n] = []byte("corrupt")
						break
					}
				}
			}
			copyFiles["index.json"] = marshal(index)
			bad, e := tarBytes(copyFiles, false)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(path, bad, 0600); e != nil {
				t.Fatal(e)
			}
			if e = VerifyApplication(path, ref); e == nil {
				t.Fatal("unsafe artifact accepted")
			}
		})
	}
}
func TestReferenceRejectsAliasesAndFloatingImages(t *testing.T) {
	for _, ref := range []string{"app:v1", "registry.local/a/../b:v1", "registry.local/a//b:v1", "registry.local/App:v1", "-host.local/a:v1", "registry.local/a:v1@sha256:bad"} {
		if !strings.Contains(ref, "@") {
			ref += "@sha256:" + strings.Repeat("a", 64)
		}
		if _, _, _, err := Reference(ref); err == nil {
			t.Fatal("accepted", ref)
		}
	}
	tag, canonical, digest, err := Reference("localhost:5000/team/app:v2@sha256:" + strings.Repeat("a", 64))
	if err != nil || tag != "localhost:5000/team/app:v2" || canonical != "localhost:5000/team/app@sha256:"+digest {
		t.Fatal(tag, canonical, err)
	}
}
