package webslice

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRoundTripActuallyWritesReadsAndDeletes(t *testing.T) {
	store := map[string][]byte{}
	methods := []string{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if !strings.HasPrefix(r.URL.Path, "/uploads/atlas-s2-probe/") || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=fixture/") {
			t.Error("wrong request boundary")
		}
		switch r.Method {
		case "PUT":
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			store[r.URL.Path] = buf.Bytes()
		case "GET":
			w.Write(store[r.URL.Path])
		case "DELETE":
			delete(store, r.URL.Path)
			w.WriteHeader(204)
		}
	}))
	defer s.Close()
	c := &Client{s.URL, "uploads", "fixture", "secret", s.Client()}
	server := httptest.NewServer(Handler(c))
	defer server.Close()
	res, e := http.Post(server.URL+"/roundtrip", "text/plain", strings.NewReader("business request"))
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	var v map[string]any
	json.NewDecoder(res.Body).Decode(&v)
	if res.StatusCode != 200 || v["sha256"] != digest([]byte("business request")) || strings.Join(methods, ",") != "PUT,GET,DELETE" || len(store) != 0 {
		t.Fatal("incomplete roundtrip", v, methods)
	}
}
func TestUnboundAndOversizedRequests(t *testing.T) {
	r := httptest.NewRequest("POST", "/roundtrip", strings.NewReader("x"))
	w := httptest.NewRecorder()
	Handler(nil).ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("unbound accepted")
	}
	c := &Client{HTTP: http.DefaultClient}
	r = httptest.NewRequest("POST", "/roundtrip", strings.NewReader(strings.Repeat("x", 65537)))
	w = httptest.NewRecorder()
	Handler(c).ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatal("oversized accepted")
	}
}
func TestFailedReadStillCleansOnlyProbeObject(t *testing.T) {
	deletes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Write([]byte("wrong"))
		case "DELETE":
			deletes++
			w.WriteHeader(204)
		}
	}))
	defer s.Close()
	c := &Client{s.URL, "uploads", "fixture", "secret", s.Client()}
	if _, e := c.RoundTrip(context.Background(), []byte("x")); e == nil || deletes != 1 {
		t.Fatal("failed read did not clean", e, deletes)
	}
}
func TestSigV4CanonicalQuery(t *testing.T) {
	a, _ := http.NewRequest("GET", "http://s3.test/uploads?z=a+b&a=", nil)
	b, _ := http.NewRequest("GET", "http://s3.test/uploads?a=&z=a%20b", nil)
	now := time.Unix(0, 0)
	Sign(a, nil, "fixture", "secret", now)
	Sign(b, nil, "fixture", "secret", now)
	if a.Header.Get("Authorization") != b.Header.Get("Authorization") || a.URL.RawQuery != "a=&z=a%20b" {
		t.Fatal("noncanonical signature")
	}
}
