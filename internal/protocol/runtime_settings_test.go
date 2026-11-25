package protocol

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectableAndHistoricalLoomBounds(t *testing.T) {
	for _, tc := range []struct {
		loom                   uint8
		selectable, historical bool
	}{{0, false, false}, {19, false, false}, {20, false, true}, {29, false, true}, {30, true, true}, {31, true, true}, {34, true, true}, {35, false, false}, {255, false, false}} {
		if ValidLoom(tc.loom) != tc.selectable || ValidRecordedLoom(tc.loom) != tc.historical {
			t.Fatalf("loom %d: selectable=%v historical=%v", tc.loom, ValidLoom(tc.loom), ValidRecordedLoom(tc.loom))
		}
	}
	if EffectiveLoom(0) != 31 || EffectiveLoom(29) != 29 {
		t.Fatal("historical defaults or explicit launch values changed")
	}
}

func TestLargerLoomRequiresInstalledDeclaration(t *testing.T) {
	old := runtimeprofile.StoreDirectory()
	defer runtimeprofile.ConfigureStore(old)
	root := t.TempDir()
	os.Chmod(root, 0700)
	runtimeprofile.ConfigureStore(filepath.Join(root, "profiles"))
	p := runtimeprofile.Profile{Format: 1, Kind: "local", Platform: runtimeprofile.Platform, VereVersion: "local-64", Image: "omabit/vere@sha256:" + strings.Repeat("a", 64), ManifestDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64), BinarySHA256: strings.Repeat("c", 64), ArchiveSHA256: strings.Repeat("d", 64), ArchiveBytes: 1234, MaxLoom: 38}
	p.ID = runtimeprofile.LocalID(p)
	if ValidProfileLoom(p.ID, 38, false) {
		t.Fatal("uninstalled runtime granted larger loom")
	}
	if err := runtimeprofile.InstallMetadata(p); err != nil {
		t.Fatal(err)
	}
	if !ValidProfileLoom(p.ID, 38, false) || ValidProfileLoom(p.ID, 39, false) || ValidProfileLoom("", 35, false) || ValidProfileLoom(runtimeprofile.JSONFixProfile, 35, false) {
		t.Fatal("loom ignored exact selected capability")
	}
	if ValidProfileLoom(p.ID, 29, false) || !ValidProfileLoom(p.ID, 29, true) {
		t.Fatal("changed historical lower bound")
	}
}
