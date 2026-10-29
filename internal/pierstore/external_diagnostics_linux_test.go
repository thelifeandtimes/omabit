//go:build linux

package pierstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestTreeReviewNamesRejectedEntryWithoutReadingItsContents(t *testing.T) {
	for _, tc := range []struct {
		name, relative, reason string
		prepare                func(string) error
	}{
		{"world-write", "desk/app.hoon", "mode 0602", func(p string) error { return os.Chmod(p, 0602) }},
		{"state-group-write", ".urb/log/state", "mode 0660", func(p string) error { return os.Chmod(p, 0660) }},
		{"quoted-name", "desk/line\nbreak", "mode 0602", func(p string) error { return os.Chmod(p, 0602) }},
		{"symlink", "desk/link", "symbolic link", func(p string) error {
			if err := os.Remove(p); err != nil {
				return err
			}
			return os.Symlink("SYNTHETIC-PRIVATE-LINK-TARGET", p)
		}},
		{"fifo", "desk/pipe", "special file", func(p string) error {
			if err := os.Remove(p); err != nil {
				return err
			}
			return unix.Mkfifo(p, 0600)
		}},
		{"hardlink", "desk/linked", "2 hard links", func(p string) error {
			return os.Link(p, filepath.Join(filepath.Dir(p), "second"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pier := t.TempDir()
			p := filepath.Join(pier, tc.relative)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			const contents = "SYNTHETIC-PRIVATE-CONTENTS"
			if err := os.WriteFile(p, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			if err := tc.prepare(p); err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(p)
			if err != nil {
				t.Fatal(err)
			}
			_, _, _, err = TreeReview(context.Background(), pier, true)
			if err == nil {
				t.Fatal("unsafe entry accepted")
			}
			message := err.Error()
			if !strings.Contains(message, fmt.Sprintf("%q", tc.relative)) || !strings.Contains(message, tc.reason) {
				t.Fatalf("missing entry or reason: %s", message)
			}
			for _, private := range []string{pier, contents, "SYNTHETIC-PRIVATE-LINK-TARGET", "\n"} {
				if strings.Contains(message, private) {
					t.Fatalf("diagnostic disclosed content, source root, link target or unescaped newline: %q", message)
				}
			}
			after, err := os.Lstat(p)
			if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatal("review changed the refused object", err)
			}
		})
	}
}

func TestTreeReviewNamesDockedLinkOutsidePier(t *testing.T) {
	pier := t.TempDir()
	run := filepath.Join(pier, ".run")
	if err := os.WriteFile(run, []byte("inert executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(run, filepath.Join(t.TempDir(), "outside")); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := TreeReview(context.Background(), pier, true)
	if err == nil || !strings.Contains(err.Error(), `pier entry ".run"`) || !strings.Contains(err.Error(), "outside the reviewed pier") {
		t.Fatal("missing docked-link diagnostic", err)
	}
}
