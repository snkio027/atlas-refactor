package installation

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostEvidenceSurvivesRetryWithoutReplacingFirstMeasurement(t *testing.T) {
	w := Workflow{Config: testConfig(t), Record: Record{InstallID: strings.Repeat("a", 32)}}
	calls := 0
	w.runCommand = func(_ context.Context, _ string, _ []byte, tool string, args ...string) ([]byte, error) {
		calls++
		switch tool {
		case "sw_vers", "orb", "sysctl", "docker":
			return []byte("measured-value"), nil
		}
		return nil, errors.New("unexpected host command")
	}
	if e := w.captureHost(context.Background(), 123, 456); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(w.Config.StateDirectory, "host-first-check.json")
	before, e := privateRead(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.captureHost(context.Background(), 1, 2); e != nil {
		t.Fatal(e)
	}
	after, e := privateRead(path)
	if e != nil || string(before) != string(after) || calls != 5 {
		t.Fatal("retry rewrote initial host evidence", e, calls)
	}
	w.Record.InstallID = strings.Repeat("b", 32)
	if w.captureHost(context.Background(), 1, 2) == nil {
		t.Fatal("accepted another installation's host evidence")
	}
}
