package workloadrun

import (
	"atlas-refactor/internal/observation"
	"atlas-refactor/internal/platform"
	"atlas-refactor/internal/workload"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
)

// Only the current, receipted, unfinished phase may wait for a newly declared
// Application's first controller observation. This never changes S1 facts or
// permits another publication; the ordinary complete Gate must still pass.
func (w *Workflow) initialApplicationOwners(ctx context.Context, result workload.Result, revision string) (map[string]string, error) {
	p, err := w.ReadPlan()
	if err != nil {
		return nil, err
	}
	phase := result.Inventory.Phase
	dir := filepath.Dir(w.publicationPath(p, phase))
	for _, name := range []string{"terminal.json", "final.json", phase + "-gate.json"} {
		if _, err := regular(filepath.Join(dir, name), true); err == nil {
			return nil, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	b, err := regular(w.publicationPath(p, phase), true)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var receipt Publication
	if err = workload.StrictDecode(b, &receipt); err != nil {
		return nil, err
	}
	prior, err := w.precedingPublication(p, phase)
	if err != nil {
		return nil, err
	}
	parent := p.Parent
	if prior != nil {
		parent = prior.Commit
	}
	if receipt.Schema != 1 || receipt.PlanSHA256 != workload.Digest(workload.JSON(p)) || receipt.Phase != phase || receipt.Parent != parent || receipt.Commit != revision || receipt.TreeSHA256 != platform.BundleDigest(result.Files) {
		return nil, errors.New("initial observation publication binding differs")
	}
	before, err := w.ReadTree(ctx, parent)
	if err != nil {
		return nil, err
	}
	inv, err := workload.InventoryOf(before, w.Context.ResourceModel)
	if err != nil {
		return nil, err
	}
	old, err := desiredObjects(before, inv)
	if err != nil {
		return nil, err
	}
	current, err := desiredObjects(result.Files, result.Inventory.Resources)
	if err != nil {
		return nil, err
	}
	return newApplicationOwners(result.Inventory.Resources, current, old), nil
}

func newApplicationOwners(resources []workload.OwnedResource, current, prior map[string]Object) map[string]string {
	owners := map[string]string{}
	for _, r := range resources {
		o := current[r.Identity]
		if o["apiVersion"] == "argoproj.io/v1alpha1" && o["kind"] == "Application" && prior[r.Identity] == nil {
			owners[str(at(o, "metadata", "name"))] = r.Owner
		}
	}
	return owners
}

func awaitingFirstObservation(want observation.ExpectedApplication, fact observation.ApplicationFact, live Object, owner string) bool {
	if owner == "" || want.UID != "" || fact.Classification != observation.Unknown || fact.Generation == nil || *fact.Generation != 1 || !fact.ArgoSSA || fact.Tracking != owner+":argoproj.io/Application:argocd/"+want.Name || live["operation"] != nil {
		return false
	}
	if status, exists := live["status"]; exists {
		fields, ok := status.(map[string]any)
		if !ok || len(fields) != 0 {
			return false
		}
	}
	// No drift, conditions, lost identity, malformed status, or explicit Unknown
	// may hide behind initialization. Facts stay UNKNOWN, never synthetic PASS.
	return slices.Equal(fact.Reasons, []string{"REVISION_EVIDENCE_MISSING", "SYNC_UNKNOWN", "HEALTH_UNKNOWN"})
}
