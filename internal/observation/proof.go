package observation

import (
	"errors"
	"sort"
)

const ProofVersion = "atlas.semantic-proof/v1"

// Proof is an evidence projection, NEVER a mutation precondition. Raw reads
// retain RVs and timestamps. Unknown fields in Application status are retained;
// only named reconciliation bookkeeping is excluded.
func Proof(o Object, readiness string) (string, error) {
	ref := Reference(o)
	if ref.Validate() != nil || String(At(o, "metadata", "uid")) == "" || String(At(o, "metadata", "resourceVersion")) == "" {
		return "", errors.New("proof identity unavailable")
	}
	f := resourceBase(ref, o, nil)
	f.ResourceVersion = ""
	generation, observed := f.Generation, f.ObservedGeneration
	f.Generation, f.ObservedGeneration = nil, nil
	sort.Slice(f.Conditions, func(i, j int) bool { return Digest(f.Conditions[i]) < Digest(f.Conditions[j]) })
	base := Object{"version": ProofVersion, "identity": f, "content": Semantic(o), "hasGeneration": generation != nil, "hasObservedGeneration": observed != nil}
	fields := []any{}
	for _, raw := range Slice(At(o, "metadata", "managedFields")) {
		m := Map(raw)
		if String(m["manager"]) == "argocd-controller" && String(m["operation"]) == "Apply" && String(m["subresource"]) == "" {
			c := Clone(m)
			delete(c, "time")
			fields = append(fields, c)
		}
	}
	sortValues(fields)
	base["argoFields"] = fields
	switch {
	case ref.APIVersion == "argoproj.io/v1alpha1" && ref.Kind == "Application":
		status := Clone(Map(o["status"]))
		for _, key := range []string{"reconciledAt", "observedAt", "observedGeneration"} {
			delete(status, key)
		}
		cleanConditions(status)
		if h := Map(status["health"]); h != nil {
			delete(h, "lastTransitionTime")
		}
		sortValues(Slice(status["resources"]))
		sortValues(Slice(At(status, "operationState", "syncResult", "resources")))
		base["status"] = status
	case ref.APIVersion == "argoproj.io/v1alpha1" && ref.Kind == "AppProject":
	case ref.APIVersion == "v1" && ref.Kind == "Node":
		status := Clone(Map(o["status"]))
		cleanConditions(status)
		// Node inventory proves identity, spec and readiness, not capacity usage.
		base["conditions"] = status["conditions"]
	default:
		check := ClassifyResource(ExpectedResource{Ref: ref, Readiness: readiness}, o, nil)
		check.ResourceVersion = ""
		check.Generation = nil
		check.ObservedGeneration = nil
		sort.Slice(check.Conditions, func(i, j int) bool { return Digest(check.Conditions[i]) < Digest(check.Conditions[j]) })
		base["readiness"] = check
		if readiness != "identity-content" {
			status := Clone(Map(o["status"]))
			delete(status, "observedGeneration")
			cleanConditions(status)
			base["status"] = status
		}
	}
	return Digest(base), nil
}
func sortValues(a []any) { sort.Slice(a, func(i, j int) bool { return Digest(a[i]) < Digest(a[j]) }) }
func cleanConditions(status Object) {
	for _, raw := range Slice(status["conditions"]) {
		c := Map(raw)
		delete(c, "lastHeartbeatTime")
		delete(c, "lastTransitionTime")
	}
	sortValues(Slice(status["conditions"]))
}
func SameProof(a, b Object, readiness string) bool {
	x, e := Proof(a, readiness)
	y, f := Proof(b, readiness)
	return e == nil && f == nil && x == y
}

// InventoryProof rejects malformed, duplicate and unexpected collection types.
func InventoryProof(objects []Object, kind Ref) (string, error) {
	values := map[string]string{}
	for _, o := range objects {
		ref := Reference(o)
		if ref.APIVersion != kind.APIVersion || ref.Kind != kind.Kind || ref.Namespace != kind.Namespace || values[ref.Key()] != "" {
			return "", errors.New("invalid inventory")
		}
		if ref.Kind != "Application" && ref.Kind != "AppProject" && ref.Kind != "Node" {
			return "", errors.New("unsupported proof inventory")
		}
		p, e := Proof(o, "identity-content")
		if e != nil {
			return "", e
		}
		values[ref.Key()] = p
	}
	return Digest(values), nil
}

// ValidateClosingProof verifies stored double reads without trusting a summary.
func ValidateClosingProof(e Envelope, resources map[string]string) error {
	if e.ProofVersion != ProofVersion || len(e.Raw) != len(e.ClosingRaw) {
		return errors.New("incomplete semantic proof")
	}
	closing := map[string]Object{}
	for _, o := range e.ClosingRaw {
		key := Reference(o).Key()
		if closing[key] != nil {
			return errors.New("duplicate closing read")
		}
		closing[key] = o
	}
	for _, o := range e.Raw {
		key := Reference(o).Key()
		rule := resources[key]
		if rule == "" {
			rule = "identity-content"
		}
		if !SameProof(o, closing[key], rule) {
			return errors.New("semantic proof changed")
		}
		delete(closing, key)
	}
	if len(closing) != 0 {
		return errors.New("unexpected closing read")
	}
	return nil
}
