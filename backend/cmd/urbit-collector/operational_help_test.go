package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationalHelpDoesNotOpenRegistry(t *testing.T) {
	for _, command := range []string{"logs", "login-code", "preview-import-restore", "restore-import"} {
		for _, suffix := range [][]string{{"--help"}, {"-h"}, {"example", "--help"}, {"example", "-h"}} {
			t.Run(command+strings.Join(suffix, "-"), func(t *testing.T) {
				dir := t.TempDir()
				args := []string{"--config-dir", filepath.Join(dir, "unopened"), command}
				var out, errs bytes.Buffer
				if code := run(context.Background(), append(args, suffix...), &out, &errs); code != 0 || errs.Len() != 0 || !strings.Contains(out.String(), "Usage:") {
					t.Fatalf("help requires live authority: code=%d stdout=%s stderr=%s", code, out.String(), errs.String())
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Fatalf("help touched enrollment state: %v %v", entries, err)
				}
			})
		}
	}
}

func TestOperationalHelpDoesNotWeakenRequiredInputs(t *testing.T) {
	for _, args := range [][]string{{"logs"}, {"logs", "example"}, {"login-code"}, {"login-code", "example"}, {"login-code", "example", "--reveal"}} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, &out, &errs); code != 2 || out.Len() != 0 || errs.Len() == 0 {
			t.Fatalf("missing binding/acknowledgement accepted for %v: %d %s %s", args, code, out.String(), errs.String())
		}
	}
}
