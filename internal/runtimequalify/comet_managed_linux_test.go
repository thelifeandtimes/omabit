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

func managedCometFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	f := cometFixture(t)
	f.d.managed = true
	f.d.hostPort = 18087
	if err := os.MkdirAll(filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), 0700); err != nil {
		t.Fatal(err)
	}
	state := ManagedState{ID: strings.Repeat("c", 32), Port: 18087, Comet: &ManagedComet{}, Storage: &pierstore.Location{Path: "livenet/comets/fixture/pier"}}
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

func TestManagedCometCreationAndPinnedResume(t *testing.T) {
	f, state, persist := managedCometFixture(t)
	ctx := context.Background()
	intent := false
	state, obs, err := applyManaged(ctx, "start", state, func(s ManagedState) error {
		if s.Comet.Uncertain && s.Comet.StartedAt != nil && !s.Initialized {
			intent = true
		}
		return persist(s)
	}, f.d)
	if err != nil || !intent || !state.Initialized || state.Comet.Ship != fixtureComet || state.Comet.FinishedAt == nil || state.UncertainProvisioning() || obs.Health != "healthy" {
		t.Fatalf("comet start %+v %v", obs, err)
	}
	for name, c := range f.containers {
		if !strings.HasPrefix(name, "urbit-managed-comet-") || mapAt(mapAt(c.doc, "Config"), "Labels")["org.urbit-runner.managed-comet"] != "true" {
			t.Fatal("wrong comet resource namespace")
		}
	}
	state, obs, err = observeManaged(ctx, state, f.d)
	if err != nil || obs.Health != "healthy" || obs.Resources == nil {
		t.Fatal("comet observation", err)
	}
	for _, kind := range []string{"pack", "meld", "roll", "inspect-epochs", "prune-epochs", "import-fake"} {
		if _, _, err := applyManaged(ctx, kind, state, persist, f.d); err == nil {
			t.Fatal("unqualified comet operation accepted", kind)
		}
	}
	state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil || obs.RecoveryRequired {
		t.Fatal("comet stop", err)
	}
	state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || state.Comet.Ship != fixtureComet || f.downloads != 1 {
		t.Fatal("comet reinitialized", err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind == "comet-boot" {
			t.Fatal("creation container on resume")
		}
	}
	_, _, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInterruptedCometCreationCannotBeLearnedByObservationOrRetried(t *testing.T) {
	f, state, persist := managedCometFixture(t)
	ctx := context.Background()
	read := f.d.hostIdentity
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "", errors.New("probe failed") }
	state, _, err := applyManaged(ctx, "start", state, persist, f.d)
	if err == nil || !state.UncertainProvisioning() || state.Initialized {
		t.Fatal("uncertain bootstrap lost")
	}
	f.d.hostIdentity = read
	state, obs, err := observeManaged(ctx, state, f.d)
	if err != nil || !obs.RecoveryRequired || state.Initialized {
		t.Fatal("observation adopted partial comet")
	}
	state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil || !obs.RecoveryRequired {
		t.Fatal("stop cleared bootstrap quarantine")
	}
	before := len(f.calls)
	if _, _, err := applyManaged(ctx, "start", state, persist, f.d); err == nil || len(f.calls) != before {
		t.Fatal("new identity generation retried")
	}
}

func TestManagedCometIdentityReplacementQuarantines(t *testing.T) {
	f, state, persist := managedCometFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	f.d.hostIdentity = func(context.Context, string) (string, error) {
		return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
	}
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err == nil || !obs.RecoveryRequired || !state.Comet.Uncertain || state.Comet.Ship != fixtureComet {
		t.Fatal("comet replacement trusted")
	}
}
