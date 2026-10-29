package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (s *store) projectStatus(ctx context.Context, id string) (protocol.ProjectStatus, error) {
	var zero protocol.ProjectStatus
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
	digest, err := p.Plan.Digest()
	if err != nil {
		return zero, err
	}
	result := protocol.ProjectStatus{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode,
		Basis: "stored-operation-history", RuntimeObservation: "not-performed", Operations: []protocol.ProjectOperationSummary{},
		Project: protocol.ProjectReservation{ID: p.ID, Revision: p.Revision, Label: p.Plan.Label, CreatedAt: p.CreatedAt, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{}}}
	for _, m := range p.Members {
		result.Project.Members = append(result.Project.Members, protocol.ProjectMemberReservation{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort, InternalHTTPPort: m.InternalHTTPPort})
	}
	if err = addExpansionInventory(tx, &result.Project); err != nil {
		return zero, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,actor,request_id,body FROM operations WHERE json_extract(body,'$.project_id')=? ORDER BY rowid DESC LIMIT ?", id, protocol.ProjectStatusOperationLimit+1)
	if err != nil {
		return zero, err
	}
	for rows.Next() {
		if len(result.Operations) == protocol.ProjectStatusOperationLimit {
			result.HasOlderOperations = true
			break
		}
		var opID, actor, request string
		var body []byte
		var op protocol.Operation
		if err = rows.Scan(&opID, &actor, &request, &body); err != nil {
			break
		}
		if json.Unmarshal(body, &op) != nil || op.ID != opID || op.Actor != actor || op.RequestID != request || op.ProjectID != id {
			err = errors.New("project operation differs from its stored ownership")
			break
		}
		result.Operations = append(result.Operations, protocol.ProjectOperationSummary{ID: op.ID, RequestID: op.RequestID, InstanceID: op.InstanceID, Kind: op.Kind, Actor: op.Actor, Status: op.Status, ConfigRevision: op.ConfigRevision, CreatedAt: op.CreatedAt, FinishedAt: op.FinishedAt})
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return zero, err
	}
	result.ReadAt = time.Now().UTC()
	for _, op := range result.Operations {
		if !protocol.ValidProjectOperationSummary(op, result.Project, result.ReadAt) {
			return zero, errors.New("invalid stored project operation summary")
		}
	}
	if len(result.Operations) == 0 {
		return zero, errors.New("registered project has no operation history")
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return result, nil
}
