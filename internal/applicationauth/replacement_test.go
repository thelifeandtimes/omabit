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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

type replacementFixture struct {
	*reconcileFixture
	capable                              bool
	prepares, replacements, observations int
	prepare                              func(context.Context, []byte) ([]byte, error)
	replace                              func(context.Context, []byte) ([]byte, error)
	observe                              func(context.Context, []byte) ([]byte, error)
}

func (r *replacementFixture) SupportsReplacement() bool { return r.capable }
func replacementResponse(data []byte, status string) []byte {
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	for _, key := range []string{"action", "origin", "ship"} {
		delete(m, key)
	}
	m["status"] = status
	result, _ := json.Marshal(m)
	return result
}
func (r *replacementFixture) PrepareReplacement(ctx context.Context, data []byte) ([]byte, error) {
	r.prepares++
	if r.prepare != nil {
		return r.prepare(ctx, data)
	}
	return replacementResponse(data, "prepared"), nil
}
func (r *replacementFixture) Replace(ctx context.Context, data []byte) ([]byte, error) {
	r.replacements++
	if r.replace != nil {
		return r.replace(ctx, data)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(data, &m)
	return ack(m["offer"]), nil
}
func (r *replacementFixture) ReplacementStatus(ctx context.Context, data []byte) ([]byte, error) {
	r.observations++
	if r.observe != nil {
		return r.observe(ctx, data)
	}
	return replacementResponse(data, "accepted"), nil
}

func replacementSetup(t *testing.T) (*fixture, *replacementFixture, collection.ApplicationAttempt, collection.ApplicationAttempt) {
	t.Helper()
	ctx := context.Background()
	f := setup(t)
	r := &replacementFixture{reconcileFixture: &reconcileFixture{receiver: newReceiver()}, capable: true}
	b := shiproute.Binding{RunnerID: "fixture", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
	fleet := collection.FleetApplication{Version: 1, RunnerName: "fixture", GrantRevision: 1, Request: protocol.ShipRouteCodeRequest{Binding: b, Selection: protocol.ShipRouteSelection{Target: b.Target, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1}, Route: shiproute.Route{ID: "managed-http", Origin: f.ship.Origin, Network: "local", Scope: b.RunnerID, Source: "managed", Enabled: true}}}
	if _, err := f.r.SetApplicationShipApproval(ctx, 0, b, r.Target(), true); err != nil {
		t.Fatal(err)
	}
	parent, err := f.r.ApproveFleetApplication(ctx, collection.NewSessionRef(), r.Target(), fleet, "file")
	if err != nil {
		t.Fatal(err)
	}
	// Actual Eyre sessions, with the existing fixture's standalone login. Runner
	// code retrieval/route binding is covered by TestFleetApplicationIssues... .
	f.e.issue = func(ctx context.Context, a collection.ApplicationAttempt) ([]byte, error) {
		c, err := eyre.New(a.Origin, a.Ship)
		if err != nil {
			return nil, err
		}
		defer c.Close()
		if err := c.Login(ctx, []byte(fixtureCode)); err != nil {
			return nil, err
		}
		return c.ExportSession()
	}
	parent, err = f.e.Connect(ctx, parent.ID, r)
	if err != nil {
		t.Fatal(err)
	}
	child, err := f.r.ApproveFleetReplacement(ctx, parent.ID, parent.Revision, collection.NewSessionRef(), r.Target(), fleet)
	if err != nil {
		t.Fatal(err)
	}
	return f, r, parent, child
}

func TestReplacementIssuerIndependentSessionAndExactOriginalCleanup(t *testing.T) {
	ctx := context.Background()
	f, r, parent, child := replacementSetup(t)
	original, err := f.store.Load(ctx, credentialKey(parent))
	if err != nil {
		t.Fatal(err)
	}
	defer clear(original)
	r.prepare = func(_ context.Context, data []byte) ([]byte, error) {
		want := replacementBinding(parent)
		want["action"], want["next_request_id"] = "prepare-replacement", child.ID
		expected, _ := json.Marshal(want)
		if !bytes.Equal(data, expected) {
			t.Fatal("unbound preparation")
		}
		return replacementResponse(data, "prepared"), nil
	}
	r.replace = func(_ context.Context, data []byte) ([]byte, error) {
		var m map[string]json.RawMessage
		if json.Unmarshal(data, &m) != nil || len(m) != 4 {
			t.Fatal("invalid replacement")
		}
		want, _ := json.Marshal(replacementBinding(parent))
		if !bytes.Equal(m["previous"], want) {
			t.Fatal("changed predecessor")
		}
		var offer struct {
			Session json.RawMessage `json:"session"`
		}
		if json.Unmarshal(m["offer"], &offer) != nil || bytes.Equal(offer.Session, original) {
			t.Fatal("reused predecessor cookie")
		}
		if _, err := f.e.Disconnect(ctx, parent.ID); !errors.Is(err, collection.ErrApplicationBusy) {
			t.Fatal("disconnect raced delivery", err)
		}
		return ack(m["offer"]), nil
	}
	got, err := f.e.ConnectReplacement(ctx, child.ID, r)
	if err != nil || got.Phase != "connected" || r.prepares != 1 || r.replacements != 1 || r.observations != 0 {
		t.Fatal(got.Phase, err)
	}
	if _, err := f.e.ConnectReplacement(ctx, child.ID, r); err != nil || r.prepares != 1 || r.replacements != 1 {
		t.Fatal("replayed connected child", err)
	}
	old, _, _ := f.r.Application(ctx, parent.ID)
	if old.Phase != "retiring" || old.Replacement.SuccessorID != child.ID {
		t.Fatal(old)
	}
	if _, err := f.e.Disconnect(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.e.Reconcile(ctx, parent.ID, r); err != nil {
		t.Fatal(err)
	}
	data, err := f.store.Load(ctx, credentialKey(got))
	if err != nil {
		t.Fatal("successor credential deleted", err)
	}
	state, err := eyre.StoredSessionHealth(ctx, got.Origin, got.Ship, data)
	if err != nil || state != "healthy" {
		t.Fatal(state, err)
	}
	if err := f.owner.Check(ctx); err != nil {
		t.Fatal("owner session changed", err)
	}
	logins, logouts, active := f.counts()
	if logins != 3 || logouts != 1 || active != 2 {
		t.Fatal(logins, logouts, active)
	}
}

func TestReplacementIssuerLostAcknowledgementNeverReissuesOrRedelivers(t *testing.T) {
	for _, kind := range []string{"accepted", "pending", "absent", "superseded", "unknown", "wrong-binding", "duplicate", "revoke-after-delivery"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			f, r, parent, child := replacementSetup(t)
			r.replace = func(context.Context, []byte) ([]byte, error) {
				if kind == "revoke-after-delivery" {
					if _, err := f.r.SetApplicationShipApproval(ctx, 1, parent.Fleet.Request.Binding, r.Target(), false); err != nil {
						t.Fatal(err)
					}
				}
				return nil, errors.New("PRIVATE lost reply")
			}
			r.observe = func(_ context.Context, data []byte) ([]byte, error) {
				switch kind {
				case "unknown":
					return nil, errors.New("PRIVATE unavailable status")
				case "wrong-binding":
					var m map[string]any
					json.Unmarshal(data, &m)
					m["request_id"] = collection.NewSessionRef()
					raw, _ := json.Marshal(m)
					return replacementResponse(raw, "accepted"), nil
				case "duplicate":
					raw := replacementResponse(data, "accepted")
					return append([]byte(`{"status":"accepted",`), raw[1:]...), nil
				case "revoke-after-delivery":
					return replacementResponse(data, "accepted"), nil
				default:
					return replacementResponse(data, kind), nil
				}
			}
			got, err := f.e.ConnectReplacement(ctx, child.ID, r)
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("private error exposed")
			}
			switch kind {
			case "accepted":
				if err != nil || got.Phase != "connected" {
					t.Fatal(got.Phase, err)
				}
			case "revoke-after-delivery":
				if !errors.Is(err, collection.ErrApplicationPermission) || got.Phase != "closed" || !got.ReceiverReconciliation {
					t.Fatal(got.Phase, err)
				}
			case "pending", "absent", "superseded":
				if !errors.Is(err, ErrDelivery) || got.Phase != "closed" || !got.ReceiverReconciliation {
					t.Fatal(got.Phase, err)
				}
			default:
				if !errors.Is(err, ErrRecovery) || got.Phase != "delivering" {
					t.Fatal(got.Phase, err)
				}
				if _, err := f.e.Recover(ctx, child.ID); !errors.Is(err, ErrRecovery) {
					t.Fatal("generic recovery skipped observation", err)
				}
				f.reopen(t)
				r.observe = nil
				got, err = f.e.AtRevision(got.Revision).RecoverReplacement(ctx, child.ID, r)
				if err != nil || got.Phase != "connected" {
					t.Fatal("restart lost accepted delivery", got.Phase, err)
				}
			}
			logins, _, _ := f.counts()
			if logins != 3 || r.prepares != 1 || r.replacements != 1 {
				t.Fatal("replayed side effect", logins, r.prepares, r.replacements)
			}
			journal, _ := os.ReadFile(filepath.Join(f.dir, "applications.json"))
			if bytes.Contains(journal, []byte("fixture-cookie")) || bytes.Contains(journal, []byte(fixtureCode)) {
				t.Fatal("secret in journal")
			}
		})
	}
}

