package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stateDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFinishPersistsBeforeReportingSuccess(t *testing.T) {
	for _, failed := range []bool{false, true} {
		w := workflow{dir: stateDir(t), evidence: map[string]any{"cluster": "test"}}
		original := errors.New("runtime verification failed")
		var runErr error
		want := "PASS"
		if failed {
			runErr, want = original, "FAIL"
		}
		// Replace a previous successful run, including on failure.
		path := filepath.Join(w.dir, "latest-run.json")
		if err := jsonFile(path, map[string]any{"result": "PASS", "old": true}); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := w.finish(runErr, &out)
		if !errors.Is(err, runErr) {
			t.Fatalf("lost run error: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err = json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result["result"] != want || result["finishedAt"] == nil || result["old"] != nil {
			t.Fatalf("stale/incomplete evidence: %s", data)
		}
		if strings.Contains(out.String(), "PASS:") == failed {
			t.Fatalf("misleading output: %q", out.String())
		}
		if failed && result["error"] != original.Error() {
			t.Fatal("failure reason missing")
		}
	}
}

func TestFinishFailsClosedWhenEvidenceCannotBeSaved(t *testing.T) {
	for _, runErr := range []error{nil, errors.New("original failure")} {
		w := workflow{dir: stateDir(t), evidence: map[string]any{}}
		if err := os.Mkdir(filepath.Join(w.dir, "latest-run.json"), 0700); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err := w.finish(runErr, &out)
		if err == nil || !strings.Contains(err.Error(), "persist development evidence") {
			t.Fatal("ignored evidence failure", err)
		}
		if runErr != nil && !errors.Is(err, runErr) {
			t.Fatal("lost original workflow error")
		}
		if out.Len() != 0 {
			t.Fatal("PASS emitted without evidence", out.String())
		}
	}
}

func TestPrivateJSONRejectsUnsafeTargetsAndPreservesExistingData(t *testing.T) {
	dir := stateDir(t)
	target := filepath.Join(dir, "evidence.json")
	if err := jsonFile(target, map[string]any{"version": 1}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err = jsonFile(target, make(chan int)); err == nil {
		t.Fatal("unsupported payload accepted")
	}
	after, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed write destroyed previous evidence", err)
	}
	link := filepath.Join(dir, "link.json")
	if err = os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err = jsonFile(link, map[string]any{"version": 2}); err == nil {
		t.Fatal("followed symlink")
	}
	after, err = os.ReadFile(target)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("symlink target overwritten", err)
	}
	if err = os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	if err = jsonFile(target, map[string]any{"private": true}); err == nil {
		t.Fatal("wrote private state into public file")
	}
	if err = os.Chmod(target, 0600); err != nil {
		t.Fatal(err)
	}
	if err = jsonFile(target, map[string]any{"version": 2}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("private mode lost", err)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".atlas-json-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatal("temporary files leaked", leftovers, err)
	}
}
