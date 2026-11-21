package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

var errRecordingHistory = errors.New("recording launch history is incomplete or changed; preserve original jobs and operation receipts")

const recordingIntentDDL = `CREATE TABLE runtime_recording_intents (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), ordinal INTEGER NOT NULL UNIQUE CHECK(ordinal>=1 AND ordinal<=1024), previous_digest TEXT NOT NULL, digest TEXT NOT NULL UNIQUE, body BLOB NOT NULL)`
const recordingHeadDDL = `CREATE TABLE runtime_recording_head (singleton INTEGER PRIMARY KEY CHECK(singleton=1), ordinal INTEGER NOT NULL CHECK(ordinal>=0 AND ordinal<=1024), digest TEXT NOT NULL)`
const maxRecordingIntents = 1024

// This private receipt associates a job file with an authenticated, journaled
// pre-Start state. It is neither current worker readiness nor permission to
// replay a service. No source paths, native output or credential bodies belong
// here. StateSHA256 binds the exact private state, including its revision.
type recordingIntent struct {
	Version               int       `json:"version"`
	JobID                 string    `json:"job_id"`
	JobSHA256             string    `json:"job_sha256"`
	RunnerID              string    `json:"runner_id"`
	Mode                  string    `json:"mode"`
	Actor                 string    `json:"actor"`
	OperationID           string    `json:"operation_id"`
	RequestID             string    `json:"request_id"`
	InstanceID            string    `json:"instance_id"`
	RuntimeID             string    `json:"runtime_id"`
	ConfigurationRevision uint64    `json:"configuration_revision"`
	RuntimeRevision       uint64    `json:"runtime_revision"`
	StateSHA256           string    `json:"state_sha256"`
	ContainerID           string    `json:"container_id"`
	ExecutableSHA256      string    `json:"executable_sha256"`
	AdmissionSHA256       string    `json:"admission_sha256"`
	CreatedAt             time.Time `json:"created_at"`
	ControlRevision       uint64    `json:"control_revision,omitempty"`
	ControlDigest         string    `json:"control_digest,omitempty"`
}

func recordingHex(s string, size int) bool {
	if len(s) != size || !protocolDigest(s) {
		return false
	}
	for _, b := range s {
		if b != '0' {
			return true
		}
	}
	return false
}

func (i recordingIntent) bytes() ([]byte, error) {
	if (i.Version != 1 && i.Version != 2) || i.Version == 1 && (i.ControlRevision != 0 || i.ControlDigest != "") || i.Version == 2 && (i.ControlRevision > 4096 || i.ControlRevision == 0 && i.ControlDigest != "" || i.ControlRevision > 0 && !recordingHex(i.ControlDigest, 64)) || !recordingHex(i.JobID, 32) || !recordingHex(i.JobSHA256, 64) || !protocol.ValidID(i.RunnerID) || i.Mode != mode && i.Mode != DevelopmentMode || i.Actor == "" || len(i.Actor) > 128 || !protocol.ValidID(i.OperationID) || !protocol.ValidID(i.RequestID) || !protocol.ValidID(i.InstanceID) || !recordingHex(i.RuntimeID, 32) || i.ConfigurationRevision == 0 || i.RuntimeRevision == 0 || !recordingHex(i.StateSHA256, 64) || !recordingHex(i.ContainerID, 64) || !recordingHex(i.ExecutableSHA256, 64) || !recordingHex(i.AdmissionSHA256, 64) || i.CreatedAt.Before(time.Unix(0, 1)) || i.CreatedAt.After(time.Unix(0, 1<<63-1)) {
		return nil, errRecordingHistory
	}
	for _, r := range i.Actor {
		if r < 33 || r > 126 {
			return nil, errRecordingHistory
		}
	}
	i.CreatedAt = i.CreatedAt.UTC()
	raw, err := json.Marshal(i)
	if err != nil || len(raw) > 4096 {
		return nil, errRecordingHistory
	}
	return raw, nil
}

func recordingStateDigest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", errRecordingHistory
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

