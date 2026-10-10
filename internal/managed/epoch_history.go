package managed

import (
	"database/sql"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Recover bounded display metadata from durable operation evidence, including
// rolls older than the 128-operation activity window. Never mutate inventories,
// their fingerprints or the driver's private state to decorate the desktop.
func loadEpochHistory(q querier, v *storedInstance) error {
	v.Public.EpochHistory, v.Public.LastRoll = nil, nil
	rows, err := q.Query(`SELECT body, json_extract(runtime, '$.id'), json_extract(runtime, '$.maintenance')
FROM operations WHERE json_extract(body, '$.instance_id')=?
AND json_extract(body, '$.kind')='roll' AND json_extract(body, '$.status')='succeeded'
AND runtime IS NOT NULL ORDER BY rowid DESC LIMIT ?`, v.Public.ID, epochs.MaxEpochs)
	if err != nil {
		return err
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var body []byte
		var id, metadata sql.NullString
		if err := rows.Scan(&body, &id, &metadata); err != nil {
			return err
		}
		if id.String != v.Runtime.ID || !metadata.Valid {
			continue
		}
		var op protocol.Operation
		var m protocol.MaintenanceReport
		if err := json.Unmarshal(body, &op); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(metadata.String), &m); err != nil {
			return err
		}
		if _, err := epochs.Number(m.CreatedEpoch); err != nil || m.CreatedEpoch == "0i0" || m.Kind != "roll" ||
			m.FinishedAt == nil || op.FinishedAt == nil || m.StartedAt.Before(op.CreatedAt) ||
			m.FinishedAt.Before(m.StartedAt) || m.FinishedAt.After(*op.FinishedAt) || seen[m.CreatedEpoch] {
			continue
		}
		seen[m.CreatedEpoch] = true
		v.Public.EpochHistory = append(v.Public.EpochHistory, protocol.EpochCreation{Name: m.CreatedEpoch, RolledAt: *m.FinishedAt})
		if v.Public.LastRoll == nil {
			v.Public.LastRoll = &m
		}
	}
	return errors.Join(rows.Err(), rows.Close())
}
