package installation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Record the first successful host check once so retries cannot erase the
// initial capacity/version evidence. This measures capacity; it does not claim
// that an untested hardware minimum has been established.
func (w *Workflow) captureHost(ctx context.Context, available, total uint64) error {
	path := filepath.Join(w.Config.StateDirectory, "host-first-check.json")
	if b, e := privateRead(path); e == nil {
		var prior map[string]any
		if e = Decode(b, &prior); e != nil {
			return e
		}
		if prior["installID"] != w.Record.InstallID {
			return errors.New("host evidence belongs to another installation")
		}
		return nil
	} else if !os.IsNotExist(e) {
		return e
	}
	evidence := map[string]any{"installID": w.Record.InstallID, "diskAvailableBytes": available, "diskTotalBytes": total, "platform": "darwin/arm64"}
	for _, q := range []struct {
		field, tool string
		args        []string
	}{
		{"macOS", "sw_vers", []string{"-productVersion"}},
		{"macOSBuild", "sw_vers", []string{"-buildVersion"}},
		{"orbStack", "orb", []string{"version"}},
		{"hostMemoryBytes", "sysctl", []string{"-n", "hw.memsize"}},
		{"docker", "docker", []string{"--context", "orbstack", "info", "--format", `{"serverVersion":{{json .ServerVersion}},"memoryBytes":{{.MemTotal}},"cpus":{{.NCPU}}}`}},
	} {
		b, e := w.command(ctx, w.Config.StateDirectory, nil, q.tool, q.args...)
		if e != nil {
			return e
		}
		value := strings.TrimSpace(string(b))
		if value == "" {
			return errors.New("empty host fact: " + q.field)
		}
		evidence[q.field] = value
	}
	return save(path, JSON(evidence), true)
}

func (w *Workflow) finalEvidence(ctx context.Context, planHash string) ([]byte, error) {
	host, e := privateRead(filepath.Join(w.Config.StateDirectory, "host-first-check.json"))
	if e != nil {
		return nil, e
	}
	var facts map[string]any
	if e = Decode(host, &facts); e != nil {
		return nil, e
	}
	if facts["installID"] != w.Record.InstallID || len(w.applications) == 0 {
		return nil, errors.New("missing installation acceptance facts")
	}
	a, e := w.app(ctx, w.Record.BaseCommit)
	if e != nil {
		return nil, e
	}
	base, e := a.Render(ctx)
	if e != nil {
		return nil, e
	}
	full, e := w.desiredFull()
	if e != nil {
		return nil, e
	}
	full[signalPath] = base[signalPath]
	hashes := map[string]string{}
	for path, b := range full {
		hashes[path] = Digest(b)
	}
	return JSON(map[string]any{"result": "PASS", "exitCode": 0, "tools": w.Product.Tools, "hostFirstCheck": facts, "renderSHA256": hashes, "applications": w.applications, "productSHA256": w.ProductDigest, "sourceCommit": w.Product.SourceCommit, "binarySHA256": w.BinaryDigest, "deploymentCommit": w.Record.FullCommit, "clusterUID": w.Record.ClusterUID, "installID": w.Record.InstallID, "planSHA256": planHash, "certificateSHA256": w.Record.CertificateSHA256, "sealedSHA256": w.Record.SealedSHA256}), nil
}
