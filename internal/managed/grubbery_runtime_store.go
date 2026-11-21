package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

const appRuntimeRecoveryDDL = `CREATE TABLE grubbery_app_runtime_recoveries (request_id TEXT PRIMARY KEY, original_id TEXT NOT NULL REFERENCES grubbery_app_operations(id), ordinal INTEGER NOT NULL CHECK(ordinal>=1 AND ordinal<=64), previous_digest TEXT NOT NULL, digest TEXT NOT NULL UNIQUE, body BLOB NOT NULL, UNIQUE(original_id,ordinal))`
const appRuntimeHeadDDL = `CREATE TABLE grubbery_app_runtime_heads (original_id TEXT PRIMARY KEY REFERENCES grubbery_app_operations(id), ordinal INTEGER NOT NULL CHECK(ordinal>=0 AND ordinal<=64), digest TEXT NOT NULL)`

// Create before the parent journal's initialization/validation. SQLite permits
// the FK declaration before its parent table exists; seeding waits for it.
func initializeAppRuntimeRecoveryTables(tx *sql.Tx, version int) error {
	for _, table := range []struct{ name, ddl string }{{"grubbery_app_runtime_recoveries", appRuntimeRecoveryDDL}, {"grubbery_app_runtime_heads", appRuntimeHeadDDL}} {
		if version < 63 {
			var count int
			if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name=?", table.name).Scan(&count); err != nil {
				return err
			}
			if count != 0 {
				return grubbery.ErrHistory
			}
			if _, err := tx.Exec(table.ddl); err != nil {
				return err
			}
		}
		var ddl string
		if err := tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table.name).Scan(&ddl); err != nil || ddl != table.ddl {
			return grubbery.ErrHistory
		}
		var triggers int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name=?", table.name).Scan(&triggers); err != nil || triggers != 0 {
			return grubbery.ErrHistory
		}
	}
	return nil
}
func seedAppRuntimeHeads(tx *sql.Tx, version int) error {
	if version >= 63 {
		return nil
	}
	_, err := tx.Exec("INSERT INTO grubbery_app_runtime_heads(original_id,ordinal,digest) SELECT id,0,'' FROM grubbery_app_operations")
	return err
}

type appRuntimeHead struct {
	ordinal uint64
	digest  string
}

func loadAppRuntimeHead(q querier, original string) (appRuntimeHead, error) {
	var head appRuntimeHead
	err := q.QueryRow("SELECT ordinal,digest FROM grubbery_app_runtime_heads WHERE original_id=?", original).Scan(&head.ordinal, &head.digest)
	if err != nil || head.ordinal > grubbery.MaxRuntimeRecoveries || head.ordinal == 0 && head.digest != "" || head.ordinal != 0 && !runtimeRecoveryID(head.digest) {
		return head, grubbery.ErrHistory
	}
	return head, nil
}
func runtimeRecoveryID(id string) bool {
	return len(id) == 64 && protocolDigest(id) && id != "0000000000000000000000000000000000000000000000000000000000000000"
}
func protocolDigest(id string) bool {
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func readAppRuntimeRecovery(q querier, id string, verified ...grubbery.Record) (grubbery.RuntimeRecoveryReceipt, error) {
	var original, previous, digest string
	var ordinal uint64
	var body []byte
	if err := q.QueryRow("SELECT original_id,ordinal,previous_digest,digest,body FROM grubbery_app_runtime_recoveries WHERE request_id=?", id).Scan(&original, &ordinal, &previous, &digest, &body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrMissing
		}
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	var receipt grubbery.RuntimeRecoveryReceipt
	if appCanonical(body, &receipt, 16<<10) != nil || receipt.RequestID != id || receipt.Review.OriginalRequestID != original || receipt.Review.PreviousRecoveryDigest != previous || receipt.Digest != digest || ordinal == 0 || ordinal > grubbery.MaxRuntimeRecoveries {
		return receipt, grubbery.ErrHistory
	}
	var record grubbery.Record
	if len(verified) == 0 {
		var err error
		record, err = loadGrubberyRecord(q, original)
		if err != nil {
			return receipt, grubbery.ErrHistory
		}
	} else if len(verified) == 1 && verified[0].Plan.RequestID == original {
		record = verified[0]
	} else {
		return receipt, grubbery.ErrHistory
	}
	if receipt.ValidateAncestry(record) != nil {
		return receipt, grubbery.ErrHistory
	}
	return receipt, nil
}

// Called from the parent journal's validation, never from readAppRuntimeRecovery
// recursively. Independent heads detect tail removal, including removal of all
// audit rows. Every original operation, including legacy ones, has one head.
func validateAppRuntimeRecoveryJournal(q querier, originals map[string]grubbery.Record) error {
	var err error
	var headCount int
	if err = q.QueryRow("SELECT count(*) FROM grubbery_app_runtime_heads").Scan(&headCount); err != nil || headCount != len(originals) {
		return grubbery.ErrHistory
	}
	for id, original := range originals {
		head, err := loadAppRuntimeHead(q, id)
		if err != nil {
			return err
		}
		rows, err := q.Query("SELECT request_id,ordinal FROM grubbery_app_runtime_recoveries WHERE original_id=? ORDER BY ordinal", id)
		if err != nil {
			return err
		}
		type item struct {
			id      string
			ordinal uint64
		}
		items := []item{}
		for rows.Next() {
			var entry item
			if err = rows.Scan(&entry.id, &entry.ordinal); err != nil {
				break
			}
			items = append(items, entry)
			if len(items) > grubbery.MaxRuntimeRecoveries {
				err = grubbery.ErrHistory
				break
			}
		}
		if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
			return err
		}
		if head.ordinal != uint64(len(items)) {
			return grubbery.ErrHistory
		}
		var previous *grubbery.RuntimeRecoveryReceipt
		for index, entry := range items {
			if entry.ordinal != uint64(index+1) {
				return grubbery.ErrHistory
			}
			receipt, err := readAppRuntimeRecovery(q, entry.id, original)
			if err != nil || receipt.Review.ValidatePredecessor(original, previous) != nil {
				return grubbery.ErrHistory
			}
			previous = &receipt
		}
		expected := ""
		if previous != nil {
			expected = previous.Digest
		}
		if head.digest != expected {
			return grubbery.ErrHistory
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM grubbery_app_runtime_recoveries r LEFT JOIN grubbery_app_operations o ON r.original_id=o.id WHERE o.id IS NULL").Scan(&orphans); err != nil || orphans != 0 {
		return grubbery.ErrHistory
	}
	return nil
}

type grubberyRuntimeJournal struct{ s *store }

func (j grubberyRuntimeJournal) Load(ctx context.Context, original, id string) (grubbery.RuntimeRecoveryReceipt, error) {
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	receipt, err := readAppRuntimeRecovery(tx, id)
	if err != nil {
		return receipt, err
	}
	if receipt.Review.OriginalRequestID != original {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrConflict
	}
	return receipt, tx.Commit()
}
func latestAppRuntimeRecovery(q querier, original string) (*grubbery.RuntimeRecoveryReceipt, error) {
	head, err := loadAppRuntimeHead(q, original)
	if err != nil {
		return nil, err
	}
	if head.ordinal == 0 {
		return nil, nil
	}
	var id string
	if err = q.QueryRow("SELECT request_id FROM grubbery_app_runtime_recoveries WHERE original_id=? AND ordinal=?", original, head.ordinal).Scan(&id); err != nil {
		return nil, grubbery.ErrHistory
	}
	receipt, err := readAppRuntimeRecovery(q, id)
	if err != nil || receipt.Digest != head.digest {
		return nil, grubbery.ErrHistory
	}
	return &receipt, nil
}
func (j grubberyRuntimeJournal) Latest(ctx context.Context, original string) (*grubbery.RuntimeRecoveryReceipt, error) {
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return nil, err
	}
	var exists int
	if err = tx.QueryRow("SELECT 1 FROM grubbery_app_operations WHERE id=?", original).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grubbery.ErrMissing
		}
		return nil, err
	}
	receipt, err := latestAppRuntimeRecovery(tx, original)
	if err != nil {
		return nil, err
	}
	return receipt, tx.Commit()
}

