//go:build darwin || linux

package atlas

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// D1 concurrency exclusion is process-bound. This does not clear or reinterpret
// the historical apply.lock used by schemas 1–3 and the frozen experiments.
func acquireInstallationLock(dir string) (func(), error) {
	path := filepath.Join(dir, "installation-apply.flock")
	if st, e := os.Lstat(path); e == nil && (!st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0) {
		return nil, errors.New("unsafe installation lock file")
	} else if e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, errors.New("another installation Bootstrap process is active")
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
