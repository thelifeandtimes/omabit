//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recordingNativeEffects(f *fakeEngine, id, verb string) int {
	count := 0
	for _, call := range f.calls {
		if len(call) >= 3 && call[0] == "container" && call[1] == verb {
			for _, arg := range call[2:] {
				if arg == id {
					count++
				}
			}
		}
	}
	return count
}

func TestRecordingHookRunsAfterDurableCreateAndBeforeFinalNativeStart(t *testing.T) {
	for _, scenario := range []string{"pass", "comet", "prepare-refused", "nil-guard", "guard-refused", "save-refused", "snapshot-alias"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			if scenario == "comet" {
				f, state, persist = managedCometFixture(t)
			}
			lastSaved := ""
			save := func(s ManagedState) error {
				if scenario == "save-refused" && s.Runtime.Stage == "starting-runtime" {
					return errors.New("inert save refusal")
				}
				if err := persist(s); err != nil {
					return err
				}
				raw, _ := json.Marshal(s)
				lastSaved = string(raw)
				return nil
			}
			prepared, checked, selectedID := 0, 0, ""
			f.d.recording = &ManagedRecordingLifecycle{
				BeforeStart: func(ctx context.Context, snapshot ManagedState, id string) (func(context.Context) error, error) {
					prepared++
					selectedID = id
					raw, _ := json.Marshal(snapshot)
					if _, bounded := ctx.Deadline(); !bounded || snapshot.Runtime.Stage != "starting-runtime" || string(raw) != lastSaved || recordingNativeEffects(f, id, "start") != 0 {
						t.Fatal("hook preceded durable creation or followed Start")
					}
					if scenario == "prepare-refused" {
						return nil, ErrRecordingSubmissionUncertain
					}
					if scenario == "nil-guard" {
						return nil, nil
					}
					if scenario == "snapshot-alias" {
						snapshot.Storage.Path = "fakeships/unrelated/pier"
						for i := range snapshot.Runtime.Containers {
							snapshot.Runtime.Containers[i].Name = "unrelated"
						}
					}
					return func(ctx context.Context) error {
						checked++
						if _, bounded := ctx.Deadline(); !bounded || recordingNativeEffects(f, id, "start") != 0 {
							t.Fatal("fresh guard missing or after dispatch")
						}
						if scenario == "guard-refused" {
							return ErrRecordingWorker
						}
						return nil
					}, nil
				},
				BeforeRemove: func(context.Context, ManagedState, string) error {
					t.Fatal("initial Start crossed runtime removal barrier")
					return nil
				},
			}
			out, observation, err := applyManaged(t.Context(), "start", state, save, f.d)
			pass := scenario == "pass" || scenario == "comet" || scenario == "snapshot-alias"
			if pass {
				if err != nil || observation.State != "running" || prepared != 1 || checked != 1 || recordingNativeEffects(f, selectedID, "start") != 1 || out.Storage.Path == "fakeships/unrelated/pier" {
					t.Fatal("recorded handoff altered launch or lost guard", scenario, err, prepared, checked)
				}
			} else {
				if err == nil || selectedID != "" && recordingNativeEffects(f, selectedID, "start") != 0 {
					t.Fatal("refused recording allowed native Start", scenario, err)
				}
				for _, event := range f.events {
					if strings.HasPrefix(event, "start-") && event != "start-probe" {
						t.Fatal("runtime started before refused save/handoff", event)
					}
				}
				if scenario == "prepare-refused" && !errors.Is(err, ErrRecordingSubmissionUncertain) {
					t.Fatal("submission uncertainty was relabeled", err)
				}
				if scenario == "save-refused" && prepared != 0 {
					t.Fatal("recording preceded durable starting stage")
				}
			}
		})
	}
}

func TestRecordingRemovalBarrierPreservesTerminalContainerAndNeverRepeatsStopSignal(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.d.logs = func(context.Context, string, string) (string, error) { return "inert terminal log", nil }
	selectedID, removals, allow := "", 0, false
	f.d.recording = &ManagedRecordingLifecycle{
		BeforeStart: func(_ context.Context, _ ManagedState, id string) (func(context.Context) error, error) {
			selectedID = id
			return func(context.Context) error { return nil }, nil
		},
		BeforeRemove: func(ctx context.Context, snapshot ManagedState, id string) error {
			removals++
			if _, bounded := ctx.Deadline(); !bounded || id != selectedID || recordingNativeEffects(f, id, "rm") != 0 || snapshot.Runtime.RetainedLogs == nil {
				t.Fatal("removal barrier lost terminal snapshot, deadline or exact container")
			}
			for _, c := range f.containers {
				if c.id == id && mapAt(c.doc, "State")["Running"] != false {
					t.Fatal("removal barrier ran before native stop")
				}
			}
			snapshot.Storage.Path = "fakeships/unrelated/pier"
			if !allow {
				return ErrRecordingWorker
			}
			return nil
		},
	}
	var err error
	state, _, err = applyManaged(t.Context(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(t.Context(), "stop", state, persist, f.d)
	if err == nil || removals != 1 || recordingNativeEffects(f, selectedID, "rm") != 0 || len(f.containers) != 1 {
		t.Fatal("failed barrier removed terminal evidence", removals, err)
	}
	signals := recordingNativeEffects(f, selectedID, "stop") + recordingNativeEffects(f, selectedID, "kill")
	if signals != 1 {
		t.Fatal("expected one graceful stop signal", signals)
	}
	allow = true
	state, observed, err := applyManaged(t.Context(), "stop", state, persist, f.d)
	if err != nil || observed.State != "stopped" || removals != 2 || recordingNativeEffects(f, selectedID, "rm") != 1 || len(f.containers) != 0 || recordingNativeEffects(f, selectedID, "stop")+recordingNativeEffects(f, selectedID, "kill") != signals {
		t.Fatal("stopped continuation repeated signal or lost exact removal", err)
	}
	if _, err := os.Stat(filepath.Join(managedPier(f, state), ".urb")); err != nil || strings.Contains(state.Storage.Path, "unrelated") {
		t.Fatal("barrier changed pier or leaked snapshot edits", err)
	}
}

func TestRecordingLifecycleRequiresBothCallbacksBeforeAnyNativeCall(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.d.recording = &ManagedRecordingLifecycle{BeforeStart: func(context.Context, ManagedState, string) (func(context.Context) error, error) {
		t.Fatal("partial lifecycle invoked")
		return nil, nil
	}}
	if _, _, err := applyManaged(t.Context(), "start", state, persist, f.d); err == nil || len(f.calls) != 0 {
		t.Fatal("incomplete recording lifecycle allowed native preparation", err)
	}
}
