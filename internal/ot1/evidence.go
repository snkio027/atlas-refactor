package ot1

import (
	"atlas-refactor/internal/observation"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

type Snapshot struct {
	Schema         int                  `json:"schema"`
	PlanSHA256     string               `json:"planSHA256"`
	Stage          string               `json:"stage"`
	Revision       string               `json:"revision"`
	Envelope       observation.Envelope `json:"observation"`
	Applications   []observation.Object `json:"applications"`
	Projects       []observation.Object `json:"projects"`
	Nodes          []observation.Object `json:"nodes"`
	InventoryError string               `json:"inventoryError,omitempty"`
}
type Assessment struct {
	Stage     string   `json:"stage"`
	Ownership string   `json:"ownership"`
	Atlas     string   `json:"atlas"`
	Reasons   []string `json:"reasons"`
}

func (a Assessment) Passed() bool {
	return a.Ownership == "VERIFIED" && (a.Atlas == "VERIFIED" || a.Atlas == "NOT_APPLICABLE")
}

func AppRef(name string) observation.Ref {
	return observation.Ref{APIVersion: "argoproj.io/v1alpha1", Kind: "Application", Namespace: "argocd", Name: name}
}
func identityRefs() []observation.Ref {
	return []observation.Ref{{APIVersion: "v1", Kind: "ConfigMap", Namespace: "kube-system", Name: "atlas-refactor-identity"}, {APIVersion: "v1", Kind: "ConfigMap", Namespace: "kube-system", Name: "atlas-refactor-handoff"}, {APIVersion: "v1", Kind: "ConfigMap", Namespace: "kube-system", Name: "atlas-refactor-receipt"}, {APIVersion: "v1", Kind: "ConfigMap", Namespace: "argocd", Name: "atlas-refactor-adoption-signal"}}
}
func rawIndex(raw []observation.Object) map[string]observation.Object {
	out := map[string]observation.Object{}
	for _, o := range raw {
		out[observation.Reference(o).Key()] = o
	}
	return out
}
func ownerFor(s Stage, object ScopeObject) string { return s.Owners[domain(object.NewOwner)] }
func readiness(ref observation.Ref) string {
	if ref.Kind == "Namespace" {
		return "namespace-active"
	}
	return "identity-content"
}

// Capture uses exactly the same read-only library as atlas-platform observe.
// Inventories are fenced independently; no atomic multi-object transaction is claimed.
func Capture(ctx context.Context, reader observation.InventoryReader, plan Plan, index int, baseline *Snapshot) (Snapshot, error) {
	if index < 0 || index >= len(plan.Phases) {
		return Snapshot{}, errors.New("phase outside plan")
	}
	phase := plan.Phases[index]
	out := Snapshot{Schema: 1, PlanSHA256: observation.Digest(plan), Stage: phase.Stage.Name, Revision: phase.Revision}
	expect := observation.Expectation{Schema: 1, Subject: "ot1/" + phase.Stage.Name, Target: plan.Target, ImplementationSHA: plan.Implementation.Revision, Revision: phase.Revision, Applications: phase.Applications}
	prior := map[string]observation.Object{}
	if baseline != nil {
		prior = rawIndex(baseline.Envelope.Raw)
	}
	for _, item := range plan.Scope.Objects {
		ref, e := ParseIdentity(item.Identity)
		if e != nil {
			return out, e
		}
		old := prior[ref.Key()]
		r := observation.ExpectedResource{Ref: ref, Tracking: observation.Tracking(ownerFor(phase.Stage, item), destination(ownerFor(phase.Stage, item)), ref), RequireSSA: true, Readiness: readiness(ref)}
		if old != nil {
			r.UID = observation.String(observation.At(old, "metadata", "uid"))
			r.SemanticSHA256 = observation.Digest(observation.Semantic(old))
		}
		expect.Resources = append(expect.Resources, r)
	}
	for _, ref := range identityRefs() {
		r := observation.ExpectedResource{Ref: ref, Readiness: "identity-content"}
		if old := prior[ref.Key()]; old != nil {
			r.UID = observation.String(observation.At(old, "metadata", "uid"))
			r.SemanticSHA256 = observation.Digest(observation.Semantic(old))
		}
		expect.Resources = append(expect.Resources, r)
	}
	kinds := []observation.Ref{{APIVersion: "argoproj.io/v1alpha1", Kind: "Application", Namespace: "argocd"}, {APIVersion: "argoproj.io/v1alpha1", Kind: "AppProject", Namespace: "argocd"}, {APIVersion: "v1", Kind: "Node"}}
	collections := []*[]observation.Object{&out.Applications, &out.Projects, &out.Nodes}
	for i, kind := range kinds {
		items, e := reader.List(ctx, kind)
		if e != nil {
			out.InventoryError = "INVENTORY_UNAVAILABLE"
			return out, nil
		}
		*collections[i] = items
	}
	var e error
	out.Envelope, e = observation.Collect(ctx, reader, expect)
	if e != nil {
		return out, e
	}
	out.Envelope.BinarySHA256 = plan.Implementation.BinarySHA256
	for i, kind := range kinds {
		items, e := reader.List(ctx, kind)
		if e != nil || inventoryFence(items) != inventoryFence(*collections[i]) {
			out.InventoryError = "INVENTORY_CHANGED_OR_UNAVAILABLE"
			break
		}
	}
	// Each Application raw GET must agree with the list, not just its name.
	raw := rawIndex(out.Envelope.Raw)
	for _, app := range out.Applications {
		if got := raw[observation.Reference(app).Key()]; got != nil && version(got) != version(app) {
			out.InventoryError = "APPLICATION_CHANGED_DURING_CAPTURE"
		}
	}
	uid, readErr := reader.ClusterIdentity(ctx)
	if readErr != nil || uid != plan.Target.ClusterUID {
		out.InventoryError = "TARGET_CHANGED_AFTER_INVENTORY"
	}
	return out, nil
}
func version(o observation.Object) string {
	return observation.String(observation.At(o, "metadata", "uid")) + "/" + observation.String(observation.At(o, "metadata", "resourceVersion"))
}
func inventoryFence(objects []observation.Object) string {
	values := []string{}
	for _, o := range objects {
		values = append(values, observation.Reference(o).Key()+":"+version(o))
	}
	sort.Strings(values)
	return observation.Digest(values)
}

// LiveDesired projects only explicit Kubernetes defaults for the four reviewed
// GVKs. It never uses subset comparison to hide additional network/policy fields.
func LiveDesired(o observation.Object) observation.Object {
	copy := observation.Clone(o)
	if observation.Reference(copy).Kind == "Namespace" {
		meta := observation.Map(copy["metadata"])
		labels := observation.Map(meta["labels"])
		if labels == nil {
			labels = observation.Object{}
			meta["labels"] = labels
		}
		labels["kubernetes.io/metadata.name"] = observation.String(meta["name"])
		copy["spec"] = observation.Object{"finalizers": []any{"kubernetes"}}
	}
	return copy
}

func operationMatches(app observation.Object, phase Phase, name, outcome string) error {
	status := observation.Map(app["status"])
	op := observation.Map(status["operationState"])
	if app["operation"] != nil || observation.String(op["finishedAt"]) == "" {
		return errors.New("operation not idle/terminal")
	}
	if _, e := time.Parse(time.RFC3339, observation.String(op["finishedAt"])); e != nil {
		return errors.New("invalid operation timestamp")
	}
	want := "Succeeded"
	if outcome == "blocked" {
		want = "Failed"
	}
	if observation.String(op["phase"]) != want || observation.String(observation.At(op, "syncResult", "revision")) != phase.Revision || observation.String(observation.At(status, "sync", "revision")) != phase.Revision {
		return errors.New("operation outcome/revision mismatch")
	}
	request := observation.Map(op["operation"])
	if !reflect.DeepEqual(observation.Slice(request["info"]), []any{observation.Object{"name": "ot1-stage", "value": name}}) || observation.String(observation.At(request, "sync", "revision")) != phase.Revision || observation.At(request, "sync", "prune") != false || observation.Digest(observation.At(request, "sync", "syncOptions")) != observation.Digest(observation.At(app, "spec", "syncPolicy", "syncOptions")) {
		return errors.New("operation request does not bind this phase")
	}
	conditions := observation.Slice(status["conditions"])
	if outcome == "blocked" {
		if !strings.Contains(observation.String(op["message"]), "Shared resource found:") || len(conditions) == 0 {
			return errors.New("not the expected shared-resource refusal")
		}
		for _, c := range conditions {
			if observation.String(observation.Map(c)["type"]) != "SharedResourceWarning" {
				return errors.New("unrelated error condition")
			}
		}
	} else if len(conditions) != 0 || observation.String(observation.At(status, "sync", "status")) != "Synced" || observation.String(observation.At(status, "health", "status")) != "Healthy" {
		return errors.New("successful phase has not converged")
	}
	return nil
}

// Assess checks ownership separately from the full Atlas gate. Gate-B evidence
// is deliberately supplied by the existing engine/runtime verifier, not inferred
// from healthy Pods or an expected Failed operation.
func Assess(plan Plan, index int, current Snapshot, baseline, previous *Snapshot, desired map[string]observation.Object, gate *GateProof) Assessment {
	if index < 0 || index >= len(plan.Phases) {
		return Assessment{Ownership: "STOP", Atlas: "NOT_PROVEN", Reasons: []string{"PHASE_OUTSIDE_PLAN"}}
	}
	phase := plan.Phases[index]
	out := Assessment{Stage: phase.Stage.Name, Ownership: "VERIFIED", Atlas: "NOT_APPLICABLE", Reasons: []string{}}
	fail := func(reason string) { out.Ownership = "STOP"; out.Reasons = append(out.Reasons, reason) }
	if current.Envelope.Schema != "atlas.observation/v1" || current.Envelope.Subject != "ot1/"+phase.Stage.Name || current.Envelope.StartedAt.IsZero() || current.Envelope.FinishedAt.Before(current.Envelope.StartedAt) || current.Schema != 1 || current.PlanSHA256 != observation.Digest(plan) || current.Stage != phase.Stage.Name || current.Revision != phase.Revision || current.Envelope.Target != plan.Target || current.Envelope.ImplementationSHA != plan.Implementation.Revision || current.Envelope.BinarySHA256 != plan.Implementation.BinarySHA256 || current.Envelope.ExpectedRevision != phase.Revision {
		fail("EVIDENCE_BINDING_MISMATCH")
	}
	if current.InventoryError != "" || len(current.Envelope.Reasons) != 0 {
		fail("READ_UNAVAILABLE_OR_CONCURRENT_CHANGE")
	}
	raw := rawIndex(current.Envelope.Raw)
	base := raw
	if baseline != nil {
		base = rawIndex(baseline.Envelope.Raw)
	}
	if len(raw) != len(current.Envelope.Raw) || len(raw) != len(phase.Applications)+13+4 {
		fail("RAW_INVENTORY_MISMATCH")
	}
	for _, item := range plan.Scope.Objects {
		ref, _ := ParseIdentity(item.Identity)
		actual := raw[ref.Key()]
		old := base[ref.Key()]
		if actual == nil || old == nil {
			fail("OBJECT_MISSING:" + ref.Key())
			continue
		}
		want := observation.ExpectedResource{Ref: ref, UID: observation.String(observation.At(old, "metadata", "uid")), SemanticSHA256: observation.Digest(observation.Semantic(old)), Tracking: observation.Tracking(ownerFor(phase.Stage, item), destination(ownerFor(phase.Stage, item)), ref), RequireSSA: true, Readiness: readiness(ref)}
		if observation.ClassifyResource(want, actual, nil).Classification != observation.Verified {
			fail("RESOURCE_INVARIANT:" + ref.Key())
		}
		if index == 0 {
			if git := desired[ref.Key()]; git == nil || observation.Digest(observation.Semantic(LiveDesired(git))) != observation.Digest(observation.Semantic(actual)) {
				fail("BASELINE_CONTENT_NOT_GIT_DEFINED:" + ref.Key())
			}
		}
	}
	for _, ref := range identityRefs() {
		actual, old := raw[ref.Key()], base[ref.Key()]
		if actual == nil || old == nil || observation.String(observation.At(old, "metadata", "uid")) == "" || observation.String(observation.At(actual, "metadata", "uid")) != observation.String(observation.At(old, "metadata", "uid")) || observation.Digest(observation.Semantic(actual)) != observation.Digest(observation.Semantic(old)) {
			fail("BOOTSTRAP_RECORD_CHANGED:" + ref.Key())
		}
	}
	apps := rawIndex(current.Applications)
	if len(apps) != len(current.Applications) || len(apps) != len(phase.Applications) {
		fail("APPLICATION_INVENTORY_MISMATCH")
	}
	prevApps := map[string]observation.Object{}
	if previous != nil {
		prevApps = rawIndex(previous.Applications)
	}
	for _, expect := range phase.Applications {
		ref := AppRef(expect.Name)
		app := apps[ref.Key()]
		if observation.Digest(app) != observation.Digest(raw[ref.Key()]) {
			fail("RAW_APPLICATION_INVENTORY_DISAGREES:" + expect.Name)
		}
		if app == nil {
			fail("APPLICATION_MISSING:" + expect.Name)
			continue
		}
		// UID persists until that exact owner has been absent in a verified stage.
		if old := prevApps[ref.Key()]; old != nil {
			expect.UID = observation.String(observation.At(old, "metadata", "uid"))
		}
		fact := observation.ClassifyApplication(expect, app, nil)
		active := phase.Stage.ActiveOwner != nil && *phase.Stage.ActiveOwner == expect.Name
		if active && phase.Stage.Outcome == "blocked" {
			for _, reason := range fact.Reasons {
				if reason != "SHARED_RESOURCE" && reason != "LAST_OPERATION_FAILED" && reason != "OUT_OF_SYNC" && reason != "HEALTH_NOT_READY" && reason != "BLOCKING_RESOURCES" {
					fail("UNEXPECTED_REFUSAL_STATE:" + reason)
				}
			}
			// Ordinary Observation still says DRIFTED/DEGRADED; only this bounded
			// ceremony interprets the exact expected refusal as a successful test.
			if fact.UID == "" || fact.ResourceVersion == "" || fact.ObservedSpecSHA256 != fact.ExpectedSpecSHA256 || expect.UID != "" && fact.UID != expect.UID || len(observation.Slice(observation.At(app, "metadata", "finalizers"))) != 0 || len(observation.Slice(observation.At(app, "metadata", "ownerReferences"))) != 0 || observation.At(app, "metadata", "deletionTimestamp") != nil {
				fail("REFUSAL_APPLICATION_BOUNDARY:" + expect.Name)
			}
		} else if fact.Classification != observation.Verified {
			fail("APPLICATION_NOT_VERIFIED:" + expect.Name)
		}
		if destination(expect.Name) != "" {
			if e := foundationInventory(plan.Scope, expect.Name, app); e != nil {
				fail("FOUNDATION_INVENTORY_MISMATCH:" + expect.Name)
			}
		}
		if active {
			if e := operationMatches(app, phase, phase.Stage.Name, phase.Stage.Outcome); e != nil {
				fail("OPERATION_FENCE:" + expect.Name)
			}
		}
		if phase.Stage.AtlasGate && destination(expect.Name) != "" && observation.String(observation.At(app, "metadata", "annotations", observation.TrackingAnnotation)) != observation.Tracking("platform-control", "argocd", ref) {
			fail("PARENT_NOT_REATTACHED:" + expect.Name)
		}
		if baseline != nil && destination(expect.Name) == "" {
			if b := rawIndex(baseline.Applications)[ref.Key()]; b == nil || observation.String(observation.At(app, "metadata", "uid")) != observation.String(observation.At(b, "metadata", "uid")) {
				fail("PERSISTENT_APPLICATION_UID_CHANGED:" + expect.Name)
			}
		}
	}
	if e := checkProjects(current.Projects, baseline, desired); e != nil {
		fail(e.Error())
	}
	if phase.Stage.AtlasGate {
		out.Atlas = "NOT_PROVEN"
		if e := checkNodes(current.Nodes, baseline, plan.Target.Cluster); e != nil {
			out.Reasons = append(out.Reasons, e.Error())
		}
		if gate != nil && gate.Validate(plan, phase, current) == nil && checkNodes(current.Nodes, baseline, plan.Target.Cluster) == nil && out.Ownership == "VERIFIED" {
			out.Atlas = "VERIFIED"
		}
	}
	return out
}
func checkProjects(projects []observation.Object, baseline *Snapshot, desired map[string]observation.Object) error {
	expected := map[string]bool{"atlas-bootstrap": true, "platform-project": true, "workload-project": true, "default": true}
	seen := map[string]bool{}
	for _, p := range projects {
		ref := observation.Reference(p)
		if ref.Kind != "AppProject" || ref.Namespace != "argocd" || !expected[ref.Name] || seen[ref.Name] || observation.String(observation.At(p, "metadata", "uid")) == "" || observation.At(p, "metadata", "deletionTimestamp") != nil {
			return errors.New("PROJECT_INVENTORY_OR_IDENTITY")
		}
		seen[ref.Name] = true
		if baseline == nil {
			want := desired[ref.Key()]
			if want == nil || observation.Digest(want["spec"]) != observation.Digest(p["spec"]) {
				return errors.New("BASELINE_PROJECT_PERMISSIONS_NOT_GIT_DEFINED")
			}
		}
	}
	if len(seen) != 4 {
		return errors.New("PROJECT_INVENTORY_OR_IDENTITY")
	}
	if baseline != nil {
		base := rawIndex(baseline.Projects)
		for _, p := range projects {
			old := base[observation.Reference(p).Key()]
			if observation.String(observation.At(p, "metadata", "uid")) != observation.String(observation.At(old, "metadata", "uid")) || observation.Digest(observation.Semantic(p)) != observation.Digest(observation.Semantic(old)) {
				return errors.New("PROJECT_PERMISSION_OR_IDENTITY_DRIFT")
			}
		}
	}
	return nil
}
func checkNodes(nodes []observation.Object, baseline *Snapshot, cluster string) error {
	names := map[string]bool{cluster + "-control-plane": true, cluster + "-worker": true, cluster + "-worker2": true, cluster + "-worker3": true}
	if len(nodes) != 4 {
		return errors.New("FOUR_NODE_INVENTORY_REQUIRED")
	}
	base := map[string]observation.Object{}
	if baseline != nil {
		base = rawIndex(baseline.Nodes)
	}
	for _, node := range nodes {
		ref := observation.Reference(node)
		if ref.Kind != "Node" || !names[ref.Name] || observation.String(observation.At(node, "metadata", "uid")) == "" || observation.At(node, "metadata", "deletionTimestamp") != nil {
			return errors.New("NODE_IDENTITY_DRIFT")
		}
		delete(names, ref.Name)
		if baseline != nil && observation.String(observation.At(node, "metadata", "uid")) != observation.String(observation.At(base[ref.Key()], "metadata", "uid")) {
			return errors.New("NODE_UID_CHANGED")
		}
		ready := false
		for _, c := range observation.Slice(observation.At(node, "status", "conditions")) {
			m := observation.Map(c)
			if observation.String(m["type"]) == "Ready" {
				ready = observation.String(m["status"]) == "True"
			}
		}
		if !ready {
			return errors.New("NODE_NOT_READY")
		}
	}
	return nil
}

// GateProof is an explicit imported evidence contract. Its digest references are
// checked by the attempt verifier against create-only local evidence files.
// A caller-written boolean alone cannot produce OT-1B success.
type GateProof struct {
	validated       bool              // set only after local evidence payload verification
	Stage           string            `json:"stage"`
	PlanSHA256      string            `json:"planSHA256"`
	SnapshotSHA256  string            `json:"snapshotSHA256"`
	EngineStatus    string            `json:"engineStatus"`
	RepeatApplyExit int               `json:"repeatApplyExit"`
	DeniedWrites    int               `json:"deniedWrites"`
	AuditBefore     int               `json:"auditBefore"`
	AuditAfter      int               `json:"auditAfter"`
	RuntimeVerified bool              `json:"runtimeVerified"`
	EvidenceFiles   map[string]string `json:"evidenceFiles"`
}

func (g GateProof) Validate(plan Plan, phase Phase, s Snapshot) error {
	if !g.validated || g.Stage != phase.Stage.Name || g.PlanSHA256 != observation.Digest(plan) || g.SnapshotSHA256 != observation.Digest(s) || g.EngineStatus != "ADOPTED" || g.RepeatApplyExit != 0 || g.DeniedWrites != 0 || g.AuditBefore < 0 || g.AuditBefore != g.AuditAfter || !g.RuntimeVerified {
		return errors.New("Atlas gate is not proven")
	}
	for _, name := range []string{"status.json", "repeat-apply.json", "audit-before.json", "audit-after.json", "runtime.json"} {
		if !observation.Hash(g.EvidenceFiles[name]) {
			return fmt.Errorf("gate evidence missing: %s", name)
		}
	}
	if len(g.EvidenceFiles) != 5 {
		return errors.New("unexpected gate evidence set")
	}
	return nil
}

func foundationInventory(scope Scope, owner string, app observation.Object) error {
	want := map[string]bool{}
	for _, item := range scope.Objects {
		if owner == Source || owner == item.NewOwner {
			want[item.Identity] = true
		}
	}
	entries, ok := observation.At(app, "status", "resources").([]any)
	if !ok || len(entries) != len(want) {
		return errors.New("foundation resources unavailable or incomplete")
	}
	for _, raw := range entries {
		r := observation.Map(raw)
		version := observation.String(r["version"])
		if group := observation.String(r["group"]); group != "" {
			version = group + "/" + version
		}
		ref := observation.Ref{APIVersion: version, Kind: observation.String(r["kind"]), Namespace: observation.String(r["namespace"]), Name: observation.String(r["name"])}
		if !want[ref.Key()] {
			return errors.New("unexpected foundation inventory resource")
		}
		delete(want, ref.Key())
	}
	return nil
}

// Argo CD 3.5.1 server.initializeDefaultProject creates this internal project.
// It is inventoried and frozen, never selected by any Atlas Application.
// Source: https://github.com/argoproj/argo-cd/blob/v3.5.1/server/server.go#L276-L300
func DefaultProject() observation.Object {
	return observation.Object{"apiVersion": "argoproj.io/v1alpha1", "kind": "AppProject", "metadata": observation.Object{"name": "default", "namespace": "argocd"}, "spec": observation.Object{"sourceRepos": []any{"*"}, "destinations": []any{observation.Object{"server": "*", "namespace": "*"}}, "clusterResourceWhitelist": []any{observation.Object{"group": "*", "kind": "*"}}}}
}
