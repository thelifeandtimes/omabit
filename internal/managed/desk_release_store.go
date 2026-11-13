package managed

import (
	"database/sql"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const maxDeskReleaseBytes = 256 << 10

func loadDeskRelease(q querier, id string) (deskwork.ReleaseRecord, error) {
	var out deskwork.ReleaseRecord
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_releases WHERE id=?", id).Scan(&body); err != nil {
		return out, err
	}
	if len(body) > maxDeskReleaseBytes {
		return out, errors.New("release record exceeds bound")
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
	if receipt.Intent != out.Intent.Intent || receipt.Actor != out.Actor {
		return out, errors.New("release journal binding mismatch")
	}
	return out, nil
}
func initializeDeskReleases(tx *sql.Tx, version int) error {
	if version < 54 {
		if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS desk_releases(id TEXT PRIMARY KEY REFERENCES desk_operations(id),body BLOB NOT NULL)"); err != nil {
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
		if deskwork.IsReleaseAction(r.Intent.Action) {
			if _, err = loadDeskRelease(tx, id); err != nil {
				return err
			}
		}
	}
	var orphaned int
	if err = tx.QueryRow("SELECT count(*) FROM desk_releases d LEFT JOIN desk_operations o ON d.id=o.id WHERE o.id IS NULL OR json_extract(o.body,'$.intent.action') NOT IN ('activate-local','publish-source','publish-treaty')").Scan(&orphaned); err != nil {
		return err
	}
	if orphaned != 0 {
		return errors.New("orphaned release review")
	}
	return nil
}

func (s *store) createDeskRelease(record deskwork.ReleaseRecord) (deskwork.Receipt, bool, error) {
	if err := record.Validate(); err != nil {
		return deskwork.Receipt{}, false, bad(err.Error())
	}
	body, _ := json.Marshal(record)
	if len(body) > maxDeskReleaseBytes {
		return deskwork.Receipt{}, false, bad("release record exceeds bound")
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
		prior, err := loadDeskRelease(tx, record.Intent.Intent.ID)
		if err != nil {
			return receipt, false, err
		}
		if prior.Intent != record.Intent {
			return receipt, false, conflict("desk_request_conflict", "release ID already records another review")
		}
		return receipt, false, nil
	}
	if _, err = tx.Exec("INSERT INTO desk_releases(id,body) VALUES(?,?)", record.Intent.Intent.ID, body); err != nil {
		return receipt, false, err
	}
	return receipt, true, tx.Commit()
}
