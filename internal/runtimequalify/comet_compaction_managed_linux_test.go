//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func TestManagedCometCompactionAndExplicitPinnedResume(t *testing.T) {
	for _, kind := range []string{"pack", "meld"} {
		t.Run(kind, func(t *testing.T) {
			f, state, persist := managedCometFixture(t)
			ctx := context.Background()
			var err error
			for _, action := range []string{"start", "stop"} {
				state, _, err = applyManaged(ctx, action, state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(managedPier(f, state), ".urb/state"), make([]byte, 4096), 0600); err != nil {
				t.Fatal(err)
			}
			intent := false
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) > 6 && args[4] == "container" && args[5] == "start" {
					for _, c := range f.containers {
						if c.id == args[6] && c.kind == "comet-"+kind && !intent {
							t.Fatal("utility dispatched before durable uncertainty")
						}
					}
				}
				return execute(ctx, args, env)
			}
			state, obs, err := applyManaged(ctx, kind, state, func(s ManagedState) error {
				intent = intent || s.MaintenanceUncertain
				return persist(s)
			}, f.d)
			if err != nil || !intent || state.UncertainMaintenance() || obs.State != "stopped" || obs.RecoveryRequired || len(f.containers) != 0 {
				t.Fatalf("comet utility %+v %v", obs, err)
			}
			m := state.Maintenance
			if m == nil || m.Kind != kind || m.FinishedAt == nil || m.BeforePierBytes < 4096 || m.AfterPierBytes < 4096 || m.MemAvailableBytes < 5<<30 || state.Comet.Ship != fixtureComet {
				t.Fatal("wrong comet measurement or identity", m)
			}
			found := false
			for _, args := range f.calls {
				call := strings.Join(args, " ")
				if strings.Contains(call, "container create ") && strings.Contains(call, "--name urbit-managed-comet-"+state.ID+"-comet-"+kind+" ") {
					found = true
					if !strings.Contains(call, "--network none") || !strings.HasSuffix(call, kind+" --loom 31 /urbit/pier") || strings.Contains(call, "/artifacts/") {
						t.Fatal("wrong managed utility containment", call)
					}
				}
			}
			if !found {
				t.Fatal("comet utility not dispatched")
			}
			state, obs, err = observeManaged(ctx, state, f.d)
			if err != nil || obs.State != "stopped" || obs.RecoveryRequired {
				t.Fatal("removed utility state was not reopenable", err)
			}
			state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
			if err != nil || obs.Health != "healthy" || state.Comet.Ship != fixtureComet || f.downloads != 1 {
				t.Fatal("post-utility identity resume", err)
			}
		})
	}
}

func TestManagedCometRoutineConsentAndFailures(t *testing.T) {
	for _, fault := range []string{"pack", "meld", "leave-stopped", "already-stopped", "memory", "consent", "utility", "identity", "provenance"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedCometFixture(t)
			ctx := context.Background()
			state, _, err := applyManaged(ctx, "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			if fault == "already-stopped" {
				state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			kind := "pack"
			if fault == "meld" {
				kind = "meld"
			}
			state.Routine = &routines.Run{Action: kind, StopRunning: true, Resume: fault != "leave-stopped", WasRunning: fault != "already-stopped", Deadline: time.Now().Add(time.Minute)}
			before := len(f.events)
			switch fault {
			case "memory":
				f.d.admission = func(string) (uint64, uint64, error) { return 0, 8 << 30, nil }
			case "consent":
				state.Routine.StopRunning = false
			case "provenance":
				state.Comet.FinishedAt = nil
			case "utility":
				execute := f.d.execute
				f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					if strings.Contains(strings.Join(args, " "), "container wait ") {
						return "", errors.New("utility interrupted")
					}
					return execute(ctx, args, env)
				}
			case "identity":
				read := f.d.hostIdentity
				f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
					for _, event := range f.events[before:] {
						if event == "start-comet-resume" {
							return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
						}
					}
					return read(ctx, endpoint)
				}
			}
			state, obs, err := applyManaged(ctx, "routine-"+kind, state, persist, f.d)
			if fault == "pack" || fault == "meld" || fault == "leave-stopped" || fault == "already-stopped" {
				want := "running"
				if fault == "leave-stopped" || fault == "already-stopped" {
					want = "stopped"
				}
				if err != nil || obs.State != want || !state.Routine.Succeeded || state.Routine.FinishedAt == nil || state.Comet.Ship != fixtureComet || f.downloads != 1 {
					t.Fatalf("routine completion %+v %v", obs, err)
				}
				return
			}
			if err == nil || state.Routine.Succeeded {
				t.Fatal("unsafe routine succeeded")
			}
			for _, event := range f.events[before:] {
				if fault != "identity" && event == "start-comet-resume" || (fault == "memory" || fault == "consent" || fault == "provenance") && event == "stop-comet-boot" {
					t.Fatal("unsafe lifecycle continuation", event)
				}
			}
			if fault == "utility" || fault == "identity" {
				if fault == "utility" && !state.UncertainMaintenance() || fault == "identity" && !state.UncertainProvisioning() {
					t.Fatal("failed outcome not quarantined")
				}
				state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
				if err != nil || !obs.RecoveryRequired {
					t.Fatal("stop erased quarantine", err)
				}
				if _, _, err := applyManaged(ctx, "start", state, persist, f.d); err == nil {
					t.Fatal("quarantined pier restarted")
				}
			}
		})
	}
}

func TestManagedCometUtilityProfilesCannotCross(t *testing.T) {
	for _, profile := range []string{ProfileHTTP, ProfileComet} {
		q := &qualifier{d: dependencies{managed: true, profile: profile}}
		kinds := []string{"comet-pack", "comet-meld"}
		if profile == ProfileComet {
			kinds = []string{"pack", "meld", "roll", "comet-post-pack", "comet-post-meld"}
		}
		for _, kind := range kinds {
			if _, err := q.create(context.Background(), kind); err == nil || !strings.Contains(err.Error(), "not allowed") {
				t.Fatal("wrong-profile utility passed dispatch admission", profile, kind, err)
			}
		}
	}
}
