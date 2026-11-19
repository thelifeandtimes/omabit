package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

const recordingControlDDL = `CREATE TABLE runtime_recording_control_events (id TEXT PRIMARY KEY, ordinal INTEGER NOT NULL UNIQUE CHECK(ordinal>=1 AND ordinal<=4096), actor TEXT NOT NULL, request_id TEXT NOT NULL, previous_digest TEXT NOT NULL, digest TEXT NOT NULL UNIQUE, body BLOB NOT NULL, UNIQUE(actor,request_id))`
const recordingControlHeadDDL = `CREATE TABLE runtime_recording_control_head (singleton INTEGER PRIMARY KEY CHECK(singleton=1), ordinal INTEGER NOT NULL CHECK(ordinal>=0 AND ordinal<=4096), digest TEXT NOT NULL)`
const maxRecordingControlBytes = 16 << 10

var errRecordingControl = errors.New("recording policy/pin history is unavailable or changed; preserve original control receipts")

func controlEventBytes(e recording.ControlEvent) ([]byte, error) {
	if e.Version != 1 || !protocol.ValidID(e.ID) || e.Request.Validate() != nil || e.RunnerID != e.Request.RunnerID || e.Actor == "" || len(e.Actor) > 128 || e.CreatedAt.Before(time.Unix(0, 1)) || e.CreatedAt.After(time.Unix(0, 1<<63-1)) {
		return nil, errRecordingControl
	}
	for _, r := range e.Actor {
		if r < 33 || r > 126 {
			return nil, errRecordingControl
		}
	}
	e.CreatedAt = e.CreatedAt.UTC()
	raw, err := json.Marshal(e)
	if err != nil || len(raw) > maxRecordingControlBytes {
		return nil, errRecordingControl
	}
	return raw, nil
}
func controlEventDigest(e recording.ControlEvent, ordinal uint64, previous string) (string, error) {
	return recordingStateDigest(struct {
		Domain   string                 `json:"domain"`
		Ordinal  uint64                 `json:"ordinal"`
		Previous string                 `json:"previous_digest"`
		Event    recording.ControlEvent `json:"event"`
	}{"omabit-recording-control-v1", ordinal, previous, e})
}

func initializeRecordingControl(tx *sql.Tx, version int) error {
	for _, table := range []struct{ name, ddl string }{{"runtime_recording_control_events", recordingControlDDL}, {"runtime_recording_control_head", recordingControlHeadDDL}} {
		if version < 65 {
			var count int
			if tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name=?", table.name).Scan(&count) != nil || count != 0 {
				return errRecordingControl
			}
			if _, err := tx.Exec(table.ddl); err != nil {
				return err
			}
		}
		var ddl string
		var triggers int
		if tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table.name).Scan(&ddl) != nil || ddl != table.ddl || tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name=?", table.name).Scan(&triggers) != nil || triggers != 0 {
			return errRecordingControl
		}
	}
	if version < 65 {
		if _, err := tx.Exec("INSERT INTO runtime_recording_control_head VALUES(1,0,'')"); err != nil {
			return err
		}
	}
	_, _, err := loadRecordingControl(tx)
	return err
}

