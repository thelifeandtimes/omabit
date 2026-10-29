//go:build linux

package pierstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBoundRenameChecksOpenedParentsSourceAndNeverReplaces(t *testing.T) {
	for _, fault := range []string{"none", "source-parent", "target-parent", "source", "source-at-boundary", "occupied-target", "boundary-error", "missing-boundary"} {
		t.Run(fault, func(t *testing.T) {
			root := t.TempDir()
			a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
			source, target := filepath.Join(a, "source"), filepath.Join(b, "target")
			for _, path := range []string{source, b} {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(source, "data"), []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			object, err := IdentifyMount(source)
			if err != nil {
				t.Fatal(err)
			}
			pa, err := IdentifyMount(a)
			if err != nil {
				t.Fatal(err)
			}
			pb, err := IdentifyMount(b)
			if err != nil {
				t.Fatal(err)
			}
			original := source
			replace := func(path string) error {
				if err := os.Rename(path, path+"-retained"); err != nil {
					return err
				}
				return os.Mkdir(path, 0700)
			}
			boundary := func() error { return nil }
			switch fault {
			case "source-parent":
				err = replace(a)
				original = filepath.Join(a+"-retained", "source")
			case "target-parent":
				err = replace(b)
			case "source":
				err = replace(source)
				original = source + "-retained"
			case "source-at-boundary":
				boundary = func() error { original = source + "-retained"; return replace(source) }
			case "occupied-target":
				err = os.Mkdir(target, 0700)
				if e := os.WriteFile(filepath.Join(target, "data"), []byte("existing target"), 0600); e != nil {
					t.Fatal(e)
				}
			case "boundary-error":
				boundary = func() error { return errors.New("replaced ancestor or busy pier") }
			case "missing-boundary":
				boundary = nil
			}
			if err != nil {
				t.Fatal(err)
			}
			err = RenameBoundDirectory(source, target, object, pa, pb, boundary)
			if fault == "none" {
				if err != nil {
					t.Fatal(err)
				}
				original = target
				if _, err := os.Lstat(source); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("old source remains", err)
				}
			} else if err == nil {
				t.Fatal("unsafe rename succeeded")
			}
			value, err := os.ReadFile(filepath.Join(original, "data"))
			if err != nil || string(value) != "original" {
				t.Fatal("source bytes lost", err)
			}
			if fault == "occupied-target" {
				value, err := os.ReadFile(filepath.Join(target, "data"))
				if err != nil || string(value) != "existing target" {
					t.Fatal("existing target overwritten", err)
				}
			}
		})
	}
}
