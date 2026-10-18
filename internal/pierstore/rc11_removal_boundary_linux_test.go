//go:build linux

package pierstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRC11RemovalParentSwapAtUnlinkCannotRedirectToReplacement(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "tree")
	pier := filepath.Join(path, "pier")
	if err := os.MkdirAll(filepath.Join(pier, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a-retained", "data"} {
		if err := os.WriteFile(filepath.Join(pier, ".urb", name), []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	tree = &RemovalTree{}
	if err := json.Unmarshal(b, tree); err != nil {
		t.Fatal(err)
	}
	retained := filepath.Join(parent, "original-pier")
	calls := 0
	err = tree.Remove(context.Background(), path, func() error {
		if Busy(pier) {
			return errors.New("native lock")
		}
		return nil
	}, func(root *os.Root, name string) error {
		calls++
		if name != "data" || calls != 1 {
			t.Fatal("unlink did not receive an immediate parent and basename", name, calls)
		}
		// The critic's substitution hook runs after the final checks and before
		// the actual unlink. A detached authorized file may be removed, but this
		// must never resolve the replacement pier's identically named file.
		if err := os.Rename(pier, retained); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(pier, ".urb"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pier, ".urb/data"), []byte("replacement"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pier, ".vere.lock"), []byte("native"), 0600); err != nil {
			t.Fatal(err)
		}
		opened, err := root.Stat(".")
		if err != nil {
			t.Fatal(err)
		}
		original, err := os.Stat(filepath.Join(retained, ".urb"))
		if err != nil || !os.SameFile(opened, original) {
			t.Fatal("unlink parent was not pinned to the original object", err)
		}
		return root.Remove(name)
	})
	if err == nil || calls != 1 {
		t.Fatal("substitution did not stop further removal", err, calls)
	}
	for p, want := range map[string]string{
		filepath.Join(pier, ".urb/data"):           "replacement",
		filepath.Join(pier, ".vere.lock"):          "native",
		filepath.Join(retained, ".urb/a-retained"): "original",
	} {
		data, err := os.ReadFile(p)
		if err != nil || string(data) != want {
			t.Fatal("unexplained/remainder data was removed", p, err)
		}
	}
}

func TestRC11RemovalBoundaryChangeBeforeUnlinkPreservesBothTrees(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "tree")
	if err := os.MkdirAll(filepath.Join(path, "pier/.urb"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(path, "pier/.urb/data")
	if err := os.WriteFile(file, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = tree.Remove(context.Background(), path, func() error {
		if err := os.Rename(filepath.Join(path, "pier"), filepath.Join(parent, "original")); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("replacement"), 0600); err != nil {
			t.Fatal(err)
		}
		return nil
	}, func(*os.Root, string) error { calls++; return errors.New("unexpected unlink") })
	if err == nil || calls != 0 {
		t.Fatal("boundary change was not revalidated", err, calls)
	}
	for _, p := range []string{file, filepath.Join(parent, "original/.urb/data")} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("tree lost", err)
		}
	}
}

func TestRC11RemovalRefusesSameSizeRewriteWithRestoredMtime(t *testing.T) {
	path := t.TempDir()
	file := filepath.Join(path, "file")
	if err := os.WriteFile(file, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("new-data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := tree.Remove(context.Background(), path, nil, nil); err == nil {
		t.Fatal("ctime change ignored")
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != "new-data" {
		t.Fatal("updated bytes lost", err)
	}
}

func TestRC11RemovalContinuesAcrossOwnedDockedHardlinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pier")
	if err := os.MkdirAll(filepath.Join(path, ".bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, ".run"), []byte("inert executable fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(path, ".run"), filepath.Join(path, ".bin/docked")); err != nil {
		t.Fatal(err)
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := tree.Remove(context.Background(), path, nil, nil); err != nil {
		t.Fatal("own hardlink unlink ctime stranded removal", err)
	}
}
