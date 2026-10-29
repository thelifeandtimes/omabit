package client

import (
	"context"
	"encoding/hex"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"slices"
)

var ErrProjectRecoveryCheckpointInvalid = errors.New("invalid saved project recovery checkpoint")

func (c *Client) ProjectRecoveryCheckpoint(ctx context.Context, expected Identity, project, operation string) (protocol.ProjectRecoveryCheckpoint, error) {
	var zero protocol.ProjectRecoveryCheckpoint
	if !protocol.ValidID(project) || !protocol.ValidID(operation) {
		return zero, ErrProjectRecoveryCheckpointInvalid
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return zero, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return zero, err
	}
	if !slices.Contains(hello.Features, protocol.ProjectRecoveryCheckpointFeature) {
		return zero, errors.New("runner does not advertise " + protocol.ProjectRecoveryCheckpointFeature)
	}
	var out protocol.ProjectRecoveryCheckpoint
	if err = c.do(ctx, http.MethodGet, protocol.ProjectRecoveryCheckpointPath(project, operation), nil, &out, true); err != nil {
		return zero, err
	}
	if err = expected.Check(out.RunnerID, out.Mode); err != nil {
		return zero, err
	}
	if !validProjectRecoveryCheckpoint(out, project, operation) {
		return zero, ErrProjectRecoveryCheckpointInvalid
	}
	return out, nil
}
func validProjectRecoveryCheckpoint(r protocol.ProjectRecoveryCheckpoint, project, operation string) bool {
	if r.SchemaVersion != 1 || r.Project.ID != project || r.Operation.ID != operation || r.Operation.Kind != "recover-project-anchor" || r.Operation.InstanceID != "" || r.Basis != "stored-recovery-checkpoint" || r.RuntimeObservation != "not-performed" || r.ActionAuthority != "not-granted" {
		return false
	}
	inventory := protocol.ProjectInventory{SchemaVersion: 1, RunnerID: r.RunnerID, Mode: r.Mode, ReadAt: r.ReadAt, Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{r.Project}}
	if !validProjectInventory(inventory) || !protocol.ValidProjectOperationSummary(r.Operation, r.Project, r.ReadAt) {
		return false
	}
	for _, digest := range []string{r.HistoryDigest, r.CheckpointDigest} {
		if len(digest) != 64 {
			return false
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	if !slices.Contains([]string{"review-planned", "reviewing", "reviewed", "draining", "namespace", "retiring", "replacing", "complete"}, r.Phase) {
		return false
	}
	switch r.Next.Kind {
	case "pending", "review", "complete":
		if r.Next.TargetKind != "" || r.Next.InstanceID != "" {
			return false
		}
		if r.Next.Kind == "complete" {
			return r.Phase == "complete" && r.Operation.Status == "succeeded"
		}
		return r.Phase != "complete" && r.Operation.Status != "succeeded"
	case "observation", "continuation":
		if r.Operation.Status != "interrupted" || r.Phase == "complete" {
			return false
		}
		member := false
		if r.Next.Kind == "observation" {
			switch r.Next.TargetKind {
			case "member-exit", "member-absence", "member-running":
				member = true
			case "namespace", "readiness":
			default:
				return false
			}
		} else {
			switch r.Next.TargetKind {
			case "stop-member", "retire-member", "create-member", "start-created-member":
				member = true
			case "begin-drain", "establish-namespace", "begin-retirement", "plan-replacements":
			default:
				return false
			}
		}
		if !member {
			return r.Next.InstanceID == ""
		}
		for _, m := range r.Project.Members {
			if m.InstanceID == r.Next.InstanceID {
				return true
			}
		}
		return false
	default:
		return false
	}
}
