package runtimeprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalProfileContentIdentityAndPrivatePersistence(t *testing.T) {
	old := StoreDirectory()
	defer ConfigureStore(old)
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	ConfigureStore(filepath.Join(root, "profiles"))
	p := Profile{Format: 1, Kind: "local", Platform: Platform, VereVersion: "test-64", Image: "omabit/vere@sha256:" + strings.Repeat("a", 64), ManifestDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64), BinarySHA256: strings.Repeat("c", 64), ArchiveSHA256: strings.Repeat("d", 64), ArchiveBytes: 1234, MaxLoom: 38, Requirement: JSONFixRequirement, CompatibleVersions: VereVersion}
	p.ID = LocalID(p)
	if err := InstallMetadata(p); err != nil {
		t.Fatal(err)
	}
	if err := InstallMetadata(p); err != nil {
		t.Fatal("idempotent installation", err)
	}
	got, err := Resolve(p.ID)
	if err != nil || got != p {
		t.Fatal(got, err)
	}
	if CompatibleSuccessor(JSONFixProfile, p.ID) != nil || CompatibleSuccessor(p.ID, OfficialProfile) == nil {
		t.Fatal("lost correction requirement")
	}
	for _, change := range []func(*Profile){func(p *Profile) { p.BinarySHA256 = strings.Repeat("e", 64) }, func(p *Profile) { p.MaxLoom++ }, func(p *Profile) { p.Requirement = "" }, func(p *Profile) { p.CompatibleVersions = "anything" }, func(p *Profile) { p.ArchiveBytes++ }} {
		changed := p
		change(&changed)
		if ValidateLocal(changed) == nil || InstallMetadata(changed) == nil {
			t.Fatal("metadata changed under existing ID")
		}
	}
	path := filepath.Join(StoreDirectory(), p.ID, "profile.json")
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = Resolve(p.ID); err == nil {
		t.Fatal("nonprivate manifest accepted")
	}
	if err = os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(root, "elsewhere"), path); err != nil {
		t.Fatal(err)
	}
	if _, err = Resolve(p.ID); err == nil {
		t.Fatal("linked manifest accepted")
	}
}

func TestChannelUpdatePreservesPreviousImmutableSelection(t *testing.T) {
	old := StoreDirectory()
	defer ConfigureStore(old)
	root := t.TempDir()
	os.Chmod(root, 0700)
	ConfigureStore(filepath.Join(root, "profiles"))
	p := Profile{Format: 1, Kind: "local", Platform: Platform, VereVersion: "4.6", Image: "omabit/vere@sha256:" + strings.Repeat("a", 64), ManifestDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64), BinarySHA256: strings.Repeat("c", 64), ArchiveSHA256: strings.Repeat("d", 64), ArchiveBytes: 1234, MaxLoom: 34, Channel: "live"}
	p.ID = LocalID(p)
	if SetChannel(p) == nil {
		t.Fatal("uninstalled channel accepted")
	}
	if err := InstallMetadata(p); err != nil {
		t.Fatal(err)
	}
	if err := SetChannel(p); err != nil {
		t.Fatal(err)
	}
	next := p
	next.VereVersion = "4.7"
	next.ID = LocalID(next)
	if err := InstallMetadata(next); err != nil {
		t.Fatal(err)
	}
	if Channels()["live"] != p.ID {
		t.Fatal("metadata installation advanced channel before archive validation")
	}
	if err := SetChannel(next); err != nil {
		t.Fatal(err)
	}
	if Channels()["live"] != next.ID {
		t.Fatal("channel failed to advance")
	}
	if got, err := Resolve(p.ID); err != nil || got != p {
		t.Fatal("channel advancement changed enrolled pin", err)
	}
	if err := os.WriteFile(filepath.Join(StoreDirectory(), "channel-live"), []byte(strings.Repeat("x", 200)), 0600); err != nil {
		t.Fatal(err)
	}
	if len(Channels()) != 0 {
		t.Fatal("invalid channel pointer advertised")
	}
}
