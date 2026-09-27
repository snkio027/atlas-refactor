package observation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appFixture() (ExpectedApplication, Object) {
	spec := Object{"project": "platform-project", "source": Object{"repoURL": "https://github.com/snkio027/atlas-refactor.git", "targetRevision": "codex/ot1-desired-state", "path": "gitops/platform/foundation/secrets/overlays/development"}, "destination": Object{"server": "https://kubernetes.default.svc", "namespace": "atlas-secrets"}, "syncPolicy": Object{"syncOptions": []any{"ServerSideApply=true", "FailOnSharedResource=true"}}}
	want := ExpectedApplication{Name: "secrets-foundation", Spec: spec, Revision: strings.Repeat("a", 40), UID: "app-uid"}
	got := Object{"apiVersion": "argoproj.io/v1alpha1", "kind": "Application", "metadata": Object{"name": want.Name, "namespace": "argocd", "uid": "app-uid", "resourceVersion": "10", "generation": 2}, "spec": spec, "status": Object{"sync": Object{"status": "Synced", "revision": want.Revision}, "health": Object{"status": "Healthy"}, "operationState": Object{"phase": "Succeeded", "finishedAt": "2026-09-27T00:00:00Z", "syncResult": Object{"revision": strings.Repeat("b", 40)}}}}
	return want, got
}
func TestApplicationFences(t *testing.T) {
	want, base := appFixture()
	if f := ClassifyApplication(want, base, nil); f.Classification != Verified || f.ObservedGeneration != nil {
		t.Fatal(f)
	}
	for _, tc := range []struct {
		name   string
		mutate func(Object)
		class  Classification
		reason string
	}{
		{"old healthy", func(o Object) { Map(At(o, "status", "sync"))["revision"] = strings.Repeat("b", 40) }, Progressing, "REVISION_NOT_CONVERGED"},
		{"revision absent", func(o Object) { delete(Map(At(o, "status", "sync")), "revision") }, Unknown, "REVISION_EVIDENCE_MISSING"},
		{"operation request", func(o Object) { o["operation"] = Object{"sync": Object{}} }, Progressing, "OPERATION_ACTIVE"},
		{"running phase", func(o Object) { Map(At(o, "status", "operationState"))["phase"] = "Running" }, Progressing, "OPERATION_ACTIVE"},
		{"failed", func(o Object) { Map(At(o, "status", "operationState"))["phase"] = "Failed" }, Degraded, "LAST_OPERATION_FAILED"},
		{"conditions", func(o Object) { Map(o["status"])["conditions"] = []any{Object{"type": "SharedResourceWarning"}} }, Drifted, "SHARED_RESOURCE"},
		{"UID", func(o Object) { Map(o["metadata"])["uid"] = "replacement" }, Drifted, "UID_CHANGED"},
		{"spec", func(o Object) { Map(o["spec"])["project"] = "other" }, Drifted, "APPLICATION_SPEC_DRIFT"},
		{"health missing", func(o Object) { delete(Map(o["status"]), "health") }, Unknown, "HEALTH_UNKNOWN"},
		{"stale generation", func(o Object) { Map(o["status"])["observedGeneration"] = 1 }, Progressing, "STALE_OBSERVED_GENERATION"},
		{"malformed generation", func(o Object) { Map(o["status"])["observedGeneration"] = "2" }, Unknown, "MALFORMED_GENERATION"},
		{"unknown condition", func(o Object) { Map(o["status"])["conditions"] = []any{Object{"type": "FutureType"}} }, Unknown, "APPLICATION_CONDITION"},
		{"finalizer", func(o Object) { Map(o["metadata"])["finalizers"] = []any{"resources-finalizer.argocd.argoproj.io"} }, Drifted, "APPLICATION_DELETION_BOUNDARY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := Clone(base)
			tc.mutate(o)
			f := ClassifyApplication(want, o, nil)
			if f.Classification != tc.class || !strings.Contains(strings.Join(f.Reasons, ","), tc.reason) {
				t.Fatal(f)
			}
		})
	}
	if f := ClassifyApplication(want, nil, errors.New("network")); f.Classification != Unknown {
		t.Fatal(f)
	}
	if f := ClassifyApplication(want, nil, nil); f.Classification != Absent {
		t.Fatal(f)
	}
}
func resourceFixture() (ExpectedResource, Object) {
	ref := Ref{"networking.k8s.io/v1", "NetworkPolicy", "workload-web", "s3-client-egress"}
	obj := Object{"apiVersion": ref.APIVersion, "kind": ref.Kind, "metadata": Object{"name": ref.Name, "namespace": ref.Namespace, "uid": "policy-uid", "resourceVersion": "20", "annotations": Object{TrackingAnnotation: Tracking("storage-foundation", "atlas-storage", ref)}, "managedFields": []any{Object{"manager": "argocd-controller", "operation": "Apply", "fieldsV1": Object{"f:spec": Object{}}}}}, "spec": Object{"podSelector": Object{"matchLabels": Object{"atlas.local/s3-client": "true"}}, "policyTypes": []any{"Egress"}, "egress": []any{Object{"ports": []any{Object{"port": 8333, "protocol": "TCP"}}}}}}
	return ExpectedResource{Ref: ref, UID: "policy-uid", SemanticSHA256: Digest(Semantic(obj)), Tracking: Tracking("storage-foundation", "atlas-storage", ref), RequireSSA: true, Readiness: "identity-content"}, obj
}
func TestResourceSemanticsDoNotHidePolicyChanges(t *testing.T) {
	want, base := resourceFixture()
	if f := ClassifyResource(want, base, nil); f.Classification != Verified {
		t.Fatal(f)
	}
	runtime := Clone(base)
	Map(runtime["metadata"])["resourceVersion"] = "21"
	runtime["status"] = Object{"used": Object{"pods": "2"}}
	if Digest(Semantic(runtime)) != want.SemanticSHA256 {
		t.Fatal("runtime state affected content digest")
	}
	for _, mutate := range []func(Object){
		func(o Object) { Map(o["metadata"])["labels"] = Object{"changed": "true"} },
		func(o Object) { Map(At(o, "metadata", "annotations"))["policy"] = "changed" },
		func(o Object) { Map(o["metadata"])["ownerReferences"] = []any{Object{"uid": "other"}} },
		func(o Object) { Map(o["metadata"])["finalizers"] = []any{"other"} },
		func(o Object) { Map(o["spec"])["policyTypes"] = []any{"Ingress"} },
	} {
		o := Clone(base)
		mutate(o)
		if f := ClassifyResource(want, o, nil); f.Classification != Drifted {
			t.Fatal(f)
		}
	}
	o := Clone(base)
	Map(At(o, "metadata", "annotations"))[TrackingAnnotation] = "capability-foundation:/NetworkPolicy:workload-web/s3-client-egress"
	if Digest(Semantic(o)) != want.SemanticSHA256 {
		t.Fatal("tracking is a separate fact")
	}
	if f := ClassifyResource(want, o, nil); f.Classification != Drifted {
		t.Fatal(f)
	}
	if Tracking("capability-foundation", "argocd", Ref{"v1", "Namespace", "", "atlas-storage"}) != "capability-foundation:/Namespace:argocd/atlas-storage" {
		t.Fatal("cluster tracking namespace")
	}
}

