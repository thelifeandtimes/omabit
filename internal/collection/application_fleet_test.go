package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func fleetAttemptFixture(t *testing.T) (*Registry, ApplicationTarget, FleetApplication) {
	t.Helper()
	r := registry(t)
	b := shipApprovalBinding()
	target := ApplicationTarget{PluginID: "fixture.plugin", Profile: "default", SHA256: strings.Repeat("a", 64)}
	f := FleetApplication{Version: 1, RunnerName: "fixture", GrantRevision: 1, Request: protocol.ShipRouteCodeRequest{Binding: b, Selection: protocol.ShipRouteSelection{Target: b.Target, ConfigRevision: 1, RuntimeID: strings.Repeat("a", 64)}, Route: shiproute.Route{ID: "web", Origin: "https://ship.example", Network: "public", Source: "manual", Enabled: true}}}
	return r, target, f
}

func TestFleetApplicationLegacyAmbiguityAndPublicationFailure(t *testing.T) {
	ctx := context.Background()
	r, ship, legacyTarget := applicationFixture(t)
	_, _, fleet := fleetAttemptFixture(t)
	old, err := r.ApproveApplication(ctx, ship, NewSessionRef(), legacyTarget)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.SetApplicationShipApproval(ctx, 0, fleet.Request.Binding, legacyTarget, true); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), legacyTarget, fleet, "file"); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("unknown legacy binding escaped through another URL", err)
	}
	closed := closeUnissued(t, r, old.ID)
	if _, err = r.ArchiveApplication(ctx, old.ID, closed.Revision); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("read-only-directory publication failure needs an unprivileged fixture")
	}
	if err = os.Chmod(r.dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(r.dir, 0700)
	_, err = r.ApproveFleetApplication(ctx, NewSessionRef(), legacyTarget, fleet, "file")
	if err == nil {
		t.Fatal("read-only directory unexpectedly published")
	}
	if err = os.Chmod(r.dir, 0700); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed publication changed original format", err)
	}
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), legacyTarget, fleet, "file"); err != nil {
		t.Fatal("cleared legacy attempt still blocks", err)
	}
}

func TestFleetApplicationRequiresFreshGrantAndBlocksOtherOrigins(t *testing.T) {
	ctx := context.Background()
	r, target, f := fleetAttemptFixture(t)
	if _, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file"); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("default grant", err)
	}
	if _, err := os.Stat(filepath.Join(r.dir, "applications.json")); !os.IsNotExist(err) {
		t.Fatal("denied attempt persisted")
	}
	if _, err := r.SetApplicationShipApproval(ctx, 0, f.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	a, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file")
	if err != nil {
		t.Fatal(err)
	}
	if a.Fleet != f || a.ConnectionID != FleetConnectionID(f.Request.Binding) || a.Phase != "approved" {
		t.Fatal(a)
	}
	other := f
	other.Request.Route.Origin = "https://other.example"
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), target, other, "file"); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("pending attempt escaped via another URL", err)
	}
	if _, err = r.ApproveFleetApplication(ctx, a.ID, target, other, "file"); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("request rebound", err)
	}
	if _, err = r.SetApplicationShipApproval(ctx, 1, f.Request.Binding, target, false); err != nil {
		t.Fatal(err)
	}
	if got, err := r.ApproveFleetApplication(ctx, a.ID, target, f, "file"); err != nil || got != a {
		t.Fatal("existing receipt unavailable", err)
	}
	if err = r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, "issuing", "") }); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("revoked copied grant issued", err)
	}
	// Reallow creates a new grant; it cannot revive the older queued attempt.
	if _, err = r.SetApplicationShipApproval(ctx, 2, f.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	if err = r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, "issuing", "") }); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("reallow revived old work", err)
	}
	closed := closeUnissued(t, r, a.ID)
	if _, err = r.ArchiveApplication(ctx, a.ID, closed.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ApproveFleetApplication(ctx, a.ID, target, f, "file"); !errors.Is(err, ErrApplicationArchived) {
		t.Fatal("archived request replayed", err)
	}
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file"); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("old grant authorized fresh request", err)
	}
	f.GrantRevision = 3
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file"); err != nil {
		t.Fatal("fresh grant refused", err)
	}
	raw, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state applicationsFile
	if json.Unmarshal(raw, &state) != nil || state.Version != 3 {
		t.Fatal("format downgraded after archive")
	}
	for _, name := range []string{"ships.json", "credentials"} {
		if _, err = os.Stat(filepath.Join(r.dir, name)); !os.IsNotExist(err) {
			t.Fatal("approval created owner login", name)
		}
	}
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	got, archived, err := reopened.Application(ctx, a.ID)
	if err != nil || !archived || got != closed {
		t.Fatal("lost archived fleet receipt", err)
	}
}

