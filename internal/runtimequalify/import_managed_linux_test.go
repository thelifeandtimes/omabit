//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func qualifiedImportFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f := importFixture(t)
	result, err := run(context.Background(), nil, f.d)
	if err != nil || !result.CheckpointPassed {
		t.Fatal("source qualification", err)
	}
	sources, err := listImportSources(context.Background(), f.d)
	if err != nil || len(sources) != 1 {
		t.Fatal("catalogue", err)
	}
	state := ManagedState{ID: strings.Repeat("b", 32), Port: 18086, Import: &ManagedImport{Source: sources[0], Uncertain: true}, Storage: &pierstore.Location{Path: "fakeships/import-fixture/pier"}}
	persist := func(s ManagedState) error {
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

func TestManagedImportCopiesWithoutBootAndStartsAsInitialized(t *testing.T) {
	f, state, persist := qualifiedImportFixture(t)
	before := len(f.events)
	intent := false
	state, obs, err := applyManaged(context.Background(), "import-fake", state, func(s ManagedState) error {
		if s.Prepared && !s.Initialized && s.Import.Uncertain && s.Import.StartedAt != nil {
			intent = true
		}
		return persist(s)
	}, f.d)
	if err != nil || !intent || !state.Initialized || state.UncertainProvisioning() || obs.State != "stopped" || state.Import.CopiedFiles != 1 {
		t.Fatalf("import: %+v %v", obs, err)
	}
	for _, e := range f.events[before:] {
		if strings.HasPrefix(e, "create-") || strings.HasPrefix(e, "start-") {
			t.Fatal("import booted a runtime")
		}
	}
	if _, err := os.Stat(state.Runtime.PillPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("import created a bootstrap pill")
	}
	f.d.profile = ProfileHTTP
	f.d.managed = true
	f.d.hostPort = state.Port
	f.d.hostProbe = func(context.Context, string) error { return nil }
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || obs.Health != "healthy" || f.downloads != 1 {
		t.Fatalf("initialized start: %+v %v", obs, err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-boot" {
			t.Fatal("imported state was bootstrapped again")
		}
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if state.UncertainProvisioning() {
		t.Fatal("complete import was quarantined")
	}
}

func TestManagedImportRejectsChangedSourceAndRetainsPartialData(t *testing.T) {
	for _, kind := range []string{"changed-source", "digest", "journal", "partial", "destination"} {
		t.Run(kind, func(t *testing.T) {
			f, state, persist := qualifiedImportFixture(t)
			dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed", state.ID)
			switch kind {
			case "changed-source":
				if err := os.WriteFile(filepath.Join(f.d.home, ".local/share/urbit-runner/qualification", state.Import.Source.RunID, "source/fake-zod/.urb/state"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "digest":
				state.Import.Source.ContentDigest = strings.Repeat("c", 64)
			case "destination":
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			state, _, err := applyManaged(context.Background(), "import-fake", state, func(s ManagedState) error {
				if kind == "journal" || kind == "partial" && s.Import.CopiedFiles > 0 {
					return errors.New("journal unavailable")
				}
				return persist(s)
			}, f.d)
			if err == nil || !state.UncertainProvisioning() {
				t.Fatal("invalid import admitted")
			}
			if kind == "partial" {
				if _, err := os.Stat(filepath.Join(managedPier(f, state), ".urb/state")); err != nil {
					t.Fatal("partial data removed")
				}
			}
			if _, _, err := applyManaged(context.Background(), "start", state, persist, f.d); err == nil {
				t.Fatal("partial import bootstrapped")
			}
			_, obs, err := observeManaged(context.Background(), state, f.d)
			if err != nil || !obs.RecoveryRequired {
				t.Fatal("observation cleared import quarantine")
			}
		})
	}
}
