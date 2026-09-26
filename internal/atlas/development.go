package atlas

import (
	"atlas-refactor/internal/platform"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const developmentSeed = "platform/development/bootstrap/argocd-self-seed.yaml"
const ciliumSeed = "platform/development/bootstrap/cilium-seed.yaml"
const developmentSignal = "gitops/platform/management/argocd-self/overlays/development/signal.json"

type developmentBundle struct {
	files               map[string][]byte
	fingerprint         string
	images              []string
	apps                map[string]Object
	root, project, kind Object
}

func (c Config) developmentProfile() bool { return c.Schema == 2 }

func (a *App) prepareDevelopment() error {
	p, e := platform.Load(a.Root, platform.Tools{Helm: "helm", Kubectl: "kubectl", YQ: "yq"})
	if e != nil {
		return e
	}
	for path, artifact := range p.Lock.Artifacts {
		b, err := readFile(a.Root, path)
		if err != nil {
			return err
		}
		if digest(b) != artifact.SHA256 {
			return fmt.Errorf("development artifact digest mismatch: %s", path)
		}
	}
	if e = p.VerifyArtifacts(); e != nil {
		return e
	}
	if a.Config.GitOpsPath != p.Config.RootPath || a.Config.RepositoryURL != p.Config.RepositoryURL || a.Config.Revision != p.Config.Revision {
		return errors.New("development Bootstrap and GitOps configurations disagree")
	}
	if p.Lock.Images["node"] != a.Lock.NodeImage || p.Lock.Images["argo"] != a.Lock.ArgoImage || p.Lock.Images["redis"] != a.Lock.RedisImage {
		return errors.New("development and base image locks disagree")
	}
	files, e := p.BundleFiles()
	if e != nil {
		return e
	}
	// Runtime reads must obey the same symlink boundary as normal Bootstrap.
	for path, want := range files {
		b, e := readFile(a.Root, path)
		if e != nil {
			return e
		}
		if digest(b) != digest(want) {
			return errors.New("development bundle changed during preparation")
		}
	}
	d := &developmentBundle{files: files, fingerprint: platform.BundleDigest(files), images: platform.ImageList(p.Lock.Images), apps: map[string]Object{}}
	for path, dst := range map[string]*Object{"platform/development/bootstrap/root.json": &d.root, "platform/development/bootstrap/project.json": &d.project, "platform/development/bootstrap/kind.json": &d.kind} {
		if e = decode(files[path], dst); e != nil {
			return e
		}
	}
	if e = validateDevelopmentKind(files["platform/development/bootstrap/kind.json"]); e != nil {
		return e
	}
	for _, path := range []string{"gitops/root/overlays/development/resources.json", "gitops/platform/applications/overlays/development/resources.json", "gitops/workloads/applications/overlays/development/resources.json"} {
		objects, e := platform.DecodeJSONManifests(files[path])
		if e != nil {
			return e
		}
		for _, obj := range objects {
			o := Object(obj)
			meta, _ := o["metadata"].(map[string]any)
			name, _ := meta["name"].(string)
			if o["kind"] != "Application" || name == "" || d.apps[name] != nil {
				return errors.New("invalid development Application inventory")
			}
			d.apps[name] = o
		}
	}
	if len(d.apps) != 14 || d.apps["cilium"] == nil || d.apps["argocd-self"] == nil {
		return errors.New("incomplete development control graph")
	}
	if a.development != nil && a.development.fingerprint != d.fingerprint {
		return errors.New("development bundle changed during command")
	}
	a.development = d
	return nil
}
func (a *App) lockedImages() []string {
	if a.development != nil {
		return a.development.images
	}
	return []string{a.Lock.NodeImage, a.Lock.ArgoImage, a.Lock.RedisImage}
}
func (a *App) seedFile() string {
	if a.development != nil {
		return developmentSeed
	}
	return "bootstrap/seed.yaml"
}
func (a *App) selfApplication() Object {
	if a.development != nil {
		return a.development.apps["argocd-self"]
	}
	return a.application("argocd-self", "platform-project", a.Config.GitOpsPath+"/platform/argocd", "0")
}
func (a *App) developmentRender() map[string][]byte {
	files := map[string][]byte{}
	for path, b := range a.development.files {
		files[path] = b
	}
	signal := configMap("argocd", "atlas-refactor-adoption-signal", map[string]string{"fingerprint": a.fingerprint()})
	signal["metadata"].(Object)["annotations"] = Object{"argocd.argoproj.io/sync-options": "Prune=confirm,Delete=false"}
	files[developmentSignal] = jsonBytes(signal)
	return files
}
func (a *App) prepareCilium(ctx context.Context, files map[string][]byte) error {
	if e := a.verifySubstrate(ctx, false); e != nil {
		return e
	}
	for _, im := range a.lockedImages() {
		if im != a.Lock.NodeImage {
			if e := a.loadNodeImage(ctx, im); e != nil {
				return e
			}
		}
	}
	if _, e := a.kube(ctx, files[ciliumSeed], "apply", "--server-side", "--field-manager=atlas-refactor-bootstrap", "-f", "-"); e != nil {
		return fmt.Errorf("Cilium Seed: %w", e)
	}
	for _, target := range []string{"daemonset/cilium", "deployment/cilium-operator"} {
		if _, e := a.kube(ctx, nil, "rollout", "status", target, "-n", "kube-system", "--timeout=180s"); e != nil {
			return e
		}
	}
	if _, e := a.kube(ctx, nil, "wait", "--for=condition=Ready", "node/"+a.Config.Cluster+"-control-plane", "--timeout=120s"); e != nil {
		return e
	}
	return a.verifyNodes(ctx)
}
func (a *App) developmentHandoffComplete(ctx context.Context, root, self, signal *Live) (bool, error) {
	if signal == nil || !ready(root) || !ready(self) {
		return false, nil
	}
	commit, e := a.sourceCommit(ctx)
	if e != nil {
		return false, e
	}
	if root.Status.Sync.Revision != commit {
		return false, nil
	}
	liveApps := map[string]*Live{"argocd-self": self}
	names := make([]string, 0, len(a.development.apps))
	for name := range a.development.apps {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		live := liveApps[name]
		if live == nil {
			live, e = a.get(ctx, "application", "argocd", name)
			if e != nil {
				return false, e
			}
		}
		if !specMatches(live, a.development.apps[name]) || !ready(live) || live.Status.Sync.Revision != commit {
			return false, nil
		}
		liveApps[name] = live
	}
	for name, path := range map[string]string{"argocd-self": developmentSeed, "cilium": ciliumSeed} {
		owned, e := a.seedPayloadOwned(ctx, liveApps[name], commit, a.development.files[path])
		if e != nil || !owned {
			return false, e
		}
	}
	return true, nil
}
func (a *App) developmentIdentity(d map[string]string) {
	d["schema"] = "atlas-refactor/identity/v2-development"
	d["substrateProfile"] = "kind-cilium-ipv4-development/v1"
	d["platformSHA256"] = a.development.fingerprint
}
func (a *App) initialWait() string {
	if a.development != nil {
		return "0s"
	}
	return "120s"
}
func (a *App) artifactPath(path string) string {
	if a.development != nil {
		return path
	}
	return strings.TrimSuffix(a.Config.GitOpsPath, "/") + "/" + path
}

// Keep runtime substrate exposure within the reviewed single-node local profile.
func validateDevelopmentKind(data []byte) error {
	const expected = `{"apiVersion":"kind.x-k8s.io/v1alpha4","kind":"Cluster","networking":{"ipFamily":"ipv4","apiServerAddress":"127.0.0.1","disableDefaultCNI":true,"kubeProxyMode":"iptables"},"nodes":[{"role":"control-plane","extraPortMappings":[{"containerPort":30080,"hostPort":8080,"listenAddress":"127.0.0.1","protocol":"TCP"},{"containerPort":30443,"hostPort":8443,"listenAddress":"127.0.0.1","protocol":"TCP"}]}]}`
	var got, want Object
	if err := strictJSON(data, &got); err != nil {
		return err
	}
	if err := strictJSON([]byte(expected), &want); err != nil {
		return err
	}
	if !reflect.DeepEqual(got, want) {
		return errors.New("development Kind configuration exceeds the reviewed local profile")
	}
	return nil
}
