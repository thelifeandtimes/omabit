//go:build linux && amd64

package runtimequalify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordingHomeRequiresExactManagedRootBeforeAccountOrStorage(t *testing.T) {
	f, _, _ := managedFixture(t)
	f.d.identity = func() (identity, error) { t.Fatal("unbound root reached owner probe"); return identity{}, nil }
	f.d.stat = func(string) (node, error) { t.Fatal("unbound root reached storage"); return node{}, nil }
	if _, err := prepareManagedRecordingHome(filepath.Join(f.d.home, "unrelated"), f.d); err == nil {
		t.Fatal("arbitrary manager directory targeted installed storage")
	}
}

func TestRecordingHomeCreatesPrivateRootsAndReusesWithoutChangingOriginalContents(t *testing.T) {
	f, _, _ := managedFixture(t)
	base := filepath.Join(f.d.home, ".local/share/urbit-runner")
	managed := filepath.Join(base, "managed")
	opts, err := prepareManagedRecordingHome(managed, f.d)
	if err != nil || opts.JobsRoot != filepath.Join(base, "recording-jobs") || opts.RecordingsRoot != filepath.Join(base, "recordings") {
		t.Fatal("selected home preparation failed", err)
	}
	info := map[string]os.FileInfo{}
	for _, leaf := range []string{"recording-jobs", "recordings", "recording-images"} {
		path := filepath.Join(base, leaf)
		stat, err := os.Lstat(path)
		if err != nil || stat.Mode() != os.ModeDir|0700 {
			t.Fatal("root is not private", leaf, err)
		}
		info[leaf] = stat
		if err := os.WriteFile(filepath.Join(path, "retained-original"), []byte(leaf), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prepareManagedRecordingHome(managed, f.d); err != nil {
		t.Fatal("private roots were replaced instead of reused", err)
	}
	for leaf, before := range info {
		path := filepath.Join(base, leaf)
		after, err := os.Lstat(path)
		raw, readErr := os.ReadFile(filepath.Join(path, "retained-original"))
		if err != nil || readErr != nil || !os.SameFile(before, after) || string(raw) != leaf {
			t.Fatal("setup changed an existing object", leaf, err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatal("home setup contacted native engine")
	}
}

func TestRecordingHomeRefusesExposedSymlinkAndFileWithoutRepair(t *testing.T) {
	for _, scenario := range []string{"exposed", "symlink", "file"} {
		t.Run(scenario, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			base := filepath.Join(f.d.home, ".local/share/urbit-runner")
			path := filepath.Join(base, "recordings")
			switch scenario {
			case "exposed":
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(path, []byte("retained user content"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(t.TempDir(), path); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := prepareManagedRecordingHome(filepath.Join(base, "managed"), f.d); err == nil {
				t.Fatal("unsafe existing root adopted")
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || len(f.calls) != 0 {
				t.Fatal("refused setup repaired original or contacted engine", err)
			}
		})
	}
}
