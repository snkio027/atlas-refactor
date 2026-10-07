package workloadrun

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReadPollStopsOnFatalAndDeduplicatesProgress(t *testing.T) {
	fatal := errors.New("identity lost")
	calls := 0
	updates := []string{}
	sequence := []error{Pending("creating"), Pending("creating"), Pending("reconciling"), fatal, nil}
	err := pollRead(context.Background(), 0, func(s string) { updates = append(updates, s) }, func(context.Context) error {
		err := sequence[calls]
		calls++
		return err
	})
	if !errors.Is(err, fatal) || calls != 4 || strings.Join(updates, ",") != "creating,reconciling" {
		t.Fatal(err, calls, updates)
	}
}
func TestReadPollCancellationAndDeadlineNeverAllowLateSuccess(t *testing.T) {
	for _, cancelInside := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if !cancelInside {
			cancel()
		}
		calls := 0
		err := pollRead(ctx, 0, nil, func(context.Context) error { calls++; cancel(); return nil })
		if !errors.Is(err, context.Canceled) || calls > 1 || !cancelInside && calls != 0 {
			t.Fatal(err, calls)
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	calls := 0
	if err := pollRead(ctx, 0, nil, func(context.Context) error { calls++; return nil }); !errors.Is(err, context.DeadlineExceeded) || calls != 0 {
		t.Fatal(err, calls)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	calls = 0
	err := pollRead(ctx2, time.Hour, func(string) { cancel2() }, func(context.Context) error { calls++; return Pending("awaiting comparison") })
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "awaiting comparison") || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestReadPollAllowsReadyWithoutIntermediateStates(t *testing.T) {
	calls := 0
	err := pollRead(context.Background(), 0, nil, func(context.Context) error { calls++; return nil })
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
}
