package collection

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func shipApprovalBinding() shiproute.Binding {
	return shiproute.Binding{RunnerID: "runner-fixture", Target: apppolicy.Target{InstanceID: "zod"}, Ship: "~zod", Continuity: strings.Repeat("1", 64)}
}

func TestApplicationShipApprovalIsExplicitPerShipAndSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	r, ship, target := applicationFixture(t)
	prior, err := r.ApproveApplication(ctx, ship, NewSessionRef(), target)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil {
		t.Fatal(err)
	}
	binding := shipApprovalBinding()
	state, err := r.ApplicationShipApprovals(ctx)
	if err != nil || state.Allows(binding, target) || state.Revision != 0 {
		t.Fatal("legacy attempt became standing consent", state, err)
	}
	state, err = r.SetApplicationShipApproval(ctx, 0, binding, target, true)
	if err != nil || !state.Allows(binding, target) || state.Revision != 1 {
		t.Fatal(state, err)
	}
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.ApplicationShipApprovals(ctx)
	if err != nil || !reflect.DeepEqual(state, loaded) {
		t.Fatal("reopen lost approval", loaded, err)
	}
	// This grant has no URL field: new verified URLs for exactly this binding do
	// not require another grant. A different incarnation, project, or helper does.
	for _, mutate := range []func(*shiproute.Binding, *ApplicationTarget){
		func(b *shiproute.Binding, a *ApplicationTarget) { b.RunnerID = "other" },
		func(b *shiproute.Binding, a *ApplicationTarget) { b.Continuity = strings.Repeat("2", 64) },
		func(b *shiproute.Binding, a *ApplicationTarget) {
			b.Target.ProjectID = "project"
			b.Target.Generation = 1
			b.Target.Creator = "created"
		},
		func(b *shiproute.Binding, a *ApplicationTarget) { a.Profile = "other" },
		func(b *shiproute.Binding, a *ApplicationTarget) { a.SHA256 = strings.Repeat("b", 64) },
	} {
		b, a := binding, target
		mutate(&b, &a)
		if loaded.Allows(b, a) {
			t.Fatal("grant crossed binding", b, a)
		}
	}
	if _, err := r.SetApplicationShipApproval(ctx, 0, binding, target, false); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("stale revoke applied", err)
	}
	state, err = r.SetApplicationShipApproval(ctx, 1, binding, target, false)
	if err != nil || state.Allows(binding, target) || state.Revision != 2 {
		t.Fatal("revoke failed", state, err)
	}
	again, err := r.SetApplicationShipApproval(ctx, 2, binding, target, false)
	if err != nil || !reflect.DeepEqual(again, state) {
		t.Fatal("idempotent choice changed history", err)
	}
	after, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("standing consent changed old attempt", err, prior)
	}
}

func TestApplicationShipApprovalHelperReplacementDoesNotLeaveOldGrant(t *testing.T) {
	ctx := context.Background()
	r, _, old := applicationFixture(t)
	b := shipApprovalBinding()
	state, err := r.SetApplicationShipApproval(ctx, 0, b, old, true)
	if err != nil {
		t.Fatal(err)
	}
	current := old
	current.SHA256 = strings.Repeat("b", 64)
	state, err = r.SetApplicationShipApproval(ctx, state.Revision, b, current, true)
	if err != nil || state.Allows(b, old) || !state.Allows(b, current) || len(state.Entries) != 1 || len(state.History) != 2 {
		t.Fatal(state, err)
	}
}

func TestApplicationShipApprovalRefusesCorruptionWithoutReset(t *testing.T) {
	for _, scenario := range []string{"version", "missing-history", "changed-current", "revision-gap", "duplicate-key", "unknown-field", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			r, _, a := applicationFixture(t)
			b := shipApprovalBinding()
			state, err := r.SetApplicationShipApproval(ctx, 0, b, a, true)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "version":
				state.Version = 2
			case "missing-history":
				state.History = nil
			case "changed-current":
				state.Entries[0].Allowed = false
			case "revision-gap":
				state.History[0].Revision = 2
			}
			body, _ := json.Marshal(state)
			if scenario == "duplicate-key" {
				body = append([]byte(`{"version":1,`), body[1:]...)
			}
			if scenario == "unknown-field" {
				body = append([]byte(`{"unexpected":true,`), body[1:]...)
			}
			path := filepath.Join(r.dir, applicationShipApprovalsFile)
			if err := os.WriteFile(path, body, 0600); err != nil {
				t.Fatal(err)
			}
			if scenario == "symlink" {
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".original", path); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := r.ApplicationShipApprovals(ctx); err == nil {
				t.Fatal("invalid state read")
			}
			if _, err := r.SetApplicationShipApproval(ctx, 1, b, a, false); err == nil {
				t.Fatal("invalid state reset")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(body) {
				t.Fatal("invalid evidence changed", err)
			}
		})
	}
}
