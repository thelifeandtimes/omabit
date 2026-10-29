package managed

import (
	"context"
	"database/sql"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"time"
)

func (s *store) projectRecoverySource(ctx context.Context, id string) (protocol.ProjectRecoverySource, error) {
	var zero protocol.ProjectRecoverySource
	if !protocol.ValidID(id) {
		return zero, bad("valid project ID required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return zero, err
	}
	p, err := loadProject(tx, id)
	if err != nil {
		return zero, err
	}
	for _, validate := range []func(querier) error{validateProjectMemberContextReadiness, validateProjectAnchorRecoveries, validateProjectRecoveryObservations, validateProjectRecoveryContinuations} {
		if err = validate(tx); err != nil {
			return zero, err
		}
	}
	// Journal validation above preserves corruption errors. A fresh reservation can
	// legitimately have no initial launches yet; only that absent input is a refusal.
	h, originals, current, err := projectRecoveryHistory(tx, id, true)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, conflict("project_not_launched", "complete all initial member launches and readiness before reviewing recovery history")
	}
	if err != nil {
		return zero, err
	}
	context, err := projectnetwork.SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return zero, err
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		return zero, err
	}
	out := protocol.ProjectRecoverySource{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode, Basis: "stored-recovery-source", RuntimeObservation: "not-performed", ActionAuthority: "not-granted", HistoryDigest: recoveryHistoryDigest(h), Project: protocol.ProjectReservation{ID: p.ID, Revision: p.Revision, Label: p.Plan.Label, CreatedAt: p.CreatedAt, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{}}, Members: []protocol.ProjectRecoverySourceMember{}}
	for _, member := range p.Members {
		out.Project.Members = append(out.Project.Members, protocol.ProjectMemberReservation{InstanceID: member.InstanceID, Ship: member.Ship, HTTPPort: member.HTTPPort, InternalHTTPPort: member.InternalHTTPPort})
		selected := protocol.ProjectRecoverySourceMember{InstanceID: member.InstanceID}
		for _, c := range context.Members {
			if c.Context.Member.InstanceID == member.InstanceID {
				selected.CreatorOperationID = c.Context.Member.OperationID
				selected.ContainerGeneration = c.Context.Member.Generation
			}
		}
		for i, ready := range h.Originals {
			if ready.InstanceID == member.InstanceID {
				selected.OriginalReadinessOperationID = originals[i]
			}
		}
		for i, ready := range h.Current {
			if ready.InstanceID == member.InstanceID {
				selected.CurrentReadinessOperationID = current[i]
			}
		}
		if !protocol.ValidID(selected.CreatorOperationID) || selected.ContainerGeneration == 0 || !protocol.ValidID(selected.OriginalReadinessOperationID) {
			return zero, errors.New("recovery source lost member references")
		}
		out.Members = append(out.Members, selected)
	}
	out.ReadAt = time.Now().UTC()
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return out, nil
}
