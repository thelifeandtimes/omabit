// Package managed provides the development-only durable fake-ship control plane.
package managed

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	_ "modernc.org/sqlite"
)

const mode = "development-managed-fake"

type storedInstance struct {
	Public                protocol.Instance           `json:"public"`
	Runtime               runtimequalify.ManagedState `json:"runtime"`
	RuntimeRevision       uint64                      `json:"runtime_revision"`
	RuntimeConfigRevision uint64                      `json:"runtime_config_revision"`
}

type store struct {
	db       *sql.DB
	runnerID string
}

type fault struct {
	status  int
	problem protocol.APIError
}

func (f *fault) Error() string { return f.problem.Message }
func conflict(code, message string) error {
	return &fault{409, protocol.APIError{Code: code, Message: message}}
}
func bad(message string) error {
	return &fault{400, protocol.APIError{Code: "invalid_request", Message: message}}
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func openStore(dir string) (*store, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, errors.New("state directory must be absolute and clean")
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || resolved != dir {
		return nil, errors.New("state directory must not contain symlinks")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, errors.New("state directory must be private 0700")
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) {
		return nil, errors.New("state directory must belong to this account")
	}
	path := filepath.Join(dir, "runner.sqlite")
	for _, name := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		info, err := os.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return nil, errors.New("unsafe state database or sidecar")
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 {
			return nil, errors.New("unsafe database ownership or link")
		}
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path}
	values := url.Values{"_pragma": {"synchronous(FULL)", "foreign_keys(1)", "busy_timeout(5000)"}, "_txlock": {"immediate"}}
	u.RawQuery = values.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &store{db: db}
	if err := s.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	var journal string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journal); err != nil || journal != "wal" {
		db.Close()
		return nil, errors.New("cannot enable the durable WAL journal")
	}
	return s, nil
}