func initializeRecordingIntents(tx *sql.Tx, version int) error {
	for _, table := range []struct{ name, ddl string }{{"runtime_recording_intents", recordingIntentDDL}, {"runtime_recording_head", recordingHeadDDL}} {
		if version < 64 {
			var count int
			if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name=?", table.name).Scan(&count); err != nil || count != 0 {
				return errRecordingHistory
			}
			if _, err := tx.Exec(table.ddl); err != nil {
				return err
			}
		}
		var ddl string
		if err := tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table.name).Scan(&ddl); err != nil || ddl != table.ddl {
			return errRecordingHistory
		}
		var triggers int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name=?", table.name).Scan(&triggers); err != nil || triggers != 0 {
			return errRecordingHistory
		}
	}
	if version < 64 {
		if _, err := tx.Exec("INSERT INTO runtime_recording_head VALUES(1,0,'')"); err != nil {
			return err
		}
	}
	return validateRecordingHistory(tx)
}

func recordingHead(q querier) (uint64, string, error) {
	var count int
	var ordinal uint64
	var digest string
	if q.QueryRow("SELECT count(*) FROM runtime_recording_head").Scan(&count) != nil || count != 1 || q.QueryRow("SELECT ordinal,digest FROM runtime_recording_head WHERE singleton=1").Scan(&ordinal, &digest) != nil || ordinal > maxRecordingIntents || ordinal == 0 && digest != "" || ordinal > 0 && !recordingHex(digest, 64) {
		return 0, "", errRecordingHistory
	}
	return ordinal, digest, nil
}

func recordingReceiptDigest(i recordingIntent, ordinal uint64, previous string) (string, error) {
	return recordingStateDigest(struct {
		Domain   string          `json:"domain"`
		Ordinal  uint64          `json:"ordinal"`
		Previous string          `json:"previous_digest"`
		Intent   recordingIntent `json:"intent"`
	}{"omabit-recording-intent-v1", ordinal, previous, i})
}

func validateRecordingHistory(q querier) error {
	head, tail, err := recordingHead(q)
	if err != nil {
		return errRecordingHistory
	}
	var count, invalid int
	if err := q.QueryRow("SELECT count(*),coalesce(sum(length(body)>4096 OR length(id)!=32 OR length(operation_id)>128 OR length(digest)!=64 OR length(previous_digest)>64),0) FROM runtime_recording_intents").Scan(&count, &invalid); err != nil || count > maxRecordingIntents || invalid != 0 || uint64(count) != head {
		return errRecordingHistory
	}
	rows, err := q.Query("SELECT id,ordinal,previous_digest,digest FROM runtime_recording_intents ORDER BY ordinal")
	if err != nil {
		return err
	}
	type entry struct {
		id, previous, digest string
		ordinal              uint64
	}
	var items []entry
	previous := ""
	for rows.Next() {
		var item entry
		if err = rows.Scan(&item.id, &item.ordinal, &item.previous, &item.digest); err != nil || !recordingHex(item.id, 32) || item.ordinal != uint64(len(items)+1) || item.previous != previous {
			rows.Close()
			return errRecordingHistory
		}
		items = append(items, item)
		previous = item.digest
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	if previous != tail {
		return errRecordingHistory
	}
	for _, item := range items {
		if _, err := loadRecordingIntent(q, item.id); err != nil {
			return err
		}
	}
	return nil
}

func loadRecordingIntent(q querier, id string) (recordingIntent, error) {
	var i recordingIntent
	var opID, digest, previous string
	var ordinal uint64
	var raw []byte
	if !recordingHex(id, 32) {
		return i, errRecordingHistory
	}
	if err := q.QueryRow("SELECT operation_id,ordinal,previous_digest,digest,body FROM runtime_recording_intents WHERE id=? AND length(body)<=4096", id).Scan(&opID, &ordinal, &previous, &digest, &raw); err != nil {
		return i, err
	}
	if json.Unmarshal(raw, &i) != nil {
		return i, errRecordingHistory
	}
	canonical, err := i.bytes()
	want, digestErr := recordingReceiptDigest(i, ordinal, previous)
	if err != nil || digestErr != nil || !bytes.Equal(raw, canonical) || i.JobID != id || i.OperationID != opID || digest != want || ordinal == 0 || ordinal > maxRecordingIntents || ordinal == 1 && previous != "" || ordinal > 1 && !recordingHex(previous, 64) {
		return i, errRecordingHistory
	}
	op, err := loadOperation(q, opID)
	var runner string
	if err != nil || q.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&runner) != nil || !recordingOperationAncestry(i, op) || i.RunnerID != runner {
		return i, errRecordingHistory
	}
	// Historical lookup survives completed/interrupted operations and later
	// instance changes/deletion. It never infers current launch authority.
	return i, nil
}

