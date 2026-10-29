package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (s *store) projectInventory(ctx context.Context) (protocol.ProjectInventory, error) {
	var result protocol.ProjectInventory
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	// Validate body and claim tables in the same snapshot. A missing claim is
	// corruption, not an empty inventory or permission to reuse its resources.
	if _, err = projectReservations(tx); err != nil {
		return result, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT body FROM projects ORDER BY id")
	if err != nil {
		return result, err
	}
	projects := []protocol.ProjectReservation{}
	for rows.Next() {
		var raw []byte
		var p storedProject
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = json.Unmarshal(raw, &p); err != nil {
			break
		}
		var digest string
		digest, err = p.Plan.Digest()
		if err != nil {
			break
		}
		item := protocol.ProjectReservation{ID: p.ID, Revision: p.Revision, Label: p.Plan.Label, CreatedAt: p.CreatedAt, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{}}
		for _, m := range p.Members {
			item.Members = append(item.Members, protocol.ProjectMemberReservation{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort, InternalHTTPPort: m.InternalHTTPPort})
		}
		projects = append(projects, item)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return result, err
	}
	for i := range projects {
		if err = addExpansionInventory(tx, &projects[i]); err != nil {
			return result, err
		}
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return protocol.ProjectInventory{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode, ReadAt: time.Now().UTC(), Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: projects}, nil
}
