package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const capabilityDir = inputDir + "/capabilities"
const platformApplications = "gitops/platform/applications/overlays/development/resources.json"
const platformProjects = "gitops/platform/management/projects/overlays/development/resources.json"

type SecretRequirement struct {
	Namespace string   `json:"namespace"`
	Name      string   `json:"name"`
	Keys      []string `json:"keys"`
}
type Capability struct {
	Path            string              `json:"path"`
	Namespace       string              `json:"namespace"`
	Wave            int                 `json:"wave"`
	DependsOn       []string            `json:"dependsOn"`
	RequiredSecrets []SecretRequirement `json:"requiredSecrets"`
}
type Partition struct {
	Component string   `json:"component"`
	Kinds     []string `json:"kinds"`
}
type CapabilityJob struct {
	ProjectedTokenMonitors []string    `json:"projectedTokenMonitors"`
	Release                string      `json:"release"`
	Chart                  string      `json:"chart"`
	Namespace              string      `json:"namespace"`
	Values                 []string    `json:"values"`
	ExcludeComponents      []string    `json:"excludeComponents"`
	Outputs                []Partition `json:"outputs"`
}
type CapabilityCatalog struct {
	Schema     int                   `json:"schema"`
	Components map[string]Capability `json:"components"`
	Jobs       []CapabilityJob       `json:"jobs"`
}
type EnabledCapabilities struct {
	Schema       int      `json:"schema"`
	Capabilities []string `json:"capabilities"`
}
type Capabilities struct {
	Catalog CapabilityCatalog
	Enabled EnabledCapabilities
	Lock    Lock
	Active  []string
}

var capabilityName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func localPath(path, prefix string) bool {
	return strings.HasPrefix(path, prefix) && filepath.Clean(path) == path && !strings.Contains(path, "..") && !strings.ContainsAny(path, ":\\")
}
func (p *Project) loadCapabilities() error {
	c := &Capabilities{}
	for path, dst := range map[string]any{"catalog.json": &c.Catalog, "enabled.json": &c.Enabled, "versions.lock.json": &c.Lock} {
		if e := readJSON(filepath.Join(p.Root, capabilityDir, path), dst); e != nil {
			return e
		}
	}
	if c.Catalog.Schema != 1 || c.Enabled.Schema != 1 || c.Lock.Schema != 1 {
		return errors.New("unsupported capability schema")
	}
	paths := map[string]bool{}
	for name, x := range c.Catalog.Components {
		if !capabilityName.MatchString(name) || p.Config.Components[name] != "" || !capabilityName.MatchString(x.Namespace) || !localPath(x.Path, "gitops/platform/") || !strings.HasSuffix(x.Path, "/overlays/development") || paths[x.Path] || x.Path == filepath.Dir(platformApplications) || x.Path == filepath.Dir(platformProjects) || x.Wave < -100 || x.Wave > 90 {
			return fmt.Errorf("invalid Tier-1 capability: %s", name)
		}
		for _, core := range p.Config.Components {
			if x.Path == core {
				return errors.New("capability shadows a core owner")
			}
		}
		paths[x.Path] = true
	}
	p.Capabilities = c
	all := []string{}
	for name := range c.Catalog.Components {
		all = append(all, name)
	}
	if _, e := p.ResolveCapabilities(all); e != nil {
		return e
	}
	var e error
	c.Active, e = p.ResolveCapabilities(c.Enabled.Capabilities)
	if e != nil {
		return e
	}
	produced := map[string]bool{}
	for _, j := range c.Catalog.Jobs {
		if !capabilityName.MatchString(j.Release) || !capabilityName.MatchString(j.Namespace) {
			return errors.New("invalid chart release/namespace")
		}
		if _, ok := c.Lock.Artifacts[j.Chart]; !ok || !localPath(j.Chart, "vendor/platform/") {
			return errors.New("unlocked capability chart")
		}
		for _, v := range j.Values {
			if !localPath(v, capabilityDir+"/values/") {
				return errors.New("invalid values path")
			}
		}
		for _, n := range j.ExcludeComponents {
			if !produced[n] {
				return errors.New("render exclusion must refer to a previous output")
			}
		}
		for _, part := range j.Outputs {
			if _, ok := c.Catalog.Components[part.Component]; !ok || produced[part.Component] {
				return errors.New("invalid/duplicate render owner")
			}
			produced[part.Component] = true
		}
	}
	for path, a := range c.Lock.Artifacts {
		if !localPath(path, "vendor/platform/") {
			return errors.New("invalid capability artifact")
		}
		b, e := os.ReadFile(filepath.Join(p.Root, path))
		if e != nil {
			return e
		}
		if hash(b) != a.SHA256 {
			return fmt.Errorf("capability artifact checksum mismatch: %s", path)
		}
	}
	for source, im := range c.Lock.Images {
		if !pinnedImage.MatchString(im) || !strings.HasPrefix(im, source+"@sha256:") {
			return errors.New("invalid capability image pin")
		}
	}
	if c.Lock.Helm != p.Lock.Helm || c.Lock.Kubectl != p.Lock.Kubectl || c.Lock.YQ != p.Lock.YQ || c.Lock.Kubernetes != p.Lock.Kubernetes {
		return errors.New("capability toolchain differs from core")
	}
	for _, name := range c.Active {
		p.Config.Components[name] = c.Catalog.Components[name].Path
	}
	// Only active capabilities expand the substrate's offline image set. All
	// candidate images are independently verified by CheckCapabilities.
	if len(c.Active) > 0 {
		for k, v := range c.Lock.Images {
			p.Lock.Images["capability:"+k] = v
		}
		for k, v := range c.Lock.Artifacts {
			p.Lock.Artifacts[k] = v
		}
	}
	return nil
}

