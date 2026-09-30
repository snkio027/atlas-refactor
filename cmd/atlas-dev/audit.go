package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Compare request identities, not event counts: rotation may remove an old
// request while adding a new one. Repeated records of one request are harmless.
func auditWrites(repo string) (map[string]bool, error) {
	files, e := filepath.Glob(filepath.Join(repo, ".state/audit/*"))
	if e != nil {
		return nil, e
	}
	if len(files) == 0 {
		return nil, errors.New("audit evidence unavailable")
	}
	ids := map[string]bool{}
	seen := false
	for _, p := range files {
		f, e := os.Open(p)
		if e != nil {
			return nil, e
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 4096), 1<<20)
		for s.Scan() {
			var event struct{ AuditID, Verb, UserAgent, Stage string }
			if e = json.Unmarshal(s.Bytes(), &event); e != nil {
				_ = f.Close()
				return nil, e
			}
			if event.AuditID == "" || event.Verb == "" || event.Stage == "" {
				_ = f.Close()
				return nil, errors.New("audit event identity missing")
			}
			seen = true
			if strings.HasPrefix(event.UserAgent, "kubectl/") && event.Stage == "ResponseComplete" {
				switch event.Verb {
				case "create", "patch", "update", "delete", "deletecollection":
					ids[event.AuditID] = true
				}
			}
		}
		e = s.Err()
		_ = f.Close()
		if e != nil {
			return nil, e
		}
	}
	if !seen {
		return nil, errors.New("audit evidence empty")
	}
	return ids, nil
}
