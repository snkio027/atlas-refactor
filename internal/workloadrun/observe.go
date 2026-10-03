package workloadrun

import (
	"atlas-refactor/internal/atlas"
	"atlas-refactor/internal/installation"
	"atlas-refactor/internal/observation"
	"atlas-refactor/internal/platform"
	"atlas-refactor/internal/workload"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type Pending string

func (e Pending) Error() string { return string(e) }

type Observation struct {
	Schema       int               `json:"schema"`
	ClusterUID   string            `json:"clusterUID"`
	Revision     string            `json:"revision"`
	Project      string            `json:"project"`
	Workload     string            `json:"workload"`
	Binding      string            `json:"binding"`
	Runtime      string            `json:"runtime"`
	Applications map[string]string `json:"applications"`
	Resources    map[string]string `json:"resources"`
	UID          map[string]string `json:"uid"`
}

func desiredObjects(files Files, inventory []workload.OwnedResource) (map[string]Object, error) {
	result := map[string]Object{}
	seen := map[string]bool{}
	for _, r := range inventory {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		xs, e := platform.DecodeJSONManifests(files[r.Path])
		if e != nil {
			return nil, e
		}
		for _, o := range xs {
			ref := observation.Reference(o)
			g, _, ok := strings.Cut(ref.APIVersion, "/")
			if !ok {
				g = ""
			}
			result[g+"/"+ref.Kind+"/"+ref.Namespace+"/"+ref.Name] = o
		}
	}
	return result, nil
}
func (w *Workflow) authority(ctx context.Context) error {
	i := w.Install
	f, e := i.Product.Project(i.Config, false)
	if e != nil {
		return e
	}
	root := filepath.Join(i.Config.StateDirectory, "runtime")
	// NewInstallation + Status only read existing frozen artifacts/state. They
	// never call D1's asset preparation or change its original completion record.
	a, e := atlas.NewInstallation(root, atlas.Config{Schema: 4, Cluster: i.Config.Cluster, RepositoryURL: i.Config.Repository, Revision: i.Config.Branch, GitOpsPath: installation.RootPath, DockerContext: "orbstack", TimeoutSeconds: 1800}, i.Product.Lock, atlas.ExecRunner{Root: root, ToolDir: installation.ToolDirectory(i.Config.StateDirectory, i.Product.Tools), DockerContext: "orbstack"}, atlas.InstallationBinding{HTTPPort: i.Config.HTTPPort, HTTPSPort: i.Config.HTTPSPort, ProductSHA256: i.ProductDigest, BinarySHA256: i.BinaryDigest, InstallID: i.Record.InstallID, DeploymentCommit: i.Record.BaseCommit, Bundle: f, Images: i.Product.Images})
	if e != nil {
		return e
	}
	r := a.Status(ctx)
	if r.State != atlas.Adopted {
		return fmt.Errorf("Bootstrap authority %s: %s", r.State, r.Detail)
	}
	return a.VerifyNodes(ctx)
}

// subset permits API-defaulted fields, but never extra list members. Network
// policy and RBAC use exact semantic comparison below, since extra permissions
// there cannot be treated as harmless defaults.
func subset(want, got any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range w {
			if !subset(v, g[k]) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !subset(w[i], g[i]) {
				return false
			}
		}
		return true
	default:
		return bytes.Equal(workload.JSON(want), workload.JSON(got))
	}
}
func normalizedSelector(v any) any { // only LabelSelector's equivalent empty map
	switch x := v.(type) {
	case map[string]any:
		out := Object{}
		for k, v := range x {
			if k == "matchLabels" && len(mapping(v)) == 0 {
				continue
			}
			out[k] = normalizedSelector(v)
		}
		return out
	case []any:
		out := []any{}
		for _, v := range x {
			out = append(out, normalizedSelector(v))
		}
		return out
	}
	return v
}

