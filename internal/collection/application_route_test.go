package collection

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestFleetRouteSelectionGrantsFallbackAndFreshBinding(t *testing.T) {
	for _, kind := range []string{"ok", "denied", "stale-grant", "revoked-after-probe", "tailnet", "tailnet-switch", "foreign-tailnet", "lan", "wrong-first", "all-unavailable", "changed-route", "changed-pier", "changed-runtime", "changed-enrollment", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, target, f := fleetAttemptFixture(t)
			binding := f.Request.Binding
			u := FleetApplicationRequest{Version: 1, Binding: binding, Target: target, GrantRevision: 1}
			if kind != "denied" {
				if _, err := r.SetApplicationShipApproval(ctx, 0, binding, target, true); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "stale-grant" {
				u.GrantRevision = 2
			}
			if kind == "lan" {
				u.LANProfiles = []string{"lab"}
			}
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			var probed atomic.Bool
			var calls atomic.Int32
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				calls.Add(1)
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: binding.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature, protocol.ShipRouteCodeFeature}})
				case protocol.AppPolicyPath:
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: binding.Target, Ship: binding.Ship, Ready: true}}})
				case "/v1/snapshot":
					encode(protocol.Snapshot{RunnerID: binding.RunnerID, Mode: "simulation", Instances: []protocol.Instance{{ID: binding.Target.InstanceID, ConfigRevision: 1, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.ShipRoutesPath:
					var selection protocol.ShipRouteSelection
					if json.NewDecoder(q.Body).Decode(&selection) != nil {
						t.Error("selection")
					}
					b := binding
					if kind == "changed-pier" && probed.Load() {
						b.Continuity = strings.Repeat("d", 64)
					}
					if kind == "changed-runtime" && probed.Load() {
						selection.RuntimeID = strings.Repeat("b", 64)
					}
					record := shiproute.DefaultRecord(b)
					record.Settings.Routes = []shiproute.Route{
						{ID: "a-web", Origin: "https://a.example", Network: "public", Source: "manual", Enabled: true},
						{ID: "b-web", Origin: "https://b.example", Network: "public", Source: "manual", Enabled: true},
						{ID: "lan", Origin: "http://192.0.2.1:8888", Network: "lan", Scope: "lab", Source: "manual", Enabled: true},
						{ID: "tail", Origin: "https://box.example.ts.net", Network: "tailnet", Scope: "example.ts.net", Source: "manual", Enabled: true},
					}
					if kind == "changed-route" && probed.Load() {
						record.Settings.Routes[0].Enabled = false
					}
					// A non-default record must have a recorded author/revision.
					record.Revision, record.Actor, record.UpdatedAt = 1, "fixture", time.Now().UTC()
					registry, err := record.Settings.Effective(b, 1, nil)
					if err != nil {
						t.Error(err)
					}
					encode(protocol.ShipRoutes{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", Selection: selection, Record: record, Registry: registry, ReadAt: time.Now().UTC()})
				default:
					t.Error("selection requested credential/effect", q.URL.Path)
					w.WriteHeader(404)
				}
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			if _, err = r.Enroll(ctx, "fixture", socket); err != nil {
				t.Fatal(err)
			}
			before := calls.Load()
			if kind == "canceled" {
				cancel()
			}
			var seen []string
			network := func(context.Context) string {
				if kind == "tailnet" || kind == "tailnet-switch" && !probed.Load() {
					return "example.ts.net"
				}
				if kind == "foreign-tailnet" || kind == "tailnet-switch" {
					return "other.ts.net"
				}
				return ""
			}
			probe := func(ctx context.Context, origin, ship string) error {
				if ship != binding.Ship {
					t.Fatal("ship rebound", ship)
				}
				deadline, bounded := ctx.Deadline()
				if !bounded || time.Until(deadline) > 5*time.Second {
					t.Fatal("unbounded route probe")
				}
				seen = append(seen, origin)
				probed.Store(true)
				if kind == "revoked-after-probe" {
					_, e := r.SetApplicationShipApproval(ctx, 1, binding, target, false)
					if e != nil {
						t.Fatal(e)
					}
				}
				if kind == "changed-enrollment" {
					e := r.Forget(ctx, "fixture", binding.RunnerID)
					if e != nil {
						t.Fatal(e)
					}
				}
				if kind == "wrong-first" && len(seen) == 1 {
					return eyre.ErrIdentity
				}
				if kind == "all-unavailable" {
					return errors.New("PRIVATE")
				}
				return nil
			}
			got, probes, err := r.selectFleetApplication(ctx, "fixture", u, network, probe)
			switch kind {
			case "denied", "stale-grant":
				if !errors.Is(err, ErrApplicationPermission) || calls.Load() != before || len(seen) > 0 {
					t.Fatal("denial performed IO", err, seen)
				}
			case "revoked-after-probe":
				if !errors.Is(err, ErrApplicationPermission) {
					t.Fatal("revoked permission accepted", err)
				}
			case "changed-route", "changed-pier", "changed-runtime", "changed-enrollment", "tailnet-switch", "all-unavailable", "canceled":
				if err == nil || got != (FleetApplication{}) {
					t.Fatal("unsafe selection", got, err)
				}
			default:
				want := "a-web"
				if kind == "tailnet" {
					want = "tail"
				}
				if kind == "lan" {
					want = "lan"
				}
				if kind == "wrong-first" {
					want = "b-web"
				}
				if err != nil || got.Request.Route.ID != want || got.Request.Binding != binding || got.GrantRevision != 1 {
					t.Fatal(got, err, seen)
				}
				if kind == "wrong-first" && (len(probes) != 2 || probes[0].Status != "wrong-ship" || probes[1].Status != "verified") {
					t.Fatal(probes)
				}
			}
			for _, name := range []string{"applications.json", "ships.json", "credentials", "application-receivers"} {
				if _, err := os.Stat(filepath.Join(r.dir, name)); !os.IsNotExist(err) {
					t.Fatal("route selection had effects", name, err)
				}
			}
		})
	}
}

func TestFleetRouteEnvironmentNeverTreatsSSHAsLocal(t *testing.T) {
	local := Connection{Transport: "unix", RunnerID: "runner-a"}
	remote := Connection{Transport: "ssh", RunnerID: "runner-b", SSH: &client.SSHConfig{Host: "fixture"}}
	if fleetEnvironment(local, "", nil).HostID != "runner-a" || fleetEnvironment(remote, "", nil).HostID != "" {
		t.Fatal("remote localhost authorized")
	}
	if got := fleetEnvironment(remote, "example.ts.net", []string{"lab"}); !reflect.DeepEqual(got, shiproute.Environment{TailnetSuffix: "example.ts.net", LANProfiles: []string{"lab"}}) {
		t.Fatal(got)
	}
}

func TestFleetRouteRequestStrictAndBounded(t *testing.T) {
	_, target, f := fleetAttemptFixture(t)
	u := FleetApplicationRequest{Version: 1, Binding: f.Request.Binding, Target: target, GrantRevision: 1}
	data, _ := json.Marshal(u)
	if _, err := DecodeFleetApplicationRequest(data); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{string(data) + "{}", strings.Replace(string(data), `"version":1`, `"version":1,"VERSION":1`, 1), strings.Replace(string(data), `"grant_revision":1`, `"grant_revision":0`, 1), strings.Replace(string(data), `"grant_revision":1`, `"grant_revision":1,"url":"https://arbitrary.example"`, 1), strings.Repeat(" ", 16<<10) + string(data)} {
		if _, err := DecodeFleetApplicationRequest([]byte(raw)); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	u.LANProfiles = []string{"lab", "lab"}
	if u.Validate() == nil {
		t.Fatal("duplicate LAN selection")
	}
}
