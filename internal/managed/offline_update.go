package managed

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// OfflineStateLock opens only existing state. Its .lock is exactly the flock
// used by runtimequalify.OpenManagedHost, not a second installer-only lock.
// Close releases this descriptor without LOCK_UN: a credential-dropped helper
// inherits the same open file description and must not unlock its parent.
type OfflineStateLock struct {
	directory string
	uid       uint32
	file      *os.File
	dirInfo   os.FileInfo
}

func LockOfflineState(directory string, uid uint32) (*OfflineStateLock, error) {
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || resolved != directory {
		return nil, errors.New("existing clean managed state directory required")
	}
	info, err := os.Lstat(filepath.Join(directory, ".lock"))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !offlineOwned(info, uid, true) {
		return nil, errors.New("existing owner-private regular runtime lock required")
	}
	f, err := os.OpenFile(filepath.Join(directory, ".lock"), os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("existing managed ownership lock required: %w", err)
	}
	l, err := AdoptOfflineStateLock(directory, uid, f)
	if err != nil {
		f.Close()
	}
	return l, err
}

// AdoptOfflineStateLock is used only with the installer's inherited descriptor.
func AdoptOfflineStateLock(directory string, uid uint32, f *os.File) (*OfflineStateLock, error) {
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || resolved != directory {
		return nil, errors.New("offline state directory must be existing, absolute and without symlinks")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !offlineOwned(info, uid, false) {
		return nil, errors.New("offline state directory ownership or privacy mismatch")
	}
	l := &OfflineStateLock{directory: directory, uid: uid, file: f, dirInfo: info}
	if err := l.Verify(); err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, errors.New("managed ownership lock is busy; a runner or another offline upgrade still owns state")
	}
	if err := l.Verify(); err != nil {
		return nil, err
	}
	return l, nil
}

func offlineOwned(info os.FileInfo, uid uint32, single bool) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uid && (!single || st.Nlink == 1)
}

func (l *OfflineStateLock) File() *os.File    { return l.file }
func (l *OfflineStateLock) Directory() string { return l.directory }
func (l *OfflineStateLock) Close() error      { return l.file.Close() }

func (l *OfflineStateLock) Verify() error {
	dir, err := os.Lstat(l.directory)
	if err != nil || !os.SameFile(l.dirInfo, dir) || dir.Mode().Perm() != 0700 || !offlineOwned(dir, l.uid, false) {
		return errors.New("managed state directory changed")
	}
	path := filepath.Join(l.directory, ".lock")
	info, err := os.Lstat(path)
	actual, fdErr := l.file.Stat()
	if err != nil || fdErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !offlineOwned(info, l.uid, true) || !os.SameFile(info, actual) {
		return errors.New("managed ownership lock identity changed")
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, err := os.Lstat(filepath.Join(l.directory, "runner.sqlite") + suffix)
		if suffix != "" && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !offlineOwned(info, l.uid, true) {
			return errors.New("existing owner-private SQLite database and sidecars required; no state will be created or adopted")
		}
	}
	return nil
}

type OfflineUpdateRequest struct {
	Action    string `json:"action"`
	Directory string `json:"directory"`
	RunnerID  string `json:"runner_id,omitempty"`
	Token     string `json:"token,omitempty"`
	Schema    int    `json:"schema,omitempty"`
}

type OfflineUpdateState struct {
	RunnerID string `json:"runner_id"`
	Schema   int    `json:"schema"`
	Quiesced bool   `json:"quiesced"`
	Token    string `json:"token,omitempty"`
}

func (l *OfflineStateLock) openDB(writable bool) (*sql.DB, error) {
	if err := l.Verify(); err != nil {
		return nil, err
	}
	// Database work runs as its existing Unix owner, including WAL/shm creation.
	if uint32(os.Geteuid()) != l.uid {
		return nil, errors.New("offline SQLite inspection/migration must run as the existing state owner")
	}
	u := url.URL{Scheme: "file", Path: filepath.Join(l.directory, "runner.sqlite")}
	q := url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "busy_timeout(5000)"}}
	if writable {
		q = url.Values{"mode": {"rw"}, "_pragma": {"synchronous(FULL)", "foreign_keys(1)", "busy_timeout(5000)"}, "_txlock": {"immediate"}}
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func inspectOfflineIdentity(q querier) (OfflineUpdateState, error) {
	var state OfflineUpdateState
	if err := q.QueryRow("PRAGMA user_version").Scan(&state.Schema); err != nil {
		return state, err
	}
	if state.Schema != 19 && (state.Schema != StateSchemaVersion || state.Schema < 20) {
		return state, errors.New("offline upgrade supports original schema19 or its journaled current-schema continuation only; unknown/future schema refused")
	}
	var check string
	if err := q.QueryRow("PRAGMA quick_check").Scan(&check); err != nil || check != "ok" {
		return state, errors.New("offline database integrity check failed")
	}
	if err := q.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&state.RunnerID); err != nil {
		return state, err
	}
	id, err := hex.DecodeString(strings.TrimPrefix(state.RunnerID, "runner-"))
	if err != nil || len(id) != 16 || len(state.RunnerID) != 39 || !strings.HasPrefix(state.RunnerID, "runner-") {
		return state, errors.New("offline database has no valid stable runner identity")
	}
	var fenceData string
	err = q.QueryRow("SELECT value FROM meta WHERE key=?", managementFenceKey).Scan(&fenceData)
	if err == nil {
		var f managementFence
		if json.Unmarshal([]byte(fenceData), &f) != nil || !updateDigest(f.Token) {
			return state, errors.New("invalid retained management fence")
		}
		state.Quiesced, state.Token = f.Held, f.Token
	} else if !errors.Is(err, sql.ErrNoRows) {
		return state, err
	}
	return state, nil
}

