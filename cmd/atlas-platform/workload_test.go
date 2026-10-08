package main

import (
	"strings"
	"testing"
)

func TestWorkloadRejectsIgnoredExecutionFlags(t *testing.T) {
	for _, args := range [][]string{{"deploy", "--phase", "infrastructure", "--approve-plan", "x"}, {"publish", "--revision", "deadbeef", "--approve-plan", "x"}, {"plan", "--wait", "1m"}} {
		if e := runWorkload(args); e == nil || !strings.Contains(e.Error(), "not valid") {
			t.Fatalf("ignored execution flag: %v %v", args, e)
		}
	}
}
