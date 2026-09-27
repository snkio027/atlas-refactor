package main

import (
	"context"
	"testing"
)

func TestCLIRejectsMutationAndIncompletePreparation(t *testing.T) {
	for _, args := range [][]string{{}, {"run"}, {"apply"}, {"delete"}, {"plan"}, {"capture"}, {"prepare-profile"}, {"prepare-action"}, {"inspect-plan", "extra"}} {
		code, e := run(context.Background(), args)
		if code != 2 || e == nil {
			t.Fatalf("unsafe/incomplete command accepted: %v", args)
		}
	}
}
