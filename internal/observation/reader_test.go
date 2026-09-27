package observation

import (
	"atlas-refactor/internal/platform"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAPIReadOnlyHTTPFences(t *testing.T) {
	root, _ := filepath.Abs("../..")
	p, e := platform.Load(root, platform.Tools{Helm: "helm", Kubectl: "kubectl", YQ: "yq"})
	if e != nil {
		t.Fatal(e)
	}
	model, e := p.ObservationResourceModel()
	if e != nil {
		t.Fatal(e)
	}
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	file := filepath.Join(dir, "kubeconfig")
	if e = os.WriteFile(file, []byte("bound"), 0600); e != nil {
		t.Fatal(e)
	}
	var mu sync.Mutex
	requests := []string{}
	mode := "ok"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method != "GET" {
			t.Error("non-GET request")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1" {
			w.Write(Bytes(Object{"kind": "APIResourceList", "groupVersion": "v1", "resources": []any{Object{"name": "namespaces", "kind": "Namespace", "namespaced": false, "verbs": []any{"get", "list"}}}}))
			return
		}
		if mode == "unavailable" {
			w.WriteHeader(503)
			return
		}
		if mode == "missing" {
			w.WriteHeader(404)
			return
		}
		if mode == "redirect" {
			http.Redirect(w, r, "/unexpected", 302)
			return
		}
		o := Object{"apiVersion": "v1", "kind": "Namespace", "metadata": Object{"name": "kube-system", "uid": "cluster", "resourceVersion": "1"}}
		if mode == "wrong-identity" {
			Map(o["metadata"])["name"] = "another"
		}
		w.Write(Bytes(o))
	}))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	reader := &APIReader{target: Target{KubeconfigSHA256: SHA([]byte("bound"))}, kubeconfig: file, server: server.URL, client: client, model: model, discovery: map[string]map[string]string{}}
	ref := Ref{APIVersion: "v1", Kind: "Namespace", Name: "kube-system"}
	if _, e = reader.Read(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
	mu.Lock()
	mode = "missing"
	mu.Unlock()
	if o, e := reader.Read(context.Background(), ref); e != nil || o != nil {
		t.Fatal("confirmed absence")
	}
	for _, bad := range []string{"unavailable", "wrong-identity", "redirect"} {
		mu.Lock()
		mode = bad
		mu.Unlock()
		if _, e = reader.Read(context.Background(), ref); e == nil {
			t.Fatalf("%s accepted", bad)
		}
	}
	mu.Lock()
	before := len(requests)
	mu.Unlock()
	if _, e = reader.Read(context.Background(), Ref{APIVersion: "v1", Kind: "Secret", Namespace: "argocd", Name: "key"}); e == nil || requestCount(&mu, requests) != before {
		t.Fatal("secret reached API")
	}
	if e = os.WriteFile(file, []byte("changed"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = reader.Read(context.Background(), ref); e == nil || requestCount(&mu, requests) != before {
		t.Fatal("changed credential binding reached API")
	}
}

func requestCount(mu *sync.Mutex, values []string) int {
	mu.Lock()
	defer mu.Unlock()
	return len(values)
}
