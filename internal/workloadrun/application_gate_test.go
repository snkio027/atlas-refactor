package workloadrun

import (
	"atlas-refactor/internal/observation"
	"atlas-refactor/internal/workload"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func gateApp(name string) (observation.ExpectedApplication, Object) {
	rev := strings.Repeat("a", 40)
	spec := Object{"project": "platform-project"}
	want := observation.ExpectedApplication{Name: name, Spec: spec, Revision: rev}
	live := Object{"apiVersion": "argoproj.io/v1alpha1", "kind": "Application", "metadata": Object{"name": name, "namespace": "argocd", "uid": name + "-uid", "resourceVersion": "1"}, "spec": spec, "status": Object{"sync": Object{"revision": rev, "status": "Synced"}, "health": Object{"status": "Healthy"}}}
	return want, live
}
func TestApplicationGateFatalAlwaysWinsOverProgressAndMissing(t *testing.T) {
	a, la := gateApp("a-progress")
	mapping(at(la, "status", "sync"))["status"] = "OutOfSync"
	b, lb := gateApp("b-fatal")
	mapping(lb["status"])["conditions"] = []any{Object{"type": "InvalidSpecError", "message": "private-diagnostic-canary"}}
	c, _ := gateApp("c-missing")
	for i := 0; i < 200; i++ {
		wants := []observation.ExpectedApplication{a, b, c}
		if i%2 == 1 {
			wants = []observation.ExpectedApplication{c, b, a}
		}
		facts, err := classifyApplications(wants, map[string]Object{a.Name: la, b.Name: lb})
		var pending Pending
		if err == nil || errors.As(err, &pending) || !strings.Contains(err.Error(), "APPLICATION_ERROR") || strings.Contains(err.Error(), "private-diagnostic-canary") {
			t.Fatal("fatal was hidden/leaked", err)
		}
		if len(facts) != 3 || facts[0].Ref.Name != a.Name || facts[1].Ref.Name != b.Name {
			t.Fatal("incomplete/non deterministic evidence")
		}
	}
	diag := applicationConditions(map[string]Object{b.Name: lb})
	if diag[b.Name][0]["message"] != "private-diagnostic-canary" {
		t.Fatal("lost private root cause")
	}
}
func TestApplicationGateExactInventoryAndUnknownRemainFatal(t *testing.T) {
	a, la := gateApp("a")
	cases := []struct {
		name          string
		live          map[string]Object
		pending, pass bool
	}{
		{"verified", map[string]Object{"a": la}, false, true},
		{"missing", map[string]Object{}, true, false},
		{"substitution", map[string]Object{"foreign": la}, false, false},
		{"extra", map[string]Object{"a": la, "foreign": la}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := classifyApplications([]observation.ExpectedApplication{a}, tc.live)
			var p Pending
			if (err == nil) != tc.pass || errors.As(err, &p) != tc.pending {
				t.Fatal(err)
			}
		})
	}
	for _, field := range []string{"sync", "health"} {
		_, app := gateApp("a")
		mapping(at(app, "status", field))["status"] = "Unknown"
		_, err := classifyApplications([]observation.ExpectedApplication{a}, map[string]Object{"a": app})
		var p Pending
		if err == nil || errors.As(err, &p) {
			t.Fatal("unknown accepted", err)
		}
	}
	before := workload.JSON(la)
	one, _ := classifyApplications([]observation.ExpectedApplication{a}, map[string]Object{"a": la})
	two, _ := classifyApplications([]observation.ExpectedApplication{a}, map[string]Object{"a": la})
	if !reflect.DeepEqual(one, two) || string(before) != string(workload.JSON(la)) {
		t.Fatal("classification changes its input")
	}
}