// ResolveCapabilities computes a stable dependency closure. A cycle or unknown
// capability fails before any output is written. Wave order encodes readiness.
func (p *Project) ResolveCapabilities(requested []string) ([]string, error) {
	state := map[string]int{}
	out := []string{}
	var visit func(string) error
	visit = func(name string) error {
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			return fmt.Errorf("capability dependency cycle at %s", name)
		}
		x, ok := p.Capabilities.Catalog.Components[name]
		if !ok {
			return fmt.Errorf("unknown capability %s", name)
		}
		state[name] = 1
		for _, dep := range x.DependsOn {
			if strings.HasPrefix(dep, "core:") {
				if p.Config.Components[strings.TrimPrefix(dep, "core:")] == "" {
					return fmt.Errorf("unknown core dependency %s", dep)
				}
				continue
			}
			d, ok := p.Capabilities.Catalog.Components[dep]
			if !ok || d.Wave >= x.Wave {
				return fmt.Errorf("%s dependency %s must have an earlier readiness wave", name, dep)
			}
			if e := visit(dep); e != nil {
				return e
			}
		}
		state[name] = 2
		out = append(out, name)
		return nil
	}
	for _, n := range requested {
		if e := visit(n); e != nil {
			return nil, e
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := p.Capabilities.Catalog.Components[out[i]], p.Capabilities.Catalog.Components[out[j]]
		if a.Wave != b.Wave {
			return a.Wave < b.Wave
		}
		return out[i] < out[j]
	})
	return out, nil
}

func (p *Project) CapabilityPlan(requested []string) (Object, error) {
	names, e := p.ResolveCapabilities(requested)
	if e != nil {
		return nil, e
	}
	steps := []any{}
	for _, n := range names {
		x := p.Capabilities.Catalog.Components[n]
		steps = append(steps, Object{"name": n, "wave": x.Wave, "namespace": x.Namespace, "path": x.Path})
	}
	missing, e := p.missingCapabilitySecrets(names)
	if e != nil {
		return nil, e
	}
	projection, e := p.CapabilityActivation(names)
	if e != nil {
		return nil, e
	}
	projects, e := decodeObjects(projection[platformProjects])
	if e != nil {
		return nil, e
	}
	coreBytes, e := os.ReadFile(filepath.Join(p.Root, capabilityDir, "core-projects.json"))
	if e != nil {
		return nil, e
	}
	core, e := decodeObjects(coreBytes)
	if e != nil {
		return nil, e
	}
	changes := Object{}
	for _, key := range []string{"destinations", "clusterResourceWhitelist", "namespaceResourceWhitelist"} {
		before := slice(field(core[0], "spec", key))
		after := slice(field(projects[0], "spec", key))
		changes[key] = after[len(before):]
	}
	return Object{"projectAdditions": changes, "requested": requested, "resolved": steps, "missingSealedSecrets": missing, "readyToEnable": len(missing) == 0, "mutationAuthority": "Argo CD / platform-project", "clusterOperations": 0}, nil
}

