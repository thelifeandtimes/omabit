package runtimeprofile

import "testing"

func TestHistoricalDefaultAndCompatibility(t *testing.T) {
	old, err := Resolve("")
	official, _ := Resolve(OfficialProfile)
	patched, _ := Resolve(JSONFixProfile)
	if err != nil || old != official || old.Image != Image || patched.BinarySHA256 == old.BinarySHA256 || patched.ManifestDigest == old.ManifestDigest {
		t.Fatal("historical default or distinct artifact lost")
	}
	if CompatibleSuccessor(JSONFixProfile, "") == nil || CompatibleSuccessor(JSONFixProfile, OfficialProfile) == nil || CompatibleSuccessor(JSONFixProfile, JSONFixProfile) != nil {
		t.Fatal("correction requirement was downgraded")
	}
	for _, id := range []string{"latest", "live", "edge", "4.7", "/bin/urbit", "docker.io/tloncorp/vere:4.6"} {
		if _, err := Resolve(id); err == nil {
			t.Fatalf("unreviewed runtime %q accepted", id)
		}
	}
	for _, id := range []string{old.ConfigDigest, old.ManifestDigest, patched.BinarySHA256, patched.Image, ""} {
		if patched.AcceptsImageID(id) {
			t.Fatalf("wrong artifact %q accepted", id)
		}
	}
	if !patched.AcceptsImageID(patched.ConfigDigest) || !patched.AcceptsImageID(patched.ManifestDigest) {
		t.Fatal("reviewed OCI identity refused")
	}
	catalogue := Catalogue()
	catalogue[1].BinarySHA256 = "changed"
	again, _ := Resolve(JSONFixProfile)
	if again != patched {
		t.Fatal("caller mutated immutable catalogue")
	}
}
