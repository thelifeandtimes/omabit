package protocol

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestHandoffRuntimeSettingsAreSigned(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	h := Handoff{ID: "transfer", VerifyKey: hex.EncodeToString(public)}
	legacy, _ := json.Marshal([]any{"omabit-stopped-handoff-v1", h.ID, h.SourceRunner, h.SourceInstance, h.DestinationRunner, h.DestinationInstance, h.Ship, h.Mode, h.AmesPublished, h.RollNeedsRuntime, h.SHA256, h.Bytes, h.TreeDigest, h.VerifyKey})
	if string(h.CommitmentBytes()) != string(legacy) {
		t.Fatal("historical commitments changed")
	}
	legacyProof := hex.EncodeToString(ed25519.Sign(private, legacy))
	if !h.ValidCommitment(legacyProof) {
		t.Fatal("legacy signature refused")
	}
	h.RuntimeSettings = &RuntimeSettings{Loom: 34, NativeControl: true}
	proof := hex.EncodeToString(ed25519.Sign(private, h.CommitmentBytes()))
	if !h.ValidCommitment(proof) || h.ValidCommitment(legacyProof) {
		t.Fatal("wrong signed contract")
	}
	h.RuntimeSettings.Loom = 31
	if h.ValidCommitment(proof) {
		t.Fatal("changed loom retained signature")
	}
	h.RuntimeSettings.Loom, h.RuntimeSettings.NativeControl = 34, false
	if h.ValidCommitment(proof) {
		t.Fatal("changed native setting retained signature")
	}
	h.RuntimeSettings = nil
	if h.ValidCommitment(proof) {
		t.Fatal("stripped settings retained signature")
	}
}

func TestContainerResourcesHandoffSignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	h := Handoff{ID: "resources", VerifyKey: hex.EncodeToString(public), RuntimeSettings: &RuntimeSettings{Loom: 31, MemoryGiB: 16, CPUCores: 8}}
	proof := hex.EncodeToString(ed25519.Sign(private, h.CommitmentBytes()))
	if !h.ValidCommitment(proof) {
		t.Fatal("valid resource contract refused")
	}
	h.RuntimeSettings.CPUCores = 2
	if h.ValidCommitment(proof) {
		t.Fatal("changed CPU limit retained signature")
	}
	h.RuntimeSettings.CPUCores = 8
	h.RuntimeSettings.MemoryGiB = 4
	if h.ValidCommitment(proof) {
		t.Fatal("changed RAM limit retained signature")
	}
}

func TestContainerResourcesExplicitDefaultsKeepLegacyHandoff(t *testing.T) {
	implicit := Configuration{Loom: 34, NativeControl: true}.HandoffRuntimeSettings()
	explicit := Configuration{Loom: 34, NativeControl: true, MemoryGiB: 4, CPUCores: 2}.HandoffRuntimeSettings()
	if implicit != explicit {
		t.Fatal("explicit defaults changed handoff compatibility")
	}
	b, err := json.Marshal(explicit)
	if err != nil || string(b) != `{"loom":34,"native_control":true}` {
		t.Fatal("historical signed settings changed", string(b), err)
	}
}