func (p *Project) missingCapabilitySecrets(names []string) ([]string, error) {
	b, e := os.ReadFile(filepath.Join(p.Root, capabilityDir, "resources/platform-credentials.json"))
	if e != nil {
		return nil, e
	}
	objects, e := decodeObjects(b)
	if e != nil {
		return nil, e
	}
	present := map[string]Object{}
	for _, o := range objects {
		if o["kind"] != "SealedSecret" || o["apiVersion"] != "bitnami.com/v1alpha1" {
			return nil, errors.New("credential input may contain only SealedSecrets")
		}
		ns, name := str(metadata(o)["namespace"]), str(metadata(o)["name"])
		if ns == "" || name == "" || field(o, "spec", "template", "metadata", "namespace") != ns || field(o, "spec", "template", "metadata", "name") != name {
			return nil, errors.New("SealedSecret must bind its exact namespace and name")
		}
		if field(o, "spec", "template", "data") != nil || field(o, "spec", "template", "stringData") != nil {
			return nil, errors.New("plaintext Secret template data prohibited")
		}
		for _, key := range []string{"sealedsecrets.bitnami.com/cluster-wide", "sealedsecrets.bitnami.com/namespace-wide"} {
			if field(o, "metadata", "annotations", key) != nil {
				return nil, errors.New("only strict-scope SealedSecrets are permitted")
			}
		}
		id := ns + "/" + name
		if present[id] != nil {
			return nil, errors.New("duplicate credential identity")
		}
		present[id] = o
	}
	missing := []string{}
	seen := map[string]bool{}
	for _, n := range names {
		for _, s := range p.Capabilities.Catalog.Components[n].RequiredSecrets {
			id := s.Namespace + "/" + s.Name
			for _, k := range s.Keys {
				cipher, decodeErr := base64.StdEncoding.DecodeString(str(field(present[id], "spec", "encryptedData", k)))
				validCipher := decodeErr == nil && len(cipher) > 2 && int(binary.BigEndian.Uint16(cipher[:2])) >= 256 && len(cipher) > 2+int(binary.BigEndian.Uint16(cipher[:2]))+16
				if !validCipher && !seen[id+"/"+k] {
					missing = append(missing, id+"/"+k)
					seen[id+"/"+k] = true
				}
			}
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func (p *Project) RenderCapabilities(ctx context.Context) (map[string][]byte, error) {
	if e := p.verifyTools(ctx); e != nil {
		return nil, e
	}
	c := p.Capabilities
	files := map[string][]byte{}
	owners := map[string][]Object{}
	// Replace only exact upstream image references; unknown images fail below.
	pin := func(s string) string {
		for old, v := range c.Lock.Images {
			if s == old {
				return v
			}
			if strings.HasSuffix(s, "="+old) {
				return strings.TrimSuffix(s, old) + v
			}
		}
		return s
	}
	var walk func(any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case string:
			return pin(x)
		case map[string]any:
			for k, a := range x {
				x[k] = walk(a)
			}
		case []any:
			for i, a := range x {
				x[i] = walk(a)
			}
		}
		return v
	}
	for _, j := range c.Catalog.Jobs {
		args := []string{"template", j.Release, j.Chart, "--namespace", j.Namespace, "--include-crds", "--skip-tests", "--kube-version", p.Lock.Kubernetes}
		for _, v := range j.Values {
			args = append(args, "--values", v)
		}
		b, e := p.run(ctx, nil, p.Tools.Helm, args...)
		if e != nil {
			return nil, e
		}
		objects, e := p.yaml(ctx, b)
		if e != nil {
			return nil, e
		}
		excluded := map[string]bool{}
		for _, n := range j.ExcludeComponents {
			for _, o := range owners[n] {
				excluded[identity(o)] = true
			}
		}
		for _, o := range objects {
			if o["kind"] == "ServiceMonitor" {
				for _, name := range j.ProjectedTokenMonitors {
					if metadata(o)["name"] == name {
						for _, endpoint := range slice(field(o, "spec", "endpoints")) {
							ep := mapping(endpoint)
							delete(ep, "authorization")
							ep["bearerTokenFile"] = "/var/run/secrets/kubernetes.io/serviceaccount/token"
						}
					}
				}
			}
			if excluded[identity(o)] {
				continue
			}
			if o["kind"] != "CustomResourceDefinition" {
				walk(o)
			}
			// Thanos is outside this capability. Avoid a dormant, mutable fallback image
			// in the Operator arguments; adding Thanos requires its own locked catalog.
			if o["kind"] == "Deployment" {
				for _, container := range slice(field(o, "spec", "template", "spec", "containers")) {
					cc := mapping(container)
					if cc["args"] != nil {
						filtered := []any{}
						for _, a := range slice(cc["args"]) {
							if !strings.HasPrefix(str(a), "--thanos-default-base-image=") {
								filtered = append(filtered, a)
							}
						}
						cc["args"] = filtered
					}
				}
			}
			matched := false
			for _, part := range j.Outputs {
				ok := len(part.Kinds) == 0
				for _, k := range part.Kinds {
					ok = ok || o["kind"] == k
				}
				if ok {
					owners[part.Component] = append(owners[part.Component], o)
					matched = true
					break
				}
			}
			if !matched {
				return nil, errors.New("rendered resource has no capability owner")
			}
		}
	}
	for name, x := range c.Catalog.Components {
		b, e := os.ReadFile(filepath.Join(p.Root, capabilityDir, "resources", name+".json"))
		if e != nil {
			return nil, e
		}
		if _, e := decodeObjects(b); e != nil {
			return nil, e
		}
		files[x.Path+"/resources.json"] = b
		if objects, ok := owners[name]; ok {
			files[x.Path+"/rendered.yaml"] = encoded(objects)
		}
	}
	return files, nil
}

func (p *Project) capabilityObjects(name string) ([]Object, error) {
	x := p.Capabilities.Catalog.Components[name]
	var k Object
	if e := readJSON(filepath.Join(p.Root, x.Path, "kustomization.yaml"), &k); e != nil {
		return nil, e
	}
	if len(k) != 3 || k["kind"] != "Kustomization" {
		return nil, errors.New("capability Kustomization must contain only local resources")
	}
	objects := []Object{}
	for _, v := range slice(k["resources"]) {
		path := str(v)
		if path == "" || filepath.Base(path) != path || strings.Contains(path, ":") {
			return nil, errors.New("nonlocal capability resource")
		}
		b, e := os.ReadFile(filepath.Join(p.Root, x.Path, path))
		if e != nil {
			return nil, e
		}
		o, e := DecodeJSONManifests(b)
		if e != nil {
			return nil, e
		}
		objects = append(objects, o...)
	}
	return objects, nil
}

// CapabilityActivation is a deterministic projection onto the existing
// platform control and canonical project. It never edits Tier-0 or credentials.
func (p *Project) CapabilityActivation(names []string) (map[string][]byte, error) {
	apps, e := os.ReadFile(filepath.Join(p.Root, capabilityDir, "core-applications.json"))
	if e != nil {
		return nil, e
	}
	projects, e := os.ReadFile(filepath.Join(p.Root, capabilityDir, "core-projects.json"))
	if e != nil {
		return nil, e
	}
	if len(names) == 0 {
		return map[string][]byte{platformApplications: apps, platformProjects: projects}, nil
	}
	aa, e := decodeObjects(apps)
	if e != nil {
		return nil, e
	}
	pp, e := decodeObjects(projects)
	if e != nil {
		return nil, e
	}
	project := pp[0]
	if metadata(project)["name"] != "platform-project" {
		return nil, errors.New("invalid core project")
	}
	spec := mapping(project["spec"])
	for _, name := range names {
		x := p.Capabilities.Catalog.Components[name]
		// Declare materialization authority from consumer contracts even before the
		// first ciphertext is available, so the review plan shows the full delta.
		for _, required := range x.RequiredSecrets {
			if !allowed(spec["namespaceResourceWhitelist"], "bitnami.com", "SealedSecret") {
				spec["namespaceResourceWhitelist"] = append(slice(spec["namespaceResourceWhitelist"]), Object{"group": "bitnami.com", "kind": "SealedSecret"})
			}
			found := false
			for _, d := range slice(spec["destinations"]) {
				found = found || field(mapping(d), "namespace") == required.Namespace
			}
			if !found {
				spec["destinations"] = append(slice(spec["destinations"]), Object{"namespace": required.Namespace, "server": "https://kubernetes.default.svc"})
			}
		}
		app := Object{"apiVersion": "argoproj.io/v1alpha1", "kind": "Application", "metadata": Object{"name": name, "namespace": "argocd", "annotations": Object{"argocd.argoproj.io/sync-wave": strconv.Itoa(x.Wave), "argocd.argoproj.io/sync-options": "Prune=confirm,Delete=false"}}, "spec": Object{"project": "platform-project", "source": Object{"repoURL": p.Config.RepositoryURL, "targetRevision": p.Config.Revision, "path": x.Path}, "destination": Object{"server": "https://kubernetes.default.svc", "namespace": x.Namespace}, "syncPolicy": Object{"automated": Object{"enabled": true, "prune": true, "selfHeal": true}, "syncOptions": []any{"ServerSideApply=true", "FailOnSharedResource=true"}}}}
		aa = append(aa, app)
		objects, e := p.capabilityObjects(name)
		if e != nil {
			return nil, e
		}
		for _, o := range objects {
			kind, g := str(o["kind"]), group(o)
			if kind == "Secret" || kind == "Application" || kind == "ApplicationSet" || kind == "AppProject" {
				return nil, fmt.Errorf("capability %s contains prohibited authority/material: %s", name, kind)
			}
			cluster := kind == "Namespace" || kind == "CustomResourceDefinition" || kind == "ClusterRole" || kind == "ClusterRoleBinding" || kind == "MutatingWebhookConfiguration" || kind == "ValidatingWebhookConfiguration"
			key := "namespaceResourceWhitelist"
			if cluster {
				key = "clusterResourceWhitelist"
			}
			if !allowed(spec[key], g, kind) {
				spec[key] = append(slice(spec[key]), Object{"group": g, "kind": kind})
			}
			if !cluster {
				ns := str(metadata(o)["namespace"])
				if ns == "" {
					ns = x.Namespace
				}
				found := false
				for _, d := range slice(spec["destinations"]) {
					found = found || field(mapping(d), "namespace") == ns
				}
				if !found {
					spec["destinations"] = append(slice(spec["destinations"]), Object{"server": "https://kubernetes.default.svc", "namespace": ns})
				}
			}
		}
	}
	encodeList := func(o []Object) []byte {
		b, _ := json.MarshalIndent(Object{"apiVersion": "v1", "kind": "List", "items": o}, "", "  ")
		return append(b, '\n')
	}
	return map[string][]byte{platformApplications: encodeList(aa), platformProjects: encodeList(pp)}, nil
}

func (p *Project) ValidateCapabilityActivation() error {
	missing, e := p.missingCapabilitySecrets(p.Capabilities.Active)
	if e != nil {
		return e
	}
	if len(missing) > 0 {
		return fmt.Errorf("capabilities require sealed credentials: %s", strings.Join(missing, ", "))
	}
	files, e := p.CapabilityActivation(p.Capabilities.Active)
	if e != nil {
		return e
	}
	for path, want := range files {
		b, e := os.ReadFile(filepath.Join(p.Root, path))
		if e != nil {
			return e
		}
		if !bytes.Equal(b, want) {
			return fmt.Errorf("capability activation differs from declared catalog: %s; run platform:render", path)
		}
	}
	return nil
}

func (p *Project) CapabilityInputs() ([]string, error) {
	paths := []string{inputDir + "/bootstrap/baseline-v3.json", capabilityDir + "/catalog.json", capabilityDir + "/enabled.json", capabilityDir + "/versions.lock.json", capabilityDir + "/kubeseal.lock.json", capabilityDir + "/core-applications.json", capabilityDir + "/core-projects.json"}
	for _, j := range p.Capabilities.Catalog.Jobs {
		paths = append(paths, j.Values...)
	}
	for name, x := range p.Capabilities.Catalog.Components {
		paths = append(paths, capabilityDir+"/resources/"+name+".json", x.Path+"/kustomization.yaml")
		var k Object
		if e := readJSON(filepath.Join(p.Root, x.Path, "kustomization.yaml"), &k); e != nil {
			return nil, e
		}
		for _, v := range slice(k["resources"]) {
			s := str(v)
			if s == "" || filepath.Base(s) != s || strings.Contains(s, ":") {
				return nil, errors.New("nonlocal capability input")
			}
			paths = append(paths, x.Path+"/"+s)
		}
	}
	return paths, nil
}

func (p *Project) CheckCapabilities(ctx context.Context) (int, error) {
	files, e := p.RenderCapabilities(ctx)
	if e != nil {
		return 0, e
	}
	for path, want := range files {
		b, e := os.ReadFile(filepath.Join(p.Root, path))
		if e != nil {
			return 0, e
		}
		if !bytes.Equal(b, want) {
			return 0, fmt.Errorf("stale capability render: %s", path)
		}
	}
	cp := *p
	cp.Lock = p.Capabilities.Lock
	inv := map[string]Object{}
	schemas := map[string]Object{}
	names := []string{}
	for n := range p.Capabilities.Catalog.Components {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		objects, e := p.capabilityObjects(n)
		if e != nil {
			return 0, e
		}
		b, e := p.run(ctx, nil, p.Tools.Kubectl, "kustomize", p.Capabilities.Catalog.Components[n].Path)
		if e != nil {
			return 0, e
		}
		built, e := p.yaml(ctx, b)
		if e != nil {
			return 0, e
		}
		if len(built) != len(objects) {
			return 0, errors.New("capability Kustomize inventory mismatch")
		}
		for _, o := range objects {
			id := identity(o)
			if inv[id] != nil {
				return 0, fmt.Errorf("duplicate capability owner: %s", id)
			}
			inv[id] = o
			if e := checkObject(o); e != nil {
				return 0, fmt.Errorf("%s: %w", id, e)
			}
			if o["kind"] == "Application" || o["kind"] == "AppProject" || o["kind"] == "ApplicationSet" {
				return 0, errors.New("capability cannot create an authority tree")
			}
			if o["kind"] == "CustomResourceDefinition" {
				for _, v := range slice(field(o, "spec", "versions")) {
					m := mapping(v)
					schemas[str(field(o, "spec", "group"))+"/"+str(m["name"])+"/"+str(field(o, "spec", "names", "kind"))] = mapping(field(m, "schema", "openAPIV3Schema"))
				}
			} else if e := cp.images(o); e != nil {
				return 0, fmt.Errorf("%s: %w", id, e)
			}
		}
	}
	for _, o := range inv {
		if s := schemas[str(o["apiVersion"])+"/"+str(o["kind"])]; s != nil {
			if e := validateSchema(o, s, "$"); e != nil {
				return 0, fmt.Errorf("%s: %w", identity(o), e)
			}
		}
	}
	if _, e := p.missingCapabilitySecrets(names); e != nil {
		return 0, e
	}
	ordered, e := p.ResolveCapabilities(names)
	if e != nil {
		return 0, e
	}
	activation, e := p.CapabilityActivation(ordered)
	if e != nil {
		return 0, e
	}
	projects, e := decodeObjects(activation[platformProjects])
	if e != nil {
		return 0, e
	}
	for _, project := range projects {
		if e := checkObject(project); e != nil {
			return 0, e
		}
	}
	return len(inv), nil
}
