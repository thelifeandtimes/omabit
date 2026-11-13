package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"io"
	"time"
)

const maxDeskDeploymentBytes = 64 << 20

func decodeDeskDeployment(body []byte) (deskwork.DeploymentRecord, error) {
	var r deskwork.DeploymentRecord
	if err := decodeDeploymentData(body, &r); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func decodeOwnedSource(body []byte) (deskwork.OwnedSource, error) {
	var r deskwork.OwnedSource
	if err := decodeDeploymentData(body, &r); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func decodeDeploymentData(body []byte, out any) error {
	if len(body) > maxDeskDeploymentBytes {
		return errors.New("deployment metadata exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid deployment metadata")
	}
	canonical, _ := json.Marshal(out)
	if !bytes.Equal(canonical, body) {
		return errors.New("noncanonical deployment metadata")
	}
	return nil
}
func loadDeskDeployment(q querier, id string) (deskwork.DeploymentRecord, error) {
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_deployments WHERE id=?", id).Scan(&body); err != nil {
		return deskwork.DeploymentRecord{}, err
	}
	record, err := decodeDeskDeployment(body)
	if err != nil {
		return record, err
	}
	receipt, err := loadDeskOperation(q, id)
	if err != nil {
		return record, err
	}
	if record.Intent.Intent != receipt.Intent || record.Actor != receipt.Actor {
		return record, errors.New("deployment journal binding mismatch")
	}
	return record, nil
}
func loadOwnedSource(q querier, key string) (deskwork.OwnedSource, error) {
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_source_ownership WHERE target=?", key).Scan(&body); err != nil {
		return deskwork.OwnedSource{}, err
	}
	owned, err := decodeOwnedSource(body)
	if err != nil {
		return owned, err
	}
	if deskSettingsKey(owned.Target) != key {
		return owned, errors.New("source ownership target mismatch")
	}
	receipt, err := loadDeskOperation(q, owned.OperationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return owned, errors.New("source ownership is missing its operation receipt")
		}
		return owned, err
	}
	if receipt.Intent.Action == "adopt-source" {
		record, err := loadDeskAdoption(q, owned.OperationID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return owned, errors.New("source ownership is missing adoption evidence")
			}
			return owned, err
		}
		return owned, owned.CheckAdoption(record, receipt)
	}
	record, err := loadDeskDeployment(q, owned.OperationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return owned, errors.New("source ownership is missing deployment evidence")
		}
		return owned, err
	}
	return owned, owned.Check(record, receipt)
}
func initializeDeskDeployments(tx *sql.Tx, version int) error {
	if version < 53 {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS desk_deployments(id TEXT PRIMARY KEY REFERENCES desk_operations(id),body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS desk_source_ownership(target TEXT PRIMARY KEY,body BLOB NOT NULL);`); err != nil {
			return err
		}
	}
	ids, err := deskOperationIDs(tx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		receipt, err := loadDeskOperation(tx, id)
		if err != nil {
			return err
		}
		if receipt.Intent.Action == "commit" {
			if _, err = loadDeskDeployment(tx, id); err != nil {
				return err
			}
		}
	}
	var orphaned int
	if err = tx.QueryRow("SELECT count(*) FROM desk_deployments d LEFT JOIN desk_operations o ON d.id=o.id WHERE o.id IS NULL OR json_extract(o.body,'$.intent.action')!='commit'").Scan(&orphaned); err != nil {
		return err
	}
	if orphaned != 0 {
		return errors.New("orphaned deployment review")
	}
	rows, err := tx.Query("SELECT target FROM desk_source_ownership ORDER BY target")
	if err != nil {
		return err
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			break
		}
		keys = append(keys, key)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return err
	}
	for _, key := range keys {
		if _, err = loadOwnedSource(tx, key); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) sourceOwnership(target apppolicy.Target, sourceName, continuity string) (deskwork.Ownership, error) {
	return sourceOwnership(s.db, target, sourceName, continuity)
}
func sourceOwnership(q querier, target apppolicy.Target, sourceName, continuity string) (deskwork.Ownership, error) {
	empty := deskwork.EmptyOwnership()
	owned, err := loadOwnedSource(q, deskSettingsKey(target))
	if errors.Is(err, sql.ErrNoRows) {
		return empty, nil
	}
	if err != nil {
		return empty, err
	}
	if owned.Target != target {
		return empty, errors.New("source ownership target changed")
	}
	if owned.SourceName != sourceName || owned.Continuity != continuity {
		return empty, nil
	}
	return owned.Ownership, nil
}
func (s *store) createDeskDeployment(record deskwork.DeploymentRecord) (deskwork.Receipt, bool, error) {
	if err := record.Validate(); err != nil {
		return deskwork.Receipt{}, false, bad(err.Error())
	}
	body, _ := json.Marshal(record)
	if len(body) > maxDeskDeploymentBytes {
		return deskwork.Receipt{}, false, bad("deployment metadata exceeds bound")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return deskwork.Receipt{}, false, err
	}
	defer tx.Rollback()
	receipt, fresh, err := createDeskOperationTx(tx, record.Actor, record.Intent.Intent)
	if err != nil {
		return receipt, false, err
	}
	if !fresh {
		prior, err := loadDeskDeployment(tx, record.Intent.Intent.ID)
		if err != nil {
			return receipt, false, err
		}
		if prior.Intent != record.Intent || prior.Plan.Digest != record.Plan.Digest {
			return receipt, false, conflict("desk_request_conflict", "commit ID already records another staged review")
		}
		return receipt, false, nil
	}
	if _, err = tx.Exec("INSERT INTO desk_deployments(id,body) VALUES(?,?)", record.Intent.Intent.ID, body); err != nil {
		return receipt, false, err
	}
	return receipt, true, tx.Commit()
}
func (s *store) completeDeskDeployment(expected deskwork.Receipt, observation deskwork.Observation, ownership deskwork.Ownership, actor string) (deskwork.Receipt, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return expected, err
	}
	defer tx.Rollback()
	record, err := loadDeskDeployment(tx, expected.Intent.ID)
	if err != nil {
		return expected, err
	}
	at := time.Now().UTC()
	receipt, err := appendDeskStep(tx, expected, deskwork.Step{Kind: "commit", State: "observed", Actor: actor, At: at, Observation: &observation, VerifiedSourceDigest: expected.Intent.SourceDigest})
	if err != nil {
		return expected, err
	}
	owned := deskwork.OwnedSource{Version: 1, Target: expected.Intent.Target, SourceName: record.Intent.SourceName, Continuity: record.Intent.Continuity, OperationID: expected.Intent.ID, Ownership: ownership, Observation: observation, ObservedAt: at}
	if err = owned.Check(record, receipt); err != nil {
		return expected, err
	}
	body, _ := json.Marshal(owned)
	if len(body) > maxDeskDeploymentBytes {
		return expected, errors.New("source ownership exceeds bound")
	}
	if _, err = tx.Exec("INSERT INTO desk_source_ownership(target,body) VALUES(?,?) ON CONFLICT(target) DO UPDATE SET body=excluded.body", deskSettingsKey(owned.Target), body); err != nil {
		return expected, err
	}
	return receipt, tx.Commit()
}
