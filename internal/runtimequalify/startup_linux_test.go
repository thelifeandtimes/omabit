//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestStartupResumeReplacesCleanStoppedContainersWithoutBootstrap(t *testing.T) {
	testStartupReplacement(t, 0, false)
}

func TestStartupResumeReplaysInitializedPiersAfterUncleanExit(t *testing.T) {
	for _, exit := range []int{255, 137} {
		testStartupReplacement(t, exit, false)
	}
	testStartupReplacement(t, 137, true)
}

func testStartupReplacement(t *testing.T, exit int, oom bool) {
	t.Helper()
	for _, comet := range []bool{false, true} {
		makeFixture := managedFixture
		if comet {
			makeFixture = managedCometFixture
		}
		f, state, persist := makeFixture(t)
		state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
		ship := ""
		if comet {
			ship = state.Comet.Ship
		}
		for _, c := range f.containers {
			mapAt(c.doc, "State")["Running"] = false
			mapAt(c.doc, "State")["ExitCode"] = exit
			mapAt(c.doc, "State")["OOMKilled"] = oom
		}
		if exit != 0 {
			state.StartupResume = &StartupResumeIntent{ContainerID: state.Runtime.Containers[len(state.Runtime.Containers)-1].ID, AllowUnclean: true}
		}
		before := len(f.calls)
		state, obs, err := applyManaged(context.Background(), "startup-resume", state, persist, f.d)
		if err != nil || obs.Health != "healthy" || !state.Initialized || f.downloads != 1 || len(f.containers) != 1 || state.StartupResume != nil {
			t.Fatal("startup resume", obs, err)
		}
		if comet && state.Comet.Ship != ship {
			t.Fatal("comet identity changed")
		}
		for _, call := range f.calls[before:] {
			if len(call) > 1 && call[0] == "container" && call[1] == "create" {
				joined := " " + strings.Join(call, " ") + " "
				for _, forbidden := range []string{" -c ", " -F ", " -B ", "/artifacts/boot.pill"} {
					if strings.Contains(joined, forbidden) {
						t.Fatal("startup attempted bootstrap", call)
					}
				}
			}
		}
		for _, c := range state.Runtime.Containers {
			if !c.Removed && (c.Kind == "http-boot" || c.Kind == "comet-boot") {
				t.Fatal("old creation container reused")
			}
		}
	}
}

func TestStartupResumeRequiresFreshCompleteEvidence(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.containers {
		mapAt(c.doc, "State")["Running"] = false
	}
	state, observation, err := observeManaged(context.Background(), state, f.d)
	if err != nil || !StartupResumeReady(state, observation) {
		t.Fatal("clean fixture not eligible", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing-state", "stale", "future", "paused", "restarting", "dead", "network", "identity", "multiple", "wrong-container"} {
		var changed ManagedState
		if err := json.Unmarshal(data, &changed); err != nil {
			t.Fatal(err)
		}
		c := &changed.Runtime.Containers[len(changed.Runtime.Containers)-1]
		changed.StartupResume = &StartupResumeIntent{ContainerID: c.ID, AllowUnclean: true}
		switch fault {
		case "wrong-container":
			changed.StartupResume.ContainerID = strings.Repeat("f", 64)
		case "missing-state":
			c.LastState = nil
		case "stale":
			c.LastState.ObservedAt = time.Now().Add(-time.Minute)
		case "future":
			c.LastState.ObservedAt = time.Now().Add(time.Minute)
		case "paused":
			c.LastState.Paused = true
		case "restarting":
			c.LastState.Restarting = true
		case "dead":
			c.LastState.Dead = true
		case "network":
			changed.Runtime.Network.CreateUncertain = true
		case "identity":
			c.HostHTTPVerified = false
		case "multiple":
			changed.Runtime.Containers = append(changed.Runtime.Containers, *c)
		}
		if StartupResumeReady(changed, observation) {
			t.Fatal("unsafe startup evidence accepted", fault)
		}
	}
}

func TestStartupResumeRefusesUncertainOrUncleanState(t *testing.T) {
	for _, fault := range []string{"running", "exit", "oom", "missing", "foreign", "uncertain", "uninitialized", "maintenance", "memory", "persistence"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			if fault != "exit" && fault != "oom" {
				state.StartupResume = &StartupResumeIntent{ContainerID: state.Runtime.Containers[len(state.Runtime.Containers)-1].ID, AllowUnclean: true}
			}
			for _, c := range f.containers {
				mapAt(c.doc, "State")["Running"] = fault == "running"
				if fault == "exit" {
					mapAt(c.doc, "State")["ExitCode"] = 137
				}
				if fault == "oom" {
					mapAt(c.doc, "State")["OOMKilled"] = true
				}
			}
			switch fault {
			case "missing":
				clear(f.containers)
			case "foreign":
				f.editInspect = func(_ string, doc map[string]any) { mapAt(mapAt(doc, "Config"), "Labels")[f.runLabelName()] = "other" }
			case "uncertain":
				state.Runtime.Containers[len(state.Runtime.Containers)-1].CreateUncertain = true
			case "uninitialized":
				state.Initialized = false
			case "maintenance":
				state.MaintenanceUncertain = true
			case "memory":
				f.d.admission = func(string) (uint64, uint64, error) { return 1, 8 << 30, nil }
			case "persistence":
				persist = func(ManagedState) error { return errors.New("journal failed") }
			}
			before := len(f.events)
			if _, _, err := applyManaged(context.Background(), "startup-resume", state, persist, f.d); err == nil {
				t.Fatal("unsafe startup admitted", fault)
			}
			for _, event := range f.events[before:] {
				for _, prefix := range []string{"create-", "start-", "stop-", "rm-"} {
					if strings.HasPrefix(event, prefix) {
						t.Fatal("mutated refused runtime", fault, event)
					}
				}
			}
		})
	}
}

func TestRebootResumeRechecksContainerBeforeRemoval(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.containers {
		mapAt(c.doc, "State")["Running"] = false
		mapAt(c.doc, "State")["ExitCode"] = 255
	}
	state.StartupResume = &StartupResumeIntent{ContainerID: state.Runtime.Containers[len(state.Runtime.Containers)-1].ID, AllowUnclean: true}
	inspections := 0
	f.editInspect = func(_ string, doc map[string]any) {
		inspections++
		if inspections > 1 {
			mapAt(doc, "State")["Running"] = true
		}
	}
	before := len(f.events)
	if _, _, err := applyManaged(context.Background(), "startup-resume", state, persist, f.d); err == nil {
		t.Fatal("running container accepted for reboot removal")
	}
	for _, event := range f.events[before:] {
		for _, prefix := range []string{"create-", "start-", "stop-", "rm-"} {
			if strings.HasPrefix(event, prefix) {
				t.Fatal("mutated changed container", event)
			}
		}
	}
}
