//go:build linux

package projectusage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func testBinding() Binding {
	return Binding{RunnerID: "runner", Mode: "development-managed", ProjectID: "project", Revision: 1, Generation: 1, PlanDigest: strings.Repeat("a", 64), LedgerSHA256: strings.Repeat("b", 64)}
}

func testRoot(t *testing.T, kind string) Root {
	t.Helper()
	p, err := os.MkdirTemp(os.TempDir(), "pu-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(p) })
	if err := os.Chmod(p, 0700); err != nil {
		t.Fatal(err)
	}
	r := Root{ID: "root", Kind: kind, Path: p}
	if kind != "project-cache" {
		r.InstanceID = "member"
		r.RuntimeID = strings.Repeat("a", 32)
	}
	if kind == "member-mount" {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		fp, _, err := object(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		r.Expected = fp.identity()
	}
	return r
}

func sourceBytes(t *testing.T, p string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if err := filepath.WalkDir(p, func(name string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			b, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			h := sha256.Sum256(b)
			out[name] = hex.EncodeToString(h[:])
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAllocationCountsSparseInodesHardlinksDirectoriesAndSpecialMetadata(t *testing.T) {
	r := testRoot(t, "member-mount")
	if err := os.Mkdir(filepath.Join(r.Path, "child"), 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(r.Path, "data"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(8 << 20); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("original private fixture bytes"), 0); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err := os.Link(filepath.Join(r.Path, "data"), filepath.Join(r.Path, "child", "linked")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(r.Path, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(r.Path, "socket"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	before := sourceBytes(t, r.Path)
	start := time.Now()
	out, err := Measure(t.Context(), testBinding(), []Root{r}, MaxEntries)
	if err != nil || out.Validate() != nil || out.Status != "measured" || out.Totals == nil {
		t.Fatal("measurement unavailable", out.Reason, err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("FIFO/socket inspection blocked")
	}
	got := *out.Totals
	if got.RegularFiles != 2 || got.Directories != 2 || got.SpecialEntries != 2 || got.UniqueFileInodes != 3 || got.HardlinkEntries != 2 || got.LogicalBytes != 8<<20 || got.LogicalBytes <= got.AllocatedBytes {
		t.Fatal("inode or sparse accounting changed", got)
	}
	var allocated uint64
	seen := map[inodeKey]bool{}
	err = filepath.WalkDir(r.Path, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		s := info.Sys().(*syscall.Stat_t)
		key := inodeKey{s.Dev, s.Ino}
		if !seen[key] {
			seen[key] = true
			allocated += uint64(s.Blocks) * 512
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AllocatedBytes != allocated || out.FullProjectBudgetKnown || out.SharedAllowanceStatus != "not-configured" || !reflect.DeepEqual(out.ExcludedComponents, Exclusions()) || !reflect.DeepEqual(before, sourceBytes(t, r.Path)) {
		t.Fatal("scope omitted or source contents changed", out)
	}
}

func TestAllocationUnknownNeverPublishesPartialTotalsOrAdoptsMissingRoots(t *testing.T) {
	for _, variant := range []string{"symlink", "external-hardlink", "missing", "wrong-identity", "public-cache", "entry-limit", "depth-limit", "cancelled"} {
		t.Run(variant, func(t *testing.T) {
			r := testRoot(t, "member-mount")
			good := testRoot(t, "project-cache")
			good.ID = "first"
			if err := os.WriteFile(filepath.Join(r.Path, "data"), []byte("untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			limit := MaxEntries
			want := ""
			switch variant {
			case "symlink":
				if err := os.Symlink(filepath.Join(r.Path, "data"), filepath.Join(r.Path, "link")); err != nil {
					t.Fatal(err)
				}
				want = "unsafe-storage"
			case "external-hardlink":
				if err := os.Link(filepath.Join(r.Path, "data"), filepath.Join(t.TempDir(), "outside")); err != nil {
					t.Fatal(err)
				}
				want = "external-hardlink"
			case "missing":
				if err := os.Remove(filepath.Join(r.Path, "data")); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(r.Path); err != nil {
					t.Fatal(err)
				}
				want = "unavailable"
			case "wrong-identity":
				r.Expected.Inode++
				want = "changed"
			case "public-cache":
				r.Kind = "runtime-cache"
				r.Expected = good.Expected
				if err := os.Chmod(r.Path, 0755); err != nil {
					t.Fatal(err)
				}
				want = "unsafe-storage"
			case "entry-limit":
				limit = 2
				want = "entry-limit"
			case "depth-limit":
				p := r.Path
				for i := 0; i < MaxDepth+1; i++ {
					p = filepath.Join(p, "d")
					if err := os.Mkdir(p, 0700); err != nil {
						t.Fatal(err)
					}
				}
				want = "depth-limit"
			case "cancelled":
				cancel()
				want = "cancelled"
			}
			out, err := Measure(ctx, testBinding(), []Root{good, r}, limit)
			if err != nil || out.Status != "unknown" || out.Reason != want || out.Totals != nil || len(out.Roots) != 0 || out.Validate() != nil {
				t.Fatal("unknown exposed partial/false totals", variant, out, err)
			}
			if variant == "missing" {
				if _, err := os.Lstat(r.Path); !os.IsNotExist(err) {
					t.Fatal("missing enrolled root recreated")
				}
			}
		})
	}
}

type mutationContext struct {
	context.Context
	calls  int
	mutate func()
}

func (c *mutationContext) Err() error {
	c.calls++
	if c.calls == 4 {
		c.mutate()
	}
	return nil
}

func TestAllocationRejectsFileReplacementDuringMetadataCount(t *testing.T) {
	r := testRoot(t, "member-mount")
	path := filepath.Join(r.Path, "data")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	c := &mutationContext{Context: context.Background(), mutate: func() {
		if err := os.Rename(path, path+"-saved"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("replacement"), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	out, err := Measure(c, testBinding(), []Root{r}, MaxEntries)
	if err != nil || out.Status != "unknown" || out.Reason != "changed" || out.Totals != nil {
		t.Fatal("replaced file accepted", out, err)
	}
	data, err := os.ReadFile(path + "-saved")
	if err != nil || string(data) != "original" {
		t.Fatal("observer changed original bytes", err)
	}
}

func TestAllocationCrossMountFingerprintAndOverlappingRootsRefuse(t *testing.T) {
	r := testRoot(t, "member-mount")
	f, err := os.Open(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fp, info, err := object(f)
	if err != nil {
		t.Fatal(err)
	}
	s := treeScan{ctx: t.Context(), remaining: 2, mount: fp.mount + 1, device: fp.dev, files: map[inodeKey]*linkCount{}, directories: map[inodeKey]bool{}}
	if err := s.count(fp, info); Reason(err) != "cross-mount" {
		t.Fatal("foreign mount admitted", err)
	}
	other := r
	other.ID = "other"
	if _, err := Measure(t.Context(), testBinding(), []Root{r, other}, MaxEntries); err == nil {
		t.Fatal("same tree double charged")
	}
	other.Path = filepath.Join(r.Path, "nested")
	if _, err := Measure(t.Context(), testBinding(), []Root{r, other}, MaxEntries); err == nil {
		t.Fatal("nested declared roots double charged")
	}
}
