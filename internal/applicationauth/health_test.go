package applicationauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

type healthStore struct {
	credentials.Store
	load func(context.Context, string) ([]byte, error)
}

func (s healthStore) Load(ctx context.Context, key string) ([]byte, error) { return s.load(ctx, key) }

func TestFleetHealthDoesNotIssueDeliverOrCleanup(t *testing.T) {
	for _, kind := range []string{"healthy", "expired", "rejected", "route-unavailable", "route-error", "revoked", "stale", "changed-helper", "changed-artifact", "store-missing", "store-locked", "store-corrupt", "revoke-during-load", "pending", "concurrent-disconnect"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			f, receiver := setup(t), newReceiver()
			binding := shiproute.Binding{RunnerID: "fixture", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			fleet := collection.FleetApplication{Version: 1, RunnerName: "fixture", GrantRevision: 1, Request: protocol.ShipRouteCodeRequest{Binding: binding, Selection: protocol.ShipRouteSelection{Target: binding.Target, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1}, Route: shiproute.Route{ID: "managed-http", Origin: f.ship.Origin, Network: "local", Scope: binding.RunnerID, Source: "managed", Enabled: true}}}
			if _, err := f.r.SetApplicationShipApproval(ctx, 0, binding, receiver.target, true); err != nil {
				t.Fatal(err)
			}
			a, err := f.r.ApproveFleetApplication(ctx, collection.NewSessionRef(), receiver.target, fleet, "file")
			if err != nil {
				t.Fatal(err)
			}
			// Issuance is independently covered by the fleet engine fixture. This
			// test's actual Eyre session focuses on health and immutable evidence.
			f.e.issue = func(ctx context.Context, _ collection.ApplicationAttempt) ([]byte, error) {
				child, err := f.owner.NewIndependentSession(ctx)
				if err != nil {
					return nil, err
				}
				defer child.Close()
				return child.ExportSession()
			}
			if kind != "pending" {
				a, err = f.e.Connect(ctx, a.ID, receiver)
				if err != nil {
					t.Fatal(err)
				}
			}
			if kind == "rejected" {
				f.mu.Lock()
				f.sessions = map[string]bool{}
				f.mu.Unlock()
			}
			if kind == "revoked" {
				if _, err = f.r.SetApplicationShipApproval(ctx, 1, binding, receiver.target, false); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "changed-helper" {
				receiver.target.SHA256 = strings.Repeat("b", 64)
			}
			if kind == "changed-artifact" {
				receiver.verify = func(context.Context) error { return errors.New("PRIVATE altered helper") }
			}
			reads, eligibilityReads := 0, 0
			f.e.store = func(collection.ApplicationAttempt) (credentials.Store, error) {
				return healthStore{Store: f.store, load: func(ctx context.Context, key string) ([]byte, error) {
					reads++
					switch kind {
					case "store-missing":
						return nil, credentials.ErrMissing
					case "store-locked":
						return nil, errors.New("PRIVATE keyring unavailable")
					case "store-corrupt":
						return []byte("PRIVATE invalid session"), nil
					case "revoke-during-load":
						if _, err := f.r.SetApplicationShipApproval(ctx, 1, binding, receiver.target, false); err != nil {
							t.Error(err)
						}
					}
					data, err := f.store.Load(ctx, key)
					if kind == "expired" && err == nil {
						// Model an elapsed expiry on read; immutable credential slots
						// correctly refuse overwriting a previously saved token.
						var s map[string]any
						if json.Unmarshal(data, &s) != nil {
							t.Fatal("fixture session invalid")
						}
						clear(data)
						s["expires"] = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
						return json.Marshal(s)
					}
					return data, err
				}}, nil
			}
			eligible := func(ctx context.Context, selected collection.ApplicationAttempt, lan []string) (bool, error) {
				eligibilityReads++
				if selected != a || len(lan) != 0 {
					t.Error("health selection changed")
				}
				if kind == "concurrent-disconnect" {
					if _, err := f.e.Disconnect(ctx, a.ID); !errors.Is(err, collection.ErrApplicationBusy) {
						t.Error("health failed to serialize disconnect", err)
					}
				}
				if kind == "route-error" {
					return false, collection.ErrApplicationConflict
				}
				return kind != "route-unavailable", nil
			}
			before, err := os.ReadFile(filepath.Join(f.dir, "applications.json"))
			if err != nil {
				t.Fatal(err)
			}
			logins, logouts, _ := f.counts()
			deliveries := receiver.calls
			revision := a.Revision
			if kind == "stale" {
				revision++
			}
			state, err := f.e.AtRevision(revision).checkFleet(ctx, a.ID, nil, receiver, eligible)
			want, wantReads := "", 0
			var wantError error
			switch kind {
			case "healthy", "expired", "rejected":
				want, wantReads = kind, 1
			case "concurrent-disconnect":
				want, wantReads = "healthy", 1
			case "route-unavailable":
				want = kind
			case "route-error", "stale", "changed-helper", "pending":
				wantError = collection.ErrApplicationConflict
			case "revoked", "revoke-during-load":
				wantError = collection.ErrApplicationPermission
				if kind == "revoke-during-load" {
					wantReads = 1
				}
			case "changed-artifact":
				wantError = ErrArtifact
			case "store-missing", "store-locked":
				wantError, wantReads = credentials.ErrStore, 1
			case "store-corrupt":
				wantError, wantReads = eyre.ErrStoredSession, 1
			}
			if state != want || !errors.Is(err, wantError) || reads != wantReads {
				t.Fatalf("state=%q err=%v credential reads=%d", state, err, reads)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("private error exposed")
			}
			if (kind == "stale" || kind == "pending" || kind == "revoked" || kind == "changed-helper" || kind == "changed-artifact") && eligibilityReads != 0 {
				t.Fatal("invalid selection performed eligibility IO")
			}
			after, err := os.ReadFile(filepath.Join(f.dir, "applications.json"))
			nowLogins, nowLogouts, _ := f.counts()
			if err != nil || !bytes.Equal(before, after) || logins != nowLogins || logouts != nowLogouts || deliveries != receiver.calls {
				t.Fatal("health changed journal, login, logout or receiver", err)
			}
		})
	}
}
