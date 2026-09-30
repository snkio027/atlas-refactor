package installation

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type gitFixture struct {
	remote, head    string
	commits, pushes int
	deny, loseAck   bool
}

func (f *gitFixture) run(_ context.Context, _ string, _ []byte, name string, args ...string) ([]byte, error) {
	if name == "gh" {
		if f.deny {
			return []byte(`{"full_name":"example/independent","private":false,"archived":false,"permissions":{"push":false}}`), nil
		}
		return []byte(`{"full_name":"example/independent","private":false,"archived":false,"permissions":{"push":true}}`), nil
	}
	if name != "git" {
		return nil, errors.New("unexpected executable")
	}
	for len(args) > 0 && args[0] == "-c" {
		args = args[2:]
	}
	switch args[0] {
	case "ls-remote":
		if f.remote == "" {
			return nil, nil
		}
		return []byte(f.remote + "\trefs/heads/atlas-development\n"), nil
	case "init", "add", "read-tree":
		return nil, nil
	case "write-tree":
		return []byte(strings.Repeat("d", 40)), nil
	case "commit-tree":
		f.commits++
		if len(args) > 2 && args[3] != f.head {
			return nil, errors.New("parent fence")
		}
		return []byte(strings.Repeat(string(rune('a'+f.commits)), 40)), nil
	case "update-ref":
		f.head = args[2]
		return nil, nil
	case "rev-parse":
		return []byte(f.head), nil
	case "push":
		f.pushes++
		if f.deny {
			return nil, errors.New("permission denied")
		}
		f.remote = strings.Split(args[len(args)-1], ":")[0]
		if f.loseAck {
			return nil, errors.New("lost acknowledgement")
		}
		return nil, nil
	}
	return nil, errors.New("unexpected Git operation")
}
func TestPublishUnknownResultReconcilesWithoutNewCommit(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	f := &gitFixture{loseAck: true}
	w := Workflow{Config: c, Product: Product{Version: "v0.1.0"}, runCommand: f.run}
	files := Files{"gitops/desired.json": []byte("safe-public-state"), "private/credentials.json": []byte("must never publish")}
	got, e := w.publish(context.Background(), "base", files, "")
	if e != nil || got != f.remote {
		t.Fatal("lost acknowledgement should be inspected", e)
	}
	again, e := w.publish(context.Background(), "base", files, "")
	if e != nil || again != got || f.commits != 1 || f.pushes != 1 {
		t.Fatal("retry resent a confirmed commit", e)
	}
	f.remote = strings.Repeat("e", 40)
	if _, e = w.publish(context.Background(), "base", files, ""); e == nil {
		t.Fatal("accepted changed remote")
	}
	if f.commits != 1 || f.pushes != 1 {
		t.Fatal("mutated after fence changed")
	}
}
func TestPublishDeniedRetainsExactIntent(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	f := &gitFixture{deny: true}
	w := Workflow{Config: c, Product: Product{Version: "v0.1.0"}, runCommand: f.run}
	files := Files{"gitops/desired.json": []byte("fixed")}
	if _, e := w.publish(context.Background(), "base", files, ""); e == nil {
		t.Fatal("permission failure ignored")
	}
	if f.commits != 1 {
		t.Fatal("missing saved intent")
	}
	f.deny = false
	if _, e := w.publish(context.Background(), "base", files, ""); e != nil {
		t.Fatal(e)
	}
	if f.commits != 1 || f.pushes != 2 {
		t.Fatal("retry created a second commit")
	}
	files["gitops/desired.json"] = []byte("different")
	if _, e := w.publish(context.Background(), "base", files, ""); e == nil {
		t.Fatal("retry changed intended deployment")
	}
}
func TestGitPermissionPreflightDoesNotMutate(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	f := &gitFixture{deny: true}
	w := Workflow{Config: c, runCommand: f.run}
	if w.checkGit(context.Background()) == nil {
		t.Fatal("read-only collaborator accepted")
	}
	if f.commits != 0 || f.pushes != 0 {
		t.Fatal("permission preflight mutated Git")
	}
}

