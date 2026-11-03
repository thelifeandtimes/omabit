package managed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

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
		if err := appendEpochHistory(v, epochHistoryRow{body: body, id: id, metadata: metadata}, seen); err != nil {
			return err
		}
	}
	return errors.Join(rows.Err(), rows.Close())
}

// Snapshot history is read from one transaction. Rank per instance before
// extracting runtime metadata, preserving the original 128-record limit even
// when older, excluded runtime evidence is malformed. Rank only row IDs, not
// potentially large private runtime blobs. No stored state is changed.
func loadEpochHistories(q querier, all []storedInstance) error {
	if len(all) == 0 {
		return nil
	}
	if len(all) == 1 {
		return loadEpochHistory(q, &all[0])
	}
	// Bound SQL parameters; large inventories remain independent batches.
	for start := 0; start < len(all); start += 512 {
		end := min(start+512, len(all))
		if err := loadEpochHistoryBatch(q, all[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func loadEpochHistoryBatch(q querier, all []storedInstance) error {
	selected := make(map[string]*storedInstance, len(all))
	seen := make(map[string]map[string]bool, len(all))
	args := make([]any, 0, len(all)+1)
	for i := range all {
		v := &all[i]
		v.Public.EpochHistory, v.Public.LastRoll = nil, nil
		selected[v.Public.ID] = v
		seen[v.Public.ID] = make(map[string]bool)
		args = append(args, v.Public.ID)
	}
	args = append(args, epochs.MaxEpochs)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(all)), ",")
	rows, err := q.Query(`SELECT ranked.instance, evidence.body, json_extract(evidence.runtime, '$.id'), json_extract(evidence.runtime, '$.maintenance') FROM (
SELECT json_extract(body, '$.instance_id') AS instance, rowid AS position,
row_number() OVER (PARTITION BY json_extract(body, '$.instance_id') ORDER BY rowid DESC) AS ordinal
FROM operations WHERE json_extract(body, '$.instance_id') IN (`+placeholders+`)
AND json_extract(body, '$.kind')='roll' AND json_extract(body, '$.status')='succeeded' AND runtime IS NOT NULL
) AS ranked JOIN operations AS evidence ON evidence.rowid=ranked.position
WHERE ranked.ordinal <= ? ORDER BY ranked.position DESC`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var instance string
		var row epochHistoryRow
		if err := rows.Scan(&instance, &row.body, &row.id, &row.metadata); err != nil {
			return err
		}
		if err := appendEpochHistory(selected[instance], row, seen[instance]); err != nil {
			return err
		}
	}
	return errors.Join(rows.Err(), rows.Close())
}

type epochHistoryRow struct {
	body         []byte
	id, metadata sql.NullString
}

func appendEpochHistory(v *storedInstance, row epochHistoryRow, seen map[string]bool) error {
	if row.id.String != v.Runtime.ID || !row.metadata.Valid {
		return nil
	}
	var op protocol.Operation
	var m protocol.MaintenanceReport
	if err := json.Unmarshal(row.body, &op); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(row.metadata.String), &m); err != nil {
		return err
	}
	if _, err := epochs.Number(m.CreatedEpoch); err != nil || m.CreatedEpoch == "0i0" || m.Kind != "roll" ||
		m.FinishedAt == nil || op.FinishedAt == nil || m.StartedAt.Before(op.CreatedAt) ||
		m.FinishedAt.Before(m.StartedAt) || m.FinishedAt.After(*op.FinishedAt) || seen[m.CreatedEpoch] {
		return nil
	}
	seen[m.CreatedEpoch] = true
	v.Public.EpochHistory = append(v.Public.EpochHistory, protocol.EpochCreation{Name: m.CreatedEpoch, RolledAt: *m.FinishedAt})
	if v.Public.LastRoll == nil {
		v.Public.LastRoll = &m
	}
	return nil
}
