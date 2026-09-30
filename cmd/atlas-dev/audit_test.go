package main

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

func TestRepeatApplyDetectsSameCountDifferentRequests(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".state/audit")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "audit.log")
	write := func(data string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(data+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	const old = `{"auditID":"old","verb":"patch","userAgent":"kubectl/v1","stage":"ResponseComplete"}`
	const newRequest = `{"auditID":"new","verb":"patch","userAgent":"kubectl/v1","stage":"ResponseComplete"}`
	write(old)
	before, err := auditWrites(repo)
	if err != nil {
		t.Fatal(err)
	}
	write(newRequest) // rotation removed one old request and added one new request
	after, err := auditWrites(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) || maps.Equal(before, after) {
		t.Fatal("same-count new write was not detected")
	}
	write(old + "\n" + old + "\n" + `{"auditID":"read","verb":"get","userAgent":"kubectl/v1","stage":"ResponseComplete"}`)
	after, err = auditWrites(repo)
	if err != nil || !maps.Equal(before, after) {
		t.Fatal("duplicate record/read counted as new mutation", err)
	}
	for _, data := range []string{"", "{", "{}", `{"verb":"patch","userAgent":"kubectl/v1","stage":"ResponseComplete"}`} {
		write(data)
		if _, err = auditWrites(repo); err == nil {
			t.Fatalf("invalid audit accepted: %q", data)
		}
	}
	if _, err := auditWrites(t.TempDir()); err == nil {
		t.Fatal("absent audit accepted")
	}
}
