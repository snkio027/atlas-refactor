package main

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestLocalCommandBothConventionsAndStrictArguments(t *testing.T) {
	for _, tc := range []struct {
		args               []string
		command, requested string
		invalid, help      bool
	}{
		{args: []string{"plan", "--capabilities", "monitoring"}, command: "plan", requested: "monitoring"},
		{args: []string{"--capabilities", "monitoring", "plan"}, command: "plan", requested: "monitoring"},
		{args: []string{"check"}, command: "check"},
		{args: []string{"select", "--capabilities="}, command: "select"},
		{args: []string{"--help"}, help: true},
		{args: []string{"plan", "--help"}, help: true},
		{args: []string{}, invalid: true},
		{args: []string{"unknown"}, invalid: true},
		{args: []string{"plan", "extra"}, invalid: true},
		{args: []string{"plan", "--unknown"}, invalid: true},
		{args: []string{"--capabilities", "monitoring", "plan", "--root", "elsewhere"}, invalid: true},
		{args: []string{"plan", "select"}, invalid: true},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			capabilities := fs.String("capabilities", "", "")
			fs.String("root", ".", "")
			command, err := parseLocalCommand(fs, tc.args)
			if tc.help {
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatal("help must not execute", err)
				}
			} else if tc.invalid {
				if err == nil {
					t.Fatal("invalid/ignored arguments accepted")
				}
			} else if err != nil || command != tc.command || *capabilities != tc.requested {
				t.Fatalf("command=%q capabilities=%q err=%v", command, *capabilities, err)
			}
		})
	}
}