func TestFullPublishAcknowledgementPrecedesBootstrapReentry(t *testing.T) {
	p := testProduct(t)
	c := testConfig(t)
	w := Workflow{Config: c, Product: p, Record: Record{BaseCommit: strings.Repeat("a", 40), CertificateSHA256: strings.Repeat("c", 64), CredentialsSHA256: strings.Repeat("d", 64)}}
	sealed := SealedRecord{CertificateSHA256: w.Record.CertificateSHA256, CredentialsSHA256: w.Record.CredentialsSHA256, Objects: []map[string]any{{"kind": "SealedSecret"}}}
	if e := save(filepath.Join(c.StateDirectory, "sealed.json"), JSON(sealed), true); e != nil {
		t.Fatal(e)
	}
	w.Record.SealedSHA256 = Digest(JSON(map[string]any{"apiVersion": "v1", "kind": "List", "items": sealed.Objects}))
	full, e := w.desiredFull()
	if e != nil {
		t.Fatal(e)
	}
	signal := []byte("bound signal")
	full[signalPath] = signal
	tree := Files{}
	for p, b := range full {
		if strings.HasPrefix(p, "gitops/") {
			tree[p] = b
		}
	}
	intended := strings.Repeat("b", 40)
	if e = save(filepath.Join(c.StateDirectory, "publish-full.json"), JSON(publishIntent{Parent: w.Record.BaseCommit, Commit: intended, TreeSHA256: Digest(JSON(tree))}), true); e != nil {
		t.Fatal(e)
	}
	fixture := &gitFixture{remote: intended}
	w.runCommand = fixture.run
	got, e := w.reconcileFullPublication(context.Background(), signal)
	if e != nil || got != intended || fixture.pushes != 0 || fixture.commits != 0 {
		t.Fatal("confirmed full publish was not recovered read-only", e)
	}
	fixture.remote = w.Record.BaseCommit
	got, e = w.reconcileFullPublication(context.Background(), signal)
	if e != nil || got != "" || fixture.pushes != 0 {
		t.Fatal("unconfirmed intent was resent ahead of preconditions", e)
	}
	fixture.remote = strings.Repeat("f", 40)
	if _, e = w.reconcileFullPublication(context.Background(), signal); e == nil {
		t.Fatal("accepted unplanned Git result")
	}
}

func TestPublicationUsesClosedTreeAndDirectParentWithRealGit(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	w := Workflow{Config: c, Product: Product{Version: "v0.1.0"}}
	remote := ""
	w.runCommand = func(ctx context.Context, dir string, input []byte, name string, args ...string) ([]byte, error) {
		actual := args
		for len(actual) > 0 && actual[0] == "-c" {
			actual = actual[2:]
		}
		if name == "git" && actual[0] == "ls-remote" {
			if remote == "" {
				return nil, nil
			}
			return []byte(remote + "\trefs/heads/" + c.Branch + "\n"), nil
		}
		if name == "git" && actual[0] == "push" {
			remote = strings.Split(actual[len(actual)-1], ":")[0]
			return nil, nil
		}
		return command(ctx, dir, input, name, args...)
	}
	if e := save(filepath.Join(w.repoDir(), "gitops/unapproved.json"), []byte("synthetic private file outside generated inventory"), true); e != nil {
		t.Fatal(e)
	}
	files := Files{"gitops/approved.json": []byte("base definition\n")}
	base, e := w.publish(context.Background(), "base", files, "")
	if e != nil {
		t.Fatal(e)
	}
	files["gitops/approved.json"] = []byte("full definition\n")
	full, e := w.publish(context.Background(), "full", files, base)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{base, full} {
		b, e := w.git(context.Background(), nil, "ls-tree", "-r", "--name-only", id)
		if e != nil || string(b) != "gitops/approved.json\n" {
			t.Fatal("unapproved files entered Git tree", e)
		}
	}
	parent, e := w.git(context.Background(), nil, "rev-parse", full+"^")
	if e != nil || strings.TrimSpace(string(parent)) != base {
		t.Fatal("publication did not extend its exact parent", e)
	}
}

func TestExistingBranchCannotBeRepurposedForFreshInstallation(t *testing.T) {
	c := testConfig(t)
	if e := privateDir(c.StateDirectory); e != nil {
		t.Fatal(e)
	}
	f := &gitFixture{remote: strings.Repeat("e", 40)}
	w := Workflow{Config: c, runCommand: f.run}
	if w.checkGit(context.Background()) == nil {
		t.Fatal("accepted existing remote branch")
	}
	if _, e := w.publish(context.Background(), "base", Files{"gitops/object.json": []byte("new")}, ""); e == nil {
		t.Fatal("overwrote existing branch")
	}
	if f.commits != 0 || f.pushes != 0 {
		t.Fatal("mutated existing branch")
	}
}
