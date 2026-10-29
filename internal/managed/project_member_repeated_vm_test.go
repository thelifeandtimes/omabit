//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func exerciseVMRepeatedMemberCycle(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage, registerCleanup func(func())) {
	t.Helper()
	var original map[string]any
	for _, m := range members {
		if m["instance_id"] == "vm-zod" {
			original = m
		}
	}
	if original == nil {
		t.Error("repeated cycle target missing")
		return
	}
	resumed, ok := original["resume"].(map[string]any)
	if !ok || resumed["ok"] != true {
		t.Error("first replacement incomplete")
		return
	}
	stopped, ok := resumed["stop"].(map[string]any)
	if !ok || stopped["ok"] != true {
		t.Error("current Stop incomplete")
		return
	}
	cycle := map[string]any{"instance_id": "vm-zod", "ok": false, "stage": "exit", "stop": stopped}
	original["later_generations"] = []map[string]any{cycle}
	if !exerciseVMMemberStopExit(t, ctx, r, b, p, "vm-zod", stopped) {
		original["ok"] = false
		return
	}
	cycle["stage"] = "retire"
	exerciseVMMemberRetirement(t, ctx, r, b, p, []map[string]any{cycle}, docker)
	if retired, ok := cycle["retirement"].(map[string]any); ok && retired["ok"] == true {
		resumed["retirement"] = retired
	} else {
		original["ok"] = false
		return
	}
	cycle["stage"] = "resume"
	exerciseVMMemberResume(t, ctx, r, b, p, []map[string]any{cycle}, docker, registerCleanup)
	if t.Failed() {
		original["ok"] = false
		return
	}
	j, ok := cycle["resume"].(map[string]any)
	if !ok || j["ok"] != true {
		t.Error("later replacement incomplete")
		original["ok"] = false
		return
	}
	state := j["journal"].(projectMemberResumeJob)
	if state.Resume == nil || state.Resume.PreviousGeneration != 2 {
		t.Error("later replacement did not reach generation3")
		original["ok"] = false
		return
	}
	cycle["stage"], cycle["ok"] = "complete", true
}

func (b *vmMemberLaunchBackend) OpenResumedProjectMemberStopExit(ctx context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	d, err := runtimequalify.OpenResumedProjectMemberStopExit(ctx, s)
	if err != nil {
		b.stopExitErrors = append(b.stopExitErrors, vmLaunchError(err))
		return nil, err
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_stop_exits j JOIN operations o ON j.operation_id=o.id WHERE j.stop_operation_id=? AND json_extract(o.body,'$.status')='queued'", s.Stop.OperationID).Scan(&id); err != nil {
		return nil, err
	}
	b.stopExit = &vmMemberStopExitDriver{store: b.store, driver: d, opID: id, original: s.Stop, peers: s.StoppedPeers}
	return b.stopExit, nil
}
func (b *vmMemberLaunchBackend) OpenResumedProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectResumedMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	initial, d, err := runtimequalify.OpenResumedProjectMemberRetirement(ctx, s)
	if err != nil {
		b.retirementErrors = append(b.retirementErrors, vmLaunchError(err))
		return initial, nil, err
	}
	b.retirement = &vmMemberRetirementDriver{store: b.store, driver: d, opID: s.OperationID, peers: s.Stop.StoppedPeers}
	return initial, b.retirement, nil
}

func vmReopenLaterMemberCycle(t *testing.T, s *store, member map[string]any, anchorReview bool) {
	t.Helper()
	cycles, _ := member["later_generations"].([]map[string]any)
	for _, cycle := range cycles {
		if cycle["ok"] != true {
			t.Fatal("later generation incomplete")
		}
		stop := cycle["stop"].(map[string]any)
		res := cycle["resume"].(map[string]any)
		checks := []struct {
			kind   string
			result map[string]any
		}{
			{"stop", stop}, {"exit", stop["exit_observation"].(map[string]any)},
			{"retire", cycle["retirement"].(map[string]any)}, {"resume", res},
			{"ready", res["readiness"].(map[string]any)},
		}
		if anchorReview && res["anchor_review_cleanup_only"] == true {
			if _, exists := res["stop"]; exists {
				t.Fatal("review-only final generation has Stop")
			}
		} else {
			finalStop, ok := res["stop"].(map[string]any)
			if !ok {
				t.Fatal("later final Stop missing; fixture-only cleanup was not recorded")
			}
			checks = append(checks, struct {
				kind   string
				result map[string]any
			}{"stop", finalStop})
		}
		for _, check := range checks {
			result := check.result
			if result["ok"] != true {
				t.Fatal("later step incomplete", check.kind)
			}
			id := result["operation_id"].(string)
			var actual any
			var err error
			switch check.kind {
			case "stop":
				actual, err = loadProjectMemberStop(s.db, id)
			case "exit":
				actual, err = loadProjectMemberStopExit(s.db, id)
			case "retire":
				actual, err = loadProjectMemberRetirement(s.db, id)
			case "resume":
				actual, err = loadProjectMemberResume(s.db, id)
			case "ready":
				actual, err = loadProjectMemberResumedReadiness(s.db, id)
			default:
				err = errors.New("unknown later step")
			}
			if err != nil || !reflect.DeepEqual(actual, result["journal"]) {
				t.Fatal("reopen changed later journal", check.kind, err)
			}
			op, err := loadOperation(s.db, id)
			if err != nil || !reflect.DeepEqual(op, result["operation"]) {
				t.Fatal("reopen changed later receipt", check.kind, err)
			}
			result["reopen_unchanged"] = true
		}
		cycle["reopen_unchanged"] = true
	}
}
