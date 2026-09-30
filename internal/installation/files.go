package installation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func privateDir(dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return errors.New("private directory must be an absolute clean path")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	resolved, e := filepath.EvalSymlinks(dir)
	if e != nil {
		return e
	}
	if resolved != dir {
		return errors.New("private directory must not traverse symlinks; use its canonical path")
	}
	info, e := os.Lstat(dir)
	if e != nil {
		return e
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("directory must be owner-only: %s", dir)
	}
	return nil
}
func privateRead(path string) ([]byte, error) {
	st, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("private input must be a regular owner-only file")
	}
	return os.ReadFile(path)
}

// Save publishes complete data atomically. Immutable records never replace an
// existing file; they reconcile byte-for-byte after a lost acknowledgement.
func save(path string, b []byte, immutable bool) error {
	if e := privateDir(filepath.Dir(path)); e != nil {
		return e
	}
	if old, e := privateRead(path); e == nil {
		if Digest(old) == Digest(b) {
			return nil
		}
		if immutable {
			return errors.New("immutable installation record differs")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".pending-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if immutable {
		if e = os.Link(name, path); e != nil {
			return e
		}
	} else if e = os.Rename(name, path); e != nil {
		return e
	}
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer dir.Close()
	return dir.Sync()
}