func TestFleetApplicationRevocationStopsDeliveryButPreservesCleanup(t *testing.T) {
	ctx := context.Background()
	r, target, f := fleetAttemptFixture(t)
	if _, err := r.SetApplicationShipApproval(ctx, 0, f.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	a, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file")
	if err != nil {
		t.Fatal(err)
	}
	advance := func(phase, outcome string) error {
		return r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, phase, outcome) })
	}
	if err = advance("issuing", ""); err != nil {
		t.Fatal(err)
	}
	if err = advance("saved", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = r.SetApplicationShipApproval(ctx, 1, f.Request.Binding, target, false); err != nil {
		t.Fatal(err)
	}
	if err = advance("delivering", ""); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("revoked session handed to receiver", err)
	}
	for _, step := range [][2]string{{"revoking", ""}, {"revoked", "revoked"}, {"closed", "revoked"}} {
		if err = advance(step[0], step[1]); err != nil {
			t.Fatal("revocation blocked cleanup", err)
		}
	}
	got, _, err := r.Application(ctx, a.ID)
	if err != nil || got.Fleet != f || got.Phase != "closed" || got.ReceiverReconciliation {
		t.Fatal(got, err)
	}
}

func TestFleetApplicationIndependentFakesAndCorruptBindings(t *testing.T) {
	ctx := context.Background()
	r, target, f := fleetAttemptFixture(t)
	if _, err := r.SetApplicationShipApproval(ctx, 0, f.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	a, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file")
	if err != nil {
		t.Fatal(err)
	}
	other := f
	other.Request.Binding.Continuity = strings.Repeat("2", 64)
	other.Request.Route.Origin = "https://other.example"
	other.GrantRevision = 2
	if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), target, other, "file"); !errors.Is(err, ErrApplicationPermission) {
		t.Fatal("independent fake inherited permission", err)
	}
	if _, err = r.SetApplicationShipApproval(ctx, 1, other.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	b, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, other, "file")
	if err != nil || a.ConnectionID == b.ConnectionID {
		t.Fatal("fake bindings collided", err)
	}
	path := filepath.Join(r.dir, "applications.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"old-version", "partial-binding", "changed-origin", "changed-identity", "duplicate-active-fleet"} {
		t.Run(kind, func(t *testing.T) {
			var state applicationsFile
			if json.Unmarshal(original, &state) != nil {
				t.Fatal("decode")
			}
			switch kind {
			case "old-version":
				state.Version = 2
			case "partial-binding":
				state.Attempts[0].Fleet.Version = 0
			case "changed-origin":
				state.Attempts[0].Origin = "https://changed.example"
			case "changed-identity":
				state.Attempts[0].ConnectionID = NewSessionRef()
			case "duplicate-active-fleet":
				copy := state.Attempts[0]
				copy.ID = NewSessionRef()
				copy.SessionRef = NewSessionRef()
				copy.Origin = "https://changed.example"
				copy.Fleet.Request.Route.Origin = copy.Origin
				state.Attempts = append(state.Attempts, copy)
			}
			raw, _ := json.Marshal(state)
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = r.Applications(ctx); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("corrupt fleet record accepted", kind, err)
			}
			if _, err = r.ApproveFleetApplication(ctx, NewSessionRef(), target, f, "file"); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("corrupt fleet journal reset", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(raw) {
				t.Fatal("corrupt journal changed")
			}
		})
	}
}
