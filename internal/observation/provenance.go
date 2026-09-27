package observation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"runtime/debug"
)

type Implementation struct {
	Revision     string `json:"revision"`
	Dirty        bool   `json:"dirty"`
	BinarySHA256 string `json:"binarySHA256"`
}

func ExecutableIdentity() (Implementation, error) {
	var result Implementation
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result, errors.New("build provenance unavailable")
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			result.Revision = s.Value
		case "vcs.modified":
			result.Dirty = s.Value == "true"
		}
	}
	if !FullSHA(result.Revision) {
		return result, errors.New("build a VCS-stamped binary with go build; source identity unavailable")
	}
	path, e := os.Executable()
	if e != nil {
		return result, e
	}
	f, e := os.Open(path)
	if e != nil {
		return result, e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return result, e
	}
	result.BinarySHA256 = hex.EncodeToString(h.Sum(nil))
	return result, nil
}
