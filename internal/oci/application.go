package oci

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

// Reference supports one explicit registry/repository:tag@sha256 reference.
// Reject path aliases so containerd and authored intent name the same image.
func Reference(ref string) (tag, canonical, digest string, err error) {
	re := regexp.MustCompile(`^([a-z0-9]+(?:[.-][a-z0-9]+)*(?::[0-9]+)?/(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*):([A-Za-z0-9_][A-Za-z0-9_.-]{0,127})@sha256:([a-f0-9]{64})$`)
	m := re.FindStringSubmatch(ref)
	if m == nil {
		return "", "", "", errors.New("image must be registry/repository:tag@sha256:<64 lowercase hex>")
	}
	return m[1] + ":" + m[2], m[1] + "@sha256:" + m[3], m[3], nil
}

// Ordinary imports accept a single image. Unbound extra index entries can cause
// containerd to register additional tags, so reject them before node mutation.
func VerifyApplication(path, ref string) error {
	tag, _, digest, err := Reference(ref)
	if err != nil {
		return err
	}
	if err = applicationEntries(path); err != nil {
		return err
	}
	if err = Verify(path, ref); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if err != nil {
			return errors.New("OCI application index missing")
		}
		if h.Name != "index.json" {
			continue
		}
		if h.Size > 8<<20 {
			return errors.New("OCI index too large")
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		var index struct {
			Manifests []struct {
				Digest      string
				MediaType   string
				Annotations map[string]string
			}
		}
		if err = json.Unmarshal(raw, &index); err != nil {
			return err
		}
		if len(index.Manifests) != 1 {
			return errors.New("application OCI archive must contain exactly one image manifest")
		}
		d := index.Manifests[0]
		if d.Digest != "sha256:"+digest || d.MediaType != "application/vnd.oci.image.manifest.v1+json" ||
			d.Annotations["org.opencontainers.image.ref.name"] != tag ||
			d.Annotations["io.containerd.image.name"] != tag {
			return errors.New("OCI manifest/tag differs from application intent")
		}
		for k := range d.Annotations {
			if strings.Contains(k, "image.name") && k != "io.containerd.image.name" {
				return errors.New("unsupported image alias")
			}
		}
		return nil
	}
}

// Containerd must not discover a second Docker manifest, symlink, archive alias
// or unrelated named image outside the reviewed OCI index.
func applicationEntries(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	seen := map[string]bool{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if seen[h.Name] {
			return errors.New("duplicate application archive entry")
		}
		seen[h.Name] = true
		if h.Typeflag == tar.TypeDir {
			if h.Name == "blobs/" || h.Name == "blobs/sha256/" {
				continue
			}
			return errors.New("unsupported OCI directory")
		}
		if h.Typeflag != tar.TypeReg {
			return errors.New("application archive entries must be regular files")
		}
		if h.Name == "index.json" || h.Name == "oci-layout" {
			continue
		}
		digest, ok := strings.CutPrefix(h.Name, "blobs/sha256/")
		if !ok || !regexp.MustCompile("^[a-f0-9]{64}$").MatchString(digest) {
			return errors.New("unexpected application archive entry")
		}
	}
}
