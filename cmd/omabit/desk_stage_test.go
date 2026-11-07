package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskStageArgumentsCannotRedirectOrOpenState(t *testing.T) {
	config := filepath.Join(t.TempDir(), "unopened")
	for _, args := range [][]string{
		{"desk-plan-stage", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--instance", "alpha", "--desk", "../base", "--source-dir", "/unread"},
		{"desk-stage", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--plan-json", "{} {}", "--source-dir", "/unread"},
		{"desk-stage-status", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--id", "../outside"},
		{"desk-stage-discard", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--id", strings.Repeat("a", 64)},
		{"desk-stage-discard", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--id", strings.Repeat("a", 64), "--source-digest", strings.Repeat("b", 64), "--source-dir", "/unread"},
		{"desk-stage-status", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--id", strings.Repeat("a", 64), "--source-digest", strings.Repeat("b", 64)},
	} {
		if run(context.Background(), append([]string{"--config-dir", config}, args...)) == 0 {
			t.Fatal("unsafe arguments accepted", args)
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid arguments opened state", err)
	}
}
