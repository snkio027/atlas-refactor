package main

import (
	"strings"
	"testing"
)

func TestAppConfirmationBindsExactTarget(t *testing.T) {
	for _, input := range []string{"yes\n", "deploy another/project\n", "deploy target/project", "deploy target/project extra\n", "\n"} {
		if err := confirmApp(strings.NewReader(input), "deploy target/project"); err == nil {
			t.Fatal("unbound approval", input)
		}
	}
	if err := confirmApp(strings.NewReader("deploy target/project\n"), "deploy target/project"); err != nil {
		t.Fatal(err)
	}
}
func TestAppRejectsIrrelevantFlagsBeforeLoadingInstance(t *testing.T) {
	for _, args := range [][]string{{"check", "--approve-plan", "x"}, {"status", "--s3"}, {"logs", "--artifact", "x"}, {"open", "extra"}, {"deploy", "--yes"}} {
		if err := runApp(args); err == nil {
			t.Fatal("accepted", args)
		}
	}
}
