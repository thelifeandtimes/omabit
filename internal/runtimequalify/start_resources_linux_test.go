//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
	"testing"
)

func TestResourcePolicyExistingStartAndStop(t *testing.T) {
	for _, comet := range []bool{false, true} {
		fixture := managedFixture
		if comet {
			fixture = managedCometFixture
		}
		f, state, persist := fixture(t)
		ctx := context.Background()
		enough := f.d.admission
		f.d.admission = func(string) (uint64, uint64, error) { return 1 << 30, 2 << 30, nil }
		before := len(f.calls)
		next, _, err := applyManaged(ctx, "start", state, persist, f.d)
		if err == nil || next.Prepared || next.UncertainProvisioning() || len(f.calls) != before {
			t.Fatal("new boot admitted below minimum", err)
		}
		f.d.admission = enough
		for _, action := range []string{"start", "stop"} {
			state, _, err = applyManaged(ctx, action, state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
		}
		f.d.admission = func(string) (uint64, uint64, error) { return 1 << 30, 2 << 30, nil }
		var obs ManagedObservation
		state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
		if err != nil || obs.Health != "healthy" || !state.Initialized {
			t.Fatal("initialized ship did not restart with modest capacity", obs, err)
		}
		f.d.resourcePolicy = func() (resourcepolicy.Policy, error) {
			return resourcepolicy.Policy{}, errors.New("invalid saved policy")
		}
		state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
		if err != nil {
			t.Fatal("invalid policy prevented Stop", err)
		}
		if _, _, err = applyManaged(ctx, "start", state, persist, f.d); err == nil {
			t.Fatal("invalid policy allowed start")
		}
		p := resourcepolicy.Default()
		p.Existing.Minimum.RAMMiB = 2048
		f.d.resourcePolicy = func() (resourcepolicy.Policy, error) { return p, nil }
		if _, _, err = applyManaged(ctx, "start", state, persist, f.d); err == nil {
			t.Fatal("configured minimum ignored")
		}
	}
}
