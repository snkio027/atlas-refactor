package workloadrun

import (
	"atlas-refactor/internal/observation"
	"atlas-refactor/internal/platform"
	"atlas-refactor/internal/workload"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func initialApp() (observation.ExpectedApplication, Object) {
	want, app := gateApp("new-project")
	delete(app, "status")
	m := mapping(app["metadata"])
	m["generation"] = float64(1)
	m["annotations"] = Object{observation.TrackingAnnotation: "platform-control:argoproj.io/Application:argocd/" + want.Name}
	m["managedFields"] = []any{Object{"manager": "argocd-controller", "operation": "Apply", "fieldsV1": Object{"f:spec": Object{}}}}
	return want, app
}

func TestNewApplicationFirstObservationWaitsWithoutInventingProof(t *testing.T) {
	for _, emptyStatus := range []bool{false, true} {
		want, app := initialApp()
		if emptyStatus {
			app["status"] = Object{}
		}
		facts, err := classifyApplications([]observation.ExpectedApplication{want}, map[string]Object{want.Name: app}, map[string]string{want.Name: "platform-control"})
		var pending Pending
		if !errors.As(err, &pending) || facts[0].Classification != observation.Unknown || facts[0].ObservedRevision != "" {
			t.Fatal("initialization invented proof or became fatal", facts, err)
		}
		_, healthy := gateApp(want.Name)
		app["status"] = healthy["status"]
		facts, err = classifyApplications([]observation.ExpectedApplication{want}, map[string]Object{want.Name: app}, map[string]string{want.Name: "platform-control"})
		if err != nil || facts[0].Classification != observation.Verified {
			t.Fatal(facts, err)
		}
	}
}

func TestFirstObservationCannotMaskUnknownDriftOrPreviousState(t *testing.T) {
	cases := map[string]func(*observation.ExpectedApplication, Object, map[string]string){
		"not introduced in phase": func(w *observation.ExpectedApplication, a Object, owners map[string]string) { delete(owners, w.Name) },
		"existing UID": func(w *observation.ExpectedApplication, a Object, _ map[string]string) {
			w.UID = str(at(a, "metadata", "uid"))
		},
		"later generation": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			mapping(a["metadata"])["generation"] = float64(2)
		},
		"no SSA": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			delete(mapping(a["metadata"]), "managedFields")
		},
		"wrong owner": func(w *observation.ExpectedApplication, _ Object, owners map[string]string) {
			owners[w.Name] = "foreign"
		},
		"lost UID": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			delete(mapping(a["metadata"]), "uid")
		},
		"spec drift": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			a["spec"] = Object{"project": "foreign"}
		},
		"null status":      func(_ *observation.ExpectedApplication, a Object, _ map[string]string) { a["status"] = nil },
		"malformed status": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) { a["status"] = "bad" },
		"explicit Unknown": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			a["status"] = Object{"sync": Object{"status": "Unknown"}}
		},
		"already observed": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			a["status"] = Object{"reconciledAt": "2026-10-07T16:12:56Z"}
		},
		"error condition": func(_ *observation.ExpectedApplication, a Object, _ map[string]string) {
			a["status"] = Object{"conditions": []any{Object{"type": "InvalidSpecError"}}}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			want, app := initialApp()
			owners := map[string]string{want.Name: "platform-control"}
			change(&want, app, owners)
			_, err := classifyApplications([]observation.ExpectedApplication{want}, map[string]Object{want.Name: app}, owners)
			var pending Pending
			if err == nil || errors.As(err, &pending) {
				t.Fatal("invalid initialization was tolerated", err)
			}
		})
	}
}

func TestFatalStillWinsOverFirstObservation(t *testing.T) {
	want, app := initialApp()
	bad, live := gateApp("bad")
	mapping(live["status"])["conditions"] = []any{Object{"type": "InvalidSpecError"}}
	_, err := classifyApplications([]observation.ExpectedApplication{want, bad}, map[string]Object{want.Name: app, bad.Name: live}, map[string]string{want.Name: "platform-control"})
	var pending Pending
	if err == nil || errors.As(err, &pending) {
		t.Fatal(err)
	}
}

func TestFirstObservationBoundToCurrentPhaseIntroduction(t *testing.T) {
	want, app := initialApp()
	id := "argoproj.io/Application/argocd/" + want.Name
	rs := []workload.OwnedResource{{Identity: id, Owner: "platform-control"}}
	current := map[string]Object{id: app}
	if newApplicationOwners(rs, current, map[string]Object{})[want.Name] != "platform-control" {
		t.Fatal("new declaration missing")
	}
	if len(newApplicationOwners(rs, current, current)) != 0 {
		t.Fatal("prior-phase Application treated as new")
	}
}

func TestFirstObservationDisabledAfterGateStopOrCompletion(t *testing.T) {
	for _, marker := range []string{"project-gate.json", "terminal.json", "final.json"} {
		t.Run(marker, func(t *testing.T) {
			w, p := sequenceFixture(t)
			p.CredentialTargets = []string{}
			if err := w.SavePlan(p); err != nil {
				t.Fatal(err)
			}
			if err := save(filepath.Join(filepath.Dir(w.publicationPath(p, "project")), marker), []byte("{}"), true); err != nil {
				t.Fatal(err)
			}
			owners, err := w.initialApplicationOwners(context.Background(), workload.Result{Inventory: workload.Inventory{Phase: "project"}}, p.Parent)
			if err != nil || len(owners) != 0 {
				t.Fatal("completed/stopped phase opened initialization", owners, err)
			}
		})
	}
}

func TestFirstObservationRequiresBoundCurrentPublication(t *testing.T) {
	for _, field := range []string{"schema", "plan", "phase", "parent", "commit", "tree"} {
		t.Run(field, func(t *testing.T) {
			w, p := sequenceFixture(t)
			p.CredentialTargets = []string{}
			if err := w.SavePlan(p); err != nil {
				t.Fatal(err)
			}
			result := workload.Result{Files: Files{}, Inventory: workload.Inventory{Phase: "permissions"}}
			receipt := Publication{1, workload.Digest(workload.JSON(p)), "permissions", p.Parent, p.Parent, platform.BundleDigest(result.Files)}
			switch field {
			case "schema":
				receipt.Schema = 0
			case "plan":
				receipt.PlanSHA256 = "other"
			case "phase":
				receipt.Phase = "consumer"
			case "parent":
				receipt.Parent = "other"
			case "commit":
				receipt.Commit = "other"
			case "tree":
				receipt.TreeSHA256 = "other"
			}
			if err := save(w.publicationPath(p, "permissions"), workload.JSON(receipt), true); err != nil {
				t.Fatal(err)
			}
			if _, err := w.initialApplicationOwners(context.Background(), result, p.Parent); err == nil {
				t.Fatal("unbound receipt opened initialization")
			}
		})
	}
}
