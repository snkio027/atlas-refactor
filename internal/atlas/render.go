package atlas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type App struct {
	Root           string
	Config         Config
	Lock           Lock
	Runner         Runner
	resolvedCommit string
	development    *developmentBundle
	installation   *InstallationBinding
	// In-package synthetic archive fixtures pin their own immutable snapshot.
	// This is never populated from config, flags, environment, or repository data.
	fixtureSnapshotDigest string
}
type Object map[string]any

func decode(b []byte, v any) error { return json.Unmarshal(b, v) }
func object(kind, namespace, name string) Object {
	api := "v1"
	if kind == "Application" || kind == "AppProject" {
		api = "argoproj.io/v1alpha1"
	}
	meta := Object{"name": name}
	if namespace != "" {
		meta["namespace"] = namespace
	}
	return Object{"apiVersion": api, "kind": kind, "metadata": meta}
}
func (a *App) identity() map[string]string {
	d := a.Config.Identity()
	d["lockSHA256"] = digest(jsonBytes(a.Lock))
	d["substrateProfile"] = "kind-ipv4-audit/v1"
	if a.development != nil {
		a.developmentIdentity(d)
	}
	return d
}
func (a *App) fingerprint() string { return digest(jsonBytes(a.identity())) }
func configMap(namespace, name string, data map[string]string) Object {
	o := object("ConfigMap", namespace, name)
	o["data"] = data
	o["immutable"] = true
	return o
}

func (a *App) application(name, project, path, wave string) Object {
	o := object("Application", "argocd", name)
	o["metadata"].(Object)["annotations"] = Object{"argocd.argoproj.io/sync-wave": wave, "argocd.argoproj.io/sync-options": "Prune=confirm,Delete=false"}
	options := []string{"ServerSideApply=true"}
	// The first self-sync must apply matching Seed objects to establish tracking
	// and field ownership. Selective sync skips those already-identical objects.
	if name != "argocd-self" {
		options = append(options, "ApplyOutOfSyncOnly=true")
	}
	o["spec"] = Object{"project": project, "source": Object{"repoURL": a.Config.RepositoryURL, "targetRevision": a.Config.Revision, "path": path}, "destination": Object{"server": "https://kubernetes.default.svc", "namespace": "argocd"}, "syncPolicy": Object{"automated": Object{"enabled": true, "prune": true, "selfHeal": true}, "syncOptions": options}}
	return o
}
func (a *App) rootApplication() Object {
	if a.development != nil {
		return a.development.root
	}
	o := a.application("atlas-refactor-root", "atlas-bootstrap", a.Config.GitOpsPath+"/root", "0")
	delete(o["metadata"].(Object), "annotations")
	return o
}
func (a *App) project(name string, platform bool) Object {
	if a.development != nil && name == "atlas-bootstrap" {
		return a.development.project
	}
	o := object("AppProject", "argocd", name)
	namespaced := []Object{{"group": "argoproj.io", "kind": "Application"}}
	if !platform {
		namespaced = append(namespaced, Object{"group": "argoproj.io", "kind": "AppProject"})
	} else {
		for _, kind := range []string{"ConfigMap", "Secret", "Service", "ServiceAccount"} {
			namespaced = append(namespaced, Object{"group": "", "kind": kind})
		}
		for _, k := range []struct{ g, k string }{{"apps", "Deployment"}, {"apps", "StatefulSet"}, {"batch", "Job"}, {"networking.k8s.io", "NetworkPolicy"}, {"rbac.authorization.k8s.io", "Role"}, {"rbac.authorization.k8s.io", "RoleBinding"}} {
			namespaced = append(namespaced, Object{"group": k.g, "kind": k.k})
		}
	}
	spec := Object{"sourceRepos": []string{a.Config.RepositoryURL}, "destinations": []Object{{"namespace": "argocd", "server": "https://kubernetes.default.svc"}}, "namespaceResourceWhitelist": namespaced, "clusterResourceWhitelist": []Object{}}
	if platform {
		spec["clusterResourceWhitelist"] = []Object{{"group": "apiextensions.k8s.io", "kind": "CustomResourceDefinition"}, {"group": "rbac.authorization.k8s.io", "kind": "ClusterRole"}, {"group": "rbac.authorization.k8s.io", "kind": "ClusterRoleBinding"}}
	}
	o["spec"] = spec
	return o
}

func (a *App) VerifyArtifacts() error {
	if e := a.Config.Validate(); e != nil {
		return e
	}
	if a.Config.Schema == 4 {
		if e := a.prepareInstallation(); e != nil {
			return e
		}
	} else if a.Config.developmentProfile() {
		if e := a.prepareDevelopment(); e != nil {
			return e
		}
	}
	checks := map[string]string{a.Lock.Chart: a.Lock.ChartSHA256}
	for p, h := range a.Lock.Assets {
		checks[p] = h
	}
	for p, want := range checks {
		b, e := readFile(a.Root, p)
		if e != nil {
			return e
		}
		if digest(b) != want {
			return fmt.Errorf("artifact checksum mismatch: %s", p)
		}
	}
	return nil
}

