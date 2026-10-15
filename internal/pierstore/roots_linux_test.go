//go:build linux

package pierstore

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestConfiguredRootSurvivesDeviceRenumbering(t *testing.T) {
	home, disk := t.TempDir(), t.TempDir()
	root, err := RegisterRoot(home, "ssd", disk)
	if err != nil {
		t.Fatal(err)
	}
	root.Identity.Device += 7
	data, err := json.Marshal([]StorageRoot{root})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(home, rootsFile), data, 0600); err != nil {
		t.Fatal(err)
	}
	resolved, rel, err := Resolve(home, "@ssd/fakeships/test/pier")
	if err != nil || resolved != disk || rel != "fakeships/test/pier" {
		t.Fatal(resolved, rel, err)
	}
	if _, err = RegisterRoot(home, "ssd", disk); err != nil {
		t.Fatal("same root re-add failed after reboot", err)
	}
}

func TestAdditionalRootsStayPinnedAndDiscoveryRetainsTheMountContract(t *testing.T) {
	home, disk := t.TempDir(), t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	r, err := RegisterRoot(home, "ssd", disk)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := RegisterRoot(home, "ssd", disk); err != nil || again != r {
		t.Fatal("idempotent add", err)
	}
	for _, path := range []string{disk, filepath.Join(disk, "fakeships")} {
		if _, err := RegisterRoot(home, "other", path); err == nil {
			t.Fatal("overlapping root", path)
		}
	}
	loc := Location{Path: "@ssd/fakeships/test/pier"}
	mount, err := PrepareMount(home, &loc)
	if err != nil || mount != filepath.Join(disk, "fakeships/test") {
		t.Fatal(mount, err)
	}
	if err = os.MkdirAll(filepath.Join(mount, "pier/.urb"), 0700); err != nil {
		t.Fatal(err)
	}
	loc.Identity, err = Identify(filepath.Join(mount, "pier"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Check(home, loc); err != nil {
		t.Fatal(err)
	}
	rows, err := Scan(context.Background(), home)
	if err != nil || len(rows) != 1 || rows[0].Path != loc.Path {
		t.Fatal(rows, err)
	}
	old := disk + "-original"
	if err = os.Rename(disk, old); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(old) })
	if err = os.Mkdir(disk, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = Check(home, loc); err == nil {
		t.Fatal("replacement root adopted")
	}
	if _, err = RegisterRoot(home, "ssd", disk); err == nil {
		t.Fatal("root silently retargeted")
	}
	if _, err = Absolute(home, "fakeships/unrelated/pier"); err != nil {
		t.Fatal("custom root failure changed default path", err)
	}
}

func TestHostPathAllowsOnlyTheRunnersNarrowACLGrant(t *testing.T) {
	dir := t.TempDir()
	acl := func(writer uint32) []byte {
		b := make([]byte, 4+5*8)
		binary.LittleEndian.PutUint32(b, 2)
		for i, e := range [][3]uint32{{1, 7, ^uint32(0)}, {2, 3, writer}, {4, 5, ^uint32(0)}, {0x10, 7, ^uint32(0)}, {0x20, 0, ^uint32(0)}} {
			off := 4 + i*8
			binary.LittleEndian.PutUint16(b[off:], uint16(e[0]))
			binary.LittleEndian.PutUint16(b[off+2:], uint16(e[1]))
			binary.LittleEndian.PutUint32(b[off+4:], e[2])
		}
		return b
	}
	if err := unix.Setxattr(dir, "system.posix_acl_access", acl(uint32(os.Geteuid())), 0); err != nil {
		t.Fatal(err)
	}
	if err := HostPath(dir); err != nil {
		t.Fatal("narrow service ACL refused", err)
	}
	if err := unix.Setxattr(dir, "system.posix_acl_access", acl(uint32(os.Geteuid())+10000), 0); err != nil {
		t.Fatal(err)
	}
	if err := HostPath(dir); err == nil {
		t.Fatal("unrelated ACL writer accepted")
	}
}

func TestStoragePathAndExternalSourceRefuseTraversalLinksLocksAndChanges(t *testing.T) {
	for _, p := range []string{"@one/@two/fakeships/a/pier", "@/fakeships/a/pier", "@default/fakeships/a/pier", "@bad.name/fakeships/a/pier", "@disk/../a/pier", "@disk/archive/a/pier", "@disk/fakeships/a//pier"} {
		if ValidPath(p) {
			t.Fatal("accepted", p)
		}
	}
	home, source := t.TempDir(), filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(source, ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, ".urb/log"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, ".urb/chk/state")
	if err := os.WriteFile(file, []byte("state"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := InspectExternal(context.Background(), home, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, []byte("new-state"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := InspectExternal(context.Background(), home, source)
	if err != nil || a.Fingerprint == b.Fingerprint {
		t.Fatal("changed content metadata ignored", err)
	}
	if err = os.Symlink(file, filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExternal(context.Background(), home, source); err == nil {
		t.Fatal("linked source accepted")
	}
	if err = os.Remove(filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(source)
	if err = os.Chmod(parent, 0300); err != nil {
		t.Fatal(err)
	}
	_, accessErr := InspectExternal(context.Background(), home, source)
	if err = os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() != 0 && accessErr == nil {
		t.Fatal("source parent without read/sync access accepted")
	}
	if err = os.WriteFile(filepath.Join(source, ".vere.lock"), []byte("running"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExternal(context.Background(), home, source); err == nil {
		t.Fatal("busy source accepted")
	}
}
