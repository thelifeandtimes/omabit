//go:build linux

package pierstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRC11RemovalRechecksReplacedAncestorBetweenUnlinks(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "tree")
	if err := os.MkdirAll(filepath.Join(path, ".urb/log"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"z-last", ".urb/log/state"} {
		if err := os.WriteFile(filepath.Join(path, name), []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	// A restart discards all transient filesystem fields; the durable manifest
	// must still bind every object, not just its original root directory.
	data, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	tree = &RemovalTree{}
	if err := json.Unmarshal(data, tree); err != nil {
		t.Fatal(err)
	}
	unlinks := 0
	err = tree.Remove(context.Background(), path, nil, func(root *os.Root, name string) error {
		unlinks++
		if err := root.Remove(name); err != nil {
			return err
		}
		if unlinks == 1 {
			if err := os.Rename(filepath.Join(path, ".urb"), filepath.Join(parent, "original-urb")); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(path, ".urb/log"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, ".urb/log/state"), []byte("replacement"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	})
	if err == nil || unlinks != 1 {
		t.Fatal("replacement ancestor redirected deletion", err, unlinks)
	}
	for _, p := range []string{filepath.Join(path, ".urb/log/state"), filepath.Join(parent, "original-urb/log/state")} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("retained data lost", err)
		}
	}
}

func TestRC11RemovalManifestResumesWithoutUrb(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tree")
	if err := os.MkdirAll(filepath.Join(path, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	tree, err := ReviewRemoval(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	first := true
	err = tree.Remove(context.Background(), path, nil, func(root *os.Root, name string) error {
		if first {
			first = false
			if err := root.Remove(name); err != nil {
				return err
			}
			return errors.New("outcome lost")
		}
		return root.Remove(name)
	})
	if err == nil {
		t.Fatal("interruption missed")
	}
	if err := tree.Remove(context.Background(), path, nil, nil); err != nil {
		t.Fatal("manifest could not resume without .urb", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("tree not removed", err)
	}
}

func TestRC11NamedRootDestinationNamespaceBarriers(t *testing.T) {
	home, volume := t.TempDir(), t.TempDir()
	if _, err := RegisterRoot(home, "fixture", volume); err != nil {
		t.Fatal(err)
	}
	location := Location{Path: "@fixture/fakeships/group/ship/pier"}
	mount, err := PrepareMount(home, &location)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(mount, "pier")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	expected := []string{path, mount, filepath.Dir(mount), filepath.Join(volume, "fakeships"), volume, filepath.Dir(volume)}
	for fail := -1; fail < len(expected); fail++ {
		var synced []string
		err := SyncDestination(home, location, func(p string) error {
			synced = append(synced, p)
			if len(synced)-1 == fail {
				return errors.New("barrier failed")
			}
			return SyncDirectory(p)
		})
		want := expected
		if fail >= 0 {
			want = expected[:fail+1]
		}
		if !reflect.DeepEqual(synced, want) || (err != nil) != (fail >= 0) {
			t.Fatal("incorrect barrier order or ignored failure", synced, err)
		}
	}
}
