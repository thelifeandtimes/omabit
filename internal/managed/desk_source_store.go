package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const maxSourceBaselineBytes = 16 << 20

func initializeDeskSourceBaselines(tx *sql.Tx, version int) error {
	if version < 52 {
		if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS desk_source_baselines (target TEXT PRIMARY KEY, body BLOB NOT NULL)"); err != nil {
			return err
		}
	}
	rows, err := tx.Query("SELECT target,body FROM desk_source_baselines ORDER BY target")
	if err != nil {
		return err
	}
	for rows.Next() {
		var key string
		var body []byte
		if err = rows.Scan(&key, &body); err != nil {
			break
		}
		var b deskwork.SourceBaseline
		b, err = decodeSourceBaseline(body)
		if err != nil {
			break
		}
		if deskSettingsKey(b.Binding.Target) != key {
			err = errors.New("source baseline target mismatch")
			break
		}
	}
	return errors.Join(err, rows.Err(), rows.Close())
}
func decodeSourceBaseline(body []byte) (deskwork.SourceBaseline, error) {
	var b deskwork.SourceBaseline
	if len(body) > maxSourceBaselineBytes {
		return b, errors.New("source baseline exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&b) != nil || d.Decode(new(any)) != io.EOF || b.Validate() != nil {
		return b, errors.New("invalid source baseline")
	}
	canonical, _ := json.Marshal(b)
	if !bytes.Equal(canonical, body) {
		return b, errors.New("noncanonical source baseline")
	}
	return b, nil
}
func (s *store) sourceBaseline(binding deskwork.SourceBinding) (*deskwork.SourceBaseline, error) {
	if binding.Validate() != nil {
		return nil, bad("invalid source binding")
	}
	var body []byte
	err := s.db.QueryRow("SELECT body FROM desk_source_baselines WHERE target=?", deskSettingsKey(binding.Target)).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b, err := decodeSourceBaseline(body)
	if err != nil {
		return nil, err
	}
	if b.Binding.Target != binding.Target {
		return nil, errors.New("source baseline target mismatch")
	}
	if b.Binding != binding {
		return nil, nil
	}
	return &b, nil
}

// Only the live status handler calls this, after verifying canonical equality.
// It is not a public write endpoint and carries no permissions or source bytes.
func (s *store) saveSourceBaseline(b deskwork.SourceBaseline) error {
	if err := b.Validate(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = appPolicyTarget(tx, b.Binding.Target); err != nil {
		return err
	}
	setting, err := loadDeskSetting(tx, deskSettingsKey(b.Binding.Target))
	if errors.Is(err, sql.ErrNoRows) {
		setting = deskwork.DefaultSetting(b.Binding.Target)
	} else if err != nil {
		return err
	}
	if setting.Revision != b.Binding.SettingsRevision || setting.Desk == nil || setting.Desk.Workflow != "mounted" || setting.Desk.SourceName != b.Binding.SourceName {
		return conflict("desk_settings_changed", "workflow changed during source inspection")
	}
	body, _ := json.Marshal(b)
	if len(body) > maxSourceBaselineBytes {
		return errors.New("source baseline exceeds bound")
	}
	if _, err = tx.Exec("INSERT INTO desk_source_baselines(target,body) VALUES(?,?) ON CONFLICT(target) DO UPDATE SET body=excluded.body", deskSettingsKey(b.Binding.Target), body); err != nil {
		return err
	}
	return tx.Commit()
}
