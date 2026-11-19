package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingPageRejectsChangedBindingsAndRepresentation(t *testing.T) {
	raw, err := os.ReadFile("../recording/testdata/original-metadata1/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := recording.ReadMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	m.ConfigurationRevision = 9007199254740993
	raw, err = m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	digest := hex.EncodeToString(hash[:])
	payload := []byte("PRIVATE-ORIGINAL\x1b[2K\n")
	frame := recording.Frame{Segment: 1, Sequence: 1, UnixNano: time.Now().UnixNano(), Kind: recording.Stdout, Digest: sha256.Sum256(payload), Data: payload}
	export, err := recording.ExportOriginalFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	in := recording.PageRequest{RunnerID: m.RunnerID, InstanceID: m.InstanceID, JobID: strings.Repeat("b", 32), JobSHA256: strings.Repeat("c", 64), IncludePrivateData: true, Representation: "original"}
	base := recording.ReadWindow{Version: 1, Session: recording.SessionDescriptor{RunnerID: m.RunnerID, Mode: m.Mode, InstanceID: m.InstanceID, ConfigurationRevision: m.ConfigurationRevision, ContainerID: m.ContainerID, OperationID: m.OperationID, RequestID: m.RequestID, JobID: in.JobID, JobSHA256: in.JobSHA256}, Metadata: &m, MetadataSHA256: digest, Status: recording.SavedStatus{SessionID: m.SessionID}, Representation: "original", FrameCount: 1, Original: []recording.OriginalFrameExport{export}, Next: &recording.Cursor{Version: 1, SessionID: m.SessionID, MetadataSHA256: digest, Sequence: 1, Digest: hex.EncodeToString(frame.Digest[:])}, Tail: "not-scanned", ReadAt: time.Now().UTC()}
	var variant atomic.Value
	variant.Store("ok")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		calls.Add(1)
		w.Header().Set(protocol.Header, "1")
		out := base
		switch variant.Load().(string) {
		case "foreign-job":
			out.Session.JobID = strings.Repeat("d", 32)
		case "foreign-metadata":
			meta := m
			meta.OperationID = "op-other"
			out.Metadata = &meta
		case "false-hash":
			out.MetadataSHA256 = strings.Repeat("d", 64)
		case "foreign-cursor":
			next := *out.Next
			next.SessionID = strings.Repeat("e", 32)
			out.Next = &next
		case "changed-choice":
			out.Representation = "redacted"
		case "false-complete":
			out.CompleteExport = true
		case "missing-frame":
			out.Original = nil
		}
		json.NewEncoder(w).Encode(out)
	}))
	defer server.Close()
	dialer := &net.Dialer{}
	c := newTransport(func(ctx context.Context) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", server.Listener.Addr().String())
	}, nil)
	defer c.Close()
	for _, name := range []string{"ok", "foreign-job", "foreign-metadata", "false-hash", "foreign-cursor", "changed-choice", "false-complete", "missing-frame"} {
		variant.Store(name)
		before := calls.Load()
		out, err := c.RecordingPage(t.Context(), in)
		if calls.Load() != before+1 {
			t.Fatal("read retried", name)
		}
		if name == "ok" {
			if err != nil || out.Session.ConfigurationRevision != m.ConfigurationRevision || !bytes.Equal(out.Original[0].OriginalBytes, payload) {
				t.Fatal("original octets or exact revision lost", err)
			}
		} else if err == nil || out.Original != nil {
			t.Fatal("invalid response released private data", name)
		}
	}
	in.IncludePrivateData = false
	before := calls.Load()
	if _, err := c.RecordingPage(t.Context(), in); err == nil || calls.Load() != before {
		t.Fatal("missing consent reached transport")
	}
}
