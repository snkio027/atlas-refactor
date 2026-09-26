package oci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func archive(t *testing.T, damage string) (string, string) {
	t.Helper()
	blobs := map[string][]byte{}
	add := func(b []byte) descriptor {
		h := sha256.Sum256(b)
		d := descriptor{Digest: "sha256:" + hex.EncodeToString(h[:]), Size: int64(len(b))}
		blobs[d.Digest[7:]] = b
		return d
	}
	config := add([]byte(`{"os":"linux","architecture":"arm64"}`))
	layer := add([]byte("layer bytes"))
	b, _ := json.Marshal(document{Config: config, Layers: []descriptor{layer}})
	manifest := add(b)
	manifest.Platform.OS = "linux"
	manifest.Platform.Architecture = "arm64"
	b, _ = json.Marshal(document{Manifests: []descriptor{manifest}})
	index := add(b)
	if damage == "missing-layer" {
		delete(blobs, layer.Digest[7:])
	}
	if damage == "corrupt" {
		blobs[layer.Digest[7:]] = []byte("bad bytes")
	}
	if damage == "metadata-only" {
		delete(blobs, config.Digest[7:])
		delete(blobs, manifest.Digest[7:])
		delete(blobs, layer.Digest[7:])
	}
	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	_ = tw.WriteHeader(&tar.Header{Name: "blobs/sha256/", Typeflag: tar.TypeDir, Mode: 0755})
	if damage != "missing-index" {
		d := index
		if damage == "wrong-index" {
			d = manifest
		}
		b, _ := json.Marshal(document{Manifests: []descriptor{d}})
		_ = tw.WriteHeader(&tar.Header{Name: "index.json", Mode: 0600, Size: int64(len(b))})
		_, _ = tw.Write(b)
	}
	for h, b := range blobs {
		_ = tw.WriteHeader(&tar.Header{Name: "blobs/sha256/" + h, Mode: 0600, Size: int64(len(b))})
		_, _ = tw.Write(b)
	}
	_ = tw.Close()
	path := filepath.Join(t.TempDir(), "image.tar")
	if e := os.WriteFile(path, out.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	return path, "test:v1@" + index.Digest
}
func TestClosure(t *testing.T) {
	for _, damage := range []string{"", "missing-layer", "corrupt", "metadata-only", "missing-index", "wrong-index"} {
		t.Run(damage, func(t *testing.T) {
			p, im := archive(t, damage)
			e := Verify(p, im)
			if (e == nil) != (damage == "") {
				t.Fatal(e)
			}
		})
	}
}
