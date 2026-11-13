package managed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"time"
)

func loadDeskAdoption(q querier, id string) (deskwork.AdoptionRecord, error) {
	var out deskwork.AdoptionRecord
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_adoptions WHERE id=?", id).Scan(&body); err != nil {
		return out, err
	}
	if err := decodeDeploymentData(body, &out); err != nil {
		return out, err
	}
	if err := out.Validate(); err != nil {
		return out, err
	}
	receipt, err := loadDeskOperation(q, id)
	if err != nil {
		return out, err
	}
	state := receipt.Steps[len(receipt.Steps)-1].State
	if receipt.Intent != out.Intent.Intent || receipt.Actor != out.Actor || state != "observed" && state != "blocked" {
		return out, errors.New("adoption journal binding mismatch")
	}
	return out, nil
}
func initializeDeskAdoptions(tx *sql.Tx, version int) error {
	if version < 55 {
		if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS desk_adoptions(id TEXT PRIMARY KEY REFERENCES desk_operations(id),body BLOB NOT NULL)"); err != nil {
			return err
		}
	}
	ids, err := deskOperationIDs(tx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		r, err := loadDeskOperation(tx, id)
		if err != nil {
			return err
		}
		if r.Intent.Action == "adopt-source" {
			if _, err = loadDeskAdoption(tx, id); err != nil {
				return err
			}
		}
	}
	var orphaned int
	if err = tx.QueryRow("SELECT count(*) FROM desk_adoptions d LEFT JOIN desk_operations o ON d.id=o.id WHERE o.id IS NULL OR json_extract(o.body,'$.intent.action')!='adopt-source'").Scan(&orphaned); err != nil {
		return err
	}
	if orphaned != 0 {
		return errors.New("orphaned source adoption")
	}
	return nil
}

// Adoption is a single runner database transaction. It performs no ship effect
// and never leaves an intermediate dispatch to retry on restart.
func (s *store) adoptDeskSource(record deskwork.AdoptionRecord) (deskwork.Receipt, error) {
	if err := record.Validate(); err != nil {
		return deskwork.Receipt{}, bad(err.Error())
	}
	body, _ := json.Marshal(record)
	if len(body) > maxDeskDeploymentBytes {
		return deskwork.Receipt{}, bad("adoption exceeds bound")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return deskwork.Receipt{}, err
	}
	defer tx.Rollback()
	receipt, fresh, err := createDeskOperationTx(tx, record.Actor, record.Intent.Intent)
	if err != nil {
		return receipt, err
	}
	if !fresh {
		old, err := loadDeskAdoption(tx, receipt.Intent.ID)
		if err != nil {
			return receipt, err
		}
		if old.Intent != record.Intent {
			return receipt, conflict("desk_request_conflict", "adoption ID already records another review")
		}
		return receipt, nil
	}
	previous, err := sourceOwnership(tx, record.Intent.Intent.Target, record.Intent.SourceName, record.Intent.Continuity)
	if err != nil {
		return receipt, err
	}
	if previous.Raw.Digest != record.Plan.Previous.Raw.Digest || previous.Clay.Digest != record.Plan.Previous.Clay.Digest {
		return receipt, conflict("adoption_changed", "source ownership changed after review")
	}
	policy, err := loadAppPolicy(tx)
	if err != nil {
		return receipt, err
	}
	at := time.Now().UTC()
	step := deskwork.Step{Kind: "adopt-source", State: "blocked", At: at}
	if record.Intent.Intent.Check(policy, record.Intent.Intent.Baseline) == nil {
		step.State = "observed"
		observation := record.Intent.Intent.Baseline
		step.Observation = &observation
		step.VerifiedSourceDigest = record.Intent.Intent.SourceDigest
	}
	receipt, err = appendDeskStep(tx, receipt, step)
	if err != nil {
		return receipt, err
	}
	if _, err = tx.Exec("INSERT INTO desk_adoptions(id,body) VALUES(?,?)", receipt.Intent.ID, body); err != nil {
		return receipt, err
	}
	if step.State == "observed" {
		owned := deskwork.OwnedSource{Version: 1, Target: receipt.Intent.Target, SourceName: record.Intent.SourceName, Continuity: record.Intent.Continuity, OperationID: receipt.Intent.ID, Ownership: record.Plan.Next, Observation: *step.Observation, ObservedAt: at}
		if err = owned.CheckAdoption(record, receipt); err != nil {
			return receipt, err
		}
		b, _ := json.Marshal(owned)
		if len(b) > maxDeskDeploymentBytes {
			return receipt, errors.New("ownership exceeds bound")
		}
		if _, err = tx.Exec("INSERT INTO desk_source_ownership(target,body) VALUES(?,?) ON CONFLICT(target) DO UPDATE SET body=excluded.body", deskSettingsKey(owned.Target), b); err != nil {
			return receipt, err
		}
	}
	return receipt, tx.Commit()
}
