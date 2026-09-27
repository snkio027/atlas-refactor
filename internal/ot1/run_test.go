package ot1

import (
	"atlas-refactor/internal/atlas"
	"atlas-refactor/internal/observation"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runtimeFixture(s Snapshot) observation.Object {
	pods := []any{}
	for _, ns := range []string{"kube-system", "argocd", "workload-web", "atlas-secrets", "atlas-monitoring", "atlas-storage", "envoy-gateway-system"} {
		name := "test-pod"
		node := s.Envelope.Target.Cluster + "-worker2"
		if ns == "workload-web" {
			node = s.Envelope.Target.Cluster + "-worker3"
		}

		pods = append(pods, live(observation.Object{"apiVersion": "v1", "kind": "Pod", "metadata": observation.Object{"name": name, "namespace": ns}, "spec": observation.Object{"nodeName": node}, "status": observation.Object{"phase": "Running", "conditions": []any{observation.Object{"type": "Ready", "status": "True"}}}}, "pod-"+ns))
	}
	pods = append(pods, live(observation.Object{"apiVersion": "v1", "kind": "Pod", "metadata": observation.Object{"name": "envoy-atlas-gateway-proxy", "namespace": "envoy-gateway-system", "labels": observation.Object{"app.kubernetes.io/component": "proxy", "gateway.envoyproxy.io/owning-gateway-name": "development", "gateway.envoyproxy.io/owning-gateway-namespace": "atlas-gateway"}}, "spec": observation.Object{"nodeName": s.Envelope.Target.Cluster + "-worker"}, "status": observation.Object{"phase": "Running", "conditions": []any{observation.Object{"type": "Ready", "status": "True"}}}}, "gateway-proxy"))
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
	for _, tc := range []struct {
		name, user, resource, namespace string
		wantError                       bool
	}{
		{"namespace-wire-argo", "system:serviceaccount:argocd:argocd-application-controller", "namespaces", "atlas-storage", false},
		{"namespace-wire-side-channel", "admin", "namespaces", "atlas-storage", true},
		{"namespace-inconsistent", "system:serviceaccount:argocd:argocd-application-controller", "namespaces", "wrong", true},
		{"unrelated-namespaced-resource", "admin", "resourcequotas", "other-namespace", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "atlas-storage"
			if tc.resource == "resourcequotas" {
				name = "platform-budget"
			}
			writeEvents(old, event("wire", tc.user, tc.resource, name, tc.namespace, ""))
			if err := x.auditScope(); (err != nil) != tc.wantError {
				t.Fatalf("audit request identity: want error %v, got %v", tc.wantError, err)
			}
		})
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

func TestApplicationReadinessBeforeVersionFencedCapture(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshots := syntheticSnapshots(t, plan, desired)
	for i, snapshot := range snapshots {
		var previous *Snapshot
		if i > 0 {
			previous = &snapshots[i-1]
		}
		activeUID := ""
		if plan.Phases[i].Stage.ActiveOwner != nil {
			app := rawIndex(snapshot.Applications)[AppRef(*plan.Phases[i].Stage.ActiveOwner).Key()]
			activeUID = observation.String(observation.At(app, "metadata", "uid"))
		}
		if ready, e := applicationProgress(plan, i, snapshot.Applications, previous, activeUID); !ready || e != nil {
			t.Fatalf("terminal stage %s not ready: %v", snapshot.Stage, e)
		}
		if !Steps(plan)[i].Sync {
			continue
		}
		pending := cloneSnapshot(t, snapshot)
		app := rawIndex(pending.Applications)[AppRef(*plan.Phases[i].Stage.ActiveOwner).Key()]
		state := observation.Map(observation.At(app, "status", "operationState"))
		state["phase"] = "Running"
		delete(state, "finishedAt")
		app["operation"] = observation.Clone(observation.Map(state["operation"]))
		if ready, e := applicationProgress(plan, i, pending.Applications, previous, activeUID); ready || e != nil {
			t.Fatalf("running stage %s should wait without a complete capture: %v", snapshot.Stage, e)
		}
		if _, e := applicationProgress(plan, i, snapshot.Applications, previous, "recreated-owner"); e == nil {
			t.Fatal("newly created owner UID was not bound to submitted request")
		}
		observation.Map(observation.At(app, "operation", "sync"))["source"] = observation.Object{"path": "outside-plan"}
		if _, e := applicationProgress(plan, i, pending.Applications, previous, activeUID); e == nil {
			t.Fatal("override accepted while waiting")
		}
	}
	for _, test := range []struct {
		name   string
		mutate func(observation.Object)
	}{
		{"unknown health", func(o observation.Object) {
			observation.Map(observation.At(o, "status", "health"))["status"] = "Unknown"
		}},
		{"unplanned revision", func(o observation.Object) {
			observation.Map(observation.At(o, "status", "sync"))["revision"] = strings.Repeat("f", 40)
		}},
		{"recreated app", func(o observation.Object) { observation.Map(o["metadata"])["uid"] = "another" }},
		{"missing RV", func(o observation.Object) { delete(observation.Map(o["metadata"]), "resourceVersion") }},
		{"spec drift", func(o observation.Object) { observation.Map(o["spec"])["project"] = "default" }},
		{"unknown condition", func(o observation.Object) {
			observation.Map(o["status"])["conditions"] = []any{observation.Object{"type": "ComparisonError"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := cloneSnapshot(t, snapshots[1])
			test.mutate(s.Applications[0])
			if _, e := applicationProgress(plan, 1, s.Applications, &snapshots[0], ""); e == nil {
				t.Fatal("unexpected drift treated as progress")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	x := Executor{}
	if e := x.wait(ctx, func() (bool, error) { called = true; return true, nil }); e == nil || called {
		t.Fatal("canceled wait performed another read")
	}
}

type convergenceInventory struct {
	target       observation.Target
	apps         []observation.Object
	reads, lists int
}

func (r *convergenceInventory) ClusterIdentity(context.Context) (string, error) {
	return r.target.ClusterUID, nil
}
func (r *convergenceInventory) Read(context.Context, observation.Ref) (observation.Object, error) {
	r.reads++
	return nil, errors.New("complete capture must wait")
}
func (r *convergenceInventory) List(context.Context, observation.Ref) ([]observation.Object, error) {
	r.lists++
	return r.apps, nil
}

type phaseRemote struct {
	plan  Plan
	calls int
}

func (r *phaseRemote) Run(_ context.Context, q atlas.Request) ([]byte, error) {
	r.calls++
	if q.Tool != "git" || len(q.Args) == 0 || q.Args[0] != "ls-remote" {
		return nil, errors.New("unexpected runner request")
	}
	return []byte(r.plan.Phases[2].Revision + "\trefs/heads/" + r.plan.Branch + "\n"), nil
}
func TestConvergeDoesNotCaptureAcrossItsRunningOperation(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshots := syntheticSnapshots(t, plan, desired)
	pending := cloneSnapshot(t, snapshots[2])
	active := rawIndex(pending.Applications)[AppRef(*plan.Phases[2].Stage.ActiveOwner).Key()]
	state := observation.Map(observation.At(active, "status", "operationState"))
	state["phase"] = "Running"
	delete(state, "finishedAt")
	active["operation"] = observation.Clone(observation.Map(state["operation"]))
	repo := privateTemp(t)
	evidence := privateTemp(t)
	kube := []byte("synthetic binding, not an actual kubeconfig")
	plan.Target.KubeconfigSHA256 = observation.SHA(kube)
	if e := observation.CreatePrivate(filepath.Join(repo, ".state/kubeconfig"), kube); e != nil {
		t.Fatal(e)
	}
	reader := &convergenceInventory{target: plan.Target, apps: pending.Applications}
	runner := &phaseRemote{plan: plan}
	executor := Executor{Plan: plan, RuntimeRepository: repo, EvidenceDirectory: evidence, Reader: reader, Runner: runner, Desired: desired, activeUID: observation.String(observation.At(active, "metadata", "uid"))}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, e := executor.Converge(ctx, 2, &snapshots[0], &snapshots[1]); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("expected bounded read-only wait", e)
	}
	if reader.reads != 0 || reader.lists != 1 || runner.calls != 1 {
		t.Fatal("operation was resubmitted or full capture opened prematurely", reader.reads, reader.lists, runner.calls)
	}
}

func TestRunningHookRemainsNonReadyDuringConvergence(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshot := syntheticSnapshots(t, plan, desired)[0]
	app := rawIndex(snapshot.Applications)[AppRef("argocd-self").Key()]
	hook := observation.Object{"kind": "Job", "name": "init", "hookType": "PreSync", "hookPhase": "Running"}
	observation.Map(app["status"])["operationState"] = observation.Object{"phase": "Running", "syncResult": observation.Object{"resources": []any{hook}}}
	if ready, err := applicationProgress(plan, 0, snapshot.Applications, nil, ""); ready || err != nil {
		t.Fatalf("running hook must wait, not pass or escape the bounded wait: ready=%v err=%v", ready, err)
	}
	hook["hookPhase"] = "Failed"
	if _, err := applicationProgress(plan, 0, snapshot.Applications, nil, ""); err == nil {
		t.Fatal("failed hook treated as expected convergence")
	}
}

func TestRuntimeGatewayUsesControllerNamespace(t *testing.T) {
	plan, desired := syntheticPlan(t)
	snapshot := syntheticSnapshots(t, plan, desired)[0]
	valid := runtimeFixture(snapshot)
	if err := runtimeEvidence(valid, plan.Target.Cluster); err != nil {
		t.Fatal("locked Envoy deployment layout rejected:", err)
	}
	for _, tc := range []struct {
		name   string
		change func(observation.Object)
	}{
		{"wrong-namespace", func(p observation.Object) { observation.Map(p["metadata"])["namespace"] = "atlas-gateway" }},
		{"wrong-node", func(p observation.Object) { observation.Map(p["spec"])["nodeName"] = plan.Target.Cluster + "-worker2" }},
		{"wrong-gateway", func(p observation.Object) {
			observation.Map(observation.At(p, "metadata", "labels"))["gateway.envoyproxy.io/owning-gateway-name"] = "other"
		}},
		{"wrong-gateway-namespace", func(p observation.Object) {
			observation.Map(observation.At(p, "metadata", "labels"))["gateway.envoyproxy.io/owning-gateway-namespace"] = "other"
		}},
		{"wrong-component", func(p observation.Object) {
			observation.Map(observation.At(p, "metadata", "labels"))["app.kubernetes.io/component"] = "controller"
		}},
		{"not-ready", func(p observation.Object) { observation.Map(p["status"])["conditions"] = []any{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := runtimeFixture(snapshot)
			for _, value := range observation.Slice(data["pods"]) {
				pod := observation.Map(value)
				if observation.String(observation.At(pod, "metadata", "name")) == "envoy-atlas-gateway-proxy" {
					tc.change(pod)
				}
			}
			if runtimeEvidence(data, plan.Target.Cluster) == nil {
				t.Fatal("invalid gateway runtime passed")
			}
		})
	}
	data := runtimeFixture(snapshot)
	data["pods"] = observation.Slice(data["pods"])[:len(observation.Slice(data["pods"]))-1]
	if runtimeEvidence(data, plan.Target.Cluster) == nil {
		t.Fatal("controller alone substituted for missing data plane")
	}
}
