package installation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Authentication stays in the user's GitHub CLI store. No token is copied into
// config, arguments, logs, the deployment repository or installation evidence.
func command(ctx context.Context, dir string, input []byte, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(input)
	for _, k := range []string{"HOME", "PATH", "TMPDIR"} {
		if v, ok := os.LookupEnv(k); ok {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	if e := cmd.Run(); e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%s failed; subprocess output withheld: %w", name, e)
	}
	return out.Bytes(), nil
}
func (w *Workflow) git(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	prefix := []string{"-c", "core.hooksPath=/dev/null", "-c", "credential.helper=", "-c", "credential.helper=!gh auth git-credential", "-c", "user.name=Atlas Installer", "-c", "user.email=atlas-install@users.noreply.github.com", "-c", "commit.gpgsign=false"}
	return w.command(ctx, w.repoDir(), input, "git", append(prefix, args...)...)
}
func (w *Workflow) remote(ctx context.Context) (string, error) {
	b, e := w.command(ctx, w.Config.StateDirectory, nil, "git", "ls-remote", w.Config.Repository, "refs/heads/"+w.Config.Branch)
	if e != nil {
		return "", e
	}
	s := strings.Fields(string(b))
	if len(s) == 0 {
		return "", nil
	}
	if len(s) != 2 || !commit.MatchString(s[0]) || s[1] != "refs/heads/"+w.Config.Branch {
		return "", errors.New("deployment branch must resolve exactly")
	}
	return s[0], nil
}
func (w *Workflow) checkGit(ctx context.Context) error {
	name := strings.TrimSuffix(strings.TrimPrefix(w.Config.Repository, "https://github.com/"), ".git")
	b, e := w.command(ctx, w.Config.StateDirectory, nil, "gh", "api", "--hostname", "github.com", "repos/"+name, "--jq", "{full_name,private,archived,permissions}")
	if e != nil {
		return e
	}
	var repo struct {
		FullName          string `json:"full_name"`
		Private, Archived bool
		Permissions       struct{ Admin, Maintain, Push, Pull, Triage bool }
	}
	if e = Decode(b, &repo); e != nil {
		return e
	}
	if !strings.EqualFold(repo.FullName, name) || repo.Private || repo.Archived || !repo.Permissions.Push {
		return errors.New("dedicated GitOps repository must be public, writable and active")
	}
	got, e := w.remote(ctx)
	if e != nil {
		return e
	}
	want := w.Record.BaseCommit
	if w.Record.FullCommit != "" {
		want = w.Record.FullCommit
	}
	if got != want {
		// A publish intent survives an ambiguous push. Only its exact saved commit
		// may explain the remote value; this does not authorize a new commit.
		for _, phase := range []string{"base", "full"} {
			intent, e := privateRead(filepath.Join(w.Config.StateDirectory, "publish-"+phase+".json"))
			if e == nil {
				var p publishIntent
				if Decode(intent, &p) == nil && p.Commit == got && p.Parent == want {
					return nil
				}
			}
		}
		return errors.New("deployment branch differs from this installation; fresh installs require an absent branch")
	}
	return nil
}

type publishIntent struct {
	Parent     string `json:"parent"`
	Commit     string `json:"commit"`
	TreeSHA256 string `json:"treeSHA256"`
}

func (w *Workflow) publish(ctx context.Context, phase string, files Files, parent string) (string, error) {
	path := filepath.Join(w.Config.StateDirectory, "publish-"+phase+".json")
	tree := Files{}
	for p, b := range files {
		if strings.HasPrefix(p, "gitops/") {
			tree[p] = b
		}
	}
	digest := Digest(JSON(tree))
	var intent publishIntent
	b, e := privateRead(path)
	if e == nil {
		if e = Decode(b, &intent); e != nil {
			return "", e
		}
		if intent.Parent != parent || intent.TreeSHA256 != digest || !commit.MatchString(intent.Commit) {
			return "", errors.New("saved publication intent differs")
		}
	}
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	remote, e := w.remote(ctx)
	if e != nil {
		return "", e
	}
	if intent.Commit != "" && remote == intent.Commit {
		return intent.Commit, nil
	}
	if remote != parent {
		return "", errors.New("remote Git fence changed before publication")
	}
	if e = privateDir(w.repoDir()); e != nil {
		return "", e
	}
	if _, e = w.git(ctx, nil, "init", "--initial-branch", w.Config.Branch); e != nil {
		return "", e
	}
	if intent.Commit == "" {
		if parent != "" {
			head, e := w.git(ctx, nil, "rev-parse", "HEAD")
			if e != nil || strings.TrimSpace(string(head)) != parent {
				return "", errors.New("local deployment HEAD differs from confirmed parent")
			}
		}
		for _, p := range sortedFiles(tree) {
			if e = save(filepath.Join(w.repoDir(), p), tree[p], false); e != nil {
				return "", e
			}
		}
		if _, e = w.git(ctx, nil, "read-tree", "--empty"); e != nil {
			return "", e
		}
		args := append([]string{"add", "--"}, sortedFiles(tree)...)
		if _, e = w.git(ctx, nil, args...); e != nil {
			return "", e
		}
		// commit-tree has no hooks and does not require a clean source checkout. The
		// dedicated deployment tree contains only generated public manifests.
		treeID, e := w.git(ctx, nil, "write-tree")
		if e != nil {
			return "", e
		}
		args = []string{"commit-tree", strings.TrimSpace(string(treeID))}
		if parent != "" {
			args = append(args, "-p", parent)
		}
		id, e := w.git(ctx, []byte("Atlas "+w.Product.Version+" installation "+phase+"\n"), args...)
		if e != nil {
			return "", e
		}
		intent = publishIntent{Parent: parent, Commit: strings.TrimSpace(string(id)), TreeSHA256: digest}
		if !commit.MatchString(intent.Commit) {
			return "", errors.New("invalid generated commit")
		}
		if e = save(path, JSON(intent), true); e != nil {
			return "", e
		}
	}
	if _, e = w.git(ctx, nil, "update-ref", "refs/heads/"+w.Config.Branch, intent.Commit); e != nil {
		return "", e
	}
	// New branch creation or a direct descendant of the exact verified parent.
	// The explicit lease protects first-creation races as well as subsequent writes.
	_, pushErr := w.git(ctx, nil, "push", "--force-with-lease=refs/heads/"+w.Config.Branch+":"+parent, w.Config.Repository, intent.Commit+":refs/heads/"+w.Config.Branch)
	remote, e = w.remote(ctx)
	if e != nil {
		return "", errors.New("publication outcome unknown; rerun inspects saved intent first")
	}
	if remote != intent.Commit {
		if pushErr != nil {
			return "", pushErr
		}
		return "", errors.New("publication did not land the intended commit")
	}
	return intent.Commit, nil
}

func (w *Workflow) command(ctx context.Context, dir string, input []byte, name string, args ...string) ([]byte, error) {
	if w.runCommand != nil {
		return w.runCommand(ctx, dir, input, name, args...)
	}
	return command(ctx, dir, input, name, args...)
}

// Reconcile a confirmed remote result before re-entering any earlier phase.
// This only reads: an intent whose push did not land is handled by its original
// phase after all original preconditions have been checked again.
func (w *Workflow) reconcileFullPublication(ctx context.Context, signal []byte) (string, error) {
	b, e := privateRead(filepath.Join(w.Config.StateDirectory, "publish-full.json"))
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	var intent publishIntent
	if e = Decode(b, &intent); e != nil {
		return "", e
	}
	full, e := w.desiredFull()
	if e != nil {
		return "", e
	}
	full[signalPath] = signal
	tree := Files{}
	for path, b := range full {
		if strings.HasPrefix(path, "gitops/") {
			tree[path] = b
		}
	}
	if intent.Parent != w.Record.BaseCommit || !commit.MatchString(intent.Commit) || intent.TreeSHA256 != Digest(JSON(tree)) {
		return "", errors.New("full publication intent no longer matches generated state")
	}
	remote, e := w.remote(ctx)
	if e != nil {
		return "", e
	}
	if remote == intent.Commit {
		return intent.Commit, nil
	}
	if remote != intent.Parent {
		return "", errors.New("remote changed after full publication intent")
	}
	return "", nil
}
