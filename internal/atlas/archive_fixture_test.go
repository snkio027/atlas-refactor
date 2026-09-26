package atlas

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepareFixtureArchives(t *testing.T, a *App) {
	t.Helper()
	blobs := map[string][]byte{}
	add := func(b []byte) Object {
		h := digest(b)
		blobs[h] = b
		return Object{"digest": "sha256:" + h, "size": len(b)}
	}
	config := add([]byte(`{"os":"linux","architecture":"arm64"}`))
	layer := add([]byte("test-layer"))
	manifest := add(jsonBytes(Object{"config": config, "layers": []Object{layer}}))
	manifest["platform"] = Object{"os": "linux", "architecture": "arm64"}
	index := add(jsonBytes(Object{"manifests": []Object{manifest}}))
	hash := index["digest"].(string)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for h, b := range blobs {
		_ = tw.WriteHeader(&tar.Header{Name: "blobs/sha256/" + h, Mode: 0600, Size: int64(len(b))})
		_, _ = tw.Write(b)
	}
	_ = tw.Close()
	path := filepath.Join(a.Root, "platform/development/versions.lock.json")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var lock Object
	_ = json.Unmarshal(b, &lock)
	for k, v := range lock["images"].(map[string]any) {
		ref, _, _ := strings.Cut(v.(string), "@")
		lock["images"].(map[string]any)[k] = ref + "@" + hash
	}
	images := lock["images"].(map[string]any)
	a.Lock.NodeImage = images["node"].(string)
	a.Lock.ArgoImage = images["argo"].(string)
	a.Lock.RedisImage = images["redis"].(string)
	if e = os.WriteFile(path, jsonBytes(lock), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(filepath.Join(a.Root, ".state/images"), 0700); e != nil {
		t.Fatal(e)
	}
	archive, _ := a.imageArchive(a.Lock.NodeImage)
	if e = os.WriteFile(archive, buf.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestMissingOfflineArchivePreventsClusterCreation(t *testing.T) {
	a, s := developmentFixture(t)
	p, _ := a.imageArchive(a.Lock.NodeImage)
	if e := os.Remove(p); e != nil {
		t.Fatal(e)
	}
	if e := a.Apply(t.Context(), a.Config.Cluster, true); e == nil {
		t.Fatal("accepted missing archive")
	}
	if len(s.effects) != 0 {
		t.Fatal(s.effects)
	}
}