func loadRecordingControl(q querier) (recording.ControlState, []recording.ControlReceipt, error) {
	var runner string
	var count, headCount, invalid int
	var head uint64
	var tail string
	if q.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&runner) != nil || !protocol.ValidID(runner) || q.QueryRow("SELECT count(*) FROM runtime_recording_control_head").Scan(&headCount) != nil || headCount != 1 || q.QueryRow("SELECT ordinal,digest FROM runtime_recording_control_head WHERE singleton=1").Scan(&head, &tail) != nil || head > 4096 || head == 0 && tail != "" || head > 0 && !recordingHex(tail, 64) || q.QueryRow("SELECT count(*),coalesce(sum(length(body)>16384 OR length(id)>128 OR length(actor)>128 OR length(request_id)>128 OR length(previous_digest)>64 OR length(digest)!=64),0) FROM runtime_recording_control_events").Scan(&count, &invalid) != nil || invalid != 0 || uint64(count) != head {
		return recording.ControlState{}, nil, errRecordingControl
	}
	state := recording.ControlState{Version: 1, RunnerID: runner, Policy: recording.DefaultRetentionPolicy(), Pins: []recording.IncidentPin{}, Basis: "saved recording controls only; no retirement, raw availability, current health or filesystem effect is implied"}
	rows, err := q.Query("SELECT id,ordinal,actor,request_id,previous_digest,digest,body FROM runtime_recording_control_events ORDER BY ordinal")
	if err != nil {
		return state, nil, err
	}
	type saved struct {
		id, actor, request, previous, digest string
		ordinal                              uint64
		raw                                  []byte
	}
	var entries []saved
	for rows.Next() {
		var r saved
		if err := rows.Scan(&r.id, &r.ordinal, &r.actor, &r.request, &r.previous, &r.digest, &r.raw); err != nil {
			rows.Close()
			return state, nil, err
		}
		if len(r.raw) > maxRecordingControlBytes {
			rows.Close()
			return state, nil, errRecordingControl
		}
		entries = append(entries, r)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return state, nil, err
	}
	var receipts []recording.ControlReceipt
	for _, row := range entries {
		var e recording.ControlEvent
		if json.Unmarshal(row.raw, &e) != nil {
			return state, nil, errRecordingControl
		}
		canonical, err := controlEventBytes(e)
		digest, de := controlEventDigest(e, row.ordinal, row.previous)
		if err != nil || de != nil || !bytes.Equal(canonical, row.raw) || row.id != e.ID || row.actor != e.Actor || row.request != e.Request.RequestID || row.ordinal != state.Revision+1 || row.previous != state.Digest || row.digest != digest || e.RunnerID != runner || e.Request.ExpectedRevision != state.Revision || e.Request.ExpectedDigest != state.Digest {
			return state, nil, errRecordingControl
		}
		if e.Request.Kind != "configure" {
			i, err := loadRecordingIntent(q, e.Request.JobID)
			if err != nil || i.JobSHA256 != e.Request.JobSHA256 || i.RunnerID != runner {
				return state, nil, errRecordingControl
			}
		}
		switch e.Request.Kind {
		case "configure":
			state.Policy = *e.Request.Policy
		case "pin":
			found := false
			for _, p := range state.Pins {
				if p.JobID == e.Request.JobID {
					found = true
				}
			}
			if found || len(state.Pins) >= 256 {
				return state, nil, errRecordingControl
			}
			state.Pins = append(append([]recording.IncidentPin{}, state.Pins...), recording.IncidentPin{JobID: e.Request.JobID, JobSHA256: e.Request.JobSHA256, Reason: e.Request.PinReason, PinnedAt: e.CreatedAt})
			sort.Slice(state.Pins, func(i, j int) bool { return state.Pins[i].JobID < state.Pins[j].JobID })
		case "unpin":
			found := false
			pins := []recording.IncidentPin{}
			for _, p := range state.Pins {
				if p.JobID == e.Request.JobID {
					found = true
					continue
				}
				pins = append(pins, p)
			}
			if !found {
				return state, nil, errRecordingControl
			}
			state.Pins = pins
		}
		state.Revision, state.Digest = row.ordinal, row.digest
		receipts = append(receipts, recording.ControlReceipt{Event: e, State: state})
	}
	if state.Revision != head || state.Digest != tail {
		return state, nil, errRecordingControl
	}
	return state, receipts, nil
}

func (s *store) recordingControls(ctx context.Context) (recording.ControlState, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return recording.ControlState{}, err
	}
	defer tx.Rollback()
	if validateRecordingHistory(tx) != nil {
		return recording.ControlState{}, errRecordingControl
	}
	state, _, err := loadRecordingControl(tx)
	return state, err
}

func (s *store) applyRecordingControl(ctx context.Context, actor string, in recording.ControlRequest) (recording.ControlReceipt, bool, error) {
	if in.Validate() != nil || in.RunnerID != s.runnerID {
		return recording.ControlReceipt{}, false, errRecordingControl
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	defer tx.Rollback()
	if validateRecordingHistory(tx) != nil {
		return recording.ControlReceipt{}, false, errRecordingControl
	}
	state, receipts, err := loadRecordingControl(tx)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	for _, receipt := range receipts {
		if receipt.Event.Actor == actor && receipt.Event.Request.RequestID == in.RequestID {
			if !reflect.DeepEqual(receipt.Event.Request, in) {
				return recording.ControlReceipt{}, true, errRecordingControl
			}
			return receipt, true, nil
		}
	}
	if state.Revision >= 4096 || state.Revision != in.ExpectedRevision || state.Digest != in.ExpectedDigest {
		return recording.ControlReceipt{}, false, errRecordingControl
	}
	id, err := randomID()
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	e := recording.ControlEvent{Version: 1, ID: "recordctl-" + id, RunnerID: s.runnerID, Actor: actor, Request: in, CreatedAt: time.Now().UTC()}
	raw, err := controlEventBytes(e)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	digest, err := controlEventDigest(e, state.Revision+1, state.Digest)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO runtime_recording_control_events(id,ordinal,actor,request_id,previous_digest,digest,body) VALUES(?,?,?,?,?,?,?)", e.ID, state.Revision+1, actor, in.RequestID, state.Digest, digest, raw); err != nil {
		return recording.ControlReceipt{}, false, err
	}
	result, err := tx.ExecContext(ctx, "UPDATE runtime_recording_control_head SET ordinal=?,digest=? WHERE singleton=1 AND ordinal=? AND digest=?", state.Revision+1, digest, state.Revision, state.Digest)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return recording.ControlReceipt{}, false, errRecordingControl
	}
	_, receipts, err = loadRecordingControl(tx)
	if err != nil {
		return recording.ControlReceipt{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return recording.ControlReceipt{}, false, err
	}
	return receipts[len(receipts)-1], false, nil
}
