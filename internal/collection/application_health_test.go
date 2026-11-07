package collection

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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestFleetHealthOriginRequiresCurrentGrantBindingAndNetwork(t *testing.T) {
	for _, kind := range []string{"public", "local", "disabled", "removed", "other-pier", "new-runtime", "revoked", "stale-attempt", "revoke-during-network", "tailnet", "foreign-tailnet", "tailnet-switch", "lan", "lan-missing", "changed-enrollment"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			r, target, f := fleetAttemptFixture(t)
			if strings.Contains(kind, "tailnet") {
				f.Request.Route.Network, f.Request.Route.Scope = "tailnet", "example.ts.net"
				f.Request.Route.Origin = "https://box.example.ts.net"
			}
			if strings.HasPrefix(kind, "lan") {
				f.Request.Route.Network, f.Request.Route.Scope = "lan", "fixture-lan"
			}
			if kind == "local" {
				f.Request.Route = shiproute.Route{ID: "managed-http", Origin: "http://127.0.0.1:8888", Network: "local", Scope: f.Request.Binding.RunnerID, Source: "managed", Enabled: true}
			}
			if _, err := r.SetApplicationShipApproval(ctx, 0, f.Request.Binding, target, true); err != nil {
				t.Fatal(err)
			}
			a, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file")
			if err != nil {
				t.Fatal(err)
			}
			if err = r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error {
				for _, phase := range []string{"issuing", "saved", "delivering", "connected"} {
					if err := tx.Advance(ctx, phase, ""); err != nil {
						return err
					}
				}
				a = tx.Current()
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				calls.Add(1)
				w.Header().Set(protocol.Header, "1")
				b := f.Request.Binding
				switch q.URL.Path {
				case "/hello":
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: b.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature}})
				case protocol.AppPolicyPath:
					json.NewEncoder(w).Encode(protocol.AppPolicyStatus{Version: 1, RunnerID: b.RunnerID, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: b.Target, Ship: b.Ship, Ready: true}}})
				case "/v1/snapshot":
					runtime := strings.Repeat("a", 64)
					if kind == "new-runtime" {
						runtime = strings.Repeat("b", 64)
					}
					json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: b.RunnerID, Mode: "simulation", Instances: []protocol.Instance{{ID: b.Target.InstanceID, ConfigRevision: 1, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: runtime}}}})
				case protocol.ShipRoutesPath:
					var selection protocol.ShipRouteSelection
					if json.NewDecoder(q.Body).Decode(&selection) != nil {
						t.Error("invalid selection")
					}
					if kind == "other-pier" {
						b.Continuity = strings.Repeat("e", 64)
					}
					record := shiproute.DefaultRecord(b)
					route := f.Request.Route
					if kind == "disabled" {
						route.Enabled = false
					}
					var detected []shiproute.Route
					if kind == "local" {
						detected = []shiproute.Route{route}
					} else if kind != "removed" {
						record.Settings.Routes = []shiproute.Route{route}
						record.Revision, record.Actor, record.UpdatedAt = 1, "fixture", time.Now().UTC()
					}
					registry, e := record.Settings.Effective(b, record.Revision, detected)
					if e != nil {
						t.Error(e)
					}
					json.NewEncoder(w).Encode(protocol.ShipRoutes{Version: 1, RunnerID: b.RunnerID, Mode: "simulation", Selection: selection, Record: record, Registry: registry, ReadAt: time.Now().UTC()})
				default:
					t.Error("health eligibility requested credentials/effects", q.URL.Path)
					w.WriteHeader(400)
				}
			})}
			go server.Serve(ln)
			t.Cleanup(func() { server.Close() })
			if _, err = r.Enroll(ctx, f.RunnerName, socket); err != nil {
				t.Fatal(err)
			}
			if kind == "revoked" {
				if _, err = r.SetApplicationShipApproval(ctx, 1, f.Request.Binding, target, false); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "stale-attempt" {
				a.Revision--
			}
			before, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
			if err != nil {
				t.Fatal(err)
			}
			networkReads := 0
			network := func(context.Context) string {
				networkReads++
				if kind == "revoke-during-network" {
					if _, err := r.SetApplicationShipApproval(ctx, 1, f.Request.Binding, target, false); err != nil {
						t.Error(err)
					}
				}
				if kind == "changed-enrollment" {
					if err := r.Forget(ctx, f.RunnerName, f.Request.Binding.RunnerID); err != nil {
						t.Error(err)
					}
				}
				if kind == "foreign-tailnet" || kind == "tailnet-switch" && networkReads > 1 {
					return "other.ts.net"
				}
				return "example.ts.net"
			}
			var lan []string
			if kind == "lan" {
				lan = []string{"fixture-lan"}
			}
			previousCalls := calls.Load()
			ok, err := r.fleetApplicationOriginEligible(ctx, a, lan, network)
			switch kind {
			case "revoked", "revoke-during-network":
				if !errors.Is(err, ErrApplicationPermission) || ok {
					t.Fatal("revoked grant accepted", ok, err)
				}
			case "stale-attempt", "other-pier", "changed-enrollment":
				if !errors.Is(err, ErrApplicationConflict) || ok {
					t.Fatal("changed binding accepted", ok, err)
				}
			case "disabled", "removed", "foreign-tailnet", "tailnet-switch", "lan-missing":
				if err != nil || ok {
					t.Fatal("ineligible origin accepted", ok, err)
				}
			default:
				if err != nil || !ok {
					t.Fatal("current origin refused", ok, err)
				}
			}
			if (kind == "revoked" || kind == "stale-attempt") && (calls.Load() != previousCalls || networkReads != 0) {
				t.Fatal("denied check performed IO")
			}
			after, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("health changed durable attempt", err)
			}
		})
	}
}
