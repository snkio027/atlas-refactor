package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// finish runs while the workflow lock is held. PASS is visible only after the
// new evidence is safely written; failed runs replace a stale PASS with FAIL.
func (w *workflow) finish(runErr error, out io.Writer) error {
	w.evidence["finishedAt"] = time.Now().UTC().Format(time.RFC3339)
	w.evidence["result"] = "PASS"
	if runErr != nil {
		w.evidence["result"] = "FAIL"
		w.evidence["error"] = runErr.Error()
	}
	path := filepath.Join(w.dir, "latest-run.json")
	if err := jsonFile(path, w.evidence); err != nil {
		return errors.Join(runErr, fmt.Errorf("persist development evidence: %w", err))
	}
	if runErr != nil {
		return runErr
	}
	_, err := fmt.Fprintln(out, "PASS: four Ready nodes, GitOps ADOPTED, HTTPS verified; evidence: "+path)
	return err
}

// jsonFile replaces private development state atomically. It is not the
// create-only writer used for immutable ceremony/authority evidence.
func jsonFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = checkedDir(filepath.Dir(path), false); err != nil {
		return err
	}
	if info, statErr := os.Lstat(path); statErr == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return errors.New("private JSON target must be an owner-only regular file")
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".atlas-json-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
