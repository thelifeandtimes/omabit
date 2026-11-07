package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func initializeShipRoutes(tx *sql.Tx, version int) error {
	if version < 58 {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS ship_routes (binding TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS ship_routes_history (binding TEXT NOT NULL REFERENCES ship_routes(binding), revision INTEGER NOT NULL, body BLOB NOT NULL, PRIMARY KEY(binding,revision));`); err != nil {
			return err
		}
	}
	rows, err := tx.Query("SELECT binding FROM ship_routes ORDER BY binding")
	if err != nil {
		return err
	}
	var keys []string
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
		record, e := loadShipRoutes(tx, key)
		if e != nil {
			return e
		}
		var runner string
		if err = tx.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&runner); err != nil {
			return err
		}
		if record.Binding.RunnerID != runner {
			return errors.New("ship routes belong to another runner")
		}
	}
	var n int
	if err = tx.QueryRow("SELECT count(*) FROM ship_routes_history h LEFT JOIN ship_routes s ON h.binding=s.binding WHERE s.binding IS NULL").Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return errors.New("orphaned ship route history")
	}
	return nil
}

func decodeShipRoutes(body []byte) (shiproute.Record, error) {
	var out shiproute.Record
	if len(body) > 32<<10 {
		return out, errors.New("ship route record exceeds bound")
	}
	if err := decodeDeploymentData(body, &out); err != nil {
		return out, err
	}
	if out.Revision == 0 || out.Validate() != nil {
		return out, errors.New("invalid stored ship routes")
	}
	canonical, _ := json.Marshal(out)
	if !bytes.Equal(body, canonical) {
		return out, errors.New("noncanonical ship route record")
	}
	return out, nil
}

func loadShipRoutes(q querier, key string) (shiproute.Record, error) {
	var body []byte
	err := q.QueryRow("SELECT body FROM ship_routes WHERE binding=?", key).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		var n int
		if e := q.QueryRow("SELECT count(*) FROM ship_routes_history WHERE binding=?", key).Scan(&n); e != nil {
			return shiproute.Record{}, e
		}
		if n != 0 {
			return shiproute.Record{}, errors.New("ship route current record is missing")
		}
	}
	if err != nil {
		return shiproute.Record{}, err
	}
	out, err := decodeShipRoutes(body)
	if err != nil {
		return out, err
	}
	if out.Binding.Key() != key {
		return out, errors.New("ship route binding mismatch")
	}
	rows, err := q.Query("SELECT revision,body FROM ship_routes_history WHERE binding=? ORDER BY revision", key)
	if err != nil {
		return out, err
	}
	var count uint64
	var prior time.Time
	for rows.Next() {
		var revision uint64
		var saved []byte
		if err = rows.Scan(&revision, &saved); err != nil {
			break
		}
		count++
		v, e := decodeShipRoutes(saved)
		if e != nil || v.Binding != out.Binding || revision != count || v.Revision != revision || v.UpdatedAt.Before(prior) || count > out.Revision {
			err = errors.New("invalid ship route history")
			break
		}
		prior = v.UpdatedAt
		if count == out.Revision && !bytes.Equal(saved, body) {
			err = errors.New("ship route history mismatch")
			break
		}
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return out, err
	}
	if count != out.Revision {
		return out, errors.New("incomplete ship route history")
	}
	return out, nil
}

func (s *store) shipRoutes(binding shiproute.Binding) (shiproute.Record, error) {
	if binding.Validate() != nil || binding.RunnerID != s.runnerID {
		return shiproute.Record{}, bad("invalid ship route binding")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return shiproute.Record{}, err
	}
	defer tx.Rollback()
	out, err := loadShipRoutes(tx, binding.Key())
	if errors.Is(err, sql.ErrNoRows) {
		return shiproute.DefaultRecord(binding), nil
	}
	return out, err
}

// The caller holds lifecycle admission and has just verified the live pier.
// This store layer cannot turn a caller-provided ship name into that proof.
func (s *store) putShipRoutes(actor string, u shiproute.Update) (shiproute.Record, error) {
	var out shiproute.Record
	if u.Validate() != nil || u.Binding.RunnerID != s.runnerID {
		return out, bad("invalid ship route update")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = appPolicyTarget(tx, u.Binding.Target); err != nil {
		return out, err
	}
	old, err := loadShipRoutes(tx, u.Binding.Key())
	if errors.Is(err, sql.ErrNoRows) {
		old = shiproute.DefaultRecord(u.Binding)
		err = nil
	}
	if err != nil {
		return out, err
	}
	if old.Revision != u.ExpectedRevision {
		return out, conflict("ship_routes_revision_conflict", "ship URLs changed; refresh before saving")
	}
	now := time.Now().UTC()
	if now.Before(old.UpdatedAt) {
		now = old.UpdatedAt
	}
	out = shiproute.Record{Version: 1, Binding: u.Binding, Revision: old.Revision + 1, Settings: u.Settings, Actor: actor, UpdatedAt: now}
	if err = out.Validate(); err != nil {
		return out, err
	}
	body, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	if len(body) > 32<<10 {
		return out, bad("ship route record exceeds bound")
	}
	if _, err = tx.Exec("INSERT INTO ship_routes(binding,body) VALUES(?,?) ON CONFLICT(binding) DO UPDATE SET body=excluded.body", u.Binding.Key(), body); err != nil {
		return out, err
	}
	if _, err = tx.Exec("INSERT INTO ship_routes_history(binding,revision,body) VALUES(?,?,?)", u.Binding.Key(), out.Revision, body); err != nil {
		return out, err
	}
	return out, tx.Commit()
}