func recordingOperationAncestry(i recordingIntent, op protocol.Operation) bool {
	return op.ID == i.OperationID && op.Actor == i.Actor && op.RequestID == i.RequestID && op.InstanceID == i.InstanceID && op.ProjectID == "" && op.ConfigRevision == i.ConfigurationRevision && (op.Kind == "start" || op.Kind == "startup-resume")
}

func recordingLiveAuthority(q querier, i recordingIntent, runner, selectedMode string) error {
	if i.Version == 2 {
		control, _, err := loadRecordingControl(q)
		if err != nil || control.Revision != i.ControlRevision || control.Digest != i.ControlDigest {
			return errRecordingControl
		}
	}
	op, err := loadOperation(q, i.OperationID)
	if err != nil || !recordingOperationAncestry(i, op) || op.Status != "running" || i.RunnerID != runner || i.Mode != selectedMode {
		return errRecordingHistory
	}
	v, err := loadInstance(q, i.InstanceID)
	stateDigest, digestErr := recordingStateDigest(v.Runtime)
	if err != nil || digestErr != nil || stateDigest != i.StateSHA256 || v.Public.ActiveOperationID != op.ID || v.Runtime.ID != i.RuntimeID || v.RuntimeRevision != i.RuntimeRevision || v.Public.ConfigRevision != i.ConfigurationRevision || v.RuntimeConfigRevision != i.ConfigurationRevision || !v.Runtime.Prepared || v.Runtime.Storage == nil || v.Runtime.Storage.Uncertain || v.Runtime.ImportRestore != nil || v.Runtime.Deletion != nil || v.Runtime.Handoff.Fenced() || v.Runtime.MoveTarget != "" || v.Runtime.Fake != nil && v.Runtime.Fake.Uncertain {
		return errRecordingHistory
	}
	selected, live := 0, 0
	for _, c := range v.Runtime.Runtime.Containers {
		if c.CreateUncertain {
			return errRecordingHistory
		}
		if !c.Removed {
			live++
		}
		if c.ID == i.ContainerID && c.CreateDispatched && !c.Removed && c.Kind != "probe" {
			selected++
		}
	}
	if selected != 1 || live != 1 {
		return errRecordingHistory
	}
	return nil
}

// checkRecordingIntent is a fresh, read-only launch fence. Historical replay is
// deliberately separate: an interrupted/completed intent remains inspectable but
// cannot acquire current operation/state authority through its saved receipt.
// The caller holds the ordinary lifecycle guard across the whole handoff.
func (s *store) checkRecordingIntent(ctx context.Context, i recordingIntent) error {
	if ctx.Err() != nil {
		return errRecordingHistory
	}
	raw, err := i.bytes()
	if err != nil || json.Unmarshal(raw, &i) != nil {
		return errRecordingHistory
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errRecordingHistory
	}
	defer tx.Rollback()
	if validateRecordingHistory(tx) != nil {
		return errRecordingHistory
	}
	saved, err := loadRecordingIntent(tx, i.JobID)
	if err != nil || !reflect.DeepEqual(saved, i) || recordingLiveAuthority(tx, i, s.runnerID, s.mode) != nil || ctx.Err() != nil {
		return errRecordingHistory
	}
	return nil
}

func (s *store) recordingSnapshotAuthority(ctx context.Context, expected protocol.Operation, snapshot runtimequalify.ManagedState) (storedInstance, error) {
	if ctx.Err() != nil || expected.ProjectID != "" || expected.Kind != "start" && expected.Kind != "startup-resume" && expected.Kind != "stop" {
		return storedInstance{}, errRecordingHistory
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storedInstance{}, errRecordingHistory
	}
	defer tx.Rollback()
	op, err := loadOperation(tx, expected.ID)
	if err != nil || !reflect.DeepEqual(op, expected) || op.Status != "running" {
		return storedInstance{}, errRecordingHistory
	}
	v, err := loadInstance(tx, op.InstanceID)
	actual, e1 := recordingStateDigest(v.Runtime)
	want, e2 := recordingStateDigest(snapshot)
	if err != nil || e1 != nil || e2 != nil || actual != want || v.Runtime.ID != snapshot.ID || v.Public.ActiveOperationID != op.ID || ctx.Err() != nil {
		return storedInstance{}, errRecordingHistory
	}
	return v, nil
}

