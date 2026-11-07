package managed

import (
	"database/sql"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const maxDeskFrontendBytes = 256 << 10

func loadDeskFrontend(q querier, id string) (deskwork.FrontendRecord, error) {
	var out deskwork.FrontendRecord
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_frontends WHERE id=?", id).Scan(&body); err != nil {
		return out, err
	}
	if len(body) > maxDeskFrontendBytes {
		return out, errors.New("frontend upload record exceeds bound")
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
	if out.CheckReceipt(receipt) != nil {
		return out, errors.New("frontend upload journal binding mismatch")
	}
	return out, nil
}
func initializeDeskFrontends(tx *sql.Tx, version int) error {
	if version < 57 {
		if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS desk_frontends(id TEXT PRIMARY KEY REFERENCES desk_operations(id),body BLOB NOT NULL)"); err != nil {
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
		if r.Intent.Action == "upload-frontend" {
			if _, err = loadDeskFrontend(tx, id); err != nil {
				return err
			}
		}
	}
	var orphaned int
	if err = tx.QueryRow("SELECT count(*) FROM desk_frontends d LEFT JOIN desk_operations o ON d.id=o.id WHERE o.id IS NULL OR json_extract(o.body,'$.intent.action') !='upload-frontend'").Scan(&orphaned); err != nil {
		return err
	}
	if orphaned != 0 {
		return errors.New("orphaned frontend upload review")
	}
	return nil
}

func (s *store) createDeskFrontend(record deskwork.FrontendRecord) (deskwork.Receipt, bool, error) {
	if err := record.Validate(); err != nil {
		return deskwork.Receipt{}, false, bad(err.Error())
	}
	body, _ := json.Marshal(record)
	if len(body) > maxDeskFrontendBytes {
		return deskwork.Receipt{}, false, bad("frontend upload record exceeds bound")
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
		prior, err := loadDeskFrontend(tx, record.Intent.Intent.ID)
		if err != nil {
			return receipt, false, err
		}
		if prior.Intent != record.Intent {
			return receipt, false, conflict("desk_request_conflict", "frontend upload ID already records another review")
		}
		return receipt, false, nil
	}
	if _, err = tx.Exec("INSERT INTO desk_frontends(id,body) VALUES(?,?)", record.Intent.Intent.ID, body); err != nil {
		return receipt, false, err
	}
	return receipt, true, tx.Commit()
}
