package managed

import (
	"context"
	"database/sql"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"time"
)

func (s *store) projectRecoveryCheckpoint(ctx context.Context, project, operation string) (protocol.ProjectRecoveryCheckpoint, error) {
	var zero protocol.ProjectRecoveryCheckpoint
	if !protocol.ValidID(project) || !protocol.ValidID(operation) {
		return zero, bad("valid project and recovery operation IDs required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return zero, err
	}
	p, err := loadProject(tx, project)
	if err != nil {
		return zero, err
	}
	// Validate stored ownership before classifying a user-selected missing receipt.
	for _, validate := range []func(querier) error{validateProjectMemberContextReadiness, validateProjectAnchorRecoveries, validateProjectRecoveryObservations, validateProjectRecoveryContinuations} {
		if err = validate(tx); err != nil {
			return zero, err
		}
	}
	op, err := loadOperation(tx, operation)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, conflict("project_recovery_not_found", "selected recovery receipt does not exist")
	}
	if err != nil {
		return zero, err
	}
	if op.ProjectID != project || op.Kind != "recover-project-anchor" || op.InstanceID != "" {
		return zero, conflict("project_recovery_mismatch", "select this project's coordinated anchor recovery receipt")
	}
	j, err := loadProjectAnchorRecovery(tx, operation)
	if err != nil {
		return zero, err
	}
	if err = recoverySourceReferences(tx, j); err != nil {
		return zero, err
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		return zero, err
	}
	out := protocol.ProjectRecoveryCheckpoint{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode, Basis: "stored-recovery-checkpoint", RuntimeObservation: "not-performed", ActionAuthority: "not-granted", HistoryDigest: recoveryHistoryDigest(j.Source), CheckpointDigest: recoveryCheckpointDigest(j.Journal), Sequence: j.Journal.Sequence, Phase: j.Journal.Phase,
		Project:   protocol.ProjectReservation{ID: p.ID, Revision: p.Revision, Label: p.Plan.Label, CreatedAt: p.CreatedAt, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{}},
		Operation: protocol.ProjectOperationSummary{ID: op.ID, RequestID: op.RequestID, Kind: op.Kind, Actor: op.Actor, Status: op.Status, ConfigRevision: op.ConfigRevision, CreatedAt: op.CreatedAt, FinishedAt: op.FinishedAt}, Next: protocol.ProjectRecoveryHint{Kind: "review"}}
	for _, m := range p.Members {
		out.Project.Members = append(out.Project.Members, protocol.ProjectMemberReservation{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort, InternalHTTPPort: m.InternalHTTPPort})
	}
	var active int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return zero, err
	}
	switch {
	case op.Status == "succeeded":
		out.Next.Kind = "complete"
	case active != 0:
		out.Next.Kind = "pending"
	case op.Status == "interrupted":
		if target, e := projectnetwork.RecoveryCheckpointObservationTarget(j.Source, j.Journal); e == nil {
			out.Next = protocol.ProjectRecoveryHint{Kind: "observation", TargetKind: target.Kind, InstanceID: target.InstanceID}
		} else if target, e := projectnetwork.RecoveryCheckpointContinuationTarget(j.Source, j.Journal); e == nil {
			out.Next = protocol.ProjectRecoveryHint{Kind: "continuation", TargetKind: target.Kind, InstanceID: target.InstanceID}
		}
	}
	out.ReadAt = time.Now().UTC()
	if !protocol.ValidProjectOperationSummary(out.Operation, out.Project, out.ReadAt) {
		return zero, errors.New("invalid saved recovery receipt")
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return out, nil
}
