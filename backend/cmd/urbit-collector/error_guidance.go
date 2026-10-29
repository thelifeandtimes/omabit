package main

import (
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Guidance is collector navigation, not a runner capability or an instruction
// to execute a repair. Preserve the established error object and exit codes.
type guidedErrorResponse struct {
	Error    protocol.APIError `json:"error"`
	Guidance *errorGuidance    `json:"guidance,omitempty"`
}

type errorGuidance struct {
	Kind    string        `json:"kind"`
	Summary string        `json:"summary"`
	Next    []helpCommand `json:"next"`
}

func guidanceFor(err error, problem protocol.APIError) *errorGuidance {
	if errors.Is(err, client.ErrIdentityMismatch) {
		return &errorGuidance{Kind: "resolve_runner_identity", Summary: "Stop mutations through this selection. Resolve the intended runner before reconnecting; do not overwrite enrollment to suppress the mismatch.", Next: []helpCommand{
			{"runners", "Read the saved identities locally using the same collector registry."},
			{"help runners", "Read offline selection and identity guidance."},
		}}
	}
	if errors.Is(err, eyre.ErrIdentity) || errors.Is(err, eyre.ErrRedirect) {
		return &errorGuidance{Kind: "resolve_ship_origin", Summary: "Resolve the intended ship and origin before signing in or changing enrollment. Preserve the saved identity and credentials while reviewing the mismatch.", Next: []helpCommand{
			{"ships", "Read locally saved ship identities and origins using the same collector registry."},
			{"help connections", "Read offline connection and identity guidance."},
		}}
	}
	var uncertain *client.OutcomeUncertainError
	if errors.As(err, &uncertain) {
		if _, _, _, ok := protocol.ProjectRecoveryActionIDs(uncertain.Path); ok {
			return &errorGuidance{Kind: "reconcile_project_recovery", Summary: "Recovery request outcome is unknown. Preserve the same runner, actor, project, action, parent, revision, history/checkpoint digests, sequence and request ID. Use the identical project-recover inputs with --replay-only to look up its receipt without scheduling work. An interrupted receipt needs a separately reviewed observation or continuation; do not mint a retry ID.", Next: []helpCommand{{"project-recover --help", "Read explicit recovery and lookup-only flags."}, {"project-recovery-checkpoint --help", "Read the exact retained parent checkpoint without runtime effects."}}}
		}
		if _, ok := protocol.ProjectActionID(uncertain.Path); ok {
			return &errorGuidance{Kind: "reconcile_project_action", Summary: "Project action outcome is unknown. Preserve the runner, actor, project, member, action, prerequisite receipts, revision, digest and request ID. Use the same project-operate inputs with --replay-only to look up its retained receipt; this never schedules new work. An interrupted receipt needs separate reviewed recovery, not a new request ID.", Next: []helpCommand{{"project-operate --help", "Read explicit action and lookup-only flags."}, {"project-status --help", "Read saved history without replaying runtime effects."}}}
		}
		if uncertain.Path == protocol.PrivateChainProjectsPath {
			return &errorGuidance{Kind: "reconcile_private_chain_reservation", Summary: "Private-chain reservation outcome is unknown. Keep the same runner, actor, plan, review digest and request ID. Use private-chain-register --replay-only to look up the original receipt without creating a missing reservation; do not choose a new ID or start ships.", Next: []helpCommand{{"private-chain-register --help", "Read exact lookup-only flags."}, {"private-chain-projects --help", "Read saved reservations without runtime effects."}}}
		}
		if uncertain.Path == protocol.ProjectInventoryPath {
			return &errorGuidance{Kind: "reconcile_project_reservation", Summary: "Reservation outcome is unknown. Keep the same runner, actor, plan, review digest and request ID. Use project-register --replay-only to look up the original receipt without creating a missing reservation; do not choose a new ID or start ships.", Next: []helpCommand{{"project-register --help", "Read the exact lookup-only command flags."}, {"help projects", "Read project reservation and saved-history guidance."}}}
		}

		return &errorGuidance{Kind: "reconcile_uncertain_request", Summary: "The mutation outcome is unknown. Preserve the original request ID, actor, target and effect. Do not retry with a new ID or submit an inverse action. An absent receipt during an outage does not prove rejection.", Next: []helpCommand{
			{"snapshot", "When the same pinned runner is reachable, read its state and receipts; match the original request and actor before deciding any continuation."},
			{"help ships", "Read offline receipt and interruption guidance. Other mutation families have their own state/recovery rules."},
		}}
	}
	var remote *client.RemoteError
	if errors.As(err, &remote) && problem.Code == "revision_conflict" {
		return &errorGuidance{Kind: "review_current_revision", Summary: "Read the current state and review the intended effect before acting. Do not blindly replace the revision or retry.", Next: []helpCommand{
			{"snapshot", "Read current state and operation receipts from the same pinned runner."},
			{"help ships", "Read offline revision and operation guidance."},
		}}
	}
	if errors.Is(err, credentials.ErrStore) {
		return &errorGuidance{Kind: "restore_credential_access", Summary: "Credential storage is unavailable or refused access. Preserve the existing session and resolve access to the configured store; do not switch adapters or create a plaintext fallback.", Next: []helpCommand{
			{"ships", "Read local connection metadata and the configured credential adapter."},
			{"help connections", "Read offline session and cleanup guidance before changing credentials."},
		}}
	}
	if errors.Is(err, credentials.ErrMissing) || errors.Is(err, eyre.ErrAuth) {
		return &errorGuidance{Kind: "review_ship_login", Summary: "The selected session is absent or rejected. Review the intended connection before a new sign-in; keep +code on stdin and out of arguments or reports. This does not prove the ship is stopped or missing its backend.", Next: []helpCommand{
			{"ships", "Read locally saved ship identities and origins using the same collector registry."},
			{"ship-login --help", "Read the explicit sign-in contract without contacting a ship or reading credentials."},
		}}
	}
	return nil
}
