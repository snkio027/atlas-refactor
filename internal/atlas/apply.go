package atlas

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) verifyRepository(ctx context.Context, files map[string][]byte) error {
	for p, want := range files {
		actual, e := readFile(a.Root, filepath.Join(a.Config.GitOpsPath, p))
		if e != nil {
			return fmt.Errorf("render into %s and commit it first: %w", a.Config.GitOpsPath, e)
		}
		if digest(actual) != digest(want) {
			return fmt.Errorf("committed render is stale: %s", p)
		}
	}
	b, e := a.run(ctx, "git", "status", "--porcelain", "--untracked-files=all")
	if e != nil {
		return e
	}
	if strings.TrimSpace(string(b)) != "" {
		return errors.New("apply requires a clean committed repository")
	}
	head, e := a.run(ctx, "git", "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	remote, e := a.run(ctx, "git", "ls-remote", "--exit-code", a.Config.RepositoryURL, a.Config.Revision)
	if e != nil {
		return errors.New("GitOps source is not reachable; publish the reviewed commit first")
	}
	lines := strings.Fields(string(remote))
	if len(lines) != 2 || lines[0] != strings.TrimSpace(string(head)) {
		return errors.New("GitOps revision must resolve to exactly the reviewed local HEAD")
	}
	return nil
}

func (a *App) acquire() (func(), error) {
	dir, e := safePath(a.Root, ".state")
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	info, e := os.Stat(dir)
	if e != nil {
		return nil, e
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New(".state must be owner-only")
	}
	lock := filepath.Join(dir, "apply.lock")
	if e = os.Mkdir(lock, 0700); e != nil {
		return nil, errors.New("apply lock exists or cannot be acquired; inspect interrupted work before manual removal")
	}
	return func() { _ = os.Remove(lock) }, nil
}

func (a *App) Apply(ctx context.Context, approvedCluster string, tier0 bool) error {
	if approvedCluster != a.Config.Cluster || !tier0 {
		return errors.New("apply requires --approve-cluster with the exact test cluster and --approve-tier0")
	}
	release, e := a.acquire()
	if e != nil {
		return e
	}
	defer release()
	if e = a.Doctor(ctx); e != nil {
		return e
	}
	files, e := a.Render(ctx)
	if e != nil {
		return e
	}
	if e = a.verifyRepository(ctx, files); e != nil {
		return e
	}
	exists, e := a.clusterExists(ctx)
	if e != nil {
		return e
	}
	if !exists {
		if e = a.createCluster(ctx); e != nil {
			return e
		}
	}
	obs, e := a.inspect(ctx)
	if e != nil {
		return e
	}
	switch obs.report.State {
	case Adopted:
		return nil
	case Fresh:
		if e = a.seedAndHandoff(ctx, files); e != nil {
			return e
		}
	case Handoff: // Resume observation and receipt commitment only.
	default:
		return fmt.Errorf("%s: %s", obs.report.State, obs.report.Detail)
	}
	for {
		obs, e = a.inspect(ctx)
		if e != nil {
			return e
		}
		if obs.report.State == Adopted {
			return nil
		}
		if obs.report.State != Handoff {
			return fmt.Errorf("%s: %s", obs.report.State, obs.report.Detail)
		}
		if obs.root != nil && obs.self != nil && obs.signal != nil {
			// Bind the receipt to the validated evidence snapshot. A changed or missing
			// object on the following inspection fails closed; it never re-enables Seed.
			data := map[string]string{"identityUID": obs.identity.Metadata.UID, "fingerprint": a.fingerprint(), "rootUID": obs.root.Metadata.UID, "selfUID": obs.self.Metadata.UID, "signalUID": obs.signal.Metadata.UID}
			if e = a.create(ctx, configMap("kube-system", "atlas-refactor-receipt", data)); e != nil {
				return e
			}
			continue
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("handoff incomplete: %w; Seed remains denied", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
}

func (a *App) createCluster(ctx context.Context) error {
	p, e := safePath(a.Root, ".state/kubeconfig")
	if e != nil {
		return e
	}
	// A lost cluster is not a new installation. No automatic destroy/recreate.
	if _, e = os.Lstat(p); !os.IsNotExist(e) {
		return errors.New("cluster missing but local target evidence exists; choose a new test identity")
	}
	kindConfig, e := a.auditedKindConfig()
	if e != nil {
		return e
	}
	if e = WriteFiles(a.Root, ".state", map[string][]byte{"kind.yaml": kindConfig, "audit-policy.yaml": auditPolicy}); e != nil {
		return e
	}
	_, e = a.run(ctx, "kind", "create", "cluster", "--name", a.Config.Cluster, "--image", a.Lock.NodeImage, "--config", filepath.Join(a.Root, ".state/kind.yaml"), "--kubeconfig", p, "--wait", "120s")
	if e != nil {
		return e
	}
	if e = os.Chmod(p, 0600); e != nil {
		return e
	}
	b, e := readFile(a.Root, ".state/kubeconfig")
	if e != nil {
		return e
	}
	if e = WriteFiles(a.Root, ".state", map[string][]byte{"kubeconfig.sha256": []byte(digest(b))}); e != nil {
		return e
	}
	return a.create(ctx, configMap("kube-system", "atlas-refactor-identity", a.identity()))
}

func (a *App) seedAndHandoff(ctx context.Context, files map[string][]byte) error {
	if e := a.verifyNodes(ctx); e != nil {
		return e
	}
	for _, img := range []string{a.Lock.ArgoImage, a.Lock.RedisImage} {
		if e := a.loadNodeImage(ctx, img); e != nil {
			return e
		}
	}
	ns, e := a.get(ctx, "namespace", "", "argocd")
	if e != nil {
		return e
	}
	if ns == nil {
		if e = a.create(ctx, object("Namespace", "", "argocd")); e != nil {
			return e
		}
	}
	// Only an empty Secret is created. Values are initialized by Argo at runtime,
	// are never rendered into Git, and are never read back by this program.
	secret, e := a.kube(ctx, nil, "get", "secret", "argocd-secret", "-n", "argocd", "--ignore-not-found=true", "-o", "name")
	if e != nil {
		return e
	}
	if strings.TrimSpace(string(secret)) == "" {
		if e = a.create(ctx, object("Secret", "argocd", "argocd-secret")); e != nil {
			return e
		}
	}
	if _, e = a.kube(ctx, files["bootstrap/seed.yaml"], "apply", "--server-side", "--field-manager=atlas-refactor-bootstrap", "-f", "-"); e != nil {
		return e
	}
	if _, e = a.kube(ctx, nil, "wait", "--for=condition=Established", "crd/applications.argoproj.io", "crd/appprojects.argoproj.io", "--timeout=120s"); e != nil {
		return e
	}
	if _, e = a.kube(ctx, nil, "rollout", "status", "deployment/atlas-refactor-argocd-server", "-n", "argocd", "--timeout=180s"); e != nil {
		return e
	}
	project, e := a.get(ctx, "appproject", "argocd", "atlas-bootstrap")
	if e != nil {
		return e
	}
	if project == nil {
		if e = a.create(ctx, a.project("atlas-bootstrap", false)); e != nil {
			return e
		}
	} else if !specMatches(project, a.project("atlas-bootstrap", false)) {
		return errors.New("bootstrap project drift")
	}
	id, e := a.get(ctx, "configmap", "kube-system", "atlas-refactor-identity")
	if e != nil {
		return e
	}
	if !dataMatches(id, a.identity()) {
		return errors.New("identity changed before handoff")
	}
	// Persist before Root creation. An uncertain result denies Seed retries.
	if e = a.create(ctx, configMap("kube-system", "atlas-refactor-handoff", map[string]string{"identityUID": id.Metadata.UID, "fingerprint": a.fingerprint()})); e != nil {
		return e
	}
	return a.create(ctx, a.rootApplication())
}

func (a *App) verifyNodes(ctx context.Context) error {
	b, e := a.run(ctx, "kind", "get", "nodes", "--name", a.Config.Cluster)
	if e != nil {
		return e
	}
	expected := a.Config.Cluster + "-control-plane"
	if strings.TrimSpace(string(b)) != expected {
		return errors.New("only the owned single-control-plane topology is permitted")
	}
	imageID, e := a.run(ctx, "docker", "image", "inspect", a.Lock.NodeImage, "--format", "{{.Id}}")
	if e != nil {
		return e
	}
	b, e = a.run(ctx, "docker", "inspect", expected, "--format", "{{.Image}} {{.State.Running}}")
	if e != nil {
		return e
	}
	if strings.TrimSpace(string(imageID)) == "" || strings.TrimSpace(string(b)) != strings.TrimSpace(string(imageID))+" true" {
		return errors.New("Kind node image or runtime drift")
	}
	b, e = a.kube(ctx, nil, "get", "nodes", "-o", "json")
	if e != nil {
		return e
	}
	var list struct {
		Items []struct {
			Metadata struct{ Name string }
			Status   struct {
				Conditions []struct{ Type, Status string }
				NodeInfo   struct{ Architecture, OperatingSystem, KubeletVersion string }
			}
		}
	}
	if e = decode(b, &list); e != nil {
		return e
	}
	if len(list.Items) != 1 || list.Items[0].Metadata.Name != expected {
		return errors.New("Kubernetes node inventory drift")
	}
	info := list.Items[0].Status.NodeInfo
	if info.Architecture != "arm64" || info.OperatingSystem != "linux" || info.KubeletVersion != "v"+a.Lock.Kubernetes {
		return errors.New("Kubernetes node platform or version drift")
	}
	for _, c := range list.Items[0].Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			return nil
		}
	}
	return errors.New("control-plane node is not Ready")
}
