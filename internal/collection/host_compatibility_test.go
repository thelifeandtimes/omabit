package collection

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestHostReportPollingReusesBoundedTransports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	var connections atomic.Int32
	r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
	report := protocol.HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.HostCompatibilityFeature}, Report: r, ExpiresAt: r.ObservedAt.Add(protocol.HostCompatibilityMaxAge)}
	srv := &http.Server{ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(protocol.Header, "1")
		switch req.URL.Path {
		case "/hello":
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.HostCompatibilityFeature}})
		case "/v1/snapshot":
			json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner", Mode: "simulation"})
		case protocol.HostCompatibilityPath:
			json.NewEncoder(w).Encode(report)
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	polls := 0
	err = Watch(ctx, []Connection{{Name: "one", RunnerID: "runner", Mode: "simulation", Transport: "unix", Socket: path}}, 100*time.Millisecond, func(v Observation) error {
		polls++
		if v.Status != "available" || v.HostCompatibilityStatus != "available" || connections.Load() > 2 {
			t.Errorf("report polling lost transport reuse: %+v, connections=%d", v, connections.Load())
		}
		if polls == 5 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || polls != 5 {
		t.Fatal(err, polls)
	}
}

func TestHostReportIndependentOfInventory(t *testing.T) {
	for _, scenario := range []string{"healthy", "older", "report-error", "report-timeout", "report-identity", "snapshot-error"} {
		t.Run(scenario, func(t *testing.T) {
			var reports atomic.Int32
			path := filepath.Join(t.TempDir(), "r.sock")
			ln, err := net.Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set(protocol.Header, "1")
				switch req.URL.Path {
				case "/hello":
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "observer", Features: []string{protocol.HostCompatibilityFeature}}
					if scenario == "older" {
						h.Features = nil
					}
					json.NewEncoder(w).Encode(h)
				case "/v1/snapshot":
					if scenario == "snapshot-error" {
						w.WriteHeader(503)
						json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "secret-snapshot", Message: "secret-snapshot"}})
						return
					}
					json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner", Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha"}}})
				case protocol.HostCompatibilityPath:
					reports.Add(1)
					if scenario == "report-error" {
						w.WriteHeader(500)
						json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "secret-provider", Message: "secret-provider"}})
						return
					}
					if scenario == "report-timeout" {
						<-req.Context().Done()
						return
					}
					r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
					bound := protocol.HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.HostCompatibilityFeature}, Report: r, ExpiresAt: r.ObservedAt.Add(protocol.HostCompatibilityMaxAge)}
					if scenario == "report-identity" {
						bound.RunnerID = "other"
					}
					json.NewEncoder(w).Encode(bound)
				default:
					t.Error("unexpected route", req.URL.Path)
				}
			})}
			go srv.Serve(ln)
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			inventory := Collect(ctx, []Connection{{Name: "one", RunnerID: "runner", Mode: "simulation", Transport: "unix", Socket: path}})
			v := inventory.Connections[0]
			encoded, _ := json.Marshal(v)
			if strings.Contains(string(encoded), "secret-") {
				t.Fatal("remote error leaked", string(encoded))
			}
			if scenario == "snapshot-error" {
				if inventory.Complete || v.Status != "unavailable" || v.HostCompatibility == nil || v.HostCompatibilityStatus != "available" {
					t.Fatal("inventory failure hid valid host evidence", string(encoded))
				}
				return
			}
			if !inventory.Complete || v.Status != "available" || v.Snapshot == nil || len(v.Snapshot.Instances) != 1 {
				t.Fatal("host evidence failure hid ships", string(encoded))
			}
			switch scenario {
			case "healthy":
				if v.HostCompatibility == nil || v.HostCompatibilityStatus != "available" {
					t.Fatal(string(encoded))
				}
			case "older":
				if reports.Load() != 0 || v.HostCompatibilityStatus != "unsupported" || v.HostCompatibilityError == nil || v.HostCompatibilityError.Code != "host_report_unsupported" {
					t.Fatal(string(encoded))
				}
			default:
				want := map[string]string{"report-error": "host_report_unavailable", "report-timeout": "host_report_timeout", "report-identity": "host_report_identity_mismatch"}[scenario]
				if v.HostCompatibility != nil || v.HostCompatibilityError == nil || v.HostCompatibilityError.Code != want {
					t.Fatal(string(encoded))
				}
			}
		})
	}
}
