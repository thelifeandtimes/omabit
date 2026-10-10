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
