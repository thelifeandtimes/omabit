package projectdiagnostic

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestShareableBundleCannotForwardPrivateOrUntrustedText(t *testing.T) {
	const private = "PRIVATE-SENTINEL authorization=credential /home/person/pier personal.example"
	d := protocol.ProjectMemberDiagnosis{RunnerID: private, Mode: private, Actor: private, ProjectID: private, InstanceID: private, Operation: protocol.Operation{ID: private, RequestID: private, Actor: private, Kind: private, Status: private, Message: private}, Phase: private, Fence: private, ReasonCode: private, Message: private, RuntimeObservation: private, ResolvedBy: private, Review: &protocol.ProjectStopReview{Actor: private, Observation: protocol.ProjectStopObservation{ContainerID: private}}}
	b := Derive(d)
	raw, err := json.Marshal(b)
	if err != nil || bytes.Contains(raw, []byte("PRIVATE-SENTINEL")) || bytes.Contains(raw, []byte("credential")) || b.ReasonCode != "diagnosis_unavailable" || b.ActionReview || !b.PrivateBindingsOmitted || len(b.Actions) != 0 || len(raw) > 4096 {
		t.Fatal("private/unsupported diagnosis escaped bounded view", string(raw), err)
	}
}

func TestSavedAndFreshEvidenceRemainDistinctAndDoNotAuthorizeEffects(t *testing.T) {
	now := time.Now().UTC()
	d := protocol.ProjectMemberDiagnosis{Operation: protocol.Operation{Kind: "stop-project-ship", Status: "interrupted", CreatedAt: now}, ReadAt: now, Phase: "stop-dispatched", Fence: "original-interrupted-operation", ReasonCode: "stop_needs_observation", RuntimeObservation: "not-performed", Observation: &protocol.ProjectStopObservation{Running: false, ExitCode: 0, ObservedAt: now}}
	b := Derive(d)
	if b.Running != nil || b.CleanExitObserved != nil || len(b.Actions) != 1 || b.ActionReview {
		t.Fatal("saved state became runtime/recovery authority", b)
	}
	d.ReasonCode = "stop_clean_exit_verified"
	if b := Derive(d); b.ReasonCode != "diagnosis_evidence_inconsistent" || b.CleanExitObserved != nil || b.ActionReview {
		t.Fatal("saved assertion became clean-exit evidence", b)
	}
	d.RuntimeObservation = "fresh"
	d.ReasonCode = "stop_exit_abnormal"
	d.Observation.OOMKilled = true
	d.Observation.StartedAt, d.Observation.FinishedAt = now.Add(-time.Minute), now
	b = Derive(d)
	if b.Running == nil || *b.Running || b.CleanExitObserved == nil || *b.CleanExitObserved || len(b.Actions) != 0 || b.ActionReview {
		t.Fatal("abnormal exit offered clean-stop recovery", b)
	}
	d.Fence, d.ReasonCode, d.ResolvedBy = "resolved", "stop_reconciled", "private-resolution"
	b = Derive(d)
	if !b.LinkedResolutionRecorded || len(b.Actions) != 0 || b.ActionReview {
		t.Fatal("resolved receipt became Start authority", b)
	}
}