func (s *store) initialize() error {
	var check string
	if err := s.db.QueryRow("PRAGMA quick_check").Scan(&check); err != nil || check != "ok" {
		return errors.New("state database integrity check failed")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version == 0 {
		var count int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("refusing unknown existing database schema")
		}
		id, err := randomID()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE instances (id TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE operations (id TEXT PRIMARY KEY, actor TEXT NOT NULL, request_id TEXT NOT NULL,
 fingerprint TEXT NOT NULL, body BLOB NOT NULL, runtime BLOB,
 UNIQUE(actor, request_id));
PRAGMA user_version=1;`); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO meta VALUES ('runner_id', ?)", "runner-"+id); err != nil {
			return err
		}
	} else if version != 1 {
		return errors.New("unsupported state database schema")
	}
	if err := tx.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&s.runnerID); err != nil {
		return errors.New("missing stable runner identity")
	}
	if !strings.HasPrefix(s.runnerID, "runner-") || len(s.runnerID) != 39 {
		return errors.New("invalid stable runner identity")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(s.runnerID, "runner-")); err != nil {
		return errors.New("invalid stable runner identity")
	}
	return tx.Commit()
}

type querier interface {
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
}

func loadInstance(q querier, id string) (storedInstance, error) {
	var data []byte
	if err := q.QueryRow("SELECT body FROM instances WHERE id=?", id).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedInstance{}, &fault{404, protocol.APIError{Code: "instance_not_found", Message: "unknown registered instance"}}
		}
		return storedInstance{}, err
	}
	var value storedInstance
	if json.Unmarshal(data, &value) != nil || value.Public.ID != id || !protocol.ValidID(id) || value.Public.ConfigRevision == 0 || len(value.Runtime.ID) != 32 {
		return value, errors.New("invalid stored instance")
	}
	return value, nil
}

func saveInstance(tx *sql.Tx, value storedInstance) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE instances SET body=? WHERE id=?", data, value.Public.ID)
	return err
}

func loadOperation(q querier, id string) (protocol.Operation, error) {
	var data []byte
	if err := q.QueryRow("SELECT body FROM operations WHERE id=?", id).Scan(&data); err != nil {
		return protocol.Operation{}, err
	}
	var op protocol.Operation
	if json.Unmarshal(data, &op) != nil || op.ID != id {
		return op, errors.New("invalid stored operation")
	}
	return op, nil
}

func saveOperation(tx *sql.Tx, op protocol.Operation) error {
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE operations SET body=? WHERE id=?", data, op.ID)
	return err
}

func replay(tx *sql.Tx, actor, request, fingerprint string) (protocol.Operation, bool, error) {
	var previous string
	var data []byte
	err := tx.QueryRow("SELECT fingerprint, body FROM operations WHERE actor=? AND request_id=?", actor, request).Scan(&previous, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, nil
	}
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if previous != fingerprint {
		return protocol.Operation{}, true, conflict("request_conflict", "request_id already identifies different input for this actor")
	}
	var op protocol.Operation
	if json.Unmarshal(data, &op) != nil {
		return op, true, errors.New("invalid stored replay")
	}
	return op, true, nil
}

func insertOperation(tx *sql.Tx, op protocol.Operation, fingerprint string) error {
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body) VALUES(?,?,?,?,?)", op.ID, op.Actor, op.RequestID, fingerprint, data)
	return err
}

func instances(q querier) ([]storedInstance, error) {
	rows, err := q.Query("SELECT id,body FROM instances ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedInstance
	for rows.Next() {
		var id string
		var data []byte
		var value storedInstance
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		if json.Unmarshal(data, &value) != nil || value.Public.ID != id || !protocol.ValidID(id) {
			return nil, errors.New("invalid instance inventory")
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func validateConfig(c protocol.Configuration) error {
	if c.HTTPPort < 1024 || c.AmesPort != 34343 || c.Loom != 31 {
		return bad("managed-fake requires HTTP port 1024..65535, loom 31, and reserved Ames value 34343 (Ames is not published)")
	}
	return nil
}

func reservePort(tx *sql.Tx, id string, port uint16) error {
	all, err := instances(tx)
	if err != nil {
		return err
	}
	for _, v := range all {
		if v.Public.ID == id {
			continue
		}
		if v.Public.Config.HTTPPort == port {
			return conflict("port_conflict", "HTTP port is reserved by another registered instance")
		}
		if v.Runtime.Port == port {
			for _, c := range v.Runtime.Runtime.Containers {
				if !c.Removed {
					return conflict("port_conflict", "HTTP port is still assigned to another instance's runtime")
				}
			}
		}
	}
	return nil
}

func (s *store) register(actor string, req protocol.CreateFakeRequest) (protocol.Operation, bool, error) {
	if !protocol.ValidID(req.ID) || !protocol.ValidID(req.RequestID) {
		return protocol.Operation{}, false, bad("valid instance and request IDs are required")
	}
	if err := validateConfig(req.Config); err != nil {
		return protocol.Operation{}, false, err
	}
	if req.Label == "" {
		req.Label = req.ID
	}
	if !utf8.ValidString(req.Label) || len(req.Label) > 128 || strings.IndexFunc(req.Label, unicode.IsControl) >= 0 {
		return protocol.Operation{}, false, bad("label must be at most 128 UTF-8 bytes without control characters")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind    string
		Request protocol.CreateFakeRequest
	}{"register-fake", req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(fingerprint)); found || err != nil {
		return op, true, err
	}
	var count int
	if err := tx.QueryRow("SELECT count(*) FROM instances WHERE id=?", req.ID).Scan(&count); err != nil {
		return protocol.Operation{}, false, err
	}
	if count != 0 {
		return protocol.Operation{}, false, conflict("instance_exists", "instance ID is already registered")
	}
	if err := tx.QueryRow("SELECT count(*) FROM instances").Scan(&count); err != nil {
		return protocol.Operation{}, false, err
	}
	if count >= 8 {
		return protocol.Operation{}, false, conflict("development_capacity", "this development runner supports at most eight registered fake instances")
	}
	if err := reservePort(tx, req.ID, req.Config.HTTPPort); err != nil {
		return protocol.Operation{}, false, err
	}
	uuid, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	v := storedInstance{Public: protocol.Instance{ID: req.ID, Label: req.Label, Config: req.Config, ConfigRevision: 1, Mode: "fake", Ship: "~zod", DesiredState: "stopped", ObservedState: "uninitialized", Health: "unknown"}, Runtime: runtimequalify.ManagedState{ID: uuid, Port: req.Config.HTTPPort}}
	data, err := json.Marshal(v)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err := tx.Exec("INSERT INTO instances VALUES(?,?)", req.ID, data); err != nil {
		return protocol.Operation{}, false, err
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	now := time.Now().UTC()
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, InstanceID: req.ID, Kind: "register-fake", Actor: actor, Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}
	if err := insertOperation(tx, op, string(fingerprint)); err != nil {
		return op, false, err
	}
	return op, false, tx.Commit()
}

func (s *store) configure(actor, id string, req protocol.ConfigureRequest) (protocol.Instance, error) {
	if req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 {
		return protocol.Instance{}, bad("a valid positive configuration revision is required")
	}
	if err := validateConfig(req.Config); err != nil {
		return protocol.Instance{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Instance{}, err
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, id)
	if err != nil {
		return v.Public, err
	}
	if v.Public.ConfigRevision != req.ExpectedRevision {
		return v.Public, &fault{409, protocol.APIError{Code: "revision_conflict", Message: "configuration changed; inspect a fresh snapshot", CurrentRevision: &v.Public.ConfigRevision}}
	}
	if v.Public.ActiveOperationID != "" {
		return v.Public, conflict("instance_busy", "configuration cannot change during an operation")
	}
	if err := reservePort(tx, id, req.Config.HTTPPort); err != nil {
		return v.Public, err
	}
	if v.Public.Config != req.Config {
		if v.Public.ConfigRevision == math.MaxInt64 {
			return v.Public, conflict("revision_limit", "configuration revision exhausted")
		}
		v.Public.Config, v.Public.ConfigRevision = req.Config, v.Public.ConfigRevision+1
		if err := saveInstance(tx, v); err != nil {
			return v.Public, err
		}
		id, err := randomID()
		if err != nil {
			return v.Public, err
		}
		now := time.Now().UTC()
		op := protocol.Operation{ID: "op-" + id, RequestID: "server-config-" + id, InstanceID: v.Public.ID, Kind: "configure", Actor: actor, Status: "succeeded", ConfigRevision: v.Public.ConfigRevision, CreatedAt: now, FinishedAt: &now}
		fingerprint, err := json.Marshal(req)
		if err != nil {
			return v.Public, err
		}
		if err := insertOperation(tx, op, "configure:"+v.Public.ID+":"+string(fingerprint)); err != nil {
			return v.Public, err
		}
	}
	return v.Public, tx.Commit()
}

func (s *store) accept(actor, id string, req protocol.OperationRequest) (protocol.Operation, bool, error) {
	if !protocol.ValidID(id) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 || (req.Kind != "start" && req.Kind != "stop") {
		return protocol.Operation{}, false, bad("require start/stop, valid request ID, and positive revision")
	}
	fingerprint, _ := json.Marshal(struct {
		ID      string
		Request protocol.OperationRequest
	}{id, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(fingerprint)); found || err != nil {
		return op, true, err
	}
	v, err := loadInstance(tx, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if v.Public.ConfigRevision != req.ExpectedRevision {
		return protocol.Operation{}, false, &fault{409, protocol.APIError{Code: "revision_conflict", Message: "configuration changed; inspect a fresh snapshot", CurrentRevision: &v.Public.ConfigRevision}}
	}
	if v.Public.ActiveOperationID != "" {
		return protocol.Operation{}, false, conflict("instance_busy", "an operation is already active")
	}
	if req.Kind == "start" && (v.Public.RecoveryRequired || v.Public.ObservedState == "running") {
		return protocol.Operation{}, false, conflict("invalid_state", "inspect/reconcile or stop the existing runtime before starting")
	}
	if req.Kind == "start" {
		for _, c := range v.Runtime.Runtime.Containers {
			if !c.Removed || c.CreateUncertain {
				return protocol.Operation{}, false, conflict("invalid_state", "stop or reconcile the retained runtime before starting")
			}
		}
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, InstanceID: id, Kind: req.Kind, Actor: actor, Status: "queued", ConfigRevision: req.ExpectedRevision, CreatedAt: time.Now().UTC()}
	v.Public.ActiveOperationID = op.ID
	if req.Kind == "start" {
		v.Public.DesiredState = "running"
	} else {
		v.Public.DesiredState = "stopped"
	}
	if err := insertOperation(tx, op, string(fingerprint)); err != nil {
		return op, false, err
	}
	if err := saveInstance(tx, v); err != nil {
		return op, false, err
	}
	return op, false, tx.Commit()
}

func (s *store) snapshot() (protocol.Snapshot, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Snapshot{}, err
	}
	defer tx.Rollback()
	all, err := instances(tx)
	if err != nil {
		return protocol.Snapshot{}, err
	}
	result := protocol.Snapshot{RunnerID: s.runnerID, Mode: mode, Instances: []protocol.Instance{}, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{}}
	for _, v := range all {
		if v.Public.ObservedAt != nil && time.Since(*v.Public.ObservedAt) > 30*time.Second {
			v.Public.Health = "unknown"
		}
		result.Instances = append(result.Instances, v.Public)
	}
	rows, err := tx.Query("SELECT body FROM operations ORDER BY rowid DESC LIMIT 128")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var data []byte
		var op protocol.Operation
		if err := rows.Scan(&data); err != nil {
			rows.Close()
			return result, err
		}
		if err := json.Unmarshal(data, &op); err != nil {
			rows.Close()
			return result, err
		}
		result.Operations = append(result.Operations, op)
	}
	err = errors.Join(rows.Err(), rows.Close())
	return result, err
}

// Each resource intent and the matching journal entry commit together. Callbacks
// are independent of a canceled client/worker context so the outcome is recorded.
func (s *store) runtimeUpdate(id, opID string, state runtimequalify.ManagedState) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, id)
	if err != nil {
		return err
	}
	if v.Public.ActiveOperationID != opID || v.Runtime.ID != state.ID {
		return errors.New("operation no longer owns instance runtime")
	}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	if err := saveInstance(tx, v); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, opID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *store) beginOperation(opID string) (protocol.Operation, storedInstance, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, storedInstance{}, err
	}
	defer tx.Rollback()
	op, err := loadOperation(tx, opID)
	if err != nil {
		return op, storedInstance{}, err
	}
	v, err := loadInstance(tx, op.InstanceID)
	if err != nil {
		return op, v, err
	}
	if op.Status != "queued" || v.Public.ActiveOperationID != op.ID {
		return op, v, errors.New("operation is not queued for this instance")
	}
	op.Status = "running"
	if op.Kind == "start" {
		v.Public.ObservedState = "starting"
		v.Runtime.Port = v.Public.Config.HTTPPort
		v.RuntimeConfigRevision = op.ConfigRevision
	} else {
		v.Public.ObservedState = "stopping"
	}
	v.Public.Health = "unknown"
	if err := saveOperation(tx, op); err != nil {
		return op, v, err
	}
	if err := saveInstance(tx, v); err != nil {
		return op, v, err
	}
	return op, v, tx.Commit()
}

func (s *store) finish(op protocol.Operation, state runtimequalify.ManagedState, observation runtimequalify.ManagedObservation, cause error, interrupted bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, op.InstanceID)
	if err != nil {
		return err
	}
	if v.Public.ActiveOperationID != op.ID || v.Runtime.ID != state.ID {
		return errors.New("operation ownership changed")
	}
	now := time.Now().UTC()
	op.Status, op.FinishedAt = "succeeded", &now
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	v.Public.ActiveOperationID = ""
	applyObservation(&v, observation, now)
	if cause != nil {
		op.Status, op.Message = "failed", "runtime action failed: "+cause.Error()
		if interrupted {
			op.Status = "interrupted"
		}
		if len(op.Message) > 1024 {
			op.Message = op.Message[:1024]
		}
		v.Public.ObservedState, v.Public.Health, v.Public.RecoveryRequired = "unknown", "unknown", true
	} else if op.Kind == "start" {
		v.Public.AppliedRevision = op.ConfigRevision
	}
	if err := saveInstance(tx, v); err != nil {
		return err
	}
	if err := saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}

func applyObservation(v *storedInstance, observation runtimequalify.ManagedObservation, now time.Time) {
	v.Public.ObservedState, v.Public.Health, v.Public.Endpoint = observation.State, observation.Health, observation.Endpoint
	v.Public.RecoveryRequired, v.Public.ObservedAt = observation.RecoveryRequired, &now
	if v.Runtime.Initialized && observation.State == "running" && observation.Health == "healthy" {
		v.Public.AppliedRevision = v.RuntimeConfigRevision
	}
}

func (s *store) reconcileInterrupted() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	all, err := instances(tx)
	if err != nil {
		return err
	}
	for _, v := range all {
		if v.Public.ActiveOperationID == "" {
			continue
		}
		op, err := loadOperation(tx, v.Public.ActiveOperationID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		op.Status, op.Message, op.FinishedAt = "interrupted", "runner restarted before the operation outcome was committed; inspect before retrying", &now
		v.Public.ActiveOperationID, v.Public.ObservedState, v.Public.Health, v.Public.RecoveryRequired = "", "unknown", "unknown", true
		if err := saveOperation(tx, op); err != nil {
			return err
		}
		if err := saveInstance(tx, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *store) observe(v storedInstance, state runtimequalify.ManagedState, observation runtimequalify.ManagedObservation) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := loadInstance(tx, v.Public.ID)
	if err != nil {
		return err
	}
	if current.Public.ActiveOperationID != "" || current.RuntimeRevision != v.RuntimeRevision {
		return nil
	}
	if state.ID != current.Runtime.ID {
		return errors.New("observed runtime identity mismatch")
	}
	current.Runtime, current.RuntimeRevision = state, current.RuntimeRevision+1
	applyObservation(&current, observation, time.Now().UTC())
	if err := saveInstance(tx, current); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *store) close() error { return s.db.Close() }
