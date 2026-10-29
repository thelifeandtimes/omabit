//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func privateProvisionBase(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	info, err := os.Stat(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("temporary base before explicit private setup: %04o", info.Mode().Perm())
	if err = os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestWorkspaceProvisionOnlyCreatesEmptyGroupingParent(t *testing.T) {
	base := privateProvisionBase(t)
	path := filepath.Join(base, "private-projects")
	id, err := ProvisionWorkspaceParent(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatal("provisioned project/runtime data", err)
	}
	again, err := ProvisionWorkspaceParent(context.Background(), path)
	if err != nil || !id.SameObject(again) {
		t.Fatal("empty initial parent replaced", err)
	}
	inspected, err := InspectWorkspaceParent(context.Background(), path)
	if err != nil || !id.SameObject(inspected) {
		t.Fatal("parent identity differs", err)
	}
}

func TestWorkspaceProvisionPreservesPartialMetadataOnSyncFailure(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(map[int]string{1: "child", 2: "parent"}[failAt], func(t *testing.T) {
			path := filepath.Join(privateProvisionBase(t), "private-projects")
			calls := 0
			_, err := provisionWorkspaceParent(context.Background(), path, func(f *os.File) error {
				calls++
				if calls == failAt {
					return errors.New("synthetic sync failure")
				}
				return f.Sync()
			})
			if err == nil {
				t.Fatal("sync failure accepted")
			}
			id, err := InspectWorkspaceParent(context.Background(), path)
			if err != nil {
				t.Fatal("partial empty parent removed", err)
			}
			again, err := ProvisionWorkspaceParent(context.Background(), path)
			if err != nil || !id.SameObject(again) {
				t.Fatal("safe pre-receipt grouping retry replaced its object", err)
			}
		})
	}
}

func TestWorkspaceProvisionRefusesUnknownContentsAndUnsafePaths(t *testing.T) {
	for _, mode := range []string{"nonempty", "file", "link", "ancestor-link", "public-child", "public-base", "wrong-name", "relative", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			base := privateProvisionBase(t)
			path := filepath.Join(base, "private-projects")
			ctx := context.Background()
			switch mode {
			case "nonempty":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "retained"), []byte("do not adopt"), 0600); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(path, []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			case "link":
				if err := os.Symlink(base, path); err != nil {
					t.Fatal(err)
				}
			case "ancestor-link":
				link := base + "-alias"
				if err := os.Symlink(base, link); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(link) })
				path = filepath.Join(link, "private-projects")
			case "public-child":
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			case "public-base":
				if err := os.Chmod(base, 0755); err != nil {
					t.Fatal(err)
				}
			case "wrong-name":
				path = filepath.Join(base, "other")
			case "relative":
				path = "private-projects"
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := ProvisionWorkspaceParent(ctx, path); err == nil {
				t.Fatal("unsafe grouping parent admitted")
			}
			if mode == "nonempty" {
				data, err := os.ReadFile(filepath.Join(path, "retained"))
				if err != nil || string(data) != "do not adopt" {
					t.Fatal("retained contents changed")
				}
			}
			if mode == "public-child" {
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != 0755 {
					t.Fatal("permissions repaired")
				}
			}
			if mode == "cancel" || mode == "public-base" || mode == "wrong-name" || mode == "ancestor-link" {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("refusal created a directory", err)
				}
			}
		})
	}
}

func TestWorkspaceProvisionRefusesReplacedObjectsDuringSync(t *testing.T) {
	for _, which := range []string{"base", "child", "content"} {
		t.Run(which, func(t *testing.T) {
			base := privateProvisionBase(t)
			path := filepath.Join(base, "private-projects")
			calls := 0
			_, err := provisionWorkspaceParent(context.Background(), path, func(f *os.File) error {
				calls++
				if calls == 1 {
					if which == "content" {
						return os.WriteFile(filepath.Join(path, "unexpected"), nil, 0600)
					}
					p := path
					if which == "base" {
						p = base
					}
					retained := p + "-retained"
					if e := os.Rename(p, retained); e != nil {
						return e
					}
					t.Cleanup(func() { os.RemoveAll(retained) })
					return os.Mkdir(p, 0700)
				}
				return f.Sync()
			})
			if err == nil {
				t.Fatal("changed object accepted")
			}
		})
	}
}
