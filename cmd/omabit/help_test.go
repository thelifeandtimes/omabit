package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopCommandHelpNeverOpensState(t *testing.T) {
	for command := range desktopCommandHelp {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			blocked := filepath.Join(dir, "not-a-directory")
			if err := os.WriteFile(blocked, []byte("untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, flag := range []string{"--help", "-h"} {
				if code := run(context.Background(), []string{"--config-dir", blocked, "--state-dir", blocked, command, flag}); code != 0 {
					t.Fatalf("help exit = %d", code)
				}
			}
			data, err := os.ReadFile(blocked)
			if err != nil || string(data) != "untouched" {
				t.Fatalf("help touched state: %q %v", data, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("help created state: %v %v", entries, err)
			}
		})
	}
}
