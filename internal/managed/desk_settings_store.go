package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func deskSettingsKey(target apppolicy.Target) string { b, _ := json.Marshal(target); return string(b) }
func initializeDeskSettings(tx *sql.Tx, version int) error {
	if version < 51 {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS desk_settings (target TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS desk_settings_history (target TEXT NOT NULL REFERENCES desk_settings(target), revision INTEGER NOT NULL, body BLOB NOT NULL, PRIMARY KEY(target,revision));`); err != nil {
			return err
		}
	}
	rows, err := tx.Query("SELECT target FROM desk_settings ORDER BY target")
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
		if _, err = loadDeskSetting(tx, key); err != nil {
			return err
		}
	}
	var count int
	if err = tx.QueryRow(`SELECT count(*) FROM desk_settings_history h LEFT JOIN desk_settings s ON h.target=s.target WHERE s.target IS NULL`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("orphaned development setting history")
	}
	return nil
}
func decodeDeskSetting(body []byte) (deskwork.Setting, error) {
	var v deskwork.Setting
	if len(body) > 16<<10 {
		return v, errors.New("development setting exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return v, err
	}
	if d.Decode(new(any)) != io.EOF || v.Revision == 0 || v.Validate() != nil {
		return v, errors.New("invalid persisted development setting")
	}
	canonical, _ := json.Marshal(v)
	if !bytes.Equal(canonical, body) {
		return v, errors.New("noncanonical development setting")
	}
	return v, nil
}
func loadDeskSetting(q querier, key string) (deskwork.Setting, error) {
	var body []byte
	if err := q.QueryRow("SELECT body FROM desk_settings WHERE target=?", key).Scan(&body); err != nil {
		return deskwork.Setting{}, err
	}
	value, err := decodeDeskSetting(body)
	if err != nil {
		return value, err
	}
	if deskSettingsKey(value.Target) != key {
		return value, errors.New("development setting target mismatch")
	}
	rows, err := q.Query("SELECT revision,body FROM desk_settings_history WHERE target=? ORDER BY revision", key)
	if err != nil {
		return value, err
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
		v, e := decodeDeskSetting(saved)
		if e != nil || v.Target != value.Target || revision != count || v.Revision != revision || v.UpdatedAt.Before(prior) || count > value.Revision {
			err = errors.New("invalid development setting history")
			break
		}
		prior = v.UpdatedAt
		if count == value.Revision && !bytes.Equal(saved, body) {
			err = errors.New("development setting history mismatch")
			break
		}
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return value, err
	}
	if count != value.Revision {
		return value, errors.New("incomplete development setting history")
	}
	return value, nil
}
func (s *store) deskSetting(target apppolicy.Target) (deskwork.Setting, error) {
	out := deskwork.DefaultSetting(target)
	if err := out.Validate(); err != nil {
		return deskwork.Setting{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = appPolicyTarget(tx, target); err != nil {
		return out, err
	}
	saved, err := loadDeskSetting(tx, deskSettingsKey(target))
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	return saved, err
}
func (s *store) putDeskSetting(actor string, update deskwork.SettingUpdate) (deskwork.Setting, error) {
	var out deskwork.Setting
	if err := update.Validate(); err != nil {
		return out, bad(err.Error())
	}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = appPolicyTarget(tx, update.Target); err != nil {
		return out, err
	}
	key := deskSettingsKey(update.Target)
	old, err := loadDeskSetting(tx, key)
	if errors.Is(err, sql.ErrNoRows) {
		old = deskwork.DefaultSetting(update.Target)
	} else if err != nil {
		return out, err
	}
	if old.Revision != update.ExpectedRevision {
		return out, conflict("desk_settings_changed", "development settings changed; read them again")
	}
	out = deskwork.Setting{Version: 1, Target: update.Target, Revision: old.Revision + 1, Ship: update.Ship, Desk: update.Desk, Actor: actor, UpdatedAt: time.Now().UTC()}
	if out.UpdatedAt.Before(old.UpdatedAt) {
		return deskwork.Setting{}, errors.New("clock precedes saved development setting")
	}
	if err = out.Validate(); err != nil {
		return deskwork.Setting{}, bad(err.Error())
	}
	body, _ := json.Marshal(out)
	if len(body) > 16<<10 {
		return deskwork.Setting{}, bad("development setting exceeds bound")
	}
	if _, err = tx.Exec("INSERT INTO desk_settings(target,body) VALUES(?,?) ON CONFLICT(target) DO UPDATE SET body=excluded.body", key, body); err != nil {
		return out, err
	}
	if _, err = tx.Exec("INSERT INTO desk_settings_history(target,revision,body) VALUES(?,?,?)", key, out.Revision, body); err != nil {
		return out, err
	}
	if err = tx.Commit(); err != nil {
		return deskwork.Setting{}, err
	}
	return out, nil
}
