package workloadrun

import (
	"atlas-refactor/internal/installation"
	"atlas-refactor/internal/oci"
	"atlas-refactor/internal/workload"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
)

func (w *Workflow) imageArchive() ([]byte, string, error) {
	raw, e := regular(w.Config.ImageArchive, true)
	if e != nil {
		return nil, "", e
	}
	if workload.Digest(raw) != w.Config.ImageSHA256 {
		return nil, "", errors.New("OCI archive digest mismatch")
	}
	if len(w.Model.Intent.Workloads) == 0 {
		return nil, "", errors.New("missing workload")
	}
	ref := w.Model.Intent.Workloads[0].Image
	if !strings.HasPrefix(ref, "atlas.local/s2-web:v1@sha256:") {
		return nil, "", errors.New("local slice requires the verified Web OCI artifact")
	}
	for _, v := range w.Model.Intent.Workloads {
		if v.Image != ref {
			return nil, "", errors.New("slice Workloads must use the same reviewed image")
		}
	}
	if e = oci.Verify(w.Config.ImageArchive, ref); e != nil {
		return nil, "", e
	}
	return raw, ref, nil
}

func (w *Workflow) ImportImage(ctx context.Context, p Plan, approval string) error {
	if e := w.approve(p, approval); e != nil {
		return e
	}
	if e := w.bind(ctx); e != nil {
		return e
	}
	if e := w.authority(ctx); e != nil {
		return e
	}
	b, ref, e := w.imageArchive()
	if e != nil {
		return e
	}
	if e = installation.VerifyTools(w.Install.Config.StateDirectory, w.Install.Product.Tools); e != nil {
		return e
	}
	nodes := []string{p.Cluster + "-control-plane", p.Cluster + "-worker", p.Cluster + "-worker2", p.Cluster + "-worker3"}
	for _, node := range nodes {
		if _, e = command(ctx, w.Config.StateDirectory, b, "docker", "--context", "orbstack", "exec", "-i", node, "ctr", "--namespace", "k8s.io", "images", "import", "--digests", "-"); e != nil {
			return e
		}
		// containerd records the complete digest-pinned name Kubernetes will use.
		if _, e = command(ctx, w.Config.StateDirectory, nil, "docker", "--context", "orbstack", "exec", node, "ctr", "--namespace", "k8s.io", "images", "tag", "--force", "atlas.local/s2-web:v1", ref); e != nil {
			return e
		}
		raw, e := command(ctx, w.Config.StateDirectory, nil, "docker", "--context", "orbstack", "exec", node, "crictl", "inspecti", ref)
		if e != nil {
			return e
		}
		var info Object
		if e = json.Unmarshal(raw, &info); e != nil {
			return e
		}
		wanted := "sha256:" + strings.Split(ref, "@sha256:")[1]
		matched := false
		for _, d := range array(at(info, "status", "repoDigests")) {
			matched = matched || strings.HasSuffix(str(d), "@"+wanted)
		}
		if !matched {
			return errors.New("node image manifest identity differs")
		}
	}
	return save(filepath.Join(w.Config.StateDirectory, "authority", approval, "image.json"), workload.JSON(Object{"archiveSHA256": w.Config.ImageSHA256, "reference": ref, "nodes": nodes}), true)
}
