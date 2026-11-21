package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestNonemptyRecordingWindowMatchesPublishedSchema(t *testing.T) {
	raw, err := os.ReadFile("../recording/testdata/original-metadata1/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := recording.ReadMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	metadata.ConfigurationRevision = 9007199254740993
	raw, err = metadata.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	digest := hex.EncodeToString(hash[:])
	payload := []byte("private original\x00\x1b[2K\xff\n")
	frame := recording.Frame{Segment: 1, Sequence: 1, UnixNano: 1795094400123456789, Kind: recording.Stdout, Digest: sha256.Sum256(payload), Data: payload}
	original, err := recording.ExportOriginalFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	redacted, err := recording.RenderRedactedFrames([]recording.Frame{frame}, recording.RedactionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	base := recording.ReadWindow{Version: 1, Session: recording.SessionDescriptor{RunnerID: metadata.RunnerID, InstanceID: metadata.InstanceID, Mode: metadata.Mode, ConfigurationRevision: metadata.ConfigurationRevision, ContainerID: metadata.ContainerID, OperationID: metadata.OperationID, RequestID: metadata.RequestID, JobID: strings.Repeat("b", 32), JobSHA256: strings.Repeat("c", 64)}, Metadata: &metadata, MetadataSHA256: digest, Status: recording.SavedStatus{SessionID: metadata.SessionID}, FrameCount: 1, Next: &recording.Cursor{Version: 1, SessionID: metadata.SessionID, MetadataSHA256: digest, Sequence: 1, Digest: hex.EncodeToString(frame.Digest[:])}, Tail: "not-scanned", ReadAt: time.Unix(1795094400, 123456789).UTC()}
	for _, representation := range []string{"original", "redacted"} {
		t.Run(representation, func(t *testing.T) {
			s, err := New(Options{})
			if err != nil {
				t.Fatal(err)
			}
			window := base
			window.Representation = representation
			if representation == "original" {
				window.Original = []recording.OriginalFrameExport{original}
			} else {
				window.Redacted = &redacted
			}
			// Exercise the exact published operation and envelope validation without
			// opening an enrollment, socket, engine or recording on this host.
			op := s.operations["runtime.recording-page.v1"]
			op.invoke = func(context.Context, *Service, json.RawMessage) (any, error) { return window, nil }
			s.operations[op.descriptor.ID] = op
			out := s.Call(t.Context(), op.descriptor.ID, json.RawMessage(`{}`))
			if out.Error != nil {
				encoded, _ := json.Marshal(Result{Version: 1, Operation: op.descriptor.ID, Data: window})
				var value any
				decoder := json.NewDecoder(bytes.NewReader(encoded))
				decoder.UseNumber()
				if err := decoder.Decode(&value); err != nil {
					t.Fatal(err)
				}
				value, err = schemaNumbers(value)
				t.Fatalf("nonempty %s output refused: %+v; numbers=%v; schema=%v", representation, out.Error, err, op.output.Validate(value))
			}
			encoded, err := json.Marshal(out)
			if err != nil || !bytes.Contains(encoded, []byte(`9007199254740993`)) || !bytes.Contains(encoded, []byte(`1795094400123456789`)) {
				t.Fatal("exact revision or frame time lost", err)
			}
			var decoded struct {
				Data recording.ReadWindow `json:"data"`
			}
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if representation == "original" && (!bytes.Equal(decoded.Data.Original[0].OriginalBytes, payload) || decoded.Data.Original[0].Frame.Digest != frame.Digest) {
				t.Fatal("original octets or digest changed")
			}
			if representation == "redacted" && (len(decoded.Data.Original) != 0 || decoded.Data.Redacted == nil) {
				t.Fatal("representation changed")
			}
		})
	}
}

func TestByteSliceSchemaMatchesJSONWithoutChangingFixedArrays(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	type input struct {
		Bytes  []byte  `json:"bytes"`
		Digest [2]byte `json:"digest"`
	}
	err = bind(s, description("bytes-fixture.v1", "JSON byte encoding fixture", nil, []string{"offline-read"}, nil, "local-user", "exact bytes", "read-only"), func(_ context.Context, _ *Service, in input) (input, error) { return in, nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{`{"bytes":"AP8bCg==","digest":[0,255]}`, true},
		{`{"bytes":"","digest":[0,255]}`, true},
		{`{"bytes":null,"digest":[0,255]}`, true},
		{`{"bytes":[0,255],"digest":[0,255]}`, false},
		{`{"bytes":"not base64!","digest":[0,255]}`, false},
		{`{"bytes":"AP8bCg==","digest":"AP8="}`, false},
		{`{"bytes":"AP8bCg==","digest":[0]}`, false},
	} {
		out := s.Call(t.Context(), "bytes-fixture.v1", json.RawMessage(tc.value))
		if (out.Error == nil) != tc.valid {
			t.Fatalf("byte schema mismatch for %s: %+v", tc.value, out.Error)
		}
		if tc.valid {
			encoded, err := json.Marshal(out.Data)
			if err != nil || string(encoded) != tc.value {
				t.Fatalf("byte encoding changed: %s; %v", encoded, err)
			}
		}
	}
}
