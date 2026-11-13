package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeskCommandRejectsInvalidPlansBeforeOpeningProfile(t *testing.T) {
	config := filepath.Join(t.TempDir(), "must-not-exist")
	cases := [][]string{
		{"desk-app-status", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--instance", "publisher"},
		{"desk-app-status", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--instance", "publisher", "--desk", "base", "--id", "unexpected"},
		{"desk-list", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--instance", "publisher", "--desk", "base"},
		{"desk-list", "--runner", "fixture", "--expected-runner-id", "runner-fixture"},
		{"desk-create", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--plan-json", `{"version":1,"unknown":true}`},
		{"desk-create", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--plan-json", `{} {}`},
		{"desk-operation", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--id", "abc", "--desk", "example"},
		{"desk-plan-create", "--runner", "fixture", "--expected-runner-id", "runner-fixture"},
		{"desk-inspect", "--runner", "fixture", "--instance", "publisher", "--desk", "example"},
	}
	for _, args := range cases {
		if run(context.Background(), append([]string{"--config-dir", config}, args...)) == 0 {
			t.Fatal("invalid arguments accepted", args)
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid arguments opened profile", err)
	}
}
