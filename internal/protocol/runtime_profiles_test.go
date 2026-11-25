package protocol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"testing"
)

func TestRuntimeProfileAndRequirementAreSignedTogether(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	settings := Configuration{Loom: 34, RuntimeProfile: runtimeprofile.JSONFixProfile}.HandoffRuntimeSettings()
	h := Handoff{ID: "patched", VerifyKey: hex.EncodeToString(pub), RuntimeSettings: &settings}
	raw := h.CommitmentBytes()
	proof := hex.EncodeToString(ed25519.Sign(priv, raw))
	if !bytes.Contains(raw, []byte("omabit-stopped-handoff-v3")) || !h.ValidCommitment(proof) || settings.RuntimeRequirement != runtimeprofile.JSONFixRequirement {
		t.Fatal("runtime commitment missing")
	}
	settings.RuntimeRequirement = ""
	if h.ValidCommitment(proof) {
		t.Fatal("stripped correction remained signed")
	}
	settings.RuntimeRequirement = runtimeprofile.JSONFixRequirement
	settings.RuntimeProfile = runtimeprofile.OfficialProfile
	if h.ValidCommitment(proof) {
		t.Fatal("substituted runtime remained signed")
	}
	h.RuntimeSettings = nil
	if h.ValidCommitment(proof) {
		t.Fatal("legacy downgrade remained signed")
	}
}
