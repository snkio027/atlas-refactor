package ot1

import (
	"atlas-refactor/internal/observation"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runtimeFixture(s Snapshot) observation.Object {
	pods := []any{}
	for _, ns := range []string{"kube-system", "argocd", "workload-web", "atlas-secrets", "atlas-monitoring", "atlas-storage", "envoy-gateway-system", "atlas-gateway"} {
		name := "test-pod"
		node := s.Envelope.Target.Cluster + "-worker2"
		if ns == "workload-web" {
			node = s.Envelope.Target.Cluster + "-worker3"
		}
		if ns == "atlas-gateway" {
			node = s.Envelope.Target.Cluster + "-worker"
			name = "envoy-atlas-gateway-proxy"
		}
		pods = append(pods, live(observation.Object{"apiVersion": "v1", "kind": "Pod", "metadata": observation.Object{"name": name, "namespace": ns}, "spec": observation.Object{"nodeName": node}, "status": observation.Object{"phase": "Running", "conditions": []any{observation.Object{"type": "Ready", "status": "True"}}}}, "pod-"+ns))
	}
	return observation.Object{"nodes": asAny(s.Nodes), "pods": pods, "pvc": live(observation.Object{"apiVersion": "v1", "kind": "PersistentVolumeClaim", "metadata": observation.Object{"name": "web-data", "namespace": "workload-web"}, "status": observation.Object{"phase": "Bound"}, "spec": observation.Object{"volumeName": "test-pv"}}, "pvc"), "pv": live(observation.Object{"apiVersion": "v1", "kind": "PersistentVolume", "metadata": observation.Object{"name": "test-pv"}, "spec": observation.Object{"persistentVolumeReclaimPolicy": "Retain"}}, "pv"), "http": observation.Object{"status": 301, "location": "https://web.atlas.test:18443/"}, "https": observation.Object{"status": 200, "tlsVerified": true, "caSHA256": strings.Repeat("a", 64), "bodyPrefix": "Atlas development web:"}}
}
func writeGateFixture(t *testing.T, dir, stem string, plan Plan, phase Phase, s Snapshot) *GateProof {
	t.Helper()
	values := map[string]observation.Object{"status.json": {"state": "ADOPTED"}, "repeat-apply.json": {"exitCode": 0, "deniedWrites": 0, "beforeIdentitySHA256": IdentityDigest(s), "afterIdentitySHA256": IdentityDigest(s), "requests": []any{observation.Object{"tool": "git", "args": []any{"status", "--porcelain"}, "exitCode": 0}}}, "audit-before.json": {"complete": true, "rawSHA256": strings.Repeat("b", 64), "kubectlMutationAuditIDs": []any{"synthetic-id"}}, "audit-after.json": {"complete": true, "rawSHA256": strings.Repeat("b", 64), "kubectlMutationAuditIDs": []any{"synthetic-id"}}, "runtime.json": runtimeFixture(s)}
	for name, data := range values {
		artifact := GateArtifact{Schema: 1, Kind: name, PlanSHA256: observation.Digest(plan), Target: plan.Target, Implementation: plan.Implementation, Revision: phase.Revision, SnapshotSHA256: observation.Digest(s), Data: data}
		if e := observation.CreatePrivate(filepath.Join(dir, stem+"-"+name), observation.Bytes(artifact)); e != nil {
			t.Fatal(e)
		}
	}
	gate, e := LoadGateEvidence(dir, stem, plan, phase, s)
	if e != nil {
		t.Fatal(e)
	}
	return gate
}

type simulatedDriver struct {
	t         *testing.T
	plan      Plan
	snapshots []Snapshot
	calls     []int
	failAt    int
}

func (d *simulatedDriver) Transition(ctx context.Context, i int, _ *Snapshot) error {
	d.calls = append(d.calls, i)
	if i == d.failAt {
		return errors.New("injected request failure; do not retry")
	}
	return ctx.Err()
}
func (d *simulatedDriver) Converge(_ context.Context, i int, _, _ *Snapshot) (Snapshot, error) {
	return d.snapshots[i], nil
}
func (d *simulatedDriver) AtlasGate(_ context.Context, i int, s Snapshot, _ *Snapshot, dir string) (Snapshot, *GateProof, error) {
	gate := writeGateFixture(d.t, dir, stageStem(i, d.plan.Phases[i].Stage.Name), d.plan, d.plan.Phases[i], s)
	return s, gate, nil
}
func privateTemp(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(p, 0700); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestBoundedRunAndStopWithoutFurtherRequests(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshots := syntheticSnapshots(t, plan, desired)
	for _, failure := range []int{-1, 3, 20} {
		dir := filepath.Join(privateTemp(t), "attempt")
		driver := &simulatedDriver{t: t, plan: plan, snapshots: snapshots, failAt: failure}
		e := Run(context.Background(), plan, observation.Digest(plan), dir, desired, driver)
		if failure < 0 && e != nil || failure >= 0 && e == nil {
			t.Fatal("wrong run outcome", failure, e)
		}
		expected := 29
		if failure >= 0 {
			expected = failure + 1
		}
		if len(driver.calls) != expected {
			t.Fatal("requests continued/retried after stop", driver.calls)
		}
		b, e := os.ReadFile(filepath.Join(dir, "terminal.json"))
		if e != nil {
			t.Fatal(e)
		}
		want := "VERIFIED"
		if failure >= 0 {
			want = "STOP"
		}
		var terminal observation.Object
		if e = observation.Decode(b, &terminal, false); e != nil || terminal["state"] != want {
			t.Fatal(string(b), e)
		}
		// Even a successful later driver cannot reopen and rewrite an old attempt.
		driver.failAt = -1
		if e = Run(context.Background(), plan, observation.Digest(plan), dir, desired, driver); e == nil {
			t.Fatal("attempt reused")
		}
	}
	driver := &simulatedDriver{t: t, plan: plan, snapshots: snapshots, failAt: -1}
	if e := Run(context.Background(), plan, "wrong", filepath.Join(privateTemp(t), "denied"), desired, driver); e == nil || len(driver.calls) != 0 {
		t.Fatal("unapproved plan executed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := Run(ctx, plan, observation.Digest(plan), filepath.Join(privateTemp(t), "interrupted"), desired, driver); e == nil || len(driver.calls) != 0 {
		t.Fatal("interrupted run issued request")
	}
}
func TestGateEvidenceRejectsPartialAndChangedArtifacts(t *testing.T) {
	plan, desired := syntheticPlan(t)
	s := syntheticSnapshots(t, plan, desired)[0]
	phase := plan.Phases[0]
	for _, test := range []struct {
		name, file string
		mutate     func(*GateArtifact)
	}{
		{"wrong target", "status.json", func(a *GateArtifact) { a.Target.ClusterUID = "another" }},
		{"degraded", "status.json", func(a *GateArtifact) { a.Data["state"] = "ADOPTED_DEGRADED" }},
		{"denied repeat write", "repeat-apply.json", func(a *GateArtifact) { a.Data["deniedWrites"] = 1 }},
		{"runner mutation", "repeat-apply.json", func(a *GateArtifact) {
			a.Data["requests"] = []any{observation.Object{"tool": "kubectl", "args": []any{"delete", "namespace", "atlas-storage"}, "exitCode": 0}}
		}},
		{"audit write", "audit-after.json", func(a *GateArtifact) { a.Data["kubectlMutationAuditIDs"] = []any{"synthetic-id", "another"} }},
		{"audit truncation", "audit-after.json", func(a *GateArtifact) { a.Data["kubectlMutationAuditIDs"] = []any{} }},
		{"TLS verification bypassed", "runtime.json", func(a *GateArtifact) { observation.Map(a.Data["https"])["tlsVerified"] = false }},
		{"wrong gateway port", "runtime.json", func(a *GateArtifact) { observation.Map(a.Data["http"])["location"] = "https://web.atlas.test:8443/" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := privateTemp(t)
			stem := stageStem(0, phase.Stage.Name)
			writeGateFixture(t, dir, stem, plan, phase, s)
			path := filepath.Join(dir, stem+"-"+test.file)
			b, _ := os.ReadFile(path)
			var a GateArtifact
			if e := observation.Decode(b, &a, true); e != nil {
				t.Fatal(e)
			}
			test.mutate(&a)
			if e := os.WriteFile(path, observation.Bytes(a), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e := LoadGateEvidence(dir, stem, plan, phase, s); e == nil {
				t.Fatal("invalid gate passed")
			}
		})
	}
}
func TestAuditRejectsSideChannelWritesAndLostHistory(t *testing.T) {
	plan, _ := syntheticPlan(t)
	repo := privateTemp(t)
	if e := os.MkdirAll(filepath.Join(repo, ".state/audit"), 0700); e != nil {
		t.Fatal(e)
	}
	event := func(id, user, resource, name, ns, sub string) observation.Object {
		return observation.Object{"auditID": id, "level": "Metadata", "stage": "ResponseComplete", "verb": "patch", "userAgent": "test-client", "user": observation.Object{"username": user}, "objectRef": observation.Object{"apiVersion": "v1", "resource": resource, "name": name, "namespace": ns, "subresource": sub}}
	}
	old := event("baseline", "admin", "applications", "ignored", "argocd", "")
	writeEvents := func(events ...observation.Object) {
		b := []byte{}
		for _, e := range events {
			compact := strings.Join(strings.Fields(string(observation.Bytes(e))), " ")
			b = append(b, []byte(compact+"\n")...)
		}
		if e := os.WriteFile(filepath.Join(repo, ".state/audit/events.jsonl"), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	writeEvents(old)
	x := Executor{Plan: plan, RuntimeRepository: repo, baselineAudit: map[string]bool{"baseline": true}}
	argo := event("argo", "system:serviceaccount:argocd:argocd-application-controller", "namespaces", "atlas-storage", "", "")
	quota := event("quota", "system:serviceaccount:kube-system:resourcequota-controller", "resourcequotas", "platform-budget", "atlas-storage", "status")
	writeEvents(old, argo, quota)
	if e := x.auditScope(); e != nil {
		t.Fatal(e)
	}
	illegal := event("illegal", "admin", "namespaces", "atlas-storage", "", "")
	writeEvents(old, illegal)
	if e := x.auditScope(); e == nil {
		t.Fatal("direct write accepted")
	}
	writeEvents(argo)
	if e := x.auditScope(); e == nil {
		t.Fatal("lost audit history accepted")
	}
}

type fixedInventory struct{ object observation.Object }

func (f fixedInventory) Read(context.Context, observation.Ref) (observation.Object, error) {
	return observation.Clone(f.object), nil
}
func (f fixedInventory) ClusterIdentity(context.Context) (string, error) { return "fixture", nil }
func (f fixedInventory) List(context.Context, observation.Ref) ([]observation.Object, error) {
	return nil, nil
}
func TestParentPruneConfirmationMayRemainRunningWithoutGeneralPruneApproval(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshot := syntheticSnapshots(t, plan, desired)[0]
	parent := observation.Clone(rawIndex(snapshot.Applications)[AppRef("platform-control").Key()])
	status := observation.Map(parent["status"])
	status["operationState"] = observation.Object{"phase": "Running", "operation": observation.Object{"sync": observation.Object{"revision": plan.Phases[1].Revision, "prune": true}}}
	x := Executor{Plan: plan, Reader: fixedInventory{parent}}
	uid := observation.String(observation.At(parent, "metadata", "uid"))
	if _, e := x.compared(context.Background(), 1, "platform-control", parent, uid, true); e != nil {
		t.Fatal("valid pending parent prune should permit guarded child release", e)
	}
	bad := observation.Clone(parent)
	observation.Map(observation.At(bad, "status", "operationState", "operation", "sync"))["revision"] = strings.Repeat("e", 40)
	x.Reader = fixedInventory{bad}
	if _, e := x.compared(context.Background(), 1, "platform-control", parent, uid, true); e == nil {
		t.Fatal("unplanned parent sync allowed")
	}
	bad = observation.Clone(parent)
	observation.Map(bad["status"])["conditions"] = []any{observation.Object{"type": "SharedResourceWarning"}}
	x.Reader = fixedInventory{bad}
	if _, e := x.compared(context.Background(), 1, "platform-control", parent, uid, true); e == nil {
		t.Fatal("parent ownership conflict ignored")
	}
}
