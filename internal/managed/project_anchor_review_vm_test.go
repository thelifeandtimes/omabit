//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestProjectAnchorReviewDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, true)
}

// Fault injection is confined to the exact anchor created in this guarded fresh
// guest. The production observer is read-only; recovery mutations are not yet
// wired. Final members stop through existing exact-ID fixture cleanup.
func exerciseVMAnchorRecoveryReview(t *testing.T, ctx context.Context, r *Server, p storedProject, members []map[string]any, latest, sender map[string]any, docker func(...string) json.RawMessage) {
	t.Helper()
	result := map[string]any{"ok": false, "stage": "selection", "fixture_fault_injection": true, "production_mutations": false}
	sender["anchor_recovery_review"] = result
	history := currentStopHistory(t, r.s)
	j, err := loadProjectJob(r.s.db, p.ID)
	if err != nil || j.Preparation == nil {
		t.Error("review fixture preparation missing", err)
		return
	}
	selection := runtimequalify.ProjectAnchorRecoveryReviewSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, OperationID: j.OperationID, Revision: p.Revision, Generation: j.Generation, Plan: p.Plan}}
	for _, m := range members {
		launch := m["journal"].(projectMemberLaunchJob)
		selection.Launches = append(selection.Launches, *launch.Launch)
		original := m["readiness"].(map[string]any)["journal"].(projectMemberReadinessJob)
		selection.Originals = append(selection.Originals, *original.Evidence)
	}
	selection.Resumes, err = completedProjectMemberResumes(r.s.db, p.ID)
	if err != nil {
		t.Error(err)
		return
	}
	for _, row := range latest["current_context_readiness"].([]map[string]any) {
		ready := row["journal"].(projectMemberContextReadinessJob)
		selection.Current = append(selection.Current, *ready.Evidence)
	}
	c, err := projectnetwork.SelectAnchorRecoveryContext(p.Plan, selection.Launches, selection.Resumes, selection.Originals, selection.Current)
	if err != nil {
		t.Error(err)
		return
	}
	result["selection"] = selection
	preflight, err := runtimequalify.OpenProjectAnchorRecoveryReview(ctx, selection)
	if err != nil || preflight.VerifyEngine(ctx, c.Binding.EngineID) != nil {
		t.Error("anchor review preflight engine refused", err)
		return
	}
	healthy, err := preflight.InspectAnchorRecovery(ctx, c)
	if err != nil {
		t.Error("anchor review preflight observation refused", err)
		return
	}
	assessment, err := projectnetwork.AssessRuntime(p.Plan, c.Binding, healthy.Runtime, time.Now())
	if err != nil || assessment.RecoveryRequired || assessment.State != "ready" || len(assessment.Running) != len(p.Members) || preflight.VerifyEngine(ctx, c.Binding.EngineID) != nil {
		t.Error("anchor fault requires a freshly verified healthy project", err)
		return
	}
	result["healthy_observation"] = healthy
	collect := func(label string) (projectnetwork.AnchorRecoveryReview, error) {
		attempt := map[string]any{}
		result[label+"_collection"] = attempt
		driver, err := runtimequalify.OpenProjectAnchorRecoveryReview(ctx, selection)
		if err != nil {
			attempt["open_error"] = vmLaunchError(err)
			return projectnetwork.AnchorRecoveryReview{}, err
		}
		observed := &vmAnchorReviewObserver{driver: driver}
		review, err := projectnetwork.CollectAnchorRecoveryReview(ctx, p.Plan, selection.Launches, selection.Resumes, selection.Originals, selection.Current, observed, time.Now)
		attempt["events"] = observed.events
		attempt["observations"] = observed.observations
		if err != nil {
			attempt["collection_error"] = vmLaunchError(err)
		}
		return review, err
	}
	if _, err = collect("healthy"); err == nil {
		t.Error("healthy anchor accepted as recovery review")
		return
	}
	result["healthy_refused"] = true
	id := c.Binding.Anchor.ID
	result["before_fault"] = docker("container", "inspect", id)
	result["stage"] = "stopped-anchor"
	if strings.TrimSpace(string(docker("container", "stop", "--timeout", "10", id))) != id {
		t.Error("fixture anchor stop acknowledgement changed")
		return
	}
	result["stopped_inspection"] = docker("container", "inspect", id)
	stopped, err := collect("stopped")
	if err != nil || stopped.Assessment.State != "anchor-unavailable" {
		if err != nil {
			result["diagnostic"] = vmLaunchError(err)
		}
		t.Error("stopped anchor review failed", err)
		return
	}
	result["stopped_review"] = stopped
	assertContextHistory(t, r.s, history)
	if t.Failed() {
		return
	}
	result["stage"] = "restarted-anchor"
	if strings.TrimSpace(string(docker("container", "start", id))) != id {
		t.Error("fixture anchor start acknowledgement changed")
		return
	}
	result["restarted_inspection"] = docker("container", "inspect", id)
	restarted, err := collect("restarted")
	if err != nil || restarted.Assessment.State != "namespace-changed" || restarted.After.Runtime.Anchor.Namespace == c.Binding.Namespace {
		if err != nil {
			result["diagnostic"] = vmLaunchError(err)
		}
		t.Error("restarted anchor review failed", err)
		return
	}
	if !reflect.DeepEqual(stopped.Context, restarted.Context) || !reflect.DeepEqual(stopped.After.Processes, restarted.After.Processes) {
		t.Error("anchor fault changed member process evidence")
		return
	}
	result["restarted_review"] = restarted
	assertContextHistory(t, r.s, history)
	if t.Failed() {
		return
	}
	result["original_history_unchanged"] = true
	result["stage"], result["ok"] = "complete", true
}

// Retain underlying read-only adapter diagnostics even if the collector returns
// its deliberately generic refusal. This adds no engine call or mutation.
type vmAnchorReviewObserver struct {
	driver       projectnetwork.AnchorRecoveryObserver
	events       []vmResumedReadinessEvent
	observations []projectnetwork.AnchorRecoveryObservation
}

func (d *vmAnchorReviewObserver) record(action string, start time.Time, err error) {
	e := vmResumedReadinessEvent{Action: action, StartedAt: start, FinishedAt: time.Now().UTC()}
	if err != nil {
		e.Error = vmLaunchError(err)
	}
	d.events = append(d.events, e)
}
func (d *vmAnchorReviewObserver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now().UTC()
	defer func() { d.record("engine", start, err) }()
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmAnchorReviewObserver) InspectAnchorRecovery(ctx context.Context, c projectnetwork.AnchorRecoveryContext) (o projectnetwork.AnchorRecoveryObservation, err error) {
	start := time.Now().UTC()
	defer func() { d.record("project", start, err) }()
	o, err = d.driver.InspectAnchorRecovery(ctx, c)
	if err == nil {
		d.observations = append(d.observations, o)
	}
	return o, err
}
