// Package projectdiagnostic derives bounded shareable views from private typed
// diagnoses. A view is not an action review and cannot authorize recovery.
package projectdiagnostic

import (
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type Action struct {
	Kind       string `json:"kind"`
	Capability string `json:"capability"`
	Role       string `json:"role"`
	Requires   string `json:"requires"`
}

type Bundle struct {
	Version                  int        `json:"version"`
	Scope                    string     `json:"scope"`
	OperationKind            string     `json:"operation_kind"`
	OperationStatus          string     `json:"operation_status"`
	CreatedAt                time.Time  `json:"created_at"`
	FinishedAt               *time.Time `json:"finished_at,omitempty"`
	ReadAt                   time.Time  `json:"read_at"`
	Phase                    string     `json:"phase"`
	Fence                    string     `json:"fence"`
	ReasonCode               string     `json:"reason_code"`
	Message                  string     `json:"message"`
	RuntimeObservation       string     `json:"runtime_observation"`
	RuntimeObservedAt        *time.Time `json:"runtime_observed_at,omitempty"`
	Running                  *bool      `json:"running,omitempty"`
	CleanExitObserved        *bool      `json:"clean_exit_observed,omitempty"`
	ExitCode                 *int       `json:"exit_code,omitempty"`
	OOMKilled                *bool      `json:"oom_killed,omitempty"`
	LinkedResolutionRecorded bool       `json:"linked_resolution_recorded"`
	Actions                  []Action   `json:"actions"`
	PrivateBindingsOmitted   bool       `json:"private_bindings_omitted"`
	ActionReview             bool       `json:"action_review"`
}

var reasons = map[string]string{
	"stop_needs_observation":            "Read exact private bindings and probe the stopped process before reviewing reconciliation.",
	"stop_clean_exit_verified":          "A clean exit was observed; only a separate exact private review can authorize reconciliation.",
	"stop_reconciled":                   "A separate linked receipt resolved the stopped-state fence. Start remains a distinct explicit action.",
	"stop_member_running":               "The selected process is still running. Reconciliation sends no signal.",
	"stop_exit_abnormal":                "An OOM or nonzero exit cannot use clean-stop reconciliation.",
	"stop_recovery_unsupported_phase":   "This retained phase has no clean-stop reconciliation path.",
	"stop_recovery_actor_mismatch":      "The original Stop operator is required for this reconciliation.",
	"stop_recovery_evidence_missing":    "Required original process or pier evidence is missing.",
	"stop_target_changed":               "The retained project or runner binding changed.",
	"stop_original_request_changed":     "The original Stop request no longer matches retained evidence.",
	"stop_process_changed":              "Process timestamps no longer match the original Stop.",
	"stop_recovery_identity_changed":    "Fresh container, pier, engine or namespace ownership differs from retained evidence.",
	"stop_recovery_evidence_changed":    "Retained Stop dispatch evidence changed.",
	"stop_observation_unavailable":      "Exact runtime observation is unavailable; no recovery authority is established.",
	"stop_recovery_adapter_unavailable": "The runner has no supported expanded-member Stop observer.",
	"stop_already_reconciled":           "A linked resolution already exists; inspect the original exact receipt.",
}

func oneOf(value, fallback string, allowed ...string) string {
	for _, known := range allowed {
		if value == known {
			return known
		}
	}
	return fallback
}

// Derive copies only explicitly enumerated categories, timestamps and bounded
// evidence booleans. It never forwards free-form messages, request/actor/member
// names, endpoints, paths, database bodies, exact identities or sealed reviews.
// Unknown runner fields cannot become an accidental diagnostic export channel.
func Derive(d protocol.ProjectMemberDiagnosis) Bundle {
	b := Bundle{Version: 1, Scope: "expanded-member Stop diagnosis only; saved history is not current health", OperationKind: oneOf(d.Operation.Kind, "unsupported", "stop-project-ship"), OperationStatus: oneOf(d.Operation.Status, "unknown", "queued", "running", "interrupted", "failed", "succeeded"), CreatedAt: d.Operation.CreatedAt, FinishedAt: d.Operation.FinishedAt, ReadAt: d.ReadAt, Phase: oneOf(d.Phase, "unsupported", "ready", "stop-dispatched", "stopped", "retired"), Fence: oneOf(d.Fence, "unknown", "original-interrupted-operation", "resolved"), ReasonCode: d.ReasonCode, RuntimeObservation: oneOf(d.RuntimeObservation, "unavailable", "not-performed", "fresh", "unavailable"), Actions: []Action{}, PrivateBindingsOmitted: true}
	message, known := reasons[d.ReasonCode]
	if !known {
		b.ReasonCode = "diagnosis_unavailable"
		message = "This diagnostic category is unsupported by this exporter. Inspect private typed diagnosis with a matching client."
	}
	b.Message = message
	if b.RuntimeObservation == "fresh" && d.Observation != nil {
		at, running := d.Observation.ObservedAt, d.Observation.Running
		b.RuntimeObservedAt, b.Running = &at, &running
		if !running && !d.Observation.StartedAt.IsZero() && !d.Observation.FinishedAt.IsZero() && !d.Observation.FinishedAt.Before(d.Observation.StartedAt) {
			code, oom := d.Observation.ExitCode, d.Observation.OOMKilled
			clean := code == 0 && !oom
			b.CleanExitObserved, b.ExitCode, b.OOMKilled = &clean, &code, &oom
		}
	}
	b.LinkedResolutionRecorded = d.ResolvedBy != "" && b.Fence == "resolved"
	if b.ReasonCode == "stop_clean_exit_verified" && (b.CleanExitObserved == nil || !*b.CleanExitObserved || d.Review == nil) || b.ReasonCode == "stop_reconciled" && !b.LinkedResolutionRecorded {
		b.ReasonCode = "diagnosis_evidence_inconsistent"
		b.Message = "The reported category lacks its corresponding evidence. Inspect a matching private typed diagnosis; no action authority is established."
	}
	if known && b.OperationKind == "stop-project-ship" && b.OperationStatus == "interrupted" && b.Phase == "stop-dispatched" && b.Fence == "original-interrupted-operation" && (b.ReasonCode == "stop_needs_observation" || b.ReasonCode == "stop_observation_unavailable") {
		b.Actions = append(b.Actions, Action{"prepare-private-stop-review", protocol.ProjectStopRecoveryFeature, "original-operator", "fresh exact private diagnosis and verified ownership/clean exit; no effect is authorized by this bundle"})
	}
	return b
}
