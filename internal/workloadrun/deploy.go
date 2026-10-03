package workloadrun

import (
	"atlas-refactor/internal/workload"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

func (w *Workflow) wait(ctx context.Context, phase, revision string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	r, e := w.Compile(phase)
	if e != nil {
		return e
	}
	for {
		_, e := w.observeLatest(ctx, r, revision)
		if e == nil {
			return nil
		}
		var pending Pending
		if !errors.As(e, &pending) {
			return e
		}
		if w.Progress != nil {
			w.Progress(string(pending))
		}
		select {
		case <-ctx.Done():
			return errors.New("S2 convergence timed out")
		case <-time.After(10 * time.Second):
		}
	}
}

// Deploy is a finite prerequisite-ordered publication procedure. It never resumes an incomplete
// externally mutated attempt or retries a failed write. A successful repeat
// verifies the final state without publishing, credential use or probe writes.
func (w *Workflow) Deploy(ctx context.Context, p Plan, approval string) (resultErr error) {
	if e := w.approve(p, approval); e != nil {
		return e
	}
	final := filepath.Join(w.Config.StateDirectory, "authority", approval, "final.json")
	if _, e := regular(final, true); e == nil {
		b, e := regular(w.publicationPath(p, "consumer"), true)
		if e != nil {
			return e
		}
		var receipt Publication
		if e = workload.StrictDecode(b, &receipt); e != nil {
			return e
		}
		return w.wait(ctx, "consumer", receipt.Commit)
	} else if !os.IsNotExist(e) {
		return e
	}
	terminal := filepath.Join(w.Config.StateDirectory, "authority", approval, "terminal.json")
	if _, e := regular(terminal, true); e == nil {
		return errors.New("prior attempt stopped; inspect external state before a new reviewed execution")
	} else if !os.IsNotExist(e) {
		return e
	}
	for _, phase := range p.Phases {
		for _, suffix := range []string{"", ".intent"} {
			if _, e := regular(w.publicationPath(p, phase)+suffix, true); e == nil {
				return errors.New("partial publication already exists; no automatic resume")
			} else if !os.IsNotExist(e) {
				return e
			}
		}
	}
	// Discover unavailable publication authority before image writes or private
	// credential generation, not at the first push after those preparations.
	if e := w.checkPublicationAccess(ctx); e != nil {
		return e
	}
	if e := w.CaptureBaseline(ctx, p, approval); e != nil {
		return e
	}
	// Replace ephemeral development output before this attempt so a failure in
	// preparation cannot accidentally archive an unrelated older observation.
	initial := Observation{Schema: 1, Phase: "preparation", ClusterUID: p.ClusterUID, Revision: p.Parent, Project: "UNKNOWN", Workload: "UNKNOWN", Binding: "UNKNOWN", Runtime: "UNPROVEN"}
	if e := save(filepath.Join(w.Config.StateDirectory, "latest-observation.json"), workload.JSON(initial), false); e != nil {
		return e
	}
	external := false
	defer func() {
		if resultErr != nil && external {
			if b, e := regular(filepath.Join(w.Config.StateDirectory, "latest-observation.json"), true); e == nil {
				_ = save(filepath.Join(filepath.Dir(terminal), "stop-observation.json"), b, true)
			}
			_ = save(terminal, workload.JSON(Object{"result": "STOP", "planSHA256": approval, "reason": resultErr.Error(), "externalStateMayHaveChanged": true}), true)
		}
	}()
	external = true // Credential evidence and node image imports are retained too.
	if e := w.ImportImage(ctx, p, approval); e != nil {
		return e
	}
	if e := w.PrepareCredentials(ctx, p, approval); e != nil {
		return e
	}
	for _, phase := range p.Phases {
		if w.Progress != nil {
			w.Progress("Publishing " + phase)
		}
		receipt, e := w.Publish(ctx, p, approval, phase)
		if e != nil {
			return e
		}
		if e = w.wait(ctx, phase, receipt.Commit); e != nil {
			return e
		}
		gate, e := regular(filepath.Join(w.Config.StateDirectory, "latest-observation.json"), true)
		if e != nil {
			return e
		}
		if e = save(filepath.Join(filepath.Dir(terminal), phase+"-gate.json"), gate, true); e != nil {
			return e
		}
	}
	// Metrics discovery may lag otherwise-complete reconciliation. Wait for it
	// before the single functional probe so retries do not repeat S3 writes.
	if e := w.waitMetrics(ctx); e != nil {
		return e
	}
	return w.Probe(ctx, p, approval)
}
func (w *Workflow) waitMetrics(ctx context.Context) error {
	// The functional probe checks every target. This bounded read waits only for
	// Prometheus' first discovery, and never masks malformed/error responses.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	for {
		e := w.metrics(ctx)
		if e == nil {
			return nil
		}
		var p Pending
		if !errors.As(e, &p) {
			return e
		}
		select {
		case <-ctx.Done():
			return e
		case <-time.After(5 * time.Second):
		}
	}
}
