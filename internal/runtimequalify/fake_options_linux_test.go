//go:build linux && amd64

package runtimequalify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func TestNamedFakeLifecycleMaintenanceAndEpochPaths(t *testing.T) {
	f, state, persist := managedFixture(t)
	state.Fake = &ManagedFake{Ship: "~bus"}
	state.Storage.Path = "fakeships/bus-data/pier"
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "~bus", nil }
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal("custom fake boot", obs, err)
	}
	for _, c := range f.containers {
		args := mapAt(c.doc, "Config")["Cmd"].([]string)
		joined := " " + strings.Join(args, " ") + " "
		if !strings.Contains(joined, " -F bus ") || args[len(args)-1] != "/urbit/pier" {
			t.Fatal("identity/path not applied", args)
		}
	}
	if _, err := os.Stat(filepath.Join(managedPier(f, state), ".urb")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(state.Runtime.Directory, "data/fake-zod")); !os.IsNotExist(err) {
		t.Fatal("default pier unexpectedly created")
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	writePierEpochPath(t, managedPier(f, state), "0i0", false)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err == nil && a[4] == "container" && a[5] == "create" {
			for _, c := range f.containers {
				if c.id == strings.TrimSpace(out) && c.kind != "probe" {
					args := mapAt(c.doc, "Config")["Cmd"].([]string)
					if args[len(args)-1] != "/urbit/pier" {
						t.Fatal("utility/resume used a different pier", args)
					}
					for _, arg := range args {
						if arg == "-F" || arg == "-c" || arg == "-B" {
							t.Fatal("initialized pier received creation flags", args)
						}
					}
				}
			}
		}
		if err == nil && a[4] == "container" && a[5] == "wait" {
			for _, c := range f.containers {
				if c.id == a[6] && c.kind == "roll" {
					writePierEpochPath(t, managedPier(f, state), "0i100", true)
				}
			}
		}
		return out, err
	}
	for _, action := range []string{"pack", "meld", "roll", "inspect-epochs", "start"} {
		state, obs, err = applyManaged(context.Background(), action, state, persist, f.d)
		if err != nil {
			t.Fatal(action, err)
		}
	}
	if obs.Health != "healthy" || f.downloads != 1 {
		t.Fatal("custom resume", obs)
	}
	for _, c := range f.containers {
		args := mapAt(c.doc, "Config")["Cmd"].([]string)
		for _, a := range args {
			if a == "-F" || a == "-c" {
				t.Fatal("resume reinitialized fake", args)
			}
		}
		if args[len(args)-1] != "/urbit/pier" {
			t.Fatal(args)
		}
	}
	state.Routine = &routines.Run{Action: "pack", StopRunning: true, Resume: true, WasRunning: true, Deadline: time.Now().Add(time.Minute)}
	state, obs, err = applyManaged(context.Background(), "routine-pack", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || !state.Routine.Succeeded {
		t.Fatal("named fake routine", obs, err)
	}
	for _, c := range f.containers {
		mapAt(c.doc, "State")["Running"] = false
	}
	state, obs, err = applyManaged(context.Background(), "startup-resume", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || f.downloads != 1 {
		t.Fatal("named fake startup resume", obs, err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0i200", "0i300"} {
		writePierEpochPath(t, managedPier(f, state), name, true)
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	before := state.Epochs
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: before.Digest, Confirmed: true}
	state, obs, err = applyManaged(context.Background(), "prune-epochs", state, persist, f.d)
	if err != nil || obs.State != "stopped" || state.UncertainMaintenance() || !sameRetainedPrefix(before, state.Epochs, 2) {
		t.Fatal("named fake prefix pruning", obs, err)
	}
	entries, err := os.ReadDir(filepath.Join(managedPier(f, state), ".urb/log"))
	if err != nil || len(entries) != 2 || entries[0].Name() != "0i200" || entries[1].Name() != "0i300" {
		t.Fatal("named fake retained suffix", entries, err)
	}
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || state.Fake.Ship != "~bus" || state.Storage.Path != "fakeships/bus-data/pier" {
		t.Fatal("named fake post-prune resume", obs, err)
	}
}

func TestNamedFakeIdentityMismatchAndUnsafeDirectoryAreRefused(t *testing.T) {
	f, state, persist := managedFixture(t)
	state.Fake = &ManagedFake{Ship: "~bus"}
	state.Storage.Path = "fakeships/../outside/pier"
	if _, _, err := applyManaged(context.Background(), "start", state, persist, f.d); err == nil || len(f.calls) != 0 {
		t.Fatal("unsafe pier dispatched")
	}
	state.Storage.Path = "fakeships/bus-data/pier"
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "~bus", nil }
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "~zod", nil }
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || !state.UncertainProvisioning() || !obs.RecoveryRequired {
		t.Fatal("changed fake identity accepted", obs, err)
	}
	state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || !state.UncertainProvisioning() || !obs.RecoveryRequired {
		t.Fatal("stop erased identity uncertainty", obs, err)
	}
}