func inspectOfflineState(q querier) (OfflineUpdateState, error) {
	state, err := inspectOfflineIdentity(q)
	if err != nil {
		return state, err
	}
	all, err := instances(q)
	if err != nil {
		return state, err
	}
	for _, v := range all {
		if _, err := loadRoutinePolicy(q, v.Public.ID); err != nil {
			return state, err
		}
		r := v.Runtime
		// A committed, permanently fenced source is a completed transfer, not
		// unfinished work. Exclude only that proven terminal marker from this
		// read-only uncertainty check; retain its original stored bytes/fence.
		if h := r.Handoff; h != nil && h.Direction == "out" && h.State == "moved" && h.SourceRunner == state.RunnerID && h.SourceInstance == v.Public.ID && v.Public.DesiredState == "stopped" && h.ValidCommitment(h.Commitment) {
			r.Handoff = nil
		}
		busy := v.Public.ActiveOperationID != "" || v.Public.RecoveryRequired || r.UncertainProvisioning() || r.UncertainMaintenance() || r.MoveTarget != "" || r.PruneIntent != nil || r.StartupResume != nil || r.Maintenance != nil && r.Maintenance.FinishedAt == nil || r.Routine != nil && r.Routine.FinishedAt == nil
		for _, c := range r.Runtime.Containers {
			liveKind := c.Kind == "http-boot" || c.Kind == "http-restart" || c.Kind == "comet-boot" || c.Kind == "comet-resume" || c.Kind == "comet-verify"
			busy = busy || c.CreateUncertain || !c.Removed && (!liveKind || c.ID == "")
		}
		if n := r.Runtime.Network; n != nil {
			busy = busy || n.CreateUncertain
		}
		if busy {
			return state, fmt.Errorf("instance %s has active/unresolved work; resolve it with the original manager before stopping it; no work was canceled", v.Public.ID)
		}
	}
	rows, err := q.Query("SELECT id,body FROM operations")
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var body []byte
		var op protocol.Operation
		if err := rows.Scan(&id, &body); err != nil {
			return state, err
		}
		if json.Unmarshal(body, &op) != nil || op.ID != id {
			return state, errors.New("invalid operation journal")
		}
		switch op.Status {
		case "succeeded", "failed", "interrupted", "skipped":
			if op.FinishedAt == nil {
				return state, errors.New("unfinished operation receipt prevents offline upgrade")
			}
		default:
			return state, errors.New("queued, running or unrecognized operation prevents offline upgrade; finish/reconcile with original manager")
		}
	}
	return state, rows.Err()
}

// Inspect never reconciles jobs, bootstraps a database or changes user_version.
func (l *OfflineStateLock) Inspect(ctx context.Context) (OfflineUpdateState, error) {
	db, err := l.openDB(false)
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer tx.Rollback()
	return inspectOfflineState(tx)
}

// InspectContinuation is read-only forward recovery for an already migrated
// journal. A released token may have admitted work before the manager exited;
// leave those receipts/runtime intents untouched for normal server startup
// reconciliation. A held fence still requires idle state: it cannot legitimately
// have admitted new jobs. Neither case reacquires, clears or replaces a token.
func (l *OfflineStateLock) InspectContinuation(ctx context.Context, runnerID, token string, schema int) (OfflineUpdateState, error) {
	if schema < 20 || schema != StateSchemaVersion || !updateDigest(token) {
		return OfflineUpdateState{}, errors.New("forward recovery requires the exact current schema and retained journal token")
	}
	db, err := l.openDB(false)
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer tx.Rollback()
	state, err := inspectOfflineIdentity(tx)
	if err != nil {
		return state, err
	}
	if state.Schema != schema || state.RunnerID != runnerID || state.Token != token {
		return state, errors.New("forward recovery runner/schema/token does not match the activation journal")
	}
	if state.Quiesced {
		return inspectOfflineState(tx)
	}
	return state, nil
}

// Fence commits the ordinary versioned migration and fence in ONE transaction.
// The root installer must durably retain its activation journal first. No
// lifecycle reconciliation is run and instance/operation bodies are not rewritten.
func (l *OfflineStateLock) Fence(ctx context.Context, runnerID, token string) (OfflineUpdateState, error) {
	if StateSchemaVersion < 20 || !updateDigest(token) {
		return OfflineUpdateState{}, errors.New("offline activation requires schema20+ and a valid journal token")
	}
	db, err := l.openDB(true)
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return OfflineUpdateState{}, err
	}
	defer tx.Rollback()
	state, err := inspectOfflineState(tx)
	if err != nil {
		return state, err
	}
	if state.RunnerID != runnerID {
		return state, errors.New("offline runner identity changed")
	}
	if state.Schema >= 20 && (!state.Quiesced || state.Token != token) || state.Schema == 19 && state.Token != "" {
		return state, errors.New("offline continuation requires the exact held journal fence")
	}
	s := &store{db: db, directory: l.directory}
	if err := s.initializeTransaction(tx); err != nil {
		return state, err
	}
	data, _ := json.Marshal(managementFence{Token: token, Held: true})
	if _, err := tx.Exec("INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", managementFenceKey, string(data)); err != nil {
		return state, err
	}
	if err := l.Verify(); err != nil {
		return state, err
	}
	if err := tx.Commit(); err != nil {
		return state, err
	}
	state.Schema, state.Quiesced, state.Token = StateSchemaVersion, true, token
	return state, nil
}
