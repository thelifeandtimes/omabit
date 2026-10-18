package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestMoveTransferBoundRevisionAndLostActivationAcknowledgement(t *testing.T) {
	for _, revision := range []uint64{2, 17} {
		t.Run(fmt.Sprint(revision), func(t *testing.T) {
			public, private, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			h := protocol.Handoff{ID: strings.Repeat("a", 32), SourceRunner: "source", SourceInstance: "ship", DestinationRunner: "destination", DestinationInstance: "arrived", Direction: "out", State: "ready", SHA256: strings.Repeat("b", 64), VerifyKey: hex.EncodeToString(public)}
			incoming := h
			incoming.Direction, incoming.State, incoming.Error = "in", "receiving", "interrupted"
			var mu sync.Mutex
			retries, activations, commits := 0, 0, 0
			endpoint := func(id string) *Client {
				return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set(protocol.Header, "1")
					switch {
					case r.URL.Path == "/hello":
						json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, AccessRole: "operator", Features: []string{"host-handoff"}})
					case r.URL.Path == "/v1/snapshot":
						i := protocol.Instance{ID: "ship", ConfigRevision: 1, Config: protocol.Configuration{Loom: protocol.DefaultLoom}, Handoff: &h}
						if id == "destination" {
							i.ID, i.ConfigRevision, i.Handoff = "arrived", revision, &incoming
						}
						json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: id, Instances: []protocol.Instance{i}})
					case r.URL.Path == "/v1/instances/ship/handoff":
						json.NewEncoder(w).Encode(h)
					case r.URL.Path == "/v1/handoffs" && r.Method == "POST":
						// Exact reservation replay is intentionally stale.
						reserved := incoming
						reserved.State, reserved.Error = "reserved", ""
						json.NewEncoder(w).Encode(reserved)
					case r.Method == "GET" && r.URL.Path == "/v1/handoffs/"+h.ID:
						if id == "source" {
							json.NewEncoder(w).Encode(h)
						} else {
							json.NewEncoder(w).Encode(incoming)
						}
					case r.Method == "POST":
						var body protocol.FinishHandoffRequest
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						}
						if id == "destination" && body.ExpectedRevision != revision {
							t.Errorf("destination received revision %d, want %d", body.ExpectedRevision, revision)
							http.Error(w, "wrong revision", 400)
							return
						}
						switch {
						case strings.HasSuffix(r.URL.Path, "/retry"):
							retries++
							incoming.State, incoming.Error = "received", ""
							json.NewEncoder(w).Encode(incoming)
						case strings.HasSuffix(r.URL.Path, "/commit"):
							commits++
							h.State = "moved"
							h.Commitment = hex.EncodeToString(ed25519.Sign(private, h.CommitmentBytes()))
							json.NewEncoder(w).Encode(h)
						case strings.HasSuffix(r.URL.Path, "/activate"):
							activations++
							if !incoming.ValidCommitment(body.Commitment) {
								t.Error("invalid source commitment")
							}
							incoming.State, incoming.Commitment = "complete", body.Commitment
							conn, _, err := w.(http.Hijacker).Hijack()
							if err != nil {
								t.Error(err)
								return
							}
							conn.Close() // acceptance succeeded; acknowledgement was lost
						default:
							t.Error("unexpected mutation", r.URL.Path)
						}
					default:
						t.Error("unexpected request", r.Method, r.URL.Path)
						http.Error(w, "unexpected request", 400)
					}
				})
			}
			source, destination := endpoint("source"), endpoint("destination")
			request := protocol.PrepareHandoffRequest{ExpectedRevision: 1, RequestID: "move", DestinationRunner: "destination", DestinationInstance: "arrived"}
			if _, err := MoveStopped(context.Background(), source, destination, "source", "ship", request, 18080, 0); err == nil {
				t.Fatal("missing lost acknowledgement")
			}
			result, err := MoveStopped(context.Background(), source, destination, "source", "ship", request, 18080, 0)
			if err != nil || result.State != "complete" || !result.ValidCommitment(result.Commitment) {
				t.Fatal("lost acknowledgement could not be reconciled", result, err)
			}
			mu.Lock()
			defer mu.Unlock()
			if retries != 1 || commits != 1 || activations != 1 {
				t.Fatal("mutation repeated", retries, commits, activations)
			}
		})
	}
}

func TestDestinationRevisionRequiresExactTransferBinding(t *testing.T) {
	h := protocol.Handoff{ID: strings.Repeat("a", 32), SourceRunner: "source", SourceInstance: "ship", DestinationRunner: "destination", DestinationInstance: "arrived", Direction: "in", State: "receiving", VerifyKey: strings.Repeat("b", 64)}
	for _, change := range []string{"runner", "missing", "revision", "direction", "transfer", "key", "settings"} {
		t.Run(change, func(t *testing.T) {
			got := h
			i := protocol.Instance{ID: "arrived", ConfigRevision: 9, Handoff: &got}
			s := protocol.Snapshot{RunnerID: "destination"}
			switch change {
			case "runner":
				s.RunnerID = "other"
			case "missing":
				i.Handoff = nil
			case "revision":
				i.ConfigRevision = 0
			case "direction":
				got.Direction = "out"
			case "transfer":
				got.ID = strings.Repeat("c", 32)
			case "key":
				got.VerifyKey = strings.Repeat("c", 64)
			case "settings":
				got.RuntimeSettings = &protocol.RuntimeSettings{Loom: 34}
			}
			s.Instances = []protocol.Instance{i}
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				json.NewEncoder(w).Encode(s)
			})
			if _, revision, err := destinationHandoff(context.Background(), c, h); err == nil || revision != 0 {
				t.Fatal("unbound destination revision accepted", revision, err)
			}
		})
	}
}
