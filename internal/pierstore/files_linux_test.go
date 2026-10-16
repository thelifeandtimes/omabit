//go:build linux

package pierstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConventionDiscoveryMoveAndArchiveIsolation(t *testing.T) {
	home := t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"fakeships/project-one/zod/pier", "fakeships/nec/pier", "livenet/moons/sampel-palnet-moons/dister-dozzod-sampel-palnet/pier", "livenet/comets/testing-comet-1/pier", "archive/zod.20260901.bak"} {
		if err := os.MkdirAll(filepath.Join(Root(home), p, ".urb"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(Root(home), "fakeships/nec"), filepath.Join(Root(home), "fakeships/link")); err != nil {
		t.Fatal(err)
	}
	rows, err := Scan(context.Background(), home)
	if err != nil || len(rows) != 4 {
		t.Fatal(rows, err)
	}
	source, _ := Path(home, "fakeships/project-one/zod/pier")
	identity, _ := Identify(source)
	mount, _ := IdentifyMount(filepath.Dir(source))
	loc := Location{Path: "fakeships/project-one/zod/pier", MountIdentity: mount, Identity: identity}
	dest, err := Parents(home, "fakeships/project-two/zod/pier")
	if err != nil {
		t.Fatal(err)
	}
	if err := Move(source, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(home, loc); err == nil {
		t.Fatal("missing registered path was adopted")
	}
	loc.Path = "fakeships/project-two/zod/pier"
	if _, err := Check(home, loc); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(source+"/.urb", 0700); err != nil {
		t.Fatal(err)
	}
	loc.Path = "fakeships/project-one/zod/pier"
	if _, err := Check(home, loc); err == nil {
		t.Fatal("replacement directory adopted")
	}
	if err := Move(source, dest); err == nil {
		t.Fatal("overwrote destination")
	}
	if err := os.WriteFile(source+"/.vere.lock", []byte("123"), 0600); err != nil {
		t.Fatal(err)
	}
	if !Busy(source) {
		t.Fatal("Vere lock ignored")
	}
}

func TestDiscoveryFindsShipNamedLeavesAndKeepsMountExclusive(t *testing.T) {
	home := t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	path := "livenet/planets/lismyl-lismyl/lismyl-lismyl"
	full := filepath.Join(Root(home), path)
	if err := os.MkdirAll(filepath.Join(full, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	rows, err := Scan(context.Background(), home)
	if err != nil || len(rows) != 1 || rows[0].Path != path || rows[0].Problem != "" || rows[0].Fingerprint == "" {
		t.Fatal(rows, err)
	}
	if _, err := Parents(home, "livenet/planets/lismyl-lismyl/nested/zod"); err == nil {
		t.Fatal("nested named pier accepted")
	}
	if err := os.Mkdir(filepath.Join(filepath.Dir(full), "other"), 0700); err != nil {
		t.Fatal(err)
	}
	rows, err = Scan(context.Background(), home)
	if err != nil || len(rows) != 1 || rows[0].Problem == "" {
		t.Fatal("sibling data would be mounted", rows, err)
	}
}

func TestTreeReviewAllowsOnlySelfContainedDockedLinks(t *testing.T) {
	p := t.TempDir()
	bin := filepath.Join(p, ".bin/live/vere")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("inert runtime"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(bin, filepath.Join(p, ".run")); err != nil {
		t.Fatal(err)
	}
	if _, files, _, err := TreeReview(context.Background(), p); err != nil || files != 2 {
		t.Fatal(files, err)
	}
	outside := filepath.Join(t.TempDir(), "other")
	if err := os.Link(bin, outside); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := TreeReview(context.Background(), p); err == nil {
		t.Fatal("external link accepted")
	}
	if err := os.Remove(outside); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(p, ".urb/log"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(bin, filepath.Join(p, ".urb/log/data.mdb")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := TreeReview(context.Background(), p); err == nil {
		t.Fatal("shared private database accepted")
	}
}

func TestPathsStayInConventionAndCannotNestPiers(t *testing.T) {
	for _, p := range []string{"fakeships/zod", "fakeships/pier", "/fakeships/zod/pier", "../zod/pier", "fakeships/../zod/pier", "archive/zod/pier", "livenet/zod/pier", "livenet/planets/pier", "fakeships/.hidden/zod/pier", "fakeships/a,b/pier", "fakeships/a//b/pier"} {
		if ValidPath(p) {
			t.Fatal(p)
		}
	}
	home := t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(Root(home), "fakeships/zod/pier/.urb"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Parents(home, "fakeships/zod/nested/pier"); err == nil {
		t.Fatal("nested pier accepted")
	}
}

func TestPermanentMountIsExclusiveAndDoesNotAdoptDirectories(t *testing.T) {
	home := t.TempDir()
	loc := Location{Path: "fakeships/new-ship/pier"}
	mount, err := PrepareMount(home, &loc)
	if err != nil {
		t.Fatal(err)
	}
	if loc.MountIdentity.Inode == 0 {
		t.Fatal("mount identity not recorded")
	}
	if _, err = os.Stat(filepath.Join(mount, "pier")); !os.IsNotExist(err) {
		t.Fatal("precreated Vere target")
	}
	if _, err = PrepareMount(home, &Location{Path: loc.Path}); err == nil {
		t.Fatal("adopted an unrecorded mount directory")
	}
	if err = os.Mkdir(filepath.Join(mount, "sibling"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = CheckMount(home, loc); err == nil {
		t.Fatal("mounted sibling data")
	}
}

// Simulates a reboot that renumbered the backing filesystem's st_dev: the
// registered pier must still pass Check, while a recreated directory must not.
func TestCheckSurvivesDeviceRenumberingButNotReplacement(t *testing.T) {
	home := t.TempDir()
	if err := Init(home); err != nil {
		t.Fatal(err)
	}
	rel := "livenet/moons/sampel-palnet-moons/dister-dozzod-sampel-palnet/pier"
	if err := os.MkdirAll(filepath.Join(Root(home), rel, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := Path(home, rel)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := Identify(p)
	if err != nil {
		t.Skip("filesystem lacks birth timestamps:", err)
	}
	mount, err := IdentifyMount(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	stored := Location{Path: rel, Identity: identity, MountIdentity: mount}
	stored.Identity.Device += 7
	stored.Identity.UrbDevice += 7
	stored.MountIdentity.Device += 7
	if _, err := Check(home, stored); err != nil {
		t.Fatal("device renumbering reported as replacement:", err)
	}
	if err := os.Rename(p, p+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(p, ".urb"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(p+"-old", filepath.Join(home, "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(home, stored); err == nil {
		t.Fatal("replacement pier directory accepted")
	}
}
