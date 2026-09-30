package installation

import (
	"archive/tar"
	"atlas-refactor/internal/platform"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return Config{Schema: 1, Cluster: "atlas-independent", Repository: "https://github.com/example/independent.git", Branch: "atlas-development", HTTPPort: 18080, HTTPSPort: 18443, StateDirectory: filepath.Join(root, "state"), BackupDirectory: filepath.Join(root, "backup"), BackupIsolation: "same-host-development-exception"}
}
func testProduct(t *testing.T) Product {
	t.Helper()
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(root, "packaging/tools-darwin-arm64.json"))
	if e != nil {
		t.Fatal(e)
	}
	var locks []Tool
	if e = Decode(b, &locks); e != nil {
		t.Fatal(e)
	}
	p, e := BuildProduct(root, "v0.1.0-dev", strings.Repeat("a", 40), platform.Tools{Helm: "helm", Kubectl: "kubectl", YQ: "yq"}, locks)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestConfigRejectsBoundaryChanges(t *testing.T) {
	c := testConfig(t)
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(*Config){"author repo": func(c *Config) { c.Repository = SourceRepository }, "old profile": func(c *Config) { c.Cluster = "atlas-refactor-test-ot1" }, "credential URL": func(c *Config) { c.Repository = "https://token@github.com/example/a.git" }, "non-GitHub": func(c *Config) { c.Repository = "https://example.com/a/a.git" }, "branch injection": func(c *Config) { c.Branch = "-main" }, "old branch": func(c *Config) { c.Branch = SourceRevision }, "same port": func(c *Config) { c.HTTPSPort = c.HTTPPort }, "backup in state": func(c *Config) { c.BackupDirectory = c.StateDirectory + "/backup" }, "no isolation decision": func(c *Config) { c.BackupIsolation = "" }} {
		t.Run(name, func(t *testing.T) {
			v := c
			change(&v)
			if v.Validate() == nil {
				t.Fatal("accepted unsafe configuration")
			}
		})
	}
}
func TestStrictConfiguration(t *testing.T) {
	for _, s := range []string{`{"schema":1,"schema":1}`, `{"schema":1,"unknown":0}`, `{} {}`, `{"schema":{"a":1,"a":2}}`} {
		var c Config
		if Decode([]byte(s), &c) == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}
func TestPackageProjectionHasNoAuthorCredentials(t *testing.T) {
	p := testProduct(t)
	c := testConfig(t)
	base, e := p.Project(c, false)
	if e != nil {
		t.Fatal(e)
	}
	full, e := p.Project(c, true)
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range []Files{base, full} {
		for path, b := range f {
			if bytes.Contains(b, []byte(SourceRepository)) || bytes.Contains(b, []byte(SourceRevision)) {
				t.Fatalf("author state leaked: %s", path)
			}
		}
	}
	aa, e := applications(base)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"secrets-controller", "secrets-crds", "secrets-foundation"} {
		if aa[name] == nil {
			t.Fatalf("base lacks %s", name)
		}
	}
	for _, name := range []string{"platform-credentials", "monitoring", "object-storage"} {
		if aa[name] != nil {
			t.Fatalf("premature consumer: %s", name)
		}
	}
	aa, e = applications(full)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"monitoring", "object-storage", "platform-credentials", "observability-foundation", "storage-foundation"} {
		if aa[name] == nil {
			t.Fatalf("full lacks %s", name)
		}
	}
	for path, b := range base {
		if path != appCatalog && path != projectCatalog && !bytes.Equal(b, full[path]) {
			t.Fatalf("two-phase authority drift %s", path)
		}
	}
	var kind map[string]any
	if e = Decode(base[bootstrapDir+"kind.json"], &kind); e != nil {
		t.Fatal(e)
	}
	nodes := array(kind["nodes"])
	if len(nodes) != 4 {
		t.Fatal("wrong topology")
	}
	ports := array(mapping(nodes[1])["extraPortMappings"])
	if mapping(ports[0])["hostPort"] != float64(c.HTTPPort) || mapping(ports[1])["hostPort"] != float64(c.HTTPSPort) {
		t.Fatal("ingress projection failed")
	}
	if !bytes.Contains(base["gitops/workloads/web-smoke/overlays/development/resources.json"], []byte(`"port": 18443`)) {
		t.Fatal("HTTP redirect omitted selected port")
	}
}
func TestReleaseBoundBootstrapWithoutSourceCheckout(t *testing.T) {
	p := testProduct(t)
	c := testConfig(t)
	w := Workflow{Config: c, Product: p, ProductDigest: Digest(JSON(p)), BinaryDigest: strings.Repeat("f", 64)}
	if e := w.Open(true); e != nil {
		t.Fatal(e)
	}
	a, e := w.app(context.Background(), strings.Repeat("b", 40))
	if e != nil {
		t.Fatal(e)
	}
	f, e := a.Render(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if len(f[signalPath]) == 0 {
		t.Fatal("missing adoption signal")
	}
	var signal map[string]any
	if e = Decode(f[signalPath], &signal); e != nil {
		t.Fatal(e)
	}
	if nested(signal, "metadata", "namespace") != "argocd" {
		t.Fatal("signal identity")
	}
	if _, e = os.Stat(filepath.Join(w.runtimeDir(), ".git")); !os.IsNotExist(e) {
		t.Fatal("runtime should not need source checkout")
	}
	old := w.Record.InstallID
	if e = w.Open(true); e != nil || old != w.Record.InstallID {
		t.Fatal("repeat changed install identity", e)
	}
	w.Config.HTTPSPort++
	if w.Open(true) == nil {
		t.Fatal("changed input adopted old state")
	}
}
func TestAtomicImmutableFilesAndLocks(t *testing.T) {
	c := testConfig(t)
	path := filepath.Join(c.StateDirectory, "credential.json")
	if e := save(path, []byte("one"), true); e != nil {
		t.Fatal(e)
	}
	if e := save(path, []byte("two"), true); e == nil {
		t.Fatal("replaced immutable credential")
	}
	b, e := privateRead(path)
	if e != nil || string(b) != "one" {
		t.Fatal("credential changed", e)
	}
	w := Workflow{Config: c}
	unlock, e := w.Lock()
	if e != nil {
		t.Fatal(e)
	}
	if other, e := w.Lock(); e == nil {
		other()
		t.Fatal("concurrent install accepted")
	}
	unlock()
	unlock, e = w.Lock()
	if e != nil {
		t.Fatal("terminal file became stale lock", e)
	}
	unlock()
	target := filepath.Join(c.StateDirectory, "link")
	if e = os.Symlink(path, target); e != nil {
		t.Fatal(e)
	}
	if save(target, []byte("bad"), false) == nil {
		t.Fatal("followed symlink")
	}
}
func archive(t *testing.T, headers []*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for _, h := range headers {
		if e := tw.WriteHeader(h); e != nil {
			t.Fatal(e)
		}
		if h.Typeflag == tar.TypeReg {
			if _, e := tw.Write([]byte(strings.Repeat("x", int(h.Size)))); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e := tw.Close(); e != nil {
		t.Fatal(e)
	}
	if e := gz.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestToolArchiveRejectsTraversalLinksAndDuplicates(t *testing.T) {
	tool := Tool{Member: "bin/tool"}
	for name, headers := range map[string][]*tar.Header{"escape": {{Name: "../tool", Typeflag: tar.TypeReg, Size: 1}}, "symlink": {{Name: "bin/tool", Typeflag: tar.TypeSymlink, Linkname: "/tmp/elsewhere"}}, "duplicate": {{Name: "bin/tool", Typeflag: tar.TypeReg, Size: 1}, {Name: "bin/tool", Typeflag: tar.TypeReg, Size: 1}}} {
		t.Run(name, func(t *testing.T) {
			if _, e := toolBinary(tool, archive(t, headers)); e == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	b, e := toolBinary(tool, archive(t, []*tar.Header{{Name: "bin/", Typeflag: tar.TypeDir}, {Name: "bin/tool", Typeflag: tar.TypeReg, Size: 2}}))
	if e != nil || string(b) != "xx" {
		t.Fatal("valid archive rejected", e)
	}
}
func TestDamagedRuntimeToolFailsClosed(t *testing.T) {
	c := testConfig(t)
	p := testProduct(t)
	for _, tool := range p.Tools {
		path := filepath.Join(ToolDirectory(c.StateDirectory, p.Tools), tool.Name)
		if e := save(path, []byte("tampered"), true); e != nil {
			t.Fatal(e)
		}
	}
	if VerifyTools(c.StateDirectory, p.Tools) == nil {
		t.Fatal("accepted corrupt executables")
	}
}
func TestProductTamperAndSecretRejected(t *testing.T) {
	p := testProduct(t)
	copyProduct := func() Product {
		var v Product
		if e := json.Unmarshal(JSON(p), &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	for name, change := range map[string]func(*Product){"asset": func(v *Product) { v.Assets[v.Lock.Chart] = []byte("bad") }, "secret": func(v *Product) {
		v.Base[CredentialPath] = []byte(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"leak"}}`)
	}, "external kustomize": func(v *Product) {
		v.Base[RootPath+"/kustomization.yaml"] = []byte(`{"kind":"Kustomization","apiVersion":"kustomize.config.k8s.io/v1beta1","resources":["https://bad.example/x"]}`)
	}, "seed": func(v *Product) { v.Base[bootstrapDir+"cilium-seed.yaml"] = []byte(`{}`) }} {
		t.Run(name, func(t *testing.T) {
			v := copyProduct()
			change(&v)
			if v.Validate() == nil {
				t.Fatal("tampered product accepted")
			}
		})
	}
}

func TestProjectedKustomizeWithoutSourceCheckout(t *testing.T) {
	executable := os.Getenv("ATLAS_TEST_KUBECTL")
	if executable == "" {
		t.Skip("set ATLAS_TEST_KUBECTL for real packaged Kustomize verification")
	}
	p := testProduct(t)
	c := testConfig(t)
	for _, full := range []bool{false, true} {
		files, e := p.Project(c, full)
		if e != nil {
			t.Fatal(e)
		}
		root, e := filepath.EvalSymlinks(t.TempDir())
		if e != nil {
			t.Fatal(e)
		}
		files[signalPath] = JSON(map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]string{"name": "atlas-refactor-adoption-signal", "namespace": "argocd"}, "data": map[string]string{"fingerprint": strings.Repeat("a", 64)}})
		for path, b := range files {
			if e = save(filepath.Join(root, path), b, true); e != nil {
				t.Fatal(e)
			}
		}
		for path := range files {
			if strings.HasSuffix(path, "/kustomization.yaml") {
				cmd := exec.Command(executable, "kustomize", filepath.Join(root, filepath.Dir(path)))
				if output, e := cmd.CombinedOutput(); e != nil {
					t.Fatalf("standalone Kustomize %s: %v: %s", path, e, output)
				}
			}
		}
	}
}

func TestRuntimeToolPreparationIntegration(t *testing.T) {
	if os.Getenv("ATLAS_TEST_DOWNLOAD_TOOLS") != "1" {
		t.Skip("explicit online tool preparation check")
	}
	p := testProduct(t)
	c := testConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	if e := PrepareTools(ctx, c.StateDirectory, p.Tools); e != nil {
		t.Fatal(e)
	}
	if e := VerifyTools(c.StateDirectory, p.Tools); e != nil {
		t.Fatal(e)
	}
	before := map[string]time.Time{}
	for _, tool := range p.Tools {
		st, e := os.Stat(filepath.Join(ToolDirectory(c.StateDirectory, p.Tools), tool.Name))
		if e != nil {
			t.Fatal(e)
		}
		before[tool.Name] = st.ModTime()
	}
	if e := PrepareTools(ctx, c.StateDirectory, p.Tools); e != nil {
		t.Fatal(e)
	}
	for _, tool := range p.Tools {
		st, e := os.Stat(filepath.Join(ToolDirectory(c.StateDirectory, p.Tools), tool.Name))
		if e != nil || st.ModTime() != before[tool.Name] {
			t.Fatal("repeat preparation replaced executable", e)
		}
	}
}

func TestIncompleteCompletedRecordIsRejected(t *testing.T) {
	p := testProduct(t)
	c := testConfig(t)
	w := Workflow{Config: c, Product: p, ProductDigest: Digest(JSON(p)), BinaryDigest: strings.Repeat("a", 64)}
	if e := w.Open(true); e != nil {
		t.Fatal(e)
	}
	w.Record.Complete = true
	if e := w.saveRecord(); e != nil {
		t.Fatal(e)
	}
	if w.Open(false) == nil {
		t.Fatal("completion flag accepted without a deployment")
	}
}
