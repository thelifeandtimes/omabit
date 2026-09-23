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

func managedFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	f := httpFixture(t)
	f.d.managed = true
	f.d.hostPort = 18080
	base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	state := ManagedState{ID: strings.Repeat("a", 32), Port: 18080}
	persist := func(s ManagedState) error {
		// Fake Docker checks persisted intent using report.json. The real driver
		// callback writes SQLite, including before the runtime directory exists.
		if _, err := os.Stat(s.Runtime.Directory); errors.Is(err, os.ErrNotExist) {
			return nil
		}
		data, err := json.Marshal(s.Runtime)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(s.Runtime.Directory, "report.json"), data, 0600)
	}
	return f, state, persist
}

func TestManagedFakePersistsAcrossDriverSessions(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || !state.Initialized || obs.State != "running" || obs.Endpoint != "http://127.0.0.1:18080" || len(f.containers) != 1 || len(f.networks) != 1 {
		t.Fatalf("start: %+v %+v %v", state, obs, err)
	}
	for name, c := range f.containers {
		if !strings.HasPrefix(name, "urbit-managed-fake-") || mapAt(mapAt(c.doc, "Config"), "Labels")[label] != nil {
			t.Fatal("managed resources leaked into qualification namespace")
		}
	}
	state, obs, err = observeManaged(context.Background(), state, f.d)
	if err != nil || obs.Health != "healthy" || !state.Initialized {
		t.Fatalf("reopen observation: %+v %v", obs, err)
	}
	state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || obs.State != "stopped" || len(f.containers) != 0 || len(f.networks) != 1 {
		t.Fatalf("stop: %+v %v", obs, err)
	}
	state.Port = 18081
	f.d.hostPort = 18081
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Endpoint != "http://127.0.0.1:18081" || f.downloads != 1 {
		t.Fatalf("restart: %+v %v downloads %d", obs, err, f.downloads)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(state.Runtime.Directory, "data/fake-zod/.urb")); err != nil {
		t.Fatal("stop removed the pier")
	}
}

func TestManagedIntentPersistenceFailurePreventsDockerMutation(t *testing.T) {
	f, state, _ := managedFixture(t)
	_, _, err := applyManaged(context.Background(), "start", state, func(ManagedState) error { return errors.New("journal failed") }, f.d)
	if err == nil || len(f.calls) != 0 || len(f.containers) != 0 || len(f.networks) != 0 {
		t.Fatalf("failed journal allowed mutation: %v", err)
	}
}

func TestManagedInitialBootIsNeverRepeatedOverPartialPier(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.d.hostProbe = func(context.Context, string) error { return errors.New("identity mismatch") }
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err == nil || state.Initialized || len(f.containers) != 1 {
		t.Fatal("expected a retained failed initial boot")
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil || len(f.containers) != 0 {
		t.Fatalf("explicit stop: %v", err)
	}
	before := len(f.events)
	_, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err == nil || !strings.Contains(err.Error(), "partial fake pier") {
		t.Fatalf("partial bootstrap replayed: %v", err)
	}
	for _, event := range f.events[before:] {
		if strings.HasPrefix(event, "create-") {
			t.Fatal("partial pier was bootstrapped again")
		}
	}
}

func TestManagedForeignRuntimeAndPathRejected(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	old := len(f.calls)
	bad := state
	bad.Runtime.Directory = filepath.Dir(state.Runtime.Directory)
	if _, _, err := applyManaged(context.Background(), "stop", bad, persist, f.d); err == nil || len(f.calls) != old {
		t.Fatal("foreign directory reached Docker")
	}
	f.editInspect = func(_ string, doc map[string]any) { mapAt(mapAt(doc, "Config"), "Labels")[f.runLabelName()] = "other" }
	if _, _, err := applyManaged(context.Background(), "stop", state, persist, f.d); err == nil {
		t.Fatal("foreign container stopped")
	}
	if len(f.containers) != 1 {
		t.Fatal("foreign container removed")
	}
}
