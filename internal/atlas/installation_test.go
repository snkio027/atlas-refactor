package atlas

import (
	"context"
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