// comparisonObject canonicalizes only known Kubernetes zero-value omissions.
// Never drop an authored false/zero: filling the omitted default on both sides
// still detects a live true/nonzero value. Other missing fields remain failures.
func comparisonObject(o Object) Object {
	var r Object
	_ = json.Unmarshal(workload.JSON(observation.Semantic(o)), &r)
	fill := func(m Object, key string, value any) {
		if m != nil {
			if _, present := m[key]; !present {
				m[key] = value
			}
		}
	}
	switch str(r["apiVersion"]) + "/" + str(r["kind"]) {
	case "v1/Service":
		fill(mapping(r["spec"]), "publishNotReadyAddresses", false)
	case "apps/v1/Deployment", "apps/v1/DaemonSet", "apps/v1/StatefulSet":
		pod := mapping(at(r, "spec", "template", "spec"))
		fill(pod, "hostIPC", false)
		fill(pod, "hostNetwork", false)
		for _, field := range []string{"containers", "initContainers"} {
			for _, c := range array(pod[field]) {
				for _, probe := range []string{"livenessProbe", "readinessProbe", "startupProbe"} {
					fill(mapping(mapping(c)[probe]), "initialDelaySeconds", float64(0))
				}
			}
		}
	case "rbac.authorization.k8s.io/v1/RoleBinding", "rbac.authorization.k8s.io/v1/ClusterRoleBinding":
		for _, v := range array(r["subjects"]) {
			s := mapping(v)
			if s["kind"] == "ServiceAccount" {
				fill(s, "apiGroup", "")
			}
		}
	}
	if r["kind"] == "NetworkPolicy" {
		r["spec"] = normalizedSelector(r["spec"])
	}
	return r
}
func matches(want, live Object) bool {
	desired, observed := comparisonObject(want), comparisonObject(live)
	if !subset(desired, observed) {
		return false
	}
	switch want["kind"] {
	case "NetworkPolicy":
		return bytes.Equal(workload.JSON(desired["spec"]), workload.JSON(observed["spec"]))
	case "Role", "ClusterRole":
		return bytes.Equal(workload.JSON(desired["rules"]), workload.JSON(observed["rules"]))
	case "RoleBinding", "ClusterRoleBinding":
		return bytes.Equal(workload.JSON(desired["subjects"]), workload.JSON(observed["subjects"])) && bytes.Equal(workload.JSON(desired["roleRef"]), workload.JSON(observed["roleRef"]))
	}
	return true
}
func (w *Workflow) Observe(ctx context.Context, result workload.Result, revision string) (Observation, error) {
	report := Observation{Schema: 1, ClusterUID: w.Install.Record.ClusterUID, Revision: revision, Project: "UNKNOWN", Workload: "UNKNOWN", Binding: "UNKNOWN", Runtime: "UNPROVEN", Applications: map[string]string{}, Resources: map[string]string{}, UID: map[string]string{}}
	if !fullSHA.MatchString(revision) {
		return report, errors.New("exact observation revision required")
	}
	current, e := w.remote(ctx)
	if e != nil || current != revision {
		return report, errors.New("remote revision differs")
	}
	if e = w.bind(ctx); e != nil {
		return report, e
	}
	if e = w.authority(ctx); e != nil {
		return report, e
	}
	gitFiles, e := w.ReadTree(ctx, revision)
	if e != nil {
		return report, e
	}
	for name, b := range result.Files {
		if !bytes.Equal(gitFiles[name], b) {
			return report, fmt.Errorf("published output differs: %s", name)
		}
	}
	desired, e := desiredObjects(result.Files, result.Inventory.Resources)
	if e != nil {
		return report, e
	}
	apps := map[string]Object{}
	for _, o := range desired {
		if o["kind"] == "Application" {
			apps[str(at(o, "metadata", "name"))] = o
		}
	}
	raw, e := w.get(ctx, "applications.argoproj.io", "argocd", "")
	if e != nil {
		return report, e
	}
	liveApps := map[string]Object{}
	for _, v := range array(raw["items"]) {
		o := mapping(v)
		if at(o, "metadata", "name") == "atlas-refactor-root" {
			continue
		}
		liveApps[str(at(o, "metadata", "name"))] = o
	}
	if len(liveApps) != len(apps) {
		return report, Pending("Application inventory has not converged")
	}
	// Read-only equivalence is confined to the immutable D1 base and current
	// approved publication parents. It never changes a mutation fence.
	accepted := map[string]bool{revision: true, w.Install.Record.FullCommit: true}
	if p, e := w.ReadPlan(); e == nil {
		accepted[p.Parent] = true
		for _, phase := range p.Phases {
			if b, e := regular(w.publicationPath(p, phase), true); e == nil {
				var r Publication
				if workload.StrictDecode(b, &r) == nil && r.PlanSHA256 == workload.Digest(workload.JSON(p)) {
					accepted[r.Commit] = true
				}
			}
		}
	}
	first := map[string]Object{}
	for name, want := range apps {
		live := liveApps[name]
		if live == nil {
			return report, Pending("Application missing: " + name)
		}
		observed := str(at(live, "status", "sync", "revision"))
		expected := revision
		if observed != revision && accepted[observed] && name != "platform-control" && name != "project-bootstrap" && name != "workload-control" && sourceEqual(ctx, w, want, gitFiles, observed) {
			expected = observed
		}
		fact := observation.ClassifyApplication(observation.ExpectedApplication{Name: name, Spec: mapping(want["spec"]), Revision: expected}, live, nil)
		if fact.Classification != observation.Verified {
			if fact.Classification == observation.Progressing {
				return report, Pending("Application progressing: " + name)
			}
			return report, fmt.Errorf("Application %s: %s %v", name, fact.Classification, fact.Reasons)
		}
		key := "argoproj.io/Application/argocd/" + name
		report.Applications[name] = observed
		report.UID[key] = str(at(live, "metadata", "uid"))
		first[key] = live
	}
	baseline, e := w.baselineUIDs()
	if e != nil {
		return report, e
	}
	for _, r := range result.Inventory.Resources {
		if result.Inventory.Files[r.Path] == "" || desired[r.Identity]["kind"] == "Application" {
			continue
		}
		want := desired[r.Identity]
		ref := observation.Reference(want)
		// Hook resources are transient and have no durable identity to preserve.
		if str(at(want, "metadata", "annotations", "helm.sh/hook")) != "" || str(at(want, "metadata", "annotations", "argocd.argoproj.io/hook")) != "" {
			continue
		}
		live, e := w.get(ctx, resourceArgument(ref), ref.Namespace, ref.Name)
		if e != nil {
			return report, e
		}
		if baseline[r.Identity] != "" && baseline[r.Identity] != str(at(live, "metadata", "uid")) {
			return report, errors.New("existing resource UID changed: " + r.Identity)
		}
		if !matches(want, live) {
			return report, Pending("resource content has not converged: " + r.Identity)
		}
		dest := str(at(apps[r.Owner], "spec", "destination", "namespace"))
		rule := "identity-content"
		if ref.Kind == "Namespace" {
			rule = "namespace-active"
		}
		if ref.Kind == "Deployment" {
			rule = "deployment-available"
		}
		fact := observation.ClassifyResource(observation.ExpectedResource{Ref: ref, Tracking: observation.Tracking(r.Owner, dest, ref), RequireSSA: true, Readiness: rule}, live, nil)
		if fact.Classification != observation.Verified {
			if fact.Classification == observation.Progressing {
				return report, Pending("resource not Ready: " + r.Identity)
			}
			return report, fmt.Errorf("resource %s: %s %v", r.Identity, fact.Classification, fact.Reasons)
		}
		if ref.Kind == "SealedSecret" {
			ok := false
			for _, c := range array(at(live, "status", "conditions")) {
				ok = ok || at(mapping(c), "type") == "Synced" && at(mapping(c), "status") == "True"
			}
			if !ok {
				return report, Pending("SealedSecret has not materialized")
			}
		}
		if ref.Kind == "Certificate" {
			ok := false
			for _, c := range array(at(live, "status", "conditions")) {
				ok = ok || at(mapping(c), "type") == "Ready" && at(mapping(c), "status") == "True"
			}
			if !ok {
				return report, Pending("TLS certificate not Ready")
			}
		}
		report.UID[r.Identity] = fact.UID
		report.Resources[r.Identity] = workload.Digest(workload.JSON(observation.Semantic(live)))
		first[r.Identity] = live
	}
	// A closing proof rejects concurrent identity/content/operation changes.
	for id, old := range first {
		ref := observation.Reference(old)
		live, e := w.get(ctx, resourceArgument(ref), ref.Namespace, ref.Name)
		if e != nil {
			return report, e
		}
		if !observation.SameProof(old, live, "identity-content") {
			return report, Pending("observation changed during capture: " + id)
		}
	}
	if e = w.bind(ctx); e != nil {
		return report, e
	}
	current, e = w.remote(ctx)
	if e != nil || current != revision {
		return report, errors.New("Git moved during observation")
	}
	report.Project = "VERIFIED"
	if result.Inventory.Phase == "consumer" {
		report.Workload = "VERIFIED"
		report.Binding = "VERIFIED"
	}
	return report, nil
}
func resourceArgument(ref observation.Ref) string {
	group, _, ok := strings.Cut(ref.APIVersion, "/")
	if ok {
		return ref.Kind + "." + group
	}
	return ref.Kind
}
func sourceEqual(ctx context.Context, w *Workflow, app Object, current Files, old string) bool {
	files, e := w.ReadTree(ctx, old)
	if e != nil {
		return false
	}
	dir := str(at(app, "spec", "source", "path")) + "/"
	count := 0
	for name, b := range current {
		if strings.HasPrefix(name, dir) {
			count++
			if !bytes.Equal(b, files[name]) {
				return false
			}
		}
	}
	for name := range files {
		if strings.HasPrefix(name, dir) && current[name] == nil {
			return false
		}
	}
	return count > 0
}
func (w *Workflow) baselineUIDs() (map[string]string, error) {
	p, e := w.ReadPlan()
	if e != nil {
		return nil, e
	}
	b, e := regular(filepath.Join(w.Config.StateDirectory, "authority", workload.Digest(workload.JSON(p)), "baseline-uids.json"), true)
	if e != nil {
		return nil, e
	}
	m := map[string]string{}
	e = workload.StrictDecode(b, &m)
	return m, e
}
func (w *Workflow) CaptureBaseline(ctx context.Context, p Plan, approval string) error {
	if e := w.approve(p, approval); e != nil {
		return e
	}
	ids, e := w.readBaseline(ctx, p)
	if e != nil {
		return e
	}
	return save(filepath.Join(w.Config.StateDirectory, "authority", approval, "baseline-uids.json"), workload.JSON(ids), true)
}

