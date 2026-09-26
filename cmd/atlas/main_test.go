package main

import "testing"

func TestExplicitStatusCheckIsAccepted(t *testing.T) {
	if code := run([]string{"status", "--check", "--help"}); code != 0 {
		t.Fatalf("status --check not accepted: %d", code)
	}
}

func TestCLIRejectsUnknownSurface(t *testing.T) {
	for _, args := range [][]string{nil, {"recovery"}, {"apply", "--recovery"}, {"status", "--approve-tier0"}, {"doctor", "unexpected"}} {
		if code := run(args); code != 2 {
			t.Fatalf("%v returned %d", args, code)
		}
	}
}
func TestHelpAndVersionNeedNoConfiguration(t *testing.T) {
	for _, arg := range []string{"--help", "--version"} {
		if code := run([]string{arg}); code != 0 {
			t.Fatalf("%s returned %d", arg, code)
		}
	}
}