func (s *store) recordingIntentForContainer(ctx context.Context, instance, runtime, container string) (*recordingIntent, error) {
	if ctx.Err() != nil || !protocol.ValidID(instance) || !recordingHex(runtime, 32) || !recordingHex(container, 64) {
		return nil, errRecordingHistory
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errRecordingHistory
	}
	defer tx.Rollback()
	if validateRecordingHistory(tx) != nil {
		return nil, errRecordingHistory
	}
	rows, err := tx.Query("SELECT id FROM runtime_recording_intents ORDER BY ordinal")
	if err != nil {
		return nil, errRecordingHistory
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, errRecordingHistory
		}
		ids = append(ids, id)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, errRecordingHistory
	}
	var selected *recordingIntent
	for _, id := range ids {
		i, err := loadRecordingIntent(tx, id)
		if err != nil {
			return nil, errRecordingHistory
		}
		if i.InstanceID == instance && i.RuntimeID == runtime && i.ContainerID == container && i.RunnerID == s.runnerID {
			if selected != nil {
				return nil, errRecordingHistory
			}
			selected = &i
		}
	}
	if ctx.Err() != nil {
		return nil, errRecordingHistory
	}
	return selected, nil
}

// admitRecordingIntent commits once before service submission. An exact saved
// replay returns history only, even after interruption; the caller must never
// turn that replay into a second submission or a new claim. No worker or native
// lifecycle command is invoked from this store layer.
func (s *store) admitRecordingIntent(i recordingIntent) (saved recordingIntent, replay bool, err error) {
	raw, err := i.bytes()
	if err != nil {
		return saved, false, err
	}
	// Normalize before hashing/comparing so accepted UTC serialization is exact.
	if json.Unmarshal(raw, &i) != nil {
		return saved, false, errRecordingHistory
	}
	tx, err := s.db.Begin()
	if err != nil {
		return saved, false, err
	}
	defer tx.Rollback()
	if err := validateRecordingHistory(tx); err != nil {
		return saved, false, err
	}
	var existing string
	err = tx.QueryRow("SELECT id FROM runtime_recording_intents WHERE id=? OR operation_id=?", i.JobID, i.OperationID).Scan(&existing)
	if err == nil {
		saved, err = loadRecordingIntent(tx, existing)
		if err != nil || !reflect.DeepEqual(saved, i) {
			return saved, true, errRecordingHistory
		}
		return saved, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return saved, false, err
	}
	if recordingLiveAuthority(tx, i, s.runnerID, s.mode) != nil {
		return saved, false, errRecordingHistory
	}
	var count int
	if tx.QueryRow("SELECT count(*) FROM runtime_recording_intents").Scan(&count) != nil || count >= maxRecordingIntents {
		return saved, false, errRecordingHistory
	}
	ordinal, previous, err := recordingHead(tx)
	if err != nil || ordinal >= maxRecordingIntents {
		return saved, false, errRecordingHistory
	}
	digest, err := recordingReceiptDigest(i, ordinal+1, previous)
	if err != nil {
		return saved, false, err
	}
	if _, err = tx.Exec("INSERT INTO runtime_recording_intents(id,operation_id,ordinal,previous_digest,digest,body) VALUES(?,?,?,?,?,?)", i.JobID, i.OperationID, ordinal+1, previous, digest, raw); err != nil {
		return saved, false, err
	}
	result, err := tx.Exec("UPDATE runtime_recording_head SET ordinal=?,digest=? WHERE singleton=1 AND ordinal=? AND digest=?", ordinal+1, digest, ordinal, previous)
	if err != nil {
		return saved, false, err
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return saved, false, errRecordingHistory
	}
	return i, false, tx.Commit()
}
