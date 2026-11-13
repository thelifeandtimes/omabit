package applicationauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestFleetApplicationIssuesIndependentSessionAndRetainsRecovery(t *testing.T) {
	for _, kind := range []string{"ok", "revoked-before-issue", "revoked-at-code", "revoked-before-delivery", "changed-code-binding", "lost-delivery", "lost-login"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			f := setup(t)
			receiver := newReceiver()
			// A valid legacy owner session exists at the same URL. The fleet path must
			// not borrow it, replace it or use its code scry as an authority shortcut.
			f.mu.Lock()
			f.threadUnavailable = true
			f.lostLogin = kind == "lost-login"
			f.mu.Unlock()
			oldShips, err := os.ReadFile(filepath.Join(f.dir, "ships.json"))
			if err != nil {
				t.Fatal(err)
			}
			b := shiproute.Binding{RunnerID: "runner-fixture", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			request := protocol.ShipRouteCodeRequest{Selection: protocol.ShipRouteSelection{Target: b.Target, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1}, Binding: b, Route: shiproute.Route{ID: "managed-http", Origin: f.ship.Origin, Network: "local", Scope: b.RunnerID, Source: "managed", Enabled: true}}
			var codes atomic.Int32
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				w.Header().Set("Content-Type", "application/json")
				if q.URL.Path == "/hello" {
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: b.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.ShipRouteCodeFeature}})
					return
				}
				if q.URL.Path != protocol.ShipRouteCodePath {
					t.Error("unexpected runner operation", q.URL.Path)
					w.WriteHeader(400)
					return
				}
				var got protocol.ShipRouteCodeRequest
				if json.NewDecoder(q.Body).Decode(&got) != nil || got != request {
					t.Error("unbound credential request")
					w.WriteHeader(400)
					return
				}
				codes.Add(1)
				if kind == "revoked-at-code" {
					if _, err := f.r.SetApplicationShipApproval(ctx, 1, b, receiver.target, false); err != nil {
						t.Error(err)
					}
				}
				reply := protocol.ShipRouteCode{Version: 1, RunnerID: b.RunnerID, Mode: "simulation", Request: got, Code: fixtureCode}
				if kind == "changed-code-binding" {
					reply.Request.Binding.Continuity = strings.Repeat("d", 64)
				}
				json.NewEncoder(w).Encode(reply)
			})}
			go server.Serve(ln)
			t.Cleanup(func() { server.Close() })
			if _, err = f.r.Enroll(ctx, "fixture", socket); err != nil {
				t.Fatal(err)
			}
			if _, err = f.r.SetApplicationShipApproval(ctx, 0, b, receiver.target, true); err != nil {
				t.Fatal(err)
			}
			fleet := collection.FleetApplication{Version: 1, RunnerName: "fixture", Request: request, GrantRevision: 1}
			a, err := f.r.ApproveFleetApplication(ctx, collection.NewSessionRef(), receiver.target, fleet, "file")
			if err != nil {
				t.Fatal(err)
			}
			verifications := 0
			receiver.verify = func(context.Context) error {
				verifications++
				if kind == "revoked-before-issue" && verifications == 1 || kind == "revoked-before-delivery" && verifications == 2 {
					_, err := f.r.SetApplicationShipApproval(ctx, 1, b, receiver.target, false)
					return err
				}
				return nil
			}
			if kind == "lost-delivery" {
				receiver.deliver = func(context.Context, []byte) ([]byte, error) { return nil, errors.New("PRIVATE lost acknowledgement") }
			}
			got, err := f.e.Connect(ctx, a.ID, receiver)
			if err != nil && (strings.Contains(err.Error(), fixtureCode) || strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "fixture-cookie")) {
				t.Fatal("private error escaped")
			}
			logins, logouts, active := f.counts()
			switch kind {
			case "ok":
				if err != nil || got.Phase != "connected" || codes.Load() != 1 || receiver.calls != 1 || logins != 2 || active != 2 {
					t.Fatal("fleet connection failed", got.Phase, err, codes.Load(), receiver.calls, logins, active)
				}
				if _, err = f.e.Connect(ctx, a.ID, receiver); err != nil || receiver.calls != 1 || codes.Load() != 1 {
					t.Fatal("duplicate issued again", err)
				}
				if _, err = f.r.SetApplicationShipApproval(ctx, 1, b, receiver.target, false); err != nil {
					t.Fatal(err)
				}
				f.reopen(t)
				got, err = f.e.Disconnect(ctx, a.ID)
				if err != nil || got.Phase != "closed" || !got.ReceiverReconciliation {
					t.Fatal("reopened cleanup failed", err)
				}
			case "revoked-before-issue", "changed-code-binding":
				if !errors.Is(err, ErrNotIssued) || got.Phase != "closed" || got.Outcome != "not-issued" || logins != 1 || receiver.calls != 0 {
					t.Fatal("unapproved/changed credential issued", got.Phase, err, logins, receiver.calls)
				}
				if kind == "revoked-before-issue" && codes.Load() != 0 {
					t.Fatal("revoked approval read credentials")
				}
			case "revoked-at-code", "revoked-before-delivery":
				if !errors.Is(err, collection.ErrApplicationPermission) || got.Phase != "closed" || got.ReceiverReconciliation || logins != 2 || logouts != 1 || active != 1 || receiver.calls != 0 {
					t.Fatal("revoked delivery/cleanup", got.Phase, err, logins, logouts, active, receiver.calls)
				}
			case "lost-delivery":
				if !errors.Is(err, ErrDelivery) || got.Phase != "closed" || !got.ReceiverReconciliation || receiver.calls != 1 || logouts != 1 || active != 1 {
					t.Fatal("lost ack recovery failed", got.Phase, err)
				}
				if _, err = f.e.Connect(ctx, a.ID, receiver); !errors.Is(err, ErrRecovery) || receiver.calls != 1 || codes.Load() != 1 {
					t.Fatal("uncertain receiver got replay", err)
				}
			case "lost-login":
				if !errors.Is(err, ErrRecovery) || got.Phase != "unresolved" || logins != 2 || active != 2 || receiver.calls != 0 || codes.Load() != 1 {
					t.Fatal("lost login did not retain uncertainty", got.Phase, err, logins, active, receiver.calls, codes.Load())
				}
				f.reopen(t)
				if _, err = f.e.Recover(ctx, a.ID); !errors.Is(err, ErrCleanup) {
					t.Fatal("unknown session declared clean", err)
				}
				if _, err = f.r.ApproveFleetApplication(ctx, collection.NewSessionRef(), receiver.target, fleet, "file"); !errors.Is(err, collection.ErrApplicationConflict) {
					t.Fatal("uncertain login allowed second attempt", err)
				}
			}
			newShips, err := os.ReadFile(filepath.Join(f.dir, "ships.json"))
			if err != nil || !bytes.Equal(oldShips, newShips) {
				t.Fatal("fleet changed parent enrollment", err)
			}
			if err = f.owner.Check(ctx); err != nil {
				t.Fatal("fleet invalidated parent cookie", err)
			}
			journal, err := os.ReadFile(filepath.Join(f.dir, "applications.json"))
			if err != nil || bytes.Contains(journal, []byte(fixtureCode)) || bytes.Contains(journal, []byte("fixture-cookie")) {
				t.Fatal("private material in journal", err)
			}
		})
	}
}
