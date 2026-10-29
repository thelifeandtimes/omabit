//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func recoveredLifecycleFixture(t *testing.T, target string, stoppedPeer bool) (*memberStopFixture, ProjectCurrentMemberStopSelection) {
	t.Helper()
	completed, h, _ := materializeOrdinarySuccessorFixture(t)
	h = h.Clone()
	f := completed.f.x.f
	project := completed.selection.Source.Project
	ctx := context.Background()
	f.f.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) {
		for _, m := range h.Plan.Members {
			if endpoint == fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) {
				return m.Ship, nil
			}
		}
		t.Fatal("unselected recovered identity read")
		return "", nil
	}
	selected := map[string]ProjectCurrentMemberStopSelection{}
	for _, instance := range []string{"receiver", "sender"} {
		source := ProjectCurrentReadinessSelection{RecoveryHistory: &h, Project: project, Launches: h.Launches, Resumes: h.Resumes, Original: f.readiness[instance]}
		raw, err := openCurrentProjectReadiness(ctx, source, f.f.d)
		if err != nil {
			t.Fatal(err)
		}
		e := raw.(*projectCurrentReadinessEngine)
		e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
		ready, err := projectnetwork.CollectRecoveredCurrentMemberReadiness(ctx, h, instance, e, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		stop, err := projectnetwork.NewRecoveredCurrentMemberStop(h, ready, "recovered-ready-"+instance, "recovered-stop-"+instance)
		if err != nil {
			t.Fatal(err)
		}
		selected[instance] = ProjectCurrentMemberStopSelection{ReadinessSource: source, Readiness: ready, Stop: stop}
	}
	result := selected[target]
	if stoppedPeer {
		peer := "sender"
		if peer == target {
			peer = "receiver"
		}
		input := selected[peer]
		raw, err := openCurrentProjectMemberStop(ctx, input, f.f.d)
		if err != nil {
			t.Fatal(err)
		}
		e := raw.(*projectMemberStopEngine)
		e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
		stopped, err := projectnetwork.ExecuteMemberStop(ctx, h.Plan, input.Stop, e, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		result.StoppedPeers = []projectnetwork.MemberStop{stopped}
	}
	return f, result
}
func TestRecoveredCurrentContextLifecycle(t *testing.T) {
	// Reuse the ordinary lifecycle's independent command/phase, pier-byte, peer,
	// lost-Stop-reply, stopped-peer and resumed-identity assertions after recovery.
	testCurrentContextLifecycle(t, recoveredLifecycleFixture)
}
func TestRecoveredCurrentContextSourceAndRetiredGuard(t *testing.T) {
	for _, mode := range []string{"retired-before-stop", "changed-original", "changed-history", "caller-alias"} {
		t.Run(mode, func(t *testing.T) {
			f, input := recoveredLifecycleFixture(t, "sender", false)
			ctx := context.Background()
			source := input.ReadinessSource
			if mode == "changed-original" {
				source.Original.InstanceID = "receiver"
			}
			if mode == "changed-history" {
				h := source.RecoveryHistory.Clone()
				h.Epochs[0].Readiness.Members[0].Ship = "~bus"
				source.RecoveryHistory = &h
			}
			if mode == "changed-original" || mode == "changed-history" {
				if _, err := openCurrentProjectReadiness(ctx, source, f.f.d); err == nil {
					t.Fatal("mismatched recovered source opened")
				}
				return
			}
			raw, err := openCurrentProjectMemberStop(ctx, input, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectMemberStopEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			h := source.RecoveryHistory.Clone()
			signals := f.signals
			saved := input.Stop
			if mode == "caller-alias" {
				source.RecoveryHistory.Epochs[0].Readiness.Replacement.Members[0].Creation.Storage.Path = "changed"
			}
			got, err := projectnetwork.ExecuteMemberStop(ctx, h.Plan, saved, e, func(phase string, next projectnetwork.MemberStop) error {
				if phase != saved.Phase {
					t.Fatal("stale Stop commit")
				}
				saved = next
				if mode == "retired-before-stop" && next.Phase == "stop-dispatched" {
					v := projectMemberContainerFixture(t, e.target.contract)
					v.ID = h.Launches[0].Container.ID
					v.HostConfig.NetworkMode = "none"
					v.Mounts = nil
					v.Name = "/retired-elsewhere"
					f.containers[v.ID] = &v
				}
				return nil
			}, time.Now)
			if !reflect.DeepEqual(got, saved) {
				t.Fatal("Stop receipt lost")
			}
			if mode == "caller-alias" {
				if err != nil || !got.Clean() || f.signals != signals+1 {
					t.Fatal("caller altered retained source", err)
				}
			} else if err == nil || f.signals != signals || got.Phase != "stop-dispatched" {
				t.Fatal("historical ID did not fence ordinary Stop", err)
			}
		})
	}
}