func TestReplacementIssuerPreparationFailureAndPartialIssueStayFenced(t *testing.T) {
	for _, kind := range []string{"old-capability", "normal-connect", "missing-old", "invalid-old", "lost-preparation", "revoke-before-preparation", "revoke-during-preparation", "not-issued", "reused-cookie", "lost-login", "interrupted-preparing", "interrupted-prepared"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			f, r, parent, child := replacementSetup(t)
			switch kind {
			case "old-capability":
				r.capable = false
			case "missing-old":
				if err := f.store.Delete(ctx, credentialKey(parent)); err != nil {
					t.Fatal(err)
				}
			case "invalid-old":
				f.e.store = func(collection.ApplicationAttempt) (credentials.Store, error) {
					return healthStore{Store: f.store, load: func(context.Context, string) ([]byte, error) { return []byte(`{"bad":"PRIVATE"}`), nil }}, nil
				}
			case "lost-preparation":
				r.prepare = func(context.Context, []byte) ([]byte, error) { return nil, ErrDelivery }
			case "revoke-before-preparation":
				if _, err := f.r.SetApplicationShipApproval(ctx, 1, parent.Fleet.Request.Binding, r.Target(), false); err != nil {
					t.Fatal(err)
				}
			case "revoke-during-preparation":
				r.prepare = func(_ context.Context, data []byte) ([]byte, error) {
					if _, err := f.r.SetApplicationShipApproval(ctx, 1, parent.Fleet.Request.Binding, r.Target(), false); err != nil {
						t.Fatal(err)
					}
					return replacementResponse(data, "prepared"), nil
				}
			case "reused-cookie":
				f.e.issue = func(ctx context.Context, _ collection.ApplicationAttempt) ([]byte, error) {
					return f.store.Load(ctx, credentialKey(parent))
				}
			case "not-issued":
				f.e.issue = func(context.Context, collection.ApplicationAttempt) ([]byte, error) { return nil, ErrNotIssued }
			case "lost-login":
				f.mu.Lock()
				f.lostLogin = true
				f.mu.Unlock()
			case "interrupted-preparing", "interrupted-prepared":
				if err := f.r.WithApplication(ctx, child.ID, func(tx *collection.ApplicationTransaction) error {
					if err := tx.Advance(ctx, "preparing", ""); err != nil {
						return err
					}
					if kind == "interrupted-prepared" {
						return tx.Advance(ctx, "prepared", "")
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			var got collection.ApplicationAttempt
			var err error
			if kind == "normal-connect" {
				got, err = f.e.Connect(ctx, child.ID, r)
			} else {
				got, err = f.e.ConnectReplacement(ctx, child.ID, r)
			}
			if err == nil {
				t.Fatal("unexpected success", kind, got.Phase)
			}
			if kind == "lost-login" {
				if got.Phase != "unresolved" {
					t.Fatal("lost issuance not fenced", got.Phase)
				}
				if _, err = f.e.Recover(ctx, child.ID); !errors.Is(err, ErrCleanup) {
					t.Fatal(err)
				}
			} else {
				if got.Phase == "approved" || got.Phase == "preparing" || got.Phase == "prepared" {
					if got, err = f.e.Recover(ctx, child.ID); err != nil {
						t.Fatal("known unissued recovery failed", err)
					}
				}
				if kind != "normal-connect" && got.Phase != "closed" {
					t.Fatal("unissued child not closed", got.Phase)
				}
			}
			logins, logouts, _ := f.counts()
			want := 2
			if kind == "lost-login" {
				want = 3
			}
			if logins != want || logouts != 0 || r.replacements != 0 {
				t.Fatal("unsafe side effect", kind, logins, logouts, r.replacements)
			}
			old, _, _ := f.r.Application(ctx, parent.ID)
			if old.Phase != "retiring" || old.Replacement.SuccessorID != "" {
				t.Fatal("restored/activated old account by inference", old)
			}
		})
	}
}

func TestReplacementExplicitDisconnectCancelsPendingAutomationOnly(t *testing.T) {
	for _, kind := range []string{"unissued-child", "pending-parent", "accepted-parent"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			f, r, parent, child := replacementSetup(t)
			if kind == "accepted-parent" {
				var err error
				child, err = f.e.ConnectReplacement(ctx, child.ID, r)
				if err != nil {
					t.Fatal(err)
				}
			}
			id := parent.ID
			if kind == "unissued-child" {
				id = child.ID
			}
			if _, err := f.e.Disconnect(ctx, id); err != nil {
				t.Fatal(err)
			}
			p, _, err := f.r.Application(ctx, parent.ID)
			if err != nil {
				t.Fatal(err)
			}
			c, _, err := f.r.Application(ctx, child.ID)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "accepted-parent" {
				if c.Replacement.AutomaticDisabled || c.Phase != "connected" {
					t.Fatal("original cleanup stopped successor", c)
				}
			} else {
				if !p.Replacement.AutomaticDisabled || !c.Replacement.AutomaticDisabled {
					t.Fatal("pending intent could resume", p, c)
				}
				if _, err := f.e.ConnectReplacement(ctx, child.ID, r); err == nil {
					t.Fatal("disconnected handoff resumed")
				}
			}
		})
	}
}
