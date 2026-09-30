package platform

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
)

const scopeRegistryPath = inputDir + "/kubernetes-api-scope.json"

type ResourceScope string

const (
	ClusterScope    ResourceScope = "Cluster"
	NamespacedScope ResourceScope = "Namespaced"
)

type GVK struct{ APIVersion, Kind string }
type ResourceType struct {
	Scope  ResourceScope
	Schema Object // CRD structural subset only; not Kubernetes admission validation.
}
type ScopeRegistry struct {
	Schema     int    `json:"schema"`
	Kubernetes string `json:"kubernetes"`
	Source     struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	} `json:"source"`
	Resources []struct {
		APIVersion string        `json:"apiVersion"`
		Kind       string        `json:"kind"`
		Scope      ResourceScope `json:"scope"`
	} `json:"resources"`
}
type ResourceModel struct{ types map[GVK]ResourceType }

func validScope(s ResourceScope) bool { return s == ClusterScope || s == NamespacedScope }

// The registry is a reviewed, version-bound projection of upstream OpenAPI.
// CRDs supply their own served versions, scope and structural schemas. An
// absent version is unknown, never implicitly Namespaced.
func newResourceModel(reg ScopeRegistry, kubernetes string, objects []Object) (*ResourceModel, error) {
	if reg.Schema != 1 || reg.Kubernetes != kubernetes || len(reg.Resources) == 0 ||
		reg.Source.URL != "https://raw.githubusercontent.com/kubernetes/kubernetes/v"+kubernetes+"/api/openapi-spec/swagger.json" ||
		!regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(reg.Source.SHA256) {
		return nil, errors.New("invalid Kubernetes scope registry or toolchain version mismatch")
	}
	m := &ResourceModel{types: map[GVK]ResourceType{}}
	for _, x := range reg.Resources {
		if x.APIVersion == "" || x.Kind == "" || !validScope(x.Scope) {
			return nil, errors.New("invalid scope registry entry")
		}
		key := GVK{x.APIVersion, x.Kind}
		if _, ok := m.types[key]; ok {
			return nil, fmt.Errorf("duplicate scope registry GVK: %v", key)
		}
		m.types[key] = ResourceType{Scope: x.Scope}
	}
	crds := map[string]bool{}
	for _, o := range objects {
		if o["apiVersion"] != "apiextensions.k8s.io/v1" || o["kind"] != "CustomResourceDefinition" {
			continue
		}
		s := mapping(o["spec"])
		g, k := str(s["group"]), str(field(s, "names", "kind"))
		scope := ResourceScope(str(s["scope"]))
		if g == "" || k == "" || !validScope(scope) || len(slice(s["versions"])) == 0 {
			return nil, fmt.Errorf("invalid CRD scope/versions: %s", identity(o))
		}
		if crds[g+"/"+k] {
			return nil, fmt.Errorf("conflicting CRD definition: %s/%s", g, k)
		}
		crds[g+"/"+k] = true
		seen := map[string]bool{}
		served := 0
		for _, raw := range slice(s["versions"]) {
			v := mapping(raw)
			name := str(v["name"])
			if name == "" || seen[name] {
				return nil, errors.New("invalid/duplicate CRD version")
			}
			seen[name] = true
			isServed, ok := v["served"].(bool)
			if !ok {
				return nil, errors.New("CRD version must declare served")
			}
			key := GVK{g + "/" + name, k}
			if _, ok := m.types[key]; ok {
				return nil, fmt.Errorf("CRD overrides registered GVK: %v", key)
			}
			if !isServed {
				continue
			}
			schema := mapping(field(v, "schema", "openAPIV3Schema"))
			if len(schema) == 0 {
				return nil, fmt.Errorf("served CRD version lacks structural schema: %v", key)
			}
			m.types[key] = ResourceType{scope, schema}
			served++
		}
		if served == 0 {
			return nil, errors.New("CRD has no served version")
		}
	}
	return m, nil
}

func (p *Project) resourceModel(objects []Object) (*ResourceModel, error) {
	var reg ScopeRegistry
	if e := readJSON(filepath.Join(p.Root, scopeRegistryPath), &reg); e != nil {
		return nil, e
	}
	return newResourceModel(reg, p.Lock.Kubernetes, objects)
}

func (m *ResourceModel) Resolve(o Object) (ResourceType, error) {
	key := GVK{str(o["apiVersion"]), str(o["kind"])}
	r, ok := m.types[key]
	if !ok {
		return ResourceType{}, fmt.Errorf("unknown resource GVK %s/%s; review the locked registry or provide a served CRD", key.APIVersion, key.Kind)
	}
	return r, nil
}

func (m *ResourceModel) Validate(o Object, defaultNamespace string) error {
	r, e := m.Resolve(o)
	if e != nil {
		return e
	}
	ns := str(metadata(o)["namespace"])
	if raw, exists := metadata(o)["namespace"]; exists && !isString(raw) {
		return fmt.Errorf("invalid explicit namespace: %s", identity(o))
	}
	if r.Scope == ClusterScope && ns != "" {
		return fmt.Errorf("cluster resource has a namespace: %s", identity(o))
	}
	if r.Scope == NamespacedScope && ns == "" && defaultNamespace == "" {
		return fmt.Errorf("namespaced resource has no destination: %s", identity(o))
	}
	if r.Schema != nil {
		if e := validateSchema(o, r.Schema, "$"); e != nil {
			return fmt.Errorf("%s: %w", identity(o), e)
		}
	}
	return nil
}

func (p *Project) capabilityResourceModel(names []string) (*ResourceModel, error) {
	paths := []string{}
	for name, path := range p.Config.Components {
		if _, capability := p.Capabilities.Catalog.Components[name]; !capability {
			paths = append(paths, path)
		}
	}
	for _, name := range names {
		x, ok := p.Capabilities.Catalog.Components[name]
		if !ok {
			return nil, fmt.Errorf("unknown capability %s", name)
		}
		paths = append(paths, x.Path)
	}
	sort.Strings(paths)
	objects := []Object{}
	for _, path := range paths {
		objectsInPath, e := p.componentObjects(path)
		if e != nil {
			return nil, e
		}
		objects = append(objects, objectsInPath...)
	}
	return p.resourceModel(objects)
}

func isString(v any) bool { _, ok := v.(string); return ok }
