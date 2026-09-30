package installation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInterruptedArtifactResumesAndVerifiesCompleteDigest(t *testing.T) {
	data := []byte("complete pinned executable archive")
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Write(data[:7])
			return
		}
		if r.Header.Get("Range") != "bytes=7-" {
			t.Error("resume omitted exact partial offset")
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 7-%d/%d", len(data)-1, len(data)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(data[7:])
	}))
	defer server.Close()
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	got, e := downloadTool(context.Background(), server.Client(), dir, Tool{Name: "fixture", URL: server.URL, SHA256: Digest(data)})
	if e != nil || string(got) != string(data) || calls != 2 {
		t.Fatal("resumed download did not verify", e)
	}
	got, e = downloadTool(context.Background(), server.Client(), dir, Tool{Name: "fixture", URL: server.URL, SHA256: Digest(data)})
	if e != nil || string(got) != string(data) || calls != 2 {
		t.Fatal("verified cache caused another request", e)
	}
}
func TestCorruptArtifactIsNeverExtracted(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("damaged")) }))
	defer server.Close()
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	if _, e = downloadTool(context.Background(), server.Client(), dir, Tool{Name: "fixture", URL: server.URL, SHA256: Digest([]byte("expected"))}); e == nil || !strings.Contains(e.Error(), "digest mismatch") {
		t.Fatal("corrupt artifact did not fail checksum verification", e)
	}
}
func TestRangeResponseMustMatchOffsetAndTotal(t *testing.T) {
	for _, s := range []string{"bytes 1-9/10", "bytes 5-9/*", "bytes 5-10/10", "bytes 5-9/9999999999", "bytes 5-8/10"} {
		if validRange(s, 5, 5) {
			t.Fatalf("accepted invalid range %q", s)
		}
	}
	if !validRange("bytes 5-9/10", 5, 5) {
		t.Fatal("valid range rejected")
	}
}