// readBaseline is shared by the execution gate and opt-in read-only integration
// test. It neither writes authority evidence nor invokes any mutation adapter.
func (w *Workflow) readBaseline(ctx context.Context, p Plan) (map[string]string, error) {
	if e := w.bind(ctx); e != nil {
		return nil, e
	}
	if e := w.authority(ctx); e != nil {
		return nil, e
	}
	current, e := w.remote(ctx)
	if e != nil || current != p.Parent {
		return nil, errors.New("baseline Git differs")
	}
	r, e := w.Compile("infrastructure")
	if e != nil {
		return nil, e
	}
	beforeFiles, e := w.ReadTree(ctx, p.Parent)
	if e != nil {
		return nil, e
	}
	baseInv, e := workload.InventoryOf(beforeFiles, w.Context.ResourceModel)
	if e != nil {
		return nil, e
	}
	desired, e := desiredObjects(beforeFiles, baseInv)
	if e != nil {
		return nil, e
	}
	ids := map[string]string{}
	for _, v := range baseInv {
		if r.Inventory.Files[v.Path] == "" && v.Path != workload.StoragePath && v.Path != workload.CredentialsPath {
			continue
		}
		o := desired[v.Identity]
		if str(at(o, "metadata", "annotations", "helm.sh/hook")) != "" || str(at(o, "metadata", "annotations", "argocd.argoproj.io/hook")) != "" {
			continue
		}
		ref := observation.Reference(o)
		live, e := w.get(ctx, resourceArgument(ref), ref.Namespace, ref.Name)
		if e != nil {
			return nil, e
		}
		if !matches(o, live) {
			return nil, fmt.Errorf("baseline content drift: %s", v.Identity)
		}
		owner := desired["argoproj.io/Application/argocd/"+v.Owner]
		if owner == nil {
			return nil, fmt.Errorf("baseline owner missing: %s", v.Identity)
		}
		dest := str(at(owner, "spec", "destination", "namespace"))
		fact := observation.ClassifyResource(observation.ExpectedResource{Ref: ref, Tracking: observation.Tracking(v.Owner, dest, ref), RequireSSA: true, Readiness: "identity-content"}, live, nil)
		if fact.Classification != observation.Verified {
			return nil, fmt.Errorf("baseline ownership %s: %s %v", v.Identity, fact.Classification, fact.Reasons)
		}
		uid := str(at(live, "metadata", "uid"))
		if uid == "" {
			return nil, errors.New("missing baseline UID")
		}
		ids[v.Identity] = uid
	}
	// A previously occupied namespace must never be adopted merely because it
	// was absent from the immutable D1 Git inventory.
	b, e := w.kube(ctx, "get", "namespace", p.Project, "--ignore-not-found", "-o", "json")
	if e != nil {
		return nil, e
	}
	if len(bytes.TrimSpace(b)) != 0 && p.Parent == p.BaseCommit {
		return nil, errors.New("Project namespace already exists live")
	}
	return ids, nil
}
