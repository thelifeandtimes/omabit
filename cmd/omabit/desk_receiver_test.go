package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskReceiverCLIRejectsAlteredOrUnboundedInputsBeforeProfile(t *testing.T) {
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
		{"desk-review-install", "--instance", "alpha", "--desk", "example", "--publisher-ship", "~nec", "--publisher-desk", "demo"},
		{"desk-plan-install", "--instance", "alpha", "--desk", "../base", "--publisher-ship", "~nec", "--publisher-desk", "demo", "--expected-source-hash", "0v123"},
		{"desk-receiver-status", "--instance", "alpha", "--desk", "example", "--publisher-ship", "~nec"},
		{"desk-install", "--plan-file", file},
		{"desk-install", "--plan-file", link},
		{"desk-install", "--plan-file", root},
		{"desk-install", "--plan-json", "{} {}"},
		{"desk-install", "--plan-json", strings.Repeat(" ", (128<<10)+1)},
		{"desk-install", "--plan-file", file, "--instance", "alpha"},
		{"desk-install", "--plan-file", file, "--publisher-ship", "~nec"},
		{"desk-install", "--plan-file", file, "--plan-json", "{}"},
	}
	for _, input := range cases {
		args := append([]string{"--config-dir", config, input[0], "--runner", "fixture", "--expected-runner-id", "runner-fixture"}, input[1:]...)
		if run(context.Background(), args) == 0 {
			t.Fatal("invalid receiver install accepted", input[0])
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid arguments opened profile", err)
	}
}
