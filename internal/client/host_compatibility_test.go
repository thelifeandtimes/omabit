package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func hostReportFixture() protocol.HostCompatibilityReport {
	r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
	return protocol.HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Features: []string{protocol.HostCompatibilityFeature}, Report: r, ExpiresAt: r.ObservedAt.Add(protocol.HostCompatibilityMaxAge)}
}

func TestHostCompatibilityBindingCapabilitiesAndSanitizedErrors(t *testing.T) {
	for _, fault := range []string{"none", "older", "identity", "mode", "schema", "protocol", "feature", "oversized", "private-path", "remote-error", "malformed", "timeout"} {
		t.Run(fault, func(t *testing.T) {
			var reportCalls atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if req.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Features: []string{protocol.HostCompatibilityFeature}}
					if fault == "older" {
						h.Features = nil
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				reportCalls.Add(1)
				if req.Method != "GET" || req.URL.Path != protocol.HostCompatibilityPath || req.Header.Get(protocol.Header) != "1" {
					t.Error("unexpected host request", req.Method, req.URL)
				}
				r := hostReportFixture()
				switch fault {
				case "identity":
					r.RunnerID = "other"
				case "mode":
					r.Mode = "simulation"
				case "schema":
					r.SchemaVersion++
				case "protocol":
					r.ProtocolVersion++
				case "feature":
					r.Features = nil
				case "oversized":
					r.Report.Guidance = []string{strings.Repeat("x", 1025)}
				case "private-path":
					r.Report.Engine = &hostcompat.Engine{DataRoot: "/private/owner/data"}
				case "remote-error":
					w.WriteHeader(500)
					json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "private-secret", Message: "private-secret"}})
					return
				case "malformed":
					w.Write([]byte("private-secret"))
					return
				case "timeout":
					<-req.Context().Done()
					return
				}
				json.NewEncoder(w).Encode(r)
			})
			ctx := context.Background()
			if fault == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}
			r, err := c.HostCompatibility(ctx, Identity{RunnerID: "runner", Mode: "development-managed"})
			if fault == "none" {
				if err != nil || r.RunnerID != "runner" || r.Report.Provenance == "verified" {
					t.Fatal("missing provider should be report evidence, not RPC failure", r, err)
				}
				return
			}
			if err == nil || r.RunnerID != "" || strings.Contains(err.Error(), "private-secret") {
				t.Fatal("bad report accepted or unsanitized", r, err)
			}
			if fault == "older" && (!errors.Is(err, ErrHostCompatibilityUnsupported) || reportCalls.Load() != 0) {
				t.Fatal("older runner was queried for unavailable feature", err)
			}
			if (fault == "identity" || fault == "mode") && !errors.Is(err, ErrIdentityMismatch) {
				t.Fatal("identity error lost", err)
			}
			if fault == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("timeout classification lost", err)
			}
		})
	}
}

func TestHostCompatibilityPinnedReconnect(t *testing.T) {
	var replaced atomic.Bool
	var reports atomic.Int32
	unpinned := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			id := "runner"
			if replaced.Load() {
				id = "replacement"
			}
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "development-managed", Features: []string{protocol.HostCompatibilityFeature}})
			return
		}
		reports.Add(1)
		json.NewEncoder(w).Encode(hostReportFixture())
	})
	conn, err := unpinned.httpClient.Transport.(*http.Transport).DialContext(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	path := conn.RemoteAddr().String()
	conn.Close()
	identity := Identity{RunnerID: "runner", Mode: "development-managed"}
	c := NewPinned(path, identity)
	defer c.Close()
	if _, err := c.HostCompatibility(context.Background(), identity); err != nil {
		t.Fatal(err)
	}
	c.httpClient.CloseIdleConnections()
	replaced.Store(true)
	if _, err := c.HostCompatibility(context.Background(), identity); !errors.Is(err, ErrIdentityMismatch) || reports.Load() != 1 {
		t.Fatal("replacement received report request", err, reports.Load())
	}
}
