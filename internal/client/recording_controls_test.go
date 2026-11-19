package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecordingControlsRejectChangedRepliesAndNeverRetryWrites(t *testing.T) {
	policy := recording.DefaultRetentionPolicy()
	in := recording.ControlRequest{RunnerID: "runner", RequestID: "original-request", Kind: "configure", Policy: &policy}
	event := recording.ControlEvent{Version: 1, ID: "control-one", RunnerID: "runner", Actor: "owner", Request: in, CreatedAt: time.Now().UTC()}
	raw, _ := json.Marshal(struct {
		Domain   string                 `json:"domain"`
		Ordinal  uint64                 `json:"ordinal"`
		Previous string                 `json:"previous_digest"`
		Event    recording.ControlEvent `json:"event"`
	}{"omabit-recording-control-v1", 1, "", event})
	digest := sha256.Sum256(raw)
	receipt := recording.ControlReceipt{Event: event, State: recording.ControlState{Version: 1, RunnerID: "runner", Revision: 1, Digest: hex.EncodeToString(digest[:]), Policy: policy, Pins: []recording.IncidentPin{}}}
	var variant atomic.Value
	variant.Store("ok")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		calls.Add(1)
		w.Header().Set(protocol.Header, "1")
		r := receipt
		r.State.Policy.Instances = append([]recording.InstanceRetention{}, receipt.State.Policy.Instances...)
		switch variant.Load().(string) {
		case "wrong-request":
			r.Event.Request.RequestID = "replacement"
		case "wrong-runner":
			r.State.RunnerID = "foreign"
		case "wrong-policy":
			r.State.Policy.MaxJobs++
		case "wrong-digest":
			r.State.Digest = "bad"
		case "lost-write":
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		switch q.URL.Path {
		case protocol.RecordingControlStatusPath:
			json.NewEncoder(w).Encode(recording.ControlStatus{Version: 1, CapacityStatus: "pilot-disabled", State: r.State, ReadAt: event.CreatedAt})
		case protocol.RecordingControlApplyPath:
			json.NewEncoder(w).Encode(recording.ControlApplyResult{Receipt: r})
		case protocol.RecordingControlLookupPath:
			json.NewEncoder(w).Encode(recording.ControlLookupResult{Version: 1, RunnerID: "runner", RequestID: in.RequestID, Found: true, Receipt: &r})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	dialer := &net.Dialer{}
	c := newTransport(func(ctx context.Context) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", server.Listener.Addr().String())
	}, nil)
	defer c.Close()
	for _, name := range []string{"ok", "wrong-request", "wrong-runner", "wrong-policy", "wrong-digest", "lost-write"} {
		variant.Store(name)
		before := calls.Load()
		out, err := c.ApplyRecordingControl(t.Context(), in)
		if calls.Load() != before+1 {
			t.Fatal("write retried", name)
		}
		if name == "ok" {
			if err != nil || !reflect.DeepEqual(out.Receipt, receipt) {
				t.Fatal("original receipt lost", err)
			}
		} else if err == nil || out.Receipt.Event.ID != "" {
			t.Fatal("changed/unconfirmed receipt admitted", name, err)
		}
		if name != "lost-write" {
			got, err := c.LookupRecordingControl(t.Context(), recording.ControlLookupRequest{RunnerID: "runner", RequestID: in.RequestID, IncludePrivateData: true})
			if (err == nil) != (name == "ok") || err != nil && got.Receipt != nil {
				t.Fatal("lookup admitted changed receipt", name, err)
			}
		}
	}
	variant.Store("ok")
	if _, err := c.RecordingControls(t.Context(), recording.ControlReadRequest{RunnerID: "runner", IncludePrivateData: true}); err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	if _, err := c.RecordingControls(t.Context(), recording.ControlReadRequest{RunnerID: "runner"}); err == nil || calls.Load() != before {
		t.Fatal("missing consent reached transport")
	}
}
