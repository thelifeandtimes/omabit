package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskHelpOfflineNavigation(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			registry := filepath.Join(dir, "unopened")
			// Even a broken selection must not obstruct offline guidance.
			globals := []string{"--config-dir", registry, "--runner", "absent", "--expected-runner-id", "invalid selection", "--socket", "/absent/runner.sock", "--format", format}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var out, errs bytes.Buffer
			if code := run(ctx, append(globals, "help"), &out, &errs); code != 0 || errs.Len() != 0 {
				t.Fatalf("root help needs runtime state: %d %s", code, errs.String())
			}
			if format == "text" {
				if !strings.Contains(out.String(), "help [TOPIC]") || !strings.Contains(out.String(), "Offline") {
					t.Fatalf("missing text navigation: %s", out.String())
				}
			} else {
				var root helpDocument
				if err := json.Unmarshal(out.Bytes(), &root); err != nil || root.SchemaVersion != 1 || len(root.Topics) == 0 || root.Topic != nil {
					t.Fatalf("bad root document: %s, %v", out.String(), err)
				}
				for _, summary := range root.Topics {
					if len(summary.Commands) != 0 || len(summary.Guidance) != 0 {
						t.Fatal("root should disclose summaries, not all command details")
					}
					out.Reset()
					if code := run(ctx, append(globals, "help", summary.Name), &out, &errs); code != 0 || errs.Len() != 0 {
						t.Fatalf("topic not reachable: %s: %d %s", summary.Name, code, errs.String())
					}
					var detail helpDocument
					if err := json.Unmarshal(out.Bytes(), &detail); err != nil || detail.Topic == nil || detail.Topic.Name != summary.Name || len(detail.Topic.Commands) == 0 || len(detail.Topic.Guidance) == 0 || len(detail.Topics) != 0 {
						t.Fatalf("invalid topic detail: %s, %v", out.String(), err)
					}
					// Follow every advertised command into its real parser. This
					// catches stale navigation without opening the registry.
					for _, command := range detail.Topic.Commands {
						var cmdOut, cmdErr bytes.Buffer
						if code := run(ctx, []string{"--config-dir", registry, command.Name, "--help"}, &cmdOut, &cmdErr); code != 0 || cmdErr.Len() != 0 || !strings.Contains(cmdOut.String(), "Usage:") {
							t.Fatalf("broken command help %s: %d %s %s", command.Name, code, cmdOut.String(), cmdErr.String())
						}
					}
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("help touched enrollment state: %v %v", entries, err)
			}
		})
	}
}

func TestTaskHelpInputErrorsAndAliases(t *testing.T) {
	for _, args := range [][]string{{"help", "missing"}, {"help", "ships", "extra"}, {"help", "--format", "text"}} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, &out, &errs); code != 2 || out.Len() != 0 || !json.Valid(errs.Bytes()) {
			t.Fatalf("invalid help input not rejected: %v: %d %s %s", args, code, out.String(), errs.String())
		}
	}
	for _, alias := range []string{"--help", "-h"} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), []string{"help", alias}, &out, &errs); code != 0 || errs.Len() != 0 || !json.Valid(out.Bytes()) {
			t.Fatalf("help alias failed: %s: %d %s %s", alias, code, out.String(), errs.String())
		}
	}
}
