package atlas

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type deploymentRunner struct {
	commit string
	body   []byte
	calls  []string
}

func (r *deploymentRunner) Run(_ context.Context, q Request) ([]byte, error) {
	r.calls = append(r.calls, q.Tool+" "+strings.Join(q.Args, " "))
	if q.Tool != "git" {
		return nil, errors.New("unexpected mutation")
	}
	switch q.Args[0] {
	case "show":
		return r.body, nil
	case "ls-remote":
		return []byte(r.commit + "\trefs/heads/atlas-development\n"), nil
	default:
		return nil, errors.New("product HEAD must never be used")
	}
}
func TestInstallationChecksDeploymentCommitNotProductHEAD(t *testing.T) {
	r := &deploymentRunner{commit: strings.Repeat("b", 40), body: []byte("manifest")}
	a := &App{Config: Config{RepositoryURL: "https://github.com/user/deployment.git", Revision: "atlas-development"}, Runner: r, installation: &InstallationBinding{DeploymentCommit: r.commit}}
	f := map[string][]byte{"gitops/root/resource.json": r.body, "platform/development/bootstrap/input.json": []byte("private runtime")}
	if e := a.verifyRepository(context.Background(), f); e != nil {
		t.Fatal(e)
	}
	for _, c := range r.calls {
		if strings.Contains(c, "rev-parse") {
			t.Fatal("compared product source HEAD")
		}
	}
	r.commit = strings.Repeat("c", 40)
	if a.verifyRepository(context.Background(), f) == nil {
		t.Fatal("accepted changed remote after prior successful resolution")
	}
}
func TestInstallationRejectsStalePublishedBytes(t *testing.T) {
	r := &deploymentRunner{commit: strings.Repeat("b", 40), body: []byte("different")}
	a := &App{Runner: r, installation: &InstallationBinding{DeploymentCommit: r.commit}}
	if a.verifyRepository(context.Background(), map[string][]byte{"gitops/object.json": []byte("intended")}) == nil {
		t.Fatal("accepted changed desired state")
	}
	if len(r.calls) != 1 {
		t.Fatal("should reject before remote or mutation")
	}
}
func TestSchema4CannotRunThroughOrdinaryConfig(t *testing.T) {
	a := &App{Config: Config{Schema: 4, Cluster: "atlas-user", RepositoryURL: "https://github.com/user/deployment.git", Revision: "atlas-development", GitOpsPath: "gitops/root/overlays/development", DockerContext: "orbstack", TimeoutSeconds: 1800}}
	if a.VerifyArtifacts() == nil {
		t.Fatal("ordinary config gained installation authority")
	}
}
func TestInstallationLockDoesNotTouchHistoricalSTOP(t *testing.T) {
	dir := t.TempDir()
	release, e := acquireInstallationLock(dir)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := acquireInstallationLock(dir); e == nil {
		r()
		t.Fatal("parallel apply admitted")
	}
	release()
	release, e = acquireInstallationLock(dir)
	if e != nil {
		t.Fatal(e)
	}
	release()
}

func TestInstallationHandoffStopsOnCurrentTerminalSyncFailure(t *testing.T) {
	for _, tc := range []struct {
		name, phase, revision string
		active, fail          bool
	}{
		{"failed child", "Failed", strings.Repeat("b", 40), false, true},
		{"errored child", "Error", strings.Repeat("b", 40), false, true},
		{"running child", "Running", strings.Repeat("b", 40), false, false},
		{"succeeded child", "Succeeded", strings.Repeat("b", 40), false, false},
		{"old failed operation", "Failed", strings.Repeat("a", 40), false, false},
		{"retry already active", "Failed", strings.Repeat("b", 40), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := fixture(t)
			a.installation = &InstallationBinding{DeploymentCommit: strings.Repeat("b", 40)}
			if e := WriteFiles(a.Root, ".state", map[string][]byte{"kubeconfig": []byte("private fixture"), "kubeconfig.sha256": []byte(digest([]byte("private fixture")))}); e != nil {
				t.Fatal(e)
			}
			app := Object{"metadata": Object{"name": "secrets-controller", "namespace": "argocd", "uid": "controller-app"}, "status": Object{"operationState": Object{"phase": tc.phase, "syncResult": Object{"revision": tc.revision}}}}
			if tc.active {
				app["operation"] = Object{"sync": Object{}}
			}
			body, e := json.Marshal(Object{"kind": "ApplicationList", "items": []Object{app}})
			if e != nil {
				t.Fatal(e)
			}
			calls := 0
			a.Runner = runnerFunc(func(_ context.Context, q Request) ([]byte, error) {
				calls++
				if q.Tool != "kubectl" || !strings.Contains(strings.Join(q.Args, " "), " get applications -n argocd -o json") || len(q.Input) != 0 {
					t.Fatal("unexpected write", q)
				}
				return body, nil
			})
			e = a.checkInstallationHandoffFailure(context.Background())
			if (e != nil) != tc.fail || calls != 1 {
				t.Fatal(e, calls)
			}
			if e != nil && !strings.Contains(e.Error(), "secrets-controller") {
				t.Fatal(e)
			}
			a.installation = nil
			if e = a.checkInstallationHandoffFailure(context.Background()); e != nil || calls != 1 {
				t.Fatal("historical contract changed", e, calls)
			}
		})
	}
}

func TestInstallationHandoffUnknownReadFailsClosed(t *testing.T) {
	for _, body := range [][]byte{nil, []byte(`{"kind":"SecretList","items":[]}`), []byte(`{"kind":"ApplicationList","items":[{"metadata":{"name":"unbound"}}]}`)} {
		a, _ := fixture(t)
		a.installation = &InstallationBinding{DeploymentCommit: strings.Repeat("b", 40)}
		if e := WriteFiles(a.Root, ".state", map[string][]byte{"kubeconfig": []byte("private fixture"), "kubeconfig.sha256": []byte(digest([]byte("private fixture")))}); e != nil {
			t.Fatal(e)
		}
		a.Runner = runnerFunc(func(_ context.Context, q Request) ([]byte, error) {
			if q.Tool != "kubectl" || len(q.Input) != 0 {
				t.Fatal("unexpected write")
			}
			return body, nil
		})
		if a.checkInstallationHandoffFailure(context.Background()) == nil {
			t.Fatal("accepted unknown Application response")
		}
	}
}
