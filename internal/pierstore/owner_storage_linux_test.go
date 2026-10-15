//go:build linux

package pierstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnerStorageCreatesHomeDirectoriesAndPinsShortcutTarget(t *testing.T) {
	home := t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	actual := filepath.Join(home, "projects")
	if err := os.Mkdir(actual, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(actual, 0775); err != nil {
		t.Fatal(err)
	}
	shortcut := filepath.Join(home, "shortcut")
	if err := os.Symlink(actual, shortcut); err != nil {
		t.Fatal(err)
	}
	r, err := registerRoot(home, "development", filepath.Join(shortcut, "new", "piers"), true)
	if err != nil || r.Path != filepath.Join(actual, "new", "piers") {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(filepath.Join(r.Path, "fakeships")); err != nil {
		t.Fatal(err)
	}
	if _, err := registerRoot(home, "private", filepath.Join(home, ".local/share/urbit-runner/new"), true); err == nil {
		t.Fatal("private root accepted")
	}
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatal("private path was created before refusal")
	}
}

func TestOwnerExternalWorkspaceAllowsOrdinaryGroupWritesButProtectsState(t *testing.T) {
	pier := t.TempDir()
	for _, dir := range []string{".urb", ".urb/log", ".urb/chk", "base"} {
		if err := os.Mkdir(filepath.Join(pier, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(pier, 0775); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pier, "base", "app.hoon")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0664); err != nil {
		t.Fatal(err)
	}
	if _, err := identifyMode(pier, true); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := TreeReview(context.Background(), pier, true); err != nil {
		t.Fatal(err)
	}
	if _, err := identifyMode(pier, false); err == nil {
		t.Fatal("legacy identity accepted writable workspace")
	}
	if err := os.Chmod(filepath.Join(pier, ".urb"), 0775); err != nil {
		t.Fatal(err)
	}
	if _, err := identifyMode(pier, true); err == nil {
		t.Fatal("writable state accepted")
	}
	if _, _, _, err := TreeReview(context.Background(), pier, true); err == nil {
		t.Fatal("writable state tree accepted")
	}
}
