//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func privateArtifactFixture(t *testing.T) (privateMemberContract, protocol.BootPill) {
	t.Helper()
	old := privateMemberContractFixture(t, "sender", "boot")
	c, err := newPrivateMemberContract(old.selection, t.TempDir(), old.imageID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(c.root, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte("inert synthetic boot artifact")
	m := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Bytes: int64(len(raw))}
	p := bootpill.CachePath(c.root, m)
	if err = os.Mkdir(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p, raw, 0400); err != nil {
		t.Fatal(err)
	}
	return c, m
}
func TestPrivateMemberArtifactExclusiveHeldPublication(t *testing.T) {
	c, m := privateArtifactFixture(t)
	id, err := preparePrivateMemberArtifact(context.Background(), c, m)
	if err != nil {
		t.Fatal(err)
	}
	v, err := openPrivateMemberPillWithMeta(context.Background(), c, id, m)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if _, err = preparePrivateMemberArtifact(context.Background(), c, m); err == nil {
		t.Fatal("artifact replay")
	}
	if v.Verify(context.Background()) != nil {
		t.Fatal("replay changed artifact")
	}
}
func TestPrivateMemberArtifactRefusesAndRetains(t *testing.T) {
	for _, kind := range []string{"hash", "size", "existing", "symlink", "root-link", "cache-link", "cache-permissions", "canceled", "contract"} {
		t.Run(kind, func(t *testing.T) {
			c, m := privateArtifactFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cache := bootpill.CachePath(c.root, m)
			switch kind {
			case "hash":
				os.Chmod(cache, 0600)
				raw := make([]byte, m.Bytes)
				os.WriteFile(cache, raw, 0400)
			case "size":
				os.Chmod(cache, 0600)
				os.WriteFile(cache, []byte("short"), 0400)
			case "existing":
				os.Mkdir(filepath.Dir(c.pillPath), 0700)
			case "symlink":
				os.Symlink(filepath.Dir(cache), filepath.Dir(c.pillPath))
			case "root-link":
				old := c.root
				link := filepath.Join(t.TempDir(), "alias")
				os.Symlink(old, link)
				var err error
				c, err = newPrivateMemberContract(c.selection, link, c.imageID)
				if err != nil {
					t.Fatal(err)
				}
			case "cache-link":
				os.Rename(cache, cache+"-saved")
				os.Symlink(cache+"-saved", cache)
			case "cache-permissions":
				os.Chmod(filepath.Dir(cache), 0755)
			case "canceled":
				cancel()
			case "contract":
				c.pillPath += "-other"
			}
			id, err := preparePrivateMemberArtifact(ctx, c, m)
			if err == nil {
				t.Fatal("unsafe artifact accepted")
			}
			if kind == "hash" {
				if id == (pierstore.DirectoryIdentity{}) {
					t.Fatal("partial object identity lost")
				}
				got, err := pierstore.IdentifyMount(filepath.Dir(c.pillPath))
				if err != nil || !got.SameObject(id) {
					t.Fatal("partial artifact removed")
				}
				if _, err = os.Lstat(c.pillPath); err != nil {
					t.Fatal("partial pill removed")
				}
			} else if id != (pierstore.DirectoryIdentity{}) {
				t.Fatal("unexpected artifact created")
			}
		})
	}
}
func TestPrivateMemberPierWaitIsReadOnlyAndBounded(t *testing.T) {
	for _, kind := range []string{"ready", "missing", "file", "changed"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := privateArtifactFixture(t)
			path := filepath.Join(c.mountSource, "zod", ".urb")
			// The contract's selected fixture member supplies its actual ship name.
			for _, m := range c.selection.Plan.Members {
				if m.InstanceID == c.selection.InstanceID {
					path = filepath.Join(c.mountSource, m.Ship[1:], ".urb")
				}
			}
			if kind == "ready" {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "file" {
				os.MkdirAll(filepath.Dir(path), 0700)
				os.WriteFile(path, []byte("opaque"), 0600)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			err := waitPrivateMemberPier(ctx, c, func(context.Context) error {
				if kind == "changed" {
					return errPrivateMemberLaunch
				}
				return nil
			})
			if (err == nil) != (kind == "ready") {
				t.Fatal(kind, err)
			}
			if kind == "missing" {
				if _, err = os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("wait wrote state")
				}
			}
		})
	}
}

func TestPrivateMemberArtifactForeignMounts(t *testing.T) {
	for _, kind := range []string{"unrelated", "own", "foreign", "readonly", "ancestor", "alias", "changed", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			c, m := privateArtifactFixture(t)
			if _, err := preparePrivateMemberArtifact(context.Background(), c, m); err != nil {
				t.Fatal(err)
			}
			source := t.TempDir()
			id := "foreign"
			rw := true
			switch kind {
			case "own", "foreign", "readonly":
				source = c.pillPath
			case "ancestor":
				source = c.root
			case "alias":
				source = filepath.Join(t.TempDir(), "alias")
				if err := os.Link(c.pillPath, source); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "own" {
				id = "owned"
			}
			if kind == "readonly" {
				rw = false
			}
			rows := []projectContainerMounts{{ID: id, Mounts: []mount{{Type: "bind", Source: source, Destination: "/test", RW: rw}}}}
			if kind == "unknown" {
				rows[0].Mounts[0].Type = "volume"
			}
			calls := 0
			inventory := func(context.Context) ([]projectContainerMounts, error) {
				calls++
				if kind == "changed" && calls > 1 {
					return nil, nil
				}
				return rows, nil
			}
			err := checkPrivateMemberArtifactMounts(context.Background(), inventory, c, []privateGatewayMember{{Contract: c, ContainerID: "owned"}})
			if (err == nil) != (kind == "own" || kind == "unrelated") {
				t.Fatal(kind, err)
			}
		})
	}
}
