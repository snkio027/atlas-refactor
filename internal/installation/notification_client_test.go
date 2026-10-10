package installation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func testNotificationClient(t *testing.T, h http.HandlerFunc) *notificationClient {
	t.Helper()
	s := httptest.NewTLSServer(h)
	t.Cleanup(s.Close)
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(dir, "kubeconfig")
	if e = os.WriteFile(p, []byte("bound"), 0600); e != nil {
		t.Fatal(e)
	}
	client := s.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.Transport.(*http.Transport).DisableKeepAlives = true
	return &notificationClient{server: s.URL, path: p, digest: Digest([]byte("bound")), clusterUID: "cluster", version: "v1.37.0", client: client}
}
func TestNotificationHTTPOutcomeClassification(t *testing.T) {
	cases := []struct {
		name     string
		code     int
		body     string
		rejected bool
	}{
		{"conflict", 409, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":409,"reason":"Conflict"}`, true},
		{"invalid", 422, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":422,"reason":"Invalid"}`, true},
		{"wrong code", 422, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":409,"reason":"Invalid"}`, false},
		{"wrong reason", 422, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":422,"reason":"Unknown"}`, false},
		{"wrong kind", 422, `{"apiVersion":"v1","kind":"Unknown","status":"Failure","code":422,"reason":"Invalid"}`, false},
		{"bare code", 422, `{}`, false},
		{"duplicate key", 422, `{"kind":"Status","kind":"Status"}`, false},
		{"truncated", 422, `{"kind":`, false},
		{"trailing", 422, `{} {}`, false},
		{"server error", 500, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":500,"reason":"InternalError","message":"secret-marker"}`, false},
		{"redirect", 307, `{}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			c := testNotificationClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "PATCH" || r.Header.Get("Content-Type") != "application/json-patch+json" {
					t.Error("incorrect request")
				}
				w.Header().Set("Location", "/must-not-follow")
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := c.request(context.Background(), http.MethodPatch, "/app", []byte("[]"))
			var rejection notificationRejection
			if err == nil || errors.As(err, &rejection) != tc.rejected {
				t.Fatalf("classification: %v", err)
			}
			if calls.Load() != 1 || strings.Contains(err.Error(), "secret-marker") {
				t.Fatal("replayed request or leaked response")
			}
		})
	}
}
func TestNotificationTransportUnknownWriteIsNeverRetried(t *testing.T) {
	var calls atomic.Int32
	c := testNotificationClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		conn, _, e := w.(http.Hijacker).Hijack()
		if e != nil {
			t.Error(e)
			return
		}
		_ = conn.Close()
	})
	_, err := c.request(context.Background(), http.MethodPatch, "/app", []byte("[]"))
	var rejection notificationRejection
	if err == nil || errors.As(err, &rejection) || calls.Load() != 1 {
		t.Fatalf("unsafe lost response: %v calls=%d", err, calls.Load())
	}
}
func TestNotificationHTTPFences(t *testing.T) {
	for _, mode := range []string{"ok", "version", "uid", "credential", "missing", "unconfirmed missing"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			c := testNotificationClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" {
					t.Error("fence mutated")
				}
				if mode == "missing" || mode == "unconfirmed missing" {
					w.WriteHeader(404)
					if mode == "missing" {
						_, _ = io.WriteString(w, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":404,"reason":"NotFound"}`)
					}
					return
				}
				if r.URL.Path == "/version" {
					v := "v1.37.0"
					if mode == "version" {
						v = "v1.36.0"
					}
					_, _ = w.Write(JSON(map[string]any{"gitVersion": v}))
					return
				}
				uid := "cluster"
				if mode == "uid" {
					uid = "other"
				}
				_, _ = w.Write(JSON(map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "kube-system", "uid": uid}}))
			})
			if mode == "credential" {
				if err := os.WriteFile(c.path, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.fence(context.Background()); (err == nil) != (mode == "ok") {
				t.Fatalf("fence: %v", err)
			}
			if mode == "credential" && calls.Load() != 0 {
				t.Fatal("credential drift reached API")
			}
			if mode == "missing" {
				o, e := c.request(context.Background(), http.MethodGet, "/app", nil)
				if e != nil || o != nil {
					t.Fatal("confirmed absence rejected")
				}
			}
		})
	}
}

// Exercise the real HTTP classification, CAS loop and durable records together.
func TestNotificationHTTPInterleaving(t *testing.T) {
	w, live, ops, effects := notificationFixture(t)
	want := notificationClone(live)
	var calls atomic.Int32
	patch := ops.patch
	c := testNotificationClient(t, func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_, _ = rw.Write(JSON(live))
			return
		}
		b, e := io.ReadAll(r.Body)
		if e != nil {
			t.Error(e)
			return
		}
		if calls.Add(1) == 1 {
			mapping(live["metadata"])["resourceVersion"] = "2"
			live["status"] = map[string]any{"health": "updated"}
			rw.WriteHeader(422)
			_, _ = io.WriteString(rw, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":422,"reason":"Invalid"}`)
			return
		}
		result, e := patch(r.Context(), "test", b)
		if e != nil {
			t.Error(e)
			return
		}
		_, _ = rw.Write(JSON(result))
	})
	ops.get = func(ctx context.Context, _ string) (map[string]any, error) {
		return c.request(ctx, http.MethodGet, "/app", nil)
	}
	ops.patch = func(ctx context.Context, _ string, b []byte) (map[string]any, error) {
		return c.request(ctx, http.MethodPatch, "/app", b)
	}
	if err := w.notifyApplication(context.Background(), "test", want, ops); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || *effects != 1 {
		t.Fatal("CAS race was not resolved exactly once")
	}
}
