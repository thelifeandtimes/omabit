package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAppWorkflowRejectsRedirectedAndPermissionRequestsBeforeState(t *testing.T) {
	config := filepath.Join(t.TempDir(), "must-not-exist")
	cases := [][]string{
		{"app-development-workflow", "--request-json", `{"name":"fixture","runner_id":"runner-fixture","target":{"instance_id":"alpha"},"allow_commit":true}`},
		{"app-development-configure", "--request-json", `{"name":"fixture","runner_id":"runner-fixture","target":{"instance_id":"alpha"},"update":{"target":{"instance_id":"other"},"ship":{"purpose":"standard","instructions":""}}}`},
		{"app-development-workflow", "--request-json", `{} {}`},
		{"desk-settings", "--runner", "fixture", "--expected-runner-id", "runner-fixture"},
		{"desk-source-status", "--runner", "fixture", "--expected-runner-id", "runner-fixture", "--instance", "alpha"},
	}
	for _, args := range cases {
		if run(context.Background(), append([]string{"--config-dir", config}, args...)) == 0 {
			t.Fatal("invalid workflow command accepted", args)
		}
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("invalid command opened profile", err)
	}
}