// Apply is a metadata-only CAS. The authenticated server must independently
// compare fresh native review facts before entering this transaction. Exact
// saved replay precedes time, lease and current phase checks, and probes nothing.
// Commit failure is uncertain; lookup with the original audit IDs must precede
// any later apply. No original app row/event, source cache or lease is modified.
func (j grubberyRuntimeJournal) Apply(ctx context.Context, original, id string, review grubbery.RuntimeRecoveryReview, now time.Time) (grubbery.RuntimeRecoveryReceipt, error) {
	if !runtimeRecoveryID(id) || id == original || review.Digest != review.ContentDigest() || review.OriginalRequestID != original || review.OriginalBinding.RunnerID != j.s.runnerID || review.OriginalBinding.Mode != j.s.mode {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrConflict
	}
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	if saved, err := readAppRuntimeRecovery(tx, id); err == nil {
		if saved.Review.OriginalRequestID != original || saved.Review.Digest != review.Digest {
			return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrConflict
		}
		return saved, tx.Commit()
	} else if !errors.Is(err, grubbery.ErrMissing) {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	record, err := loadGrubberyRecord(tx, original)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	head, err := loadAppRuntimeHead(tx, original)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	previous, err := latestAppRuntimeRecovery(tx, original)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	if err = review.ValidatePredecessor(record, previous); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	if err = review.MatchesCurrent(record, head.digest, now); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	if head.ordinal >= grubbery.MaxRuntimeRecoveries {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrAdmission
	}
	n, _ := record.Plan.Binding.Namespace()
	var held string
	if err = tx.QueryRow("SELECT id FROM grubbery_app_leases WHERE namespace=?", n.DeskRoot).Scan(&held); err != nil || held != original {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrHistory
	}
	receipt := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: id, Review: review, AppliedAt: now.UTC()}
	receipt.Digest = receipt.ContentDigest()
	if err = receipt.ValidateAncestry(record); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	body, err := json.Marshal(receipt)
	if err != nil || len(body) > 16<<10 {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrHistory
	}
	if _, err = tx.Exec("INSERT INTO grubbery_app_runtime_recoveries(request_id,original_id,ordinal,previous_digest,digest,body) VALUES(?,?,?,?,?,?)", id, original, head.ordinal+1, head.digest, receipt.Digest, body); err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, errors.Join(grubbery.ErrCAS, err)
	}
	result, err := tx.Exec("UPDATE grubbery_app_runtime_heads SET ordinal=?,digest=? WHERE original_id=? AND ordinal=? AND digest=?", head.ordinal+1, receipt.Digest, original, head.ordinal, head.digest)
	if err != nil {
		return grubbery.RuntimeRecoveryReceipt{}, err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return grubbery.RuntimeRecoveryReceipt{}, grubbery.ErrCAS
	}
	if err = tx.Commit(); err != nil {
		return receipt, errors.Join(grubbery.ErrJournal, err)
	}
	return receipt, nil
}
