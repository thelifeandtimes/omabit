package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskDeploymentCLIRejectsUnsafeArgumentsBeforeOpeningState(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "unopened")
	bad := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(bad, []byte("{} {}"), 0600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked.json")
	if err := os.Symlink(bad, linked); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"desk-plan-commit", "--stage-id", "../outside"},
		{"desk-review-commit", "--stage-id", strings.Repeat("a", 64), "--plan-file", bad},
		{"desk-commit", "--plan-json", "{} {}"},
		{"desk-commit", "--plan-file", bad},
		{"desk-commit", "--plan-file", linked},
		{"desk-commit", "--plan-file", root},
		{"desk-commit", "--plan-file", bad, "--plan-json", "{}"},
		{"desk-commit", "--plan-json", "{}", "--stage-id", strings.Repeat("a", 64)},
	} {
		argv := append([]string{"--config-dir", config, args[0], "--runner", "fixture", "--expected-runner-id", "runner-fixture"}, args[1:]...)
		if run(context.Background(), argv) == 0 {
			t.Fatal("unsafe command accepted", args)
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid command opened state", err)
	}
}
