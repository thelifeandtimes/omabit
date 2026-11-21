package recording

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func testMetadata() Metadata {
	return Metadata{Version: 1, SessionID: strings.Repeat("1", 32), RunnerID: "runner", Mode: "development-managed", InstanceID: "member", ProjectID: "project", ConfigurationRevision: math.MaxUint64, OperationID: "operation", RequestID: "original-request", ContainerID: strings.Repeat("a", 64), LaunchContractSHA256: strings.Repeat("b", 64), ImageManifestDigest: "sha256:" + strings.Repeat("c", 64), RuntimeVersion: "4.6-5a7614f", Transport: "docker-multiplex", AttachedAt: time.Unix(1700000000, 1).UTC()}
}

func TestMetadataBindsIdentityRevisionArtifactsTransportAndClockOrigin(t *testing.T) {
	m := testMetadata()
	raw, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadMetadata(raw)
	if err != nil || got.ConfigurationRevision != math.MaxUint64 || !got.AttachedAt.Equal(m.AttachedAt) {
		t.Fatal(got, err)
	}
	digest, _ := m.Digest()
	id, _ := hex.DecodeString(m.SessionID)
	h := Header{MetadataSHA256: digest, Segment: 1, FirstSequence: 1, StartedUnixNano: m.AttachedAt.UnixNano()}
	copy(h.SessionID[:], id)
	if err := m.CheckHeader(h); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Metadata){
		func(v *Metadata) { v.InstanceID = "other" }, func(v *Metadata) { v.RunnerID = "other" }, func(v *Metadata) { v.ConfigurationRevision-- }, func(v *Metadata) { v.OperationID = "other" }, func(v *Metadata) { v.ContainerID = strings.Repeat("d", 64) }, func(v *Metadata) { v.ImageManifestDigest = "sha256:" + strings.Repeat("d", 64) }, func(v *Metadata) { v.Transport = "pty-combined" }, func(v *Metadata) { v.AttachedAt = v.AttachedAt.Add(time.Nanosecond) },
	} {
		other := m
		change(&other)
		if !errors.Is(other.CheckHeader(h), ErrInvalid) {
			t.Fatal("metadata replacement rebound an existing header")
		}
	}
	other := m
	other.AttachedAt = other.AttachedAt.In(time.FixedZone("offset", -7*3600))
	canonical, err := other.Bytes()
	if err != nil || !bytes.Equal(raw, canonical) {
		t.Fatal("equivalent time origin changed canonical digest", err)
	}
}

func TestMetadataRejectsUnknownSecretsAndNoncanonicalOrPartialBindings(t *testing.T) {
	m := testMetadata()
	raw, _ := m.Bytes()
	for _, invalid := range [][]byte{
		append([]byte(" "), raw...), bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"environment":"PRIVATE-CREDENTIAL"`), 1), bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1), bytes.Repeat([]byte("x"), MaxMetadataBytes+1),
	} {
		if _, err := ReadMetadata(invalid); !errors.Is(err, ErrInvalid) {
			t.Fatal("unbounded/ambiguous metadata accepted", err)
		}
	}
	for _, change := range []func(*Metadata){func(v *Metadata) { v.ContainerID = "" }, func(v *Metadata) { v.AttachedAt = time.Time{} }, func(v *Metadata) { v.AttachedAt = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC) }, func(v *Metadata) { v.PreviousSessionID = v.SessionID }, func(v *Metadata) { v.RuntimeVersion = "arbitrary version\nsecret" }, func(v *Metadata) { v.ImageManifestDigest = "mutable:latest" }} {
		v := m
		change(&v)
		if _, err := v.Bytes(); !errors.Is(err, ErrInvalid) {
			t.Fatal("incomplete metadata accepted", err)
		}
	}
}
