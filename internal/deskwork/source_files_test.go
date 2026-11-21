package deskwork

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceFilesPreserveBackendBytesWithoutFrontendIndex(t *testing.T) {
	values := map[string]string{"bill.json": `{"state":"/state"}`, "nex/state.hoon": ":: exact source\n", "assets/data.bin": "\x00\xff\x00"}
	dir := t.TempDir()
	for name, value := range values {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := fixture(t, values)
	files, err := ReadSourceFiles(t.Context(), dir, m)
	if err != nil || len(files) != len(values) {
		t.Fatal(err)
	}
	for _, file := range files {
		if string(file.Bytes) != values[file.Path] {
			t.Fatal("source bytes changed")
		}
	}
	if _, _, err := ReadFrontendFiles(t.Context(), dir, m); err == nil {
		t.Fatal("backend stage silently became a frontend artifact")
	}
	if err := os.WriteFile(filepath.Join(dir, "nex/state.hoon"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSourceFiles(t.Context(), dir, m); err == nil {
		t.Fatal("changed source accepted")
	}
}
func TestSourceFilesRefuseExtraFilesLinksAndCanceledReads(t *testing.T) {
	for _, kind := range []string{"extra", "file-link", "root-link", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			name := filepath.Join(dir, "file.txt")
			if err := os.WriteFile(name, []byte("exact"), 0600); err != nil {
				t.Fatal(err)
			}
			m := fixture(t, map[string]string{"file.txt": "exact"})
			ctx := t.Context()
			switch kind {
			case "extra":
				if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("unreviewed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "file-link":
				if err := os.Rename(name, filepath.Join(dir, "target.txt")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("target.txt", name); err != nil {
					t.Fatal(err)
				}
			case "root-link":
				link := filepath.Join(t.TempDir(), "source")
				if err := os.Symlink(dir, link); err != nil {
					t.Fatal(err)
				}
				dir = link
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := ReadSourceFiles(ctx, dir, m); err == nil {
				t.Fatal("unverified source read", kind)
			}
		})
	}
}