type fakeReader struct {
	objects     map[Ref]Object
	calls       int
	changeAt    int
	unavailable bool
}

func (f *fakeReader) Read(_ context.Context, r Ref) (Object, error) {
	f.calls++
	if f.unavailable {
		return nil, errors.New("private error must not escape")
	}
	o := Clone(f.objects[r])
	if f.changeAt > 0 && f.calls >= f.changeAt {
		Map(o["metadata"])["resourceVersion"] = "changed"
	}
	return o, nil
}
func (f *fakeReader) ClusterIdentity(context.Context) (string, error) { return "cluster-uid", nil }
func expectationFixture() (Expectation, *fakeReader) {
	a, o := appFixture()
	r, p := resourceFixture()
	return Expectation{1, "OT-1 test", Target{"atlas-refactor-test-ot1", "kind-atlas-refactor-test-ot1", "cluster-uid", strings.Repeat("1", 64)}, strings.Repeat("c", 40), a.Revision, []ExpectedApplication{a}, []ExpectedResource{r}}, &fakeReader{objects: map[Ref]Object{Reference(o): o, Reference(p): p}}
}
func TestCollectClosingFenceAndReadFailures(t *testing.T) {
	for _, tc := range []struct {
		name        string
		change      int
		unavailable bool
		want        Classification
	}{{"stable", 0, false, Verified}, {"changed", 3, false, Unknown}, {"unavailable", 0, true, Unknown}} {
		t.Run(tc.name, func(t *testing.T) {
			expect, reader := expectationFixture()
			reader.changeAt = tc.change
			reader.unavailable = tc.unavailable
			got, e := Collect(context.Background(), reader, expect)
			if e != nil || got.Classification != tc.want {
				t.Fatal(got, e)
			}
			if strings.Contains(got.Report(), "private error") {
				t.Fatal("leaked error")
			}
		})
	}
}
func TestStrictJSONAndSensitiveTargets(t *testing.T) {
	for _, data := range []string{`{"schema":1,"schema":2}`, `{"a":{"x":1,"x":2}}`, `{} {}`} {
		var o Object
		if Decode([]byte(data), &o, false) == nil {
			t.Fatal(data)
		}
	}
	e, _ := expectationFixture()
	e.Resources[0].Ref = Ref{"v1", "Secret", "argocd", "argocd-secret"}
	if e.Validate() == nil {
		t.Fatal("Secret admitted")
	}
	e, _ = expectationFixture()
	e.Resources[0].Ref = Ref{"v1", "ConfigMap", "argocd", "x/../../secrets"}
	if e.Validate() == nil {
		t.Fatal("path injection admitted")
	}
	e, _ = expectationFixture()
	e.Resources[0].Readiness = "deployment-available"
	if e.Validate() == nil {
		t.Fatal("unknown health treated ready")
	}
}
func TestPrivateEvidenceCreateOnlyAndSymlinks(t *testing.T) {
	baseDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(baseDir, "observations")
	e, _ := expectationFixture()
	snapshot := Envelope{Schema: "atlas.observation/v1", Subject: e.Subject}
	path, err := Save(dir, "attempt-01", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Save(dir, "attempt-01", snapshot); err == nil {
		t.Fatal("overwrote evidence")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
	link := filepath.Join(baseDir, "link")
	if err = os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err = Save(link, "attempt-02", snapshot); err == nil {
		t.Fatal("followed symlink")
	}
	if _, err = Save(dir, "../escape", snapshot); err == nil {
		t.Fatal("path escape")
	}
}
