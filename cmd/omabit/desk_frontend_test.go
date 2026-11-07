package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskFrontendCLIRejectsAlteredOrUnboundedInputsBeforeProfile(t *testing.T) {
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
		{"desk-review-frontend", "--stage-id", "not-a-stage"},
		{"desk-plan-frontend", "--stage-id", "../base"},
		{"desk-plan-frontend", "--stage-id", strings.Repeat("d", 64), "--checks-json", "null"},
		{"desk-plan-frontend", "--stage-id", strings.Repeat("d", 64), "--checks-json", `[{"force":true}]`},
		{"desk-review-frontend", "--stage-id", strings.Repeat("d", 64), "--checks-json", "[]"},
		{"desk-upload-frontend", "--plan-file", file},
		{"desk-upload-frontend", "--plan-file", link},
		{"desk-upload-frontend", "--plan-file", root},
		{"desk-upload-frontend", "--plan-json", "{} {}"},
		{"desk-upload-frontend", "--plan-json", strings.Repeat(" ", (1<<20)+1)},
		{"desk-upload-frontend", "--plan-file", file, "--stage-id", strings.Repeat("d", 64)},
		{"desk-upload-frontend", "--plan-file", file, "--checks-json", "[]"},
		{"desk-upload-frontend", "--plan-file", file, "--plan-json", "{}"},
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
