//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"golang.org/x/sys/unix"
)

func privatePillFixture(t *testing.T) (privateMemberContract, pierstore.DirectoryIdentity, protocol.BootPill, []byte) {
	t.Helper()
	original := privateMemberContractFixture(t, "sender", "boot")
	c, err := newPrivateMemberContract(original.selection, t.TempDir(), original.imageID)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Chmod(c.root, 0700))
	must(os.Mkdir(filepath.Dir(c.pillPath), 0700))
	raw := []byte("small synthetic private pill\n")
	must(os.WriteFile(c.pillPath, raw, 0400))
	id, err := pierstore.IdentifyMount(filepath.Dir(c.pillPath))
	must(err)
	return c, id, protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Bytes: int64(len(raw))}, raw
}
func TestPrivateMemberPillHeldVerification(t *testing.T) {
	c, id, meta, raw := privatePillFixture(t)
	v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, meta)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	c.pillPath = "changed"
	id.Inode++
	meta.SHA256 = "changed"
	for range 3 {
		if err = v.Verify(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(filepath.Join(v.root, v.runtime, candidate.PillName))
	if err != nil || string(got) != string(raw) {
		t.Fatal("verifier changed file")
	}
	if err = v.Verify(context.Background()); err != nil {
		t.Fatal("atime read invalidated pill", err)
	}
	if !v.verified {
		t.Fatal("full hash never accepted")
	}
	if err = v.Close(); err != nil {
		t.Fatal(err)
	}
	if err = v.Close(); err != nil {
		t.Fatal(err)
	}
	if v.Verify(context.Background()) == nil {
		t.Fatal("closed verifier accepted")
	}
}
func TestPrivateMemberPillProductionPinsAndBootOnly(t *testing.T) {
	c, id, _, _ := privatePillFixture(t)
	if v, err := openPrivateMemberPill(context.Background(), c, id); err == nil {
		v.Close()
		t.Fatal("production accepted tiny substitute")
	}
	for _, kind := range []string{"resume", "path", "digest", "workspace", "missing-workspace", "meta-size", "meta-hash", "meta-zero"} {
		t.Run(kind, func(t *testing.T) {
			c, id, meta, _ := privatePillFixture(t)
			switch kind {
			case "resume":
				c = privateMemberContractFixture(t, "sender", "resume")
			case "path":
				c.pillPath += "-other"
			case "digest":
				c.digest = strings.Repeat("a", 64)
			case "workspace":
				id.Inode++
			case "missing-workspace":
				id = pierstore.DirectoryIdentity{}
			case "meta-size":
				meta.Bytes++
			case "meta-hash":
				meta.SHA256 = strings.Repeat("a", 64)
			case "meta-zero":
				meta.SHA256 = strings.Repeat("0", 64)
			}
			if v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, meta); err == nil {
				v.Close()
				t.Fatal("unbound pill accepted")
			}
		})
	}
}
func TestPrivateMemberPillUnsafeInputsPreserved(t *testing.T) {
	for _, kind := range []string{"empty", "large", "writable", "public", "hardlink", "symlink", "fifo", "directory", "extra", "public-workspace", "public-root", "symlink-root"} {
		t.Run(kind, func(t *testing.T) {
			c, id, meta, _ := privatePillFixture(t)
			original := c.pillPath
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "empty", "large":
				must(os.Chmod(c.pillPath, 0600))
				size := int64(0)
				if kind == "large" {
					size = meta.Bytes + 1
				}
				must(os.Truncate(c.pillPath, size))
				must(os.Chmod(c.pillPath, 0400))
			case "writable":
				must(os.Chmod(c.pillPath, 0600))
			case "public":
				must(os.Chmod(c.pillPath, 0444))
			case "hardlink":
				must(os.Link(c.pillPath, filepath.Join(c.root, "alias")))
			case "symlink":
				kept := filepath.Join(c.root, "retained-pill")
				must(os.Rename(c.pillPath, kept))
				must(os.Symlink(kept, c.pillPath))
			case "fifo":
				must(os.Remove(c.pillPath))
				must(unix.Mkfifo(c.pillPath, 0400))
			case "directory":
				must(os.Remove(c.pillPath))
				must(os.Mkdir(c.pillPath, 0700))
			case "extra":
				must(os.WriteFile(filepath.Join(filepath.Dir(c.pillPath), "unselected"), []byte("retained"), 0600))
			case "public-workspace":
				must(os.Chmod(filepath.Dir(c.pillPath), 0755))
			case "public-root":
				must(os.Chmod(c.root, 0755))
			case "symlink-root":
				link := filepath.Join(t.TempDir(), "linked-root")
				must(os.Symlink(c.root, link))
				var err error
				c, err = newPrivateMemberContract(c.selection, link, c.imageID)
				must(err)
			}
			v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, meta)
			if err == nil {
				v.Close()
				t.Fatal("unsafe pill accepted")
			}
			if strings.Contains(err.Error(), original) {
				t.Fatal("path leaked")
			}
			if _, err = os.Lstat(original); err != nil {
				t.Fatal("refusal removed retained file")
			}
		})
	}
}
func TestPrivateMemberPillRejectsChangedVerifiedObjects(t *testing.T) {
	for _, kind := range []string{"root", "workspace", "pill", "contents-restored-mtime", "metadata", "mode", "extra"} {
		t.Run(kind, func(t *testing.T) {
			c, id, meta, raw := privatePillFixture(t)
			v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, meta)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "root", "workspace":
				target := c.root
				if kind == "workspace" {
					target = filepath.Dir(c.pillPath)
				}
				kept := filepath.Join(t.TempDir(), "retained")
				must(os.Rename(target, kept))
				must(os.Mkdir(target, 0700))
				entries, err := os.ReadDir(kept)
				must(err)
				for _, entry := range entries {
					must(os.Rename(filepath.Join(kept, entry.Name()), filepath.Join(target, entry.Name())))
				}
			case "pill":
				must(os.Rename(c.pillPath, filepath.Join(c.root, "retained-pill")))
				must(os.WriteFile(c.pillPath, raw, 0400))
			case "contents-restored-mtime":
				before, err := os.Stat(c.pillPath)
				must(err)
				must(os.Chmod(c.pillPath, 0600))
				must(os.WriteFile(c.pillPath, []byte(strings.Repeat("x", len(raw))), 0400))
				must(os.Chmod(c.pillPath, 0400))
				must(os.Chtimes(c.pillPath, before.ModTime(), before.ModTime()))
			case "metadata":
				must(os.Chtimes(c.pillPath, time.Unix(1, 0), time.Unix(1, 0)))
			case "mode":
				must(os.Chmod(c.pillPath, 0600))
			case "extra":
				must(os.WriteFile(filepath.Join(filepath.Dir(c.pillPath), "new"), []byte("retained"), 0600))
			}
			if v.Verify(context.Background()) == nil {
				t.Fatal("changed verified object accepted")
			}
			if _, err = os.Lstat(c.pillPath); err != nil {
				t.Fatal("refusal removed pill")
			}
		})
	}
}
func TestPrivateMemberPillCancellationAndConcurrentClose(t *testing.T) {
	c, id, meta, _ := privatePillFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := openPrivateMemberPillWithMeta(ctx, c, id, meta); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, meta)
	if err != nil {
		t.Fatal(err)
	}
	if err = v.Verify(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() { defer group.Done(); _ = v.Verify(context.Background()); _ = v.Close() }()
	}
	group.Wait()
	if v.Verify(context.Background()) == nil {
		t.Fatal("closed verifier accepted")
	}
}
