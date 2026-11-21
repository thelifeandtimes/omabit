package recording

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func attachmentMetadata() Metadata {
	m := testMetadata()
	m.Version = 2
	m.Identity = &AttachmentIdentity{Target: apppolicy.Target{InstanceID: m.InstanceID, ProjectID: m.ProjectID, Generation: 9, Creator: "project-creator"}, NativeShip: "~zod", PierContinuitySHA256: strings.Repeat("d", 64), EngineID: "daemon-fixture:one", EngineEndpointSHA256: strings.Repeat("e", 64), HostBootID: "11111111-2222-3333-4444-555555555555", RecorderNamespaces: HostNamespaces{User: 100, Mount: 101, PID: 102, Network: 103}, ProjectNamespace: &ProjectNamespace{BootID: "11111111-2222-3333-4444-555555555555", Device: 4, Inode: 200}}
	return m
}

func TestAttachmentMetadataV2BindsExactScopeAndExternalCaptureContext(t *testing.T) {
	m := attachmentMetadata()
	raw, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadMetadata(raw)
	if err != nil || !reflect.DeepEqual(got, m) {
		t.Fatal("successor metadata roundtrip changed identity", err)
	}
	identityCopy, err := got.CaptureIdentity()
	if err != nil {
		t.Fatal(err)
	}
	identityCopy.Target.Generation++
	identityCopy.ProjectNamespace.Inode++
	if got.Identity.Target.Generation != 9 || got.Identity.ProjectNamespace.Inode != 200 {
		t.Fatal("capture identity retained caller-mutable pointer")
	}
	digest, _ := m.Digest()
	id, _ := hex.DecodeString(m.SessionID)
	h := Header{MetadataSHA256: digest, Segment: 1, FirstSequence: 1, StartedUnixNano: m.AttachedAt.UnixNano()}
	copy(h.SessionID[:], id)
	if m.CheckHeader(h) != nil {
		t.Fatal("successor header refused")
	}
	for _, change := range []func(*Metadata){
		func(m *Metadata) { m.Identity.Target.Generation++ }, func(m *Metadata) { m.Identity.Target.Creator = "different-creator" }, func(m *Metadata) { m.Identity.NativeShip = "~nec" }, func(m *Metadata) { m.Identity.PierContinuitySHA256 = strings.Repeat("f", 64) }, func(m *Metadata) { m.Identity.EngineID = "another-daemon" }, func(m *Metadata) { m.Identity.EngineEndpointSHA256 = strings.Repeat("f", 64) }, func(m *Metadata) { m.Identity.RecorderNamespaces.Mount++ }, func(m *Metadata) { m.Identity.ProjectNamespace.Inode++ },
	} {
		mutated, err := ReadMetadata(raw)
		if err != nil {
			t.Fatal(err)
		}
		change(&mutated)
		if mutated.Validate() != nil {
			t.Fatal("valid changed identity rejected before header test")
		}
		if mutated.CheckHeader(h) == nil {
			t.Fatal("changed identity accepted original frame header")
		}
	}
	for _, change := range []func(*Metadata){
		func(m *Metadata) { m.Identity = nil }, func(m *Metadata) { m.Identity.Target.InstanceID = "other" }, func(m *Metadata) { m.Identity.Target.ProjectID = "other" }, func(m *Metadata) { m.Identity.Target.Desk = "grubbery" }, func(m *Metadata) { m.Identity.Target.Creator = "" }, func(m *Metadata) { m.Identity.Target.Generation = 0 }, func(m *Metadata) { m.Identity.NativeShip = "not-a-ship" }, func(m *Metadata) { m.Identity.PierContinuitySHA256 = strings.Repeat("0", 64) }, func(m *Metadata) { m.Identity.EngineID = "/run/docker.sock" }, func(m *Metadata) { m.Identity.EngineEndpointSHA256 = "bad" }, func(m *Metadata) { m.Identity.HostBootID = "111111112222-3333-4444-5555-555555555555" }, func(m *Metadata) { m.Identity.RecorderNamespaces.Network = 0 }, func(m *Metadata) { m.Identity.ProjectNamespace = nil }, func(m *Metadata) { m.Identity.ProjectNamespace.BootID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" }, func(m *Metadata) { m.Identity.ProjectNamespace.Device = 0 },
	} {
		mutated, err := ReadMetadata(raw)
		if err != nil {
			t.Fatal(err)
		}
		change(&mutated)
		if mutated.Validate() == nil {
			t.Fatal("incomplete or changed target accepted")
		}
	}
	ordinary, err := ReadMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	ordinary.ProjectID = ""
	ordinary.Identity.Target.ProjectID = ""
	ordinary.Identity.Target.Generation = 0
	ordinary.Identity.Target.Creator = ""
	ordinary.Identity.ProjectNamespace = nil
	if ordinary.Validate() != nil {
		t.Fatal("ordinary attachment invented project requirements")
	}
	ordinary.Identity.ProjectNamespace = m.Identity.ProjectNamespace
	if ordinary.Validate() == nil {
		t.Fatal("ordinary attachment claimed project namespace")
	}
}

func TestLegacyRecordingMetadataNeverAcquiresInventedIdentity(t *testing.T) {
	m := testMetadata()
	raw, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("identity")) {
		t.Fatal("legacy bytes gained successor identity")
	}
	legacy, err := ReadMetadata(raw)
	if err != nil || legacy.Identity != nil || legacy.Version != 1 {
		t.Fatal("legacy metadata changed", err)
	}
	if _, err = legacy.CaptureIdentity(); err != ErrLegacyMetadata {
		t.Fatal("legacy names invented current target identity", err)
	}
	identity := attachmentMetadata().Identity
	m.Identity = identity
	if m.Validate() == nil {
		t.Fatal("legacy version silently admitted successor fields")
	}
	var v map[string]json.RawMessage
	if json.Unmarshal(raw, &v) != nil {
		t.Fatal("legacy fixture failed")
	}
	v["identity"] = json.RawMessage("null")
	malformed, _ := json.Marshal(v)
	if _, err = ReadMetadata(malformed); err == nil {
		t.Fatal("noncanonical legacy identity field accepted")
	}
}
