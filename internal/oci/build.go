// Static application packaging is offline, deterministic and has no deployment authority.
package oci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func marshal(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func tarBytes(files map[string][]byte, executable bool) ([]byte, error) {
	var b bytes.Buffer
	t := tar.NewWriter(&b)
	names := []string{}
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		mode := int64(0644)
		if executable {
			mode = 0755
		}
		if e := t.WriteHeader(&tar.Header{Name: n, Mode: mode, Size: int64(len(files[n])), ModTime: time.Unix(0, 0), Uid: 0, Gid: 0, Typeflag: tar.TypeReg}); e != nil {
			return nil, e
		}
		if _, e := t.Write(files[n]); e != nil {
			return nil, e
		}
	}
	if e := t.Close(); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
func BuildStatic(binary []byte, tag string) ([]byte, string, error) {
	if _, _, _, err := Reference(tag + "@sha256:" + strings.Repeat("0", 64)); err != nil {
		return nil, "", err
	}
	e, eErr := elf.NewFile(bytes.NewReader(binary))
	if eErr != nil {
		return nil, "", eErr
	}
	if e.Machine != elf.EM_AARCH64 || e.Class != elf.ELFCLASS64 || e.Type != elf.ET_EXEC && e.Type != elf.ET_DYN {
		return nil, "", errors.New("expected static linux/arm64 executable")
	}
	for _, p := range e.Progs {
		if p.Type == elf.PT_INTERP {
			return nil, "", errors.New("dynamic executable requires unprovided runtime")
		}
	}
	layer, err := tarBytes(map[string][]byte{"app": binary}, true)
	if err != nil {
		return nil, "", err
	}
	config := marshal(map[string]any{"architecture": "arm64", "os": "linux", "created": "1970-01-01T00:00:00Z", "config": map[string]any{"User": "65532:65532", "Entrypoint": []string{"/app"}, "Env": []string{"PORT=8080"}, "WorkingDir": "/"}, "rootfs": map[string]any{"type": "layers", "diff_ids": []string{"sha256:" + hash(layer)}}})
	descriptor := func(media string, b []byte) map[string]any {
		return map[string]any{"mediaType": media, "digest": "sha256:" + hash(b), "size": len(b)}
	}
	manifest := marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": descriptor("application/vnd.oci.image.config.v1+json", config), "layers": []any{descriptor("application/vnd.oci.image.layer.v1.tar", layer)}})
	d := descriptor("application/vnd.oci.image.manifest.v1+json", manifest)
	d["annotations"] = map[string]string{"org.opencontainers.image.ref.name": tag, "io.containerd.image.name": tag}
	d["platform"] = map[string]string{"os": "linux", "architecture": "arm64"}
	index := marshal(map[string]any{"schemaVersion": 2, "manifests": []any{d}})
	out, err := tarBytes(map[string][]byte{"oci-layout": []byte(`{"imageLayoutVersion":"1.0.0"}`), "index.json": index, "blobs/sha256/" + hash(layer): layer, "blobs/sha256/" + hash(config): config, "blobs/sha256/" + hash(manifest): manifest}, false)
	return out, tag + "@sha256:" + hash(manifest), err
}
