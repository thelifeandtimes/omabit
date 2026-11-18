package recording

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math"
	"testing"
)

func TestLifecycleKeepsUnknownCoverageAndObservedExitDistinct(t *testing.T) {
	unknown, err := (GapEvent{Version: 1, Reason: "transport-interrupted"}).Bytes()
	if err != nil || bytes.Contains(unknown, []byte("known_missing_bytes")) {
		t.Fatal("unknown loss became zero", string(unknown), err)
	}
	count := uint64(math.MaxUint64)
	known, err := (GapEvent{Version: 1, Reason: "write-failed", KnownMissingBytes: &count}).Bytes()
	decoded, readErr := ReadGap(known)
	if err != nil || readErr != nil || decoded.KnownMissingBytes == nil || *decoded.KnownMissingBytes != count {
		t.Fatal("loss precision changed", decoded, err, readErr)
	}
	e := ExitEvent{Version: 1, ContainerID: string(bytes.Repeat([]byte("a"), 64)), StartedUnixNano: 200, FinishedUnixNano: 100, Code: 137, OOMKilled: true}
	raw, err := e.Bytes()
	got, readErr := ReadExit(raw)
	if err != nil || readErr != nil || got != e {
		t.Fatal("observed wall-clock reversal or OOM lost", got, err, readErr)
	}
	e.StartedUnixNano = 0
	if _, err := e.Bytes(); !errors.Is(err, ErrInvalid) {
		t.Fatal("unstarted process fabricated an exit", err)
	}
	if _, err := ReadExit(unknown); !errors.Is(err, ErrInvalid) {
		t.Fatal("transport gap became process exit", err)
	}
}

func TestLifecycleRejectsAmbiguousOrUnboundedMetadataWithoutWriting(t *testing.T) {
	var out bytes.Buffer
	w, err := NewEncoder(&out, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), out.Bytes()...)
	for _, raw := range [][]byte{
		[]byte(`{"version":1,"reason":"disk-pressure","reason":"write-failed"}`),
		[]byte(`{"version":1,"reason":"disk-pressure","environment":"private"}`),
		[]byte(`{"version":1,"reason":"unknown-free-form-secret"}`),
		[]byte(`{"version":1,"reason":"disk-pressure","known_missing_bytes":-1}`),
		[]byte(`{"version":1,"reason":"disk-pressure","known_missing_bytes":1e2}`),
		[]byte(` {"version":1,"reason":"disk-pressure"}`),
		bytes.Repeat([]byte("x"), MaxFrameBytes+1),
	} {
		if _, err := w.Append(Gap, raw, testHeader().StartedUnixNano, 0); !errors.Is(err, ErrInvalid) || !bytes.Equal(out.Bytes(), before) {
			t.Fatal("invalid lifecycle wrote bytes", err)
		}
	}
	if _, err := w.Append(Stdout, []byte("0wdiagnostic Authorization: original-byte-fixture\x1b[2K"), testHeader().StartedUnixNano, 0); err != nil {
		t.Fatal("invalid metadata poisoned a writer before I/O", err)
	}
}

func TestLifecycleDecoderRejectsSemanticCorruptionEvenWithValidChecksum(t *testing.T) {
	var out bytes.Buffer
	w, err := NewEncoder(&out, testHeader())
	if err != nil {
		t.Fatal(err)
	}
	data, _ := (GapEvent{Version: 1, Reason: "disk-pressure"}).Bytes()
	if _, err := w.Append(Gap, data, testHeader().StartedUnixNano, 0); err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), out.Bytes()...)
	start := headerBytes + frameHeaderBytes
	copy(raw[start:start+len(data)], bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1))
	previous := sha256.Sum256(raw[:headerBytes])
	hash := sha256.New()
	hash.Write(previous[:])
	hash.Write(raw[headerBytes : len(raw)-sha256.Size])
	copy(raw[len(raw)-sha256.Size:], hash.Sum(nil))
	r, err := NewDecoder(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Next(); !errors.Is(err, ErrCorrupt) || r.CommittedOffset() != int64(headerBytes) || r.LastDigest() != previous {
		t.Fatal("valid checksum promoted invalid lifecycle payload", err)
	}
}