func (a *App) Render(ctx context.Context) (map[string][]byte, error) {
	if e := a.VerifyArtifacts(); e != nil {
		return nil, e
	}
	if a.development != nil {
		return a.developmentRender(), nil
	}
	if e := a.verifyTools(ctx, false); e != nil {
		return nil, e
	}
	seed, e := a.run(ctx, "helm", "template", "atlas-refactor-argocd", a.Lock.Chart, "--namespace", "argocd", "--include-crds", "--skip-tests", "--kube-version", a.Lock.Kubernetes, "--values", "assets/argocd-values.yaml", "--set", "configs.secret.createSecret=false")
	if e != nil {
		return nil, e
	}
	if !strings.Contains(string(seed), "name: applications.argoproj.io") || !strings.Contains(string(seed), a.Lock.ArgoImage) || !strings.Contains(string(seed), a.Lock.RedisImage) {
		return nil, errors.New("Seed is missing CRDs or locked images")
	}
	if regexp.MustCompile(`(?m)^kind: (Secret|Application|AppProject)$`).Match(seed) {
		return nil, errors.New("Seed must not contain Secret data or control Applications")
	}
	for _, match := range regexp.MustCompile(`(?m)^\s*image:\s*["']?([^\s"']+)`).FindAllSubmatch(seed, -1) {
		s := string(match[1])
		if s != a.Lock.ArgoImage && s != a.Lock.RedisImage {
			return nil, fmt.Errorf("unlocked Seed image: %s", s)
		}
	}
	cm, e := readFile(a.Root, "assets/argocd-cm.yaml")
	if e != nil {
		return nil, e
	}
	// Anchor the complete top-level key: "data:" is also a suffix of "metadata:".
	if strings.Count(string(cm), "\ndata:\n") != 1 {
		return nil, errors.New("shared Argo ConfigMap must have one top-level data mapping")
	}
	cm = []byte(strings.Replace(string(cm), "\ndata:\n", "\ndata:\n  application.resourceTrackingMethod: annotation\n", 1))
	seed = append(seed, []byte("\n---\n")...)
	seed = append(seed, cm...)
	files := map[string][]byte{"platform/argocd/seed.yaml": seed, "bootstrap/seed.yaml": seed, "bootstrap/root.json": jsonBytes(a.rootApplication()), "bootstrap/project.json": jsonBytes(a.project("atlas-bootstrap", false))}
	signal := configMap("argocd", "atlas-refactor-adoption-signal", map[string]string{"fingerprint": a.fingerprint()})
	signal["metadata"].(Object)["annotations"] = Object{"argocd.argoproj.io/sync-options": "Prune=confirm,Delete=false"}
	files["platform/argocd/signal.json"] = jsonBytes(signal)
	files["projects/platform.json"] = jsonBytes(a.project("platform-project", true))
	files["root/projects.json"] = jsonBytes(a.application("project-bootstrap", "atlas-bootstrap", a.Config.GitOpsPath+"/projects", "-20"))
	files["root/platform.json"] = jsonBytes(a.application("platform-control", "platform-project", a.Config.GitOpsPath+"/platform/applications", "-10"))
	files["platform/applications/argocd-self.json"] = jsonBytes(a.application("argocd-self", "platform-project", a.Config.GitOpsPath+"/platform/argocd", "0"))
	for dir, res := range map[string][]string{"root": {"projects.json", "platform.json"}, "projects": {"platform.json"}, "platform/applications": {"argocd-self.json"}, "platform/argocd": {"seed.yaml", "signal.json"}} {
		files[dir+"/kustomization.yaml"] = jsonBytes(Object{"apiVersion": "kustomize.config.k8s.io/v1beta1", "kind": "Kustomization", "resources": res})
	}
	return files, nil
}

func WriteFiles(root, out string, files map[string][]byte) error {
	keys := make([]string, 0, len(files))
	for p := range files {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, p := range keys {
		path, e := safePath(root, filepath.Join(out, p))
		if e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return e
		}
		f, e := os.CreateTemp(filepath.Dir(path), ".atlas-write-*")
		if e != nil {
			return e
		}
		name := f.Name()
		_, e = f.Write(files[p])
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(name, path)
		}
		if e != nil {
			os.Remove(name)
			return e
		}
	}
	return nil
}

func (a *App) Doctor(ctx context.Context) error {
	if e := a.VerifyArtifacts(); e != nil {
		return e
	}
	if e := a.verifyTools(ctx, true); e != nil {
		return e
	}
	if a.Config.fourNodeProfile() {
		if e := a.verifyImageArchives(); e != nil {
			return e
		}
	}
	for _, img := range a.lockedImages() {
		b, e := a.run(ctx, "docker", "image", "inspect", "--platform", "linux/arm64", img, "--format", "{{.Os}}/{{.Architecture}}")
		if e != nil {
			return fmt.Errorf("locked image unavailable locally: %s", img)
		}
		if strings.TrimSpace(string(b)) != "linux/arm64" {
			return fmt.Errorf("locked image must provide the supported linux/arm64 platform: %s", img)
		}
	}
	return nil
}
