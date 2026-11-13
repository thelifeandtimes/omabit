package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskReleaseCLIRejectsAlteredOrUnboundedInputsBeforeProfile(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "must-not-exist")
	file := filepath.Join(root, "plan.json")
	if err := os.WriteFile(file, []byte(`{"unknown":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.json")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	cases := [][]string{
		{"desk-review-release", "--instance", "alpha", "--desk", "example", "--action", "arbitrary-dojo"},
		{"desk-plan-release", "--instance", "alpha", "--desk", "../base", "--action", "publish-source"},
		{"desk-plan-release", "--instance", "alpha", "--desk", "example", "--action", "publish-source", "--checks-json", "null"},
		{"desk-plan-release", "--instance", "alpha", "--desk", "example", "--action", "publish-source", "--checks-json", `[{"force":true}]`},
		{"desk-review-release", "--instance", "alpha", "--desk", "example", "--action", "publish-source", "--checks-json", "[]"},
		{"desk-release", "--plan-file", file},
		{"desk-release", "--plan-file", link},
		{"desk-release", "--plan-file", root},
		{"desk-release", "--plan-json", "{} {}"},
		{"desk-release", "--plan-json", strings.Repeat(" ", (1<<20)+1)},
		{"desk-release", "--plan-file", file, "--action", "activate-local"},
		{"desk-release", "--plan-file", file, "--checks-json", "[]"},
		{"desk-release", "--plan-file", file, "--plan-json", "{}"},
	}
	for _, input := range cases {
		args := append([]string{"--config-dir", config, input[0], "--runner", "fixture", "--expected-runner-id", "runner-fixture"}, input[1:]...)
		if run(context.Background(), args) == 0 {
			t.Fatal("invalid release accepted", input[0])
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid arguments opened profile", err)
	}
}
