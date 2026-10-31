//go:build linux

package piercopy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func tree(t *testing.T) (string, string, *os.Root, *os.Root) {
	t.Helper()
	s, d := t.TempDir(), t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s, ".urb/chk"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s, ".urb/chk/image.bin"), []byte(strings.Repeat("pier\x00", 120000)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s, ".urb/lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	source, err := os.OpenRoot(s)
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.OpenRoot(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close(); target.Close() })
	return s, d, source, target
}

func TestCopyVerifiesContentAndPrivatePermissions(t *testing.T) {
	_, d, source, target := tree(t)
	ctx := context.Background()
	before, err := Measure(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	lastFiles, lastBytes := 0, int64(0)
	copied, err := Copy(ctx, source, target, func(files int, bytes int64) error {
		calls++
		if files < lastFiles || bytes < lastBytes {
			t.Error("copy progress went backwards")
		}
		lastFiles, lastBytes = files, bytes
		return nil
	})
	if err != nil || copied != before || copied.Files != 2 || copied.Bytes != 600000 || calls < 2 || lastFiles != 2 || lastBytes != 600000 {
		t.Fatalf("copy summary: %+v %v", copied, err)
	}
	for _, name := range []string{".urb", ".urb/chk", ".urb/chk/image.bin", ".urb/lock"} {
		info, err := os.Stat(filepath.Join(d, name))
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0600)
		if info.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Fatal("permissions not private", name)
		}
	}
	if _, err := Copy(ctx, source, target, func(int, int64) error { return nil }); err == nil {
		t.Fatal("overwrote existing destination")
	}
}

func TestCopyRejectsUnsafeTreesWithoutFollowingLinks(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "fifo", "oversize", "depth"} {
		t.Run(kind, func(t *testing.T) {
			s, _, source, target := tree(t)
			switch kind {
			case "symlink":
				if err := os.Symlink("/etc/passwd", filepath.Join(s, "escape")); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(s, ".urb/chk/image.bin"), filepath.Join(s, "shared")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := syscall.Mkfifo(filepath.Join(s, "pipe"), 0600); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.Truncate(filepath.Join(s, ".urb/chk/image.bin"), MaxBytes+1); err != nil {
					t.Fatal(err)
				}
			case "depth":
				if err := os.MkdirAll(filepath.Join(s, strings.Repeat("x/", MaxDepth+1)), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Copy(context.Background(), source, target, func(int, int64) error { return nil }); err == nil {
				t.Fatal("unsafe source accepted")
			}
		})
	}
}

func TestCopyPreservesInertDockedBinaryBytesAndRejectsExternalLinks(t *testing.T) {
	s, d, source, target := tree(t)
	bin := filepath.Join(s, ".bin/live/vere-v4.6")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("inert docked executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(bin, filepath.Join(s, ".run")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "shared-binary")
	if err := os.Link(bin, outside); err != nil {
		t.Fatal(err)
	}
	if _, err := Copy(context.Background(), source, target, func(int, int64) error { return nil }); err == nil {
		t.Fatal("external dock link accepted")
	}
	if err := os.Remove(outside); err != nil {
		t.Fatal(err)
	}
	verified := false
	if _, err := Copy(context.Background(), source, target, func(int, int64) error { return nil }, func() error { verified = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("verification phase was not reported")
	}
	for _, name := range []string{".run", ".bin/live/vere-v4.6"} {
		data, err := os.ReadFile(filepath.Join(d, name))
		if err != nil || string(data) != "inert docked executable" {
			t.Fatal(name, err)
		}
		info, err := os.Stat(filepath.Join(d, name))
		if err != nil || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			t.Fatal("destination still shares docked files", err)
		}
	}
}

func TestInventoryAcceptsLargeLocalPierAndRejectsAggregateOverflow(t *testing.T) {
	s, _, source, _ := tree(t)
	const limit int64 = MaxBytes
	// Only metadata is traversed here: the sparse file allocates no multi-terabyte payload.
	if err := os.Truncate(filepath.Join(s, ".urb/chk/image.bin"), limit); err != nil {
		t.Fatal(err)
	}
	entries, err := inventory(context.Background(), source)
	if err != nil {
		t.Fatal("large local tree refused", err)
	}
	if got := digest(entries).Bytes; got != limit {
		t.Fatalf("truncated logical size: %d", got)
	}
	if err := os.WriteFile(filepath.Join(s, "overflow"), []byte{1}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := inventory(context.Background(), source); err == nil {
		t.Fatal("tree larger than local bound accepted")
	}
}

func TestCopyDetectsMutationAndKeepsInterruptedTargets(t *testing.T) {
	for _, kind := range []string{"source-write", "source-add", "target-write", "journal", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			s, d, source, target := tree(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			called := false
			_, err := Copy(ctx, source, target, func(int, int64) error {
				if called {
					return nil
				}
				called = true
				switch kind {
				case "source-write":
					return os.WriteFile(filepath.Join(s, ".urb/chk/image.bin"), []byte("changed"), 0600)
				case "source-add":
					return os.WriteFile(filepath.Join(s, "extra"), []byte("new"), 0600)
				case "target-write":
					return os.WriteFile(filepath.Join(d, ".urb/chk/image.bin"), []byte("changed"), 0600)
				case "journal":
					return errors.New("journal unavailable")
				case "cancel":
					cancel()
				}
				return nil
			})
			if err == nil || !called {
				t.Fatal("mutation or interruption accepted")
			}
			if _, err := os.Stat(filepath.Join(d, ".urb/chk/image.bin")); err != nil {
				t.Fatal("partial target was deleted")
			}
		})
	}
}
