//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectMemberStopExitAdapterObservesWithoutReplaying(t *testing.T) {
	for _, mode := range []string{"lost-signal-reply", "lost-wait-reply", "completed", "unclean", "still-running", "changed-process", "changed-known-exit", "changed-selection", "caller-alias", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f := memberStopAdapterFixture(t)
			stopDriver := f.open(t)
			initial := f.selection.Stop
			before, err := stopDriver.Inspect(context.Background(), initial)
			if err != nil {
				t.Fatal(err)
			}
			retained, err := initial.Advance(f.selection.Project.Plan, "planned", "stop-dispatched", &before)
			if err != nil {
				t.Fatal(err)
			}
			// Represent a recorded dispatch whose acknowledgement may have been lost.
			// These synthetic engine state changes send no real signal.
			if mode != "still-running" {
				f.exit(stopDriver.target.id, 0, false)
			}
			if mode == "unclean" {
				f.exit(stopDriver.target.id, 137, true)
			}
			if mode == "lost-wait-reply" || mode == "completed" || mode == "unclean" || mode == "changed-known-exit" {
				retained, err = retained.Advance(f.selection.Project.Plan, "stop-dispatched", "stop-signalled", nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			if mode == "completed" || mode == "unclean" || mode == "changed-known-exit" {
				after, _, err := stopDriver.observe(context.Background(), true)
				if err != nil {
					t.Fatal(err)
				}
				retained, err = retained.Advance(f.selection.Project.Plan, "stop-signalled", "stopped", &after)
				if err != nil {
					t.Fatal(err)
				}
			}
			originalPhase := retained.Phase
			f.selection.Stop = retained
			raw, err := openProjectMemberStopExit(context.Background(), f.selection, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			driver := raw.(*projectMemberStopExitEngine)
			driver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			f.signal = func(context.Context, string) (string, error) {
				t.Fatal("read-only recovery signalled")
				return "", errors.New("unexpected signal")
			}
			switch mode {
			case "changed-process":
				f.containers[stopDriver.target.id].State.StartedAt = time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
			case "changed-known-exit":
				*f.containers[stopDriver.target.id].State.ExitCode = 137
			case "changed-selection":
				retained.Storage.Identity.Inode++
			case "caller-alias":
				f.selection.Stop.Before.Runtime.Members[0].Resource.ID = "changed"
				if driver.retained.Before.Runtime.Members[0].Resource.ID == "changed" {
					t.Fatal("caller rewrote retained evidence")
				}
				retained = driver.retained
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			proof, err := projectnetwork.CollectMemberStopExit(ctx, f.selection.Project.Plan, retained, "observe-retained-stop", driver, time.Now)
			pass := mode == "lost-signal-reply" || mode == "lost-wait-reply" || mode == "completed" || mode == "unclean" || mode == "caller-alias"
			if (err == nil) != pass || f.signals != 0 {
				t.Fatal(mode, proof, err)
			}
			if pass {
				if proof.Validate(f.selection.Project.Plan) != nil || proof.Stop.Phase != originalPhase || proof.Clean() != (mode != "unclean") {
					t.Fatal("original outcome changed", proof)
				}
				if _, ok := any(driver).(projectnetwork.MemberStopDriver); ok {
					t.Fatal("recovery driver exposes mutation interface")
				}
			} else if !reflect.DeepEqual(proof, projectnetwork.MemberStopExitEvidence{}) {
				t.Fatal("incomplete proof retained")
			}
		})
	}
}
