package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// Desk operations have their own journal. The ordinary lifecycle queue must
// never replay them, including a plan whose dispatch was interrupted.
func initializeDeskOperations(tx *sql.Tx, version int) error {
	if version < 50 {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS desk_operations (id TEXT PRIMARY KEY, actor TEXT NOT NULL, body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS desk_operation_history (id TEXT NOT NULL REFERENCES desk_operations(id), sequence INTEGER NOT NULL, body BLOB NOT NULL, PRIMARY KEY(id,sequence));`); err != nil {
			return err
		}
	}
	all, err := deskOperationIDs(tx)
	if err != nil {
		return err
	}
	for _, id := range all {
		if _, err = loadDeskOperation(tx, id); err != nil {
			return err
		}
	}
	var orphaned int
	if err = tx.QueryRow(`SELECT count(*) FROM desk_operation_history h LEFT JOIN desk_operations o ON h.id=o.id WHERE o.id IS NULL`).Scan(&orphaned); err != nil {
		return err
	}
	if orphaned != 0 {
		return errors.New("orphaned desk operation receipt")
	}
	return nil
}
func deskOperationIDs(q querier) ([]string, error) {
	rows, err := q.Query("SELECT id FROM desk_operations ORDER BY id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	return ids, errors.Join(err, rows.Err(), rows.Close())
}
func decodeDeskReceipt(body []byte) (deskwork.Receipt, error) {
	var r deskwork.Receipt
	if len(body) > 64<<10 {
		return r, errors.New("desk receipt exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return r, errors.New("invalid trailing desk receipt data")
	}
	return r, r.Validate()
}
func loadDeskOperation(q querier, id string) (deskwork.Receipt, error) {
	var body []byte
	var actor string
	if err := q.QueryRow("SELECT actor,body FROM desk_operations WHERE id=?", id).Scan(&actor, &body); err != nil {
		return deskwork.Receipt{}, err
	}
	r, err := decodeDeskReceipt(body)
	if err != nil {
		return r, err
	}
	if r.Intent.ID != id || r.Actor != actor {
		return r, errors.New("desk operation identity mismatch")
	}
	rows, err := q.Query("SELECT sequence,body FROM desk_operation_history WHERE id=? ORDER BY sequence", id)
	if err != nil {
		return r, err
	}
	count := 0
	for rows.Next() {
		var sequence int
		var saved []byte
		if err = rows.Scan(&sequence, &saved); err != nil {
			break
		}
		count++
		if sequence != count || count > len(r.Steps) {
			err = errors.New("desk operation history sequence mismatch")
			break
		}
		prefix := r
		prefix.Steps = r.Steps[:count]
		expected, _ := json.Marshal(prefix)
		if !bytes.Equal(expected, saved) {
			err = errors.New("desk operation history mismatch")
			break
		}
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return r, err
	}
	if count != len(r.Steps) {
		return r, errors.New("desk operation history is incomplete")
	}
	return r, nil
}

// createDeskOperation is idempotent only for the same actor and exact intent.
// A duplicate never becomes permission to dispatch again.
func (s *store) createDeskOperation(actor string, intent deskwork.Intent) (deskwork.Receipt, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return deskwork.Receipt{}, false, err
	}
	defer tx.Rollback()
	receipt, fresh, err := createDeskOperationTx(tx, actor, intent)
	if err != nil {
		return receipt, false, err
	}
	return receipt, fresh, tx.Commit()
}
func createDeskOperationTx(tx *sql.Tx, actor string, intent deskwork.Intent) (deskwork.Receipt, bool, error) {
	r := deskwork.Receipt{Version: 1, Intent: intent, Actor: actor, Steps: []deskwork.Step{{Kind: intent.Action, State: "planned", At: time.Now().UTC()}}}
	if err := r.Validate(); err != nil {
		return r, false, bad(err.Error())
	}
	old, err := loadDeskOperation(tx, intent.ID)
	if err == nil {
		if old.Actor != actor || old.Intent != intent {
			return r, false, conflict("desk_request_conflict", "desk request ID already records another intent")
		}
		return old, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return r, false, err
	}
	ids, err := deskOperationIDs(tx)
	if err != nil {
		return r, false, err
	}
	for _, id := range ids {
		old, err := loadDeskOperation(tx, id)
		if err != nil {
			return r, false, err
		}
		state := old.Steps[len(old.Steps)-1].State
		if old.Intent.Target == intent.Target && (state == "planned" || state == "dispatched" || state == "acknowledged" || state == "uncertain") {
			return r, false, conflict("desk_outcome_unresolved", "reconcile the earlier desk operation before another effect")
		}
	}
	body, _ := json.Marshal(r)
	if _, err = tx.Exec("INSERT INTO desk_operations(id,actor,body) VALUES(?,?,?)", intent.ID, actor, body); err != nil {
		return r, false, err
	}
	if _, err = tx.Exec("INSERT INTO desk_operation_history(id,sequence,body) VALUES(?,1,?)", intent.ID, body); err != nil {
		return r, false, err
	}
	return r, true, nil
}

func appendDeskStep(tx *sql.Tx, expected deskwork.Receipt, step deskwork.Step) (deskwork.Receipt, error) {
	current, err := loadDeskOperation(tx, expected.Intent.ID)
	if err != nil {
		return current, err
	}
	before, _ := json.Marshal(expected)
	actual, _ := json.Marshal(current)
	if !bytes.Equal(before, actual) {
		return current, conflict("desk_receipt_changed", "desk operation changed; read its receipt")
	}
	current.Steps = append(append([]deskwork.Step{}, current.Steps...), step)
	if err = current.Validate(); err != nil {
		return current, err
	}
	body, _ := json.Marshal(current)
	if _, err = tx.Exec("INSERT INTO desk_operation_history(id,sequence,body) VALUES(?,?,?)", current.Intent.ID, len(current.Steps), body); err != nil {
		return current, err
	}
	if _, err = tx.Exec("UPDATE desk_operations SET body=? WHERE id=?", body, current.Intent.ID); err != nil {
		return current, err
	}
	return current, nil
}
func (s *store) appendDeskStep(expected deskwork.Receipt, step deskwork.Step) (deskwork.Receipt, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return expected, err
	}
	defer tx.Rollback()
	result, err := appendDeskStep(tx, expected, step)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s *store) reconcileDeskOperations() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids, err := deskOperationIDs(tx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, id := range ids {
		r, err := loadDeskOperation(tx, id)
		if err != nil {
			return err
		}
		next, err := r.Recover(now)
		if err != nil {
			return err
		}
		if len(next.Steps) > len(r.Steps) {
			if _, err = appendDeskStep(tx, r, next.Steps[len(next.Steps)-1]); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
