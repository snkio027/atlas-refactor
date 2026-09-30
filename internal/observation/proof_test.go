package observation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSemanticProofPermitsBookkeepingButRejectsControlChanges(t *testing.T) {
	_, app := appFixture()
	Map(app["status"])["observedGeneration"] = 2
	Map(app["metadata"])["managedFields"] = []any{Object{"manager": "argocd-controller", "operation": "Apply", "fieldsV1": Object{"f:spec": Object{}}}}
	for _, tc := range []struct {
		name   string
		change func(Object)
		same   bool
	}{
		{"rv", func(o Object) { Map(o["metadata"])["resourceVersion"] = "99" }, true},
		{"generation bookkeeping", func(o Object) { Map(o["metadata"])["generation"] = 9; Map(o["status"])["observedGeneration"] = 9 }, true},
		{"reconciledAt", func(o Object) { Map(o["status"])["reconciledAt"] = "later" }, true},
		{"managed field time", func(o Object) {
			for _, f := range Slice(At(o, "metadata", "managedFields")) {
				Map(f)["time"] = "later"
			}
		}, true},
		{"uid", func(o Object) { Map(o["metadata"])["uid"] = "replacement" }, false},
		{"missing rv", func(o Object) { delete(Map(o["metadata"]), "resourceVersion") }, false},
		{"spec", func(o Object) { Map(o["spec"])["project"] = "other" }, false},
		{"tracking", func(o Object) { Map(o["metadata"])["annotations"] = Object{TrackingAnnotation: "other"} }, false},
		{"SSA loss", func(o Object) { Map(o["metadata"])["managedFields"] = []any{} }, false},
		{"sync revision", func(o Object) { Map(At(o, "status", "sync"))["revision"] = "other" }, false},
		{"health", func(o Object) { Map(At(o, "status", "health"))["status"] = "Degraded" }, false},
		{"active operation", func(o Object) { o["operation"] = Object{"sync": Object{"prune": true}} }, false},
		{"operation result", func(o Object) { Map(At(o, "status", "operationState"))["phase"] = "Failed" }, false},
		{"conditions", func(o Object) { Map(o["status"])["conditions"] = []any{Object{"type": "ComparisonError"}} }, false},
		{"resources", func(o Object) { Map(o["status"])["resources"] = []any{Object{"name": "new", "status": "OutOfSync"}} }, false},
		{"stale observedGeneration", func(o Object) { Map(o["metadata"])["generation"] = 9 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := Clone(app)
			tc.change(other)
			if SameProof(app, other, "identity-content") != tc.same {
				t.Fatal("wrong proof equivalence")
			}
		})
	}
}
func TestNodeProofAndInventoryMembership(t *testing.T) {
	n := Object{"apiVersion": "v1", "kind": "Node", "metadata": Object{"name": "node", "uid": "uid", "resourceVersion": "1"}, "spec": Object{}, "status": Object{"conditions": []any{Object{"type": "Ready", "status": "True", "lastHeartbeatTime": "before"}}}}
	copy := Clone(n)
	Map(Slice(At(copy, "status", "conditions"))[0])["lastHeartbeatTime"] = "after"
	Map(copy["metadata"])["resourceVersion"] = "2"
	if !SameProof(n, copy, "identity-content") {
		t.Fatal("heartbeat changed proof")
	}
	Map(Slice(At(copy, "status", "conditions"))[0])["status"] = "False"
	if SameProof(n, copy, "identity-content") {
		t.Fatal("readiness ignored")
	}
	kind := Ref{APIVersion: "v1", Kind: "Node"}
	if _, e := InventoryProof([]Object{n, n}, kind); e == nil {
		t.Fatal("duplicate accepted")
	}
	a, _ := InventoryProof([]Object{n}, kind)
	b, _ := InventoryProof(nil, kind)
	if a == b {
		t.Fatal("missing member accepted")
	}
	copy = Clone(n)
	Map(copy["spec"])["unschedulable"] = true
	if SameProof(n, copy, "identity-content") {
		t.Fatal("spec ignored")
	}
}

type changedProofReader struct {
	*fakeReader
	fail bool
}

func (r *changedProofReader) Read(ctx context.Context, ref Ref) (Object, error) {
	o, e := r.fakeReader.Read(ctx, ref)
	if r.calls > 2 {
		if r.fail {
			return nil, errors.New("unavailable")
		}
		Map(o["metadata"])["uid"] = "replacement"
	}
	return o, e
}
func TestCollectSemanticChangesAreUnknownWithoutResampling(t *testing.T) {
	for _, fail := range []bool{false, true} {
		expect, reader := expectationFixture()
		r := &changedProofReader{reader, fail}
		out, e := Collect(context.Background(), r, expect)
		if e != nil || out.Classification != Unknown || reader.calls != 4 {
			t.Fatal("must reject once, not resample", e, reader.calls, out.Classification)
		}
		prefix := "SNAPSHOT_CHANGED:"
		if fail {
			prefix = "CLOSING_READ_UNAVAILABLE:"
		}
		if len(out.Reasons) != 2 {
			t.Fatal(out.Reasons)
		}
		for _, reason := range out.Reasons {
			if !strings.HasPrefix(reason, prefix) {
				t.Fatal("failure and race merged", out.Reasons)
			}
		}
	}
	expect, reader := expectationFixture()
	out, e := Collect(context.Background(), reader, expect)
	if e != nil {
		t.Fatal(e)
	}
	rules := map[string]string{}
	for _, r := range expect.Resources {
		rules[r.Ref.Key()] = r.Readiness
	}
	if e = ValidateClosingProof(out, rules); e != nil {
		t.Fatal(e)
	}
	out.ClosingRaw = out.ClosingRaw[:1]
	if ValidateClosingProof(out, rules) == nil {
		t.Fatal("partial proof accepted")
	}
}
