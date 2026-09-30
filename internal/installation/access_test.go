package installation

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAccessReadersShareButExcludeInstallation(t *testing.T) {
	w := Workflow{Config: testConfig(t)}
	if e := privateDir(w.Config.StateDirectory); e != nil {
		t.Fatal(e)
	}
	a, e := w.ReadLock()
	if e != nil {
		t.Fatal(e)
	}
	b, e := w.ReadLock()
	if e != nil {
		a()
		t.Fatal(e)
	}
	if release, e := w.Lock(); e == nil {
		release()
		t.Fatal("installation overlapped active access")
	}
	a()
	b()
	exclusive, e := w.Lock()
	if e != nil {
		t.Fatal(e)
	}
	defer exclusive()
	if release, e := w.ReadLock(); e == nil {
		release()
		t.Fatal("access overlapped active installation")
	}
}

// The child models kubectl's readiness line and later unexpected exit. No
// cluster or shell script is involved, and closing stdin controls the race.
func TestForwardProcess(t *testing.T) {
	if os.Getenv("ATLAS_FORWARD_TEST_CHILD") == "1" {
		fmt.Println("Forwarding from 127.0.0.1:12345 -> 80")
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(9)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestForwardProcess$")
	cmd.Env = append(os.Environ(), "ATLAS_FORWARD_TEST_CHILD=1")
	in, e := cmd.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	address, stop, done, e := startForward(ctx, cancel, cmd)
	if e != nil {
		in.Close()
		t.Fatal(e)
	}
	defer stop()
	if address != "http://127.0.0.1:12345" {
		t.Fatal(address)
	}
	in.Close()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("unexpected child exit lost")
		}
	case <-ctx.Done():
		t.Fatal("connection exit was not reported")
	}
	// A consumed exit event must not make deferred cleanup deadlock.
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup deadlocked")
	}
}
