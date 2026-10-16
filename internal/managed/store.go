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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	_ "modernc.org/sqlite"
)

const mode = "development-managed-fake"
const DevelopmentMode = "development-managed"
const currentSchema = 16

type storedInstance struct {
	Public                protocol.Instance           `json:"public"`
	Runtime               runtimequalify.ManagedState `json:"runtime"`
	RuntimeRevision       uint64                      `json:"runtime_revision"`
	RuntimeConfigRevision uint64                      `json:"runtime_config_revision"`
	CPU                   *cpuBaseline                `json:"cpu_baseline,omitempty"`
	Startup               *startupState               `json:"startup,omitempty"`
}

type store struct {
	bootID          string
	validateStorage func(string) error
	portAvailable   func(string, uint16) bool
	directory       string
	db              *sql.DB
	runnerID        string
	mode            string
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
	s := &store{db: db, mode: mode, directory: dir}
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
PRAGMA user_version=16;`); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO meta VALUES ('runner_id', ?)", "runner-"+id); err != nil {
			return err
		}
	} else if version >= 11 && version <= 15 {
		// Older runners must not discard durable reboot-resume authorization.
		if _, err := tx.Exec("PRAGMA user_version=16"); err != nil {
			return err
		}
	} else if version != currentSchema {
		return errors.New("state database requires owner-user runtime and storage support (schema 16)")
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS maintenance_policies (instance_id TEXT PRIMARY KEY REFERENCES instances(id), body BLOB NOT NULL)"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS deleted_instances (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL, revision INTEGER NOT NULL)"); err != nil {
		return err
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

func replay(tx querier, actor, request, fingerprint string) (protocol.Operation, bool, error) {
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
		return bad("development runtime requires HTTP port 1024..65535, loom 31, and guest Ames value 34343 (no host UDP publication)")
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
	return s.registerInstance(actor, req, nil)
}

func registrationFingerprint(req *protocol.CreateFakeRequest, source *protocol.ImportSource) (string, error) {
	if !protocol.ValidID(req.ID) || !protocol.ValidID(req.RequestID) {
		return "", bad("valid instance and request IDs are required")
	}
	if err := validateConfig(req.Config); err != nil {
		return "", err
	}
	if req.Label == "" {
		req.Label = req.ID
	}
	if !utf8.ValidString(req.Label) || len(req.Label) > 128 || strings.IndexFunc(req.Label, unicode.IsControl) >= 0 {
		return "", bad("label must be at most 128 UTF-8 bytes without control characters")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind    string
		Request protocol.CreateFakeRequest
	}{"register-fake", *req})
	if source != nil {
		fingerprint, _ = json.Marshal(struct {
			Kind           string
			Request        protocol.CreateFakeRequest
			SourceRunID    string
			ExpectedDigest string
		}{"import-fake", *req, source.RunID, source.ContentDigest})
	}
	return string(fingerprint), nil
}

func (s *store) registerInstance(actor string, req protocol.CreateFakeRequest, source *protocol.ImportSource) (protocol.Operation, bool, error) {
	return s.registerKind(actor, req, source, false)
}

func (s *store) registerKind(actor string, req protocol.CreateFakeRequest, source *protocol.ImportSource, comet bool) (protocol.Operation, bool, error) {
	if req.Bootstrap != nil && (!protocol.ValidBootstrap(req.Bootstrap) || source != nil) {
		return protocol.Operation{}, false, bad("bootstrap selection applies only to new ships; choose bare or a staged recommended pill")
	}
	location := req.PierPath
	if location == "" {
		if comet {
			location = "livenet/comets/" + req.ID + "/pier"
		} else {
			leaf := req.ID
			if req.PierName != "" {
				leaf = req.PierName
			}
			location = "fakeships/" + leaf + "/pier"
		}
	}
	if !pierstore.ValidPath(location) || (comet && pierstore.Kind(location) != "comet") || (!comet && pierstore.Kind(location) != "fake") {
		return protocol.Operation{}, false, bad("choose a relative pier path in the appropriate fakeships or livenet/comets tree")
	}
	if err := s.checkStorage(location); err != nil {
		return protocol.Operation{}, false, err
	}
	var fake *runtimequalify.ManagedFake
	if req.FakeShip != "" || req.PierName != "" {
		if comet || source != nil {
			return protocol.Operation{}, false, bad("fake identity and pier options apply only to new fake ships")
		}
		if req.FakeShip == "" {
			req.FakeShip = "~zod"
		}
		if !strings.HasPrefix(req.FakeShip, "~") {
			req.FakeShip = "~" + req.FakeShip
		}
		if req.PierName == "" {
			req.PierName = req.ID
		}
		if !protocol.ValidFakeShip(req.FakeShip) || len(req.FakeShip) > 28 || !protocol.ValidPierName(req.PierName) {
			return protocol.Operation{}, false, bad("require a valid fake Urbit name and a single pier directory name starting with a letter or number")
		}
		fake = &runtimequalify.ManagedFake{Ship: req.FakeShip}
	}
	fingerprint, err := registrationFingerprint(&req, source)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if comet {
		if s.mode != DevelopmentMode || source != nil {
			return protocol.Operation{}, false, conflict("comet_mode_required", "comet registration requires the explicit development-managed endpoint")
		}
		data, _ := json.Marshal(struct {
			Kind    string
			Request protocol.CreateFakeRequest
		}{"register-comet", req})
		fingerprint = string(data)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, fingerprint); found || err != nil {
		return op, true, err
	}
	if req.Bootstrap != nil {
		if err := s.checkStagedPill(req.Bootstrap.Pill); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	var count int
	revision, err := initialRevision(tx, req.ID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if err := tx.QueryRow("SELECT count(*) FROM instances WHERE id=?", req.ID).Scan(&count); err != nil {
		return protocol.Operation{}, false, err
	}
	if count != 0 {
		return protocol.Operation{}, false, conflict("instance_exists", "instance ID is already registered")
	}
	if err := reservePort(tx, req.ID, req.Config.HTTPPort); err != nil {
		return protocol.Operation{}, false, err
	}
	if err := s.availablePort("tcp", req.Config.HTTPPort); err != nil {
		return protocol.Operation{}, false, err
	}
	if err := reservePierPath(tx, req.ID, location); err != nil {
		return protocol.Operation{}, false, err
	}
	if source != nil {
		all, err := instances(tx)
		if err != nil {
			return protocol.Operation{}, false, err
		}
		for _, other := range all {
			if other.Runtime.Import != nil && other.Runtime.Import.Source.RunID == source.RunID {
				return protocol.Operation{}, false, conflict("source_claimed", "this source is already reserved by a registered import")
			}
		}
	}
	uuid, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	v := storedInstance{Public: protocol.Instance{ID: req.ID, Label: req.Label, Config: req.Config, ConfigRevision: revision, Mode: "fake", Ship: "~zod", DesiredState: "stopped", ObservedState: "uninitialized", Health: "unknown"}, Runtime: runtimequalify.ManagedState{ID: uuid, Port: req.Config.HTTPPort}}
	v.Runtime.Fake = fake
	if source == nil {
		v.Runtime.Bootstrap = req.Bootstrap
		if v.Runtime.Bootstrap == nil {
			v.Runtime.Bootstrap = &protocol.Bootstrap{Kind: "bare"}
		}
		v.Public.Bootstrap = v.Runtime.Bootstrap
	}
	v.Runtime.Storage = &pierstore.Location{Path: location}
	v.Public.PierPath = location
	if fake != nil {
		v.Public.Ship = fake.Ship
	}
	if comet {
		v.Public.Mode = "comet"
		v.Public.Ship = ""
		v.Runtime.Comet = &runtimequalify.ManagedComet{}
	}
	if source != nil {
		v.Runtime.Import = &runtimequalify.ManagedImport{Source: *source, Uncertain: true}
	}
	v.Public.PierName = "pier"
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
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, InstanceID: req.ID, Kind: "register-fake", Actor: actor, Status: "succeeded", ConfigRevision: revision, CreatedAt: now, FinishedAt: &now}
	if comet {
		op.Kind = "register-comet"
	}
	if source != nil {
		op.Kind, op.Status, op.FinishedAt = "import-fake", "queued", nil
		v.Public.ActiveOperationID = op.ID
		v.Public.ObservedState = "importing"
		updateImportReport(&v)
		if err := saveInstance(tx, v); err != nil {
			return op, false, err
		}
	}
	if err := insertOperation(tx, op, fingerprint); err != nil {
		return op, false, err
	}
	return op, false, tx.Commit()
}

func (s *store) configure(actor, id string, req protocol.ConfigureRequest) (protocol.Instance, error) {
	if req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 {
		return protocol.Instance{}, bad("a valid positive configuration revision is required")
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
	validate := validateConfig
	if v.Runtime.Deletion != nil {
		return v.Public, conflict("deletion_pending", "pier deletion is pending")
	}
	if v.Runtime.Handoff.Fenced() {
		return v.Public, conflict("handoff_fenced", "ship is fenced for a cross-host move")
	}
	if v.Runtime.Boot != nil {
		validate = validateOwnedConfig
	}
	if err := validate(req.Config); err != nil {
		return v.Public, err
	}
	if v.Runtime.Boot != nil {
		if err := reserveAmes(tx, id, req.Config.AmesPort); err != nil {
			return v.Public, err
		}
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
	if req.Config.HTTPPort != v.Public.Config.HTTPPort {
		if err := s.availablePort("tcp", req.Config.HTTPPort); err != nil {
			return v.Public, err
		}
	}
	if v.Runtime.Boot != nil && req.Config.AmesPort != v.Public.Config.AmesPort {
		if err := s.availablePort("udp", req.Config.AmesPort); err != nil {
			return v.Public, err
		}
	}
	if v.Public.Config != req.Config {
		if v.Public.ConfigRevision == math.MaxInt64 {
			return v.Public, conflict("revision_limit", "configuration revision exhausted")
		}
		v.Public.Config, v.Public.ConfigRevision = req.Config, v.Public.ConfigRevision+1
		v.Startup = nil
		v.Public.Epochs, v.Runtime.Epochs = nil, nil
		v.RuntimeRevision++ // prevent an in-flight poll resurrecting the old inventory
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
	if !protocol.ValidID(id) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 || (req.Kind != "start" && req.Kind != "stop" && req.Kind != "pack" && req.Kind != "meld" && req.Kind != "roll" && req.Kind != "inspect-epochs" && req.Kind != "prune-epochs" && req.Kind != "move-pier" && req.Kind != "delete-pier" && req.Kind != "recover-import") {
		return protocol.Operation{}, false, bad("require a supported operation, valid request ID, and positive revision")
	}
	if (req.Kind == "delete-pier") != req.ConfirmDelete {
		return protocol.Operation{}, false, bad("delete-pier requires explicit irreversible deletion confirmation")
	}
	if req.Kind == "prune-epochs" {
		_, cutoffErr := epochs.Number(req.EpochCutoff)
		if !req.ConfirmHistoryLoss || cutoffErr != nil || !epochs.ValidDigest(req.InventoryDigest) {
			return protocol.Operation{}, false, bad("prefix pruning requires a canonical cutoff, inventory digest and explicit history-loss confirmation")
		}
	} else if req.EpochCutoff != "" || req.InventoryDigest != "" || req.ConfirmHistoryLoss {
		return protocol.Operation{}, false, bad("pruning fields are valid only for prune-epochs")
	}
	if (req.Kind == "move-pier" && !pierstore.ValidPath(req.PierPath)) || (req.Kind != "move-pier" && req.PierPath != "") {
		return protocol.Operation{}, false, bad("move-pier requires its convention-relative destination path")
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
	if v.Runtime.Handoff.Fenced() {
		return protocol.Operation{}, false, conflict("handoff_fenced", "ship is fenced for a cross-host move")
	}
	if v.Runtime.Deletion != nil && req.Kind != "delete-pier" {
		return protocol.Operation{}, false, conflict("deletion_pending", "pier deletion is pending; only explicit deletion continuation is available")
	}
	if req.Kind == "recover-import" && (v.Public.DesiredState != "stopped" || !v.Runtime.ImportRecoveryEligible()) {
		return protocol.Operation{}, false, conflict("invalid_state", "recovery requires a stopped interrupted prepared external enrollment with retained identity")
	}
	if req.Kind == "delete-pier" {
		if v.Public.DesiredState != "stopped" || v.Runtime.Deletion == nil && v.Public.ObservedState != "stopped" && v.Public.ObservedState != "uninitialized" {
			return protocol.Operation{}, false, conflict("invalid_state", "stop the runtime before deleting its pier")
		}
		for _, c := range v.Runtime.Runtime.Containers {
			if !c.Removed || c.CreateUncertain {
				return protocol.Operation{}, false, conflict("invalid_state", "stop and reconcile retained containers before deletion")
			}
		}
		if v.Runtime.ExternalImportInFlight() {
			return protocol.Operation{}, false, conflict("recovery_required", "review the interrupted external import before deletion")
		}
	}
	if req.Kind == "move-pier" {
		if err := s.checkStorage(req.PierPath); err != nil {
			return protocol.Operation{}, false, err
		}
		if v.Public.DesiredState != "stopped" || v.Public.RecoveryRequired || (v.Public.ObservedState != "stopped" && v.Public.ObservedState != "uninitialized") {
			return protocol.Operation{}, false, conflict("invalid_state", "stop the ship cleanly before moving its pier")
		}
		if pierstore.Kind(req.PierPath) != v.Public.Mode {
			return protocol.Operation{}, false, bad("destination classification differs from the registered ship")
		}
		if err := reservePierPath(tx, id, req.PierPath); err != nil {
			return protocol.Operation{}, false, err
		}
		v.Runtime.MoveTarget = req.PierPath
	}
	if req.Kind != "stop" && req.Kind != "delete-pier" && v.Runtime.UncertainMaintenance() {
		return protocol.Operation{}, false, conflict("recovery_required", "interrupted maintenance requires manual review")
	}
	if req.Kind != "stop" && req.Kind != "delete-pier" && req.Kind != "recover-import" && v.Runtime.UncertainProvisioning() {
		return protocol.Operation{}, false, conflict("recovery_required", "incomplete provisioning or identity verification requires manual review")
	}
	if req.Kind == "roll" && v.Runtime.RollNeedsRuntime {
		return protocol.Operation{}, false, conflict("invalid_state", "resume and cleanly stop the pier before requesting another new roll")
	}
	if req.Kind == "pack" || req.Kind == "meld" || req.Kind == "roll" || req.Kind == "inspect-epochs" || req.Kind == "prune-epochs" {
		if req.Kind != "inspect-epochs" && !v.Runtime.MaintenanceQualified() {
			return protocol.Operation{}, false, conflict("invalid_state", "comet maintenance requires completed identity-pinned provisioning")
		}
		if !v.Runtime.Initialized || v.Public.DesiredState != "stopped" || v.Public.ObservedState != "stopped" || v.Public.RecoveryRequired {
			return protocol.Operation{}, false, conflict("invalid_state", "offline maintenance requires a cleanly stopped initialized pier")
		}
		for _, c := range v.Runtime.Runtime.Containers {
			if !c.Removed || c.CreateUncertain {
				return protocol.Operation{}, false, conflict("invalid_state", "stop and reconcile all retained runtime resources before maintenance")
			}
		}
	}
	if req.Kind == "prune-epochs" {
		inv := v.Runtime.Epochs
		if inv == nil || inv.PierID != v.Runtime.ID {
			return protocol.Operation{}, false, conflict("stale_inventory", "inspect the selected stopped pier before pruning")
		}
		if _, err := epochs.PrunePlan(inv, req.EpochCutoff, req.InventoryDigest, time.Now().UTC()); err != nil {
			return protocol.Operation{}, false, conflict("stale_or_unqualified_inventory", err.Error())
		}
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
		if err := s.availablePort("tcp", v.Public.Config.HTTPPort); err != nil {
			return protocol.Operation{}, false, err
		}
		if v.Runtime.Boot != nil {
			if err := s.availablePort("udp", v.Public.Config.AmesPort); err != nil {
				return protocol.Operation{}, false, err
			}
		}
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, InstanceID: id, Kind: req.Kind, Actor: actor, Status: "queued", ConfigRevision: req.ExpectedRevision, CreatedAt: time.Now().UTC()}
	v.Public.ActiveOperationID = op.ID
	v.Startup, v.Runtime.StartupResume = nil, nil
	v.Public.Epochs, v.Runtime.Epochs = nil, nil
	v.Runtime.PruneIntent = nil
	if req.Kind == "prune-epochs" {
		v.Runtime.PruneIntent = &runtimequalify.ManagedPruneIntent{Cutoff: req.EpochCutoff, InventoryDigest: req.InventoryDigest, Confirmed: req.ConfirmHistoryLoss}
	}
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
	result := protocol.Snapshot{RunnerID: s.runnerID, Mode: s.mode, Instances: []protocol.Instance{}, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{}}
	for _, v := range all {
		if err := loadEpochHistory(tx, &v); err != nil {
			return result, err
		}
		if err := attachRuntimeStatus(tx, &v); err != nil {
			return result, err
		}
		policy, err := loadRoutinePolicy(tx, v.Public.ID)
		if err != nil {
			return result, err
		}
		if policy != nil {
			policy.NextSlot = routines.Next(*policy, time.Now().UTC())
			v.Public.Schedule = policy
		}
		if v.Public.ObservedAt != nil && time.Since(*v.Public.ObservedAt) > 30*time.Second {
			v.Public.Health = "unknown"
		}
		ageResources(v.Public.Resources, time.Now().UTC())
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
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if op.Kind == "recover-import" {
		// Recovery is read-only; it has no incremental resource intents. Keep
		// the original private fence durable until finish atomically publishes
		// the completed state, stopped observation and succeeded receipt.
		return nil
	}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	updateImportReport(&v)
	v.Public.Routine = state.Routine
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
	if strings.HasSuffix(op.Kind, "-handoff") || op.Kind == "delete-pier" || op.Kind == "recover-import" || op.Kind == "import-pier" || op.Kind == "start" || op.Kind == "startup-resume" || op.Kind == "move-pier" || op.Kind == "enroll-pier" || op.Kind == "pack" || op.Kind == "meld" || op.Kind == "roll" || op.Kind == "prune-epochs" || op.Kind == "import-fake" || routineKind(op.Kind) {
		// A worker timeout does not stop a daemon-side utility. Recheck after
		// acquiring the host gate, including jobs queued before that timeout and
		// retained intents loaded after management restart. Explicit stop remains
		// available to resolve resources without clearing pier quarantine.
		all, err := instances(tx)
		if err != nil {
			return op, v, err
		}
		for _, other := range all {
			for _, c := range other.Runtime.Runtime.Containers {
				if (c.Kind == "pack" || c.Kind == "meld" || c.Kind == "roll" || c.Kind == "comet-pack" || c.Kind == "comet-meld" || c.Kind == "comet-roll") && (!c.Removed || c.CreateUncertain) {
					return op, v, conflict("host_recovery_required", "resolve retained maintenance resources on "+other.Public.ID+" with explicit stop/review before more runtime work")
				}
			}
		}
	}
	op.Status = "running"
	if op.Kind == "start" || op.Kind == "startup-resume" {
		v.Public.ObservedState = "starting"
		v.Runtime.Port = v.Public.Config.HTTPPort
		if v.Runtime.Boot != nil {
			v.Runtime.AmesPort = v.Public.Config.AmesPort
		}
		v.RuntimeConfigRevision = op.ConfigRevision
	} else if op.Kind == "pack" || op.Kind == "meld" || op.Kind == "roll" || op.Kind == "prune-epochs" || routineKind(op.Kind) {
		v.Public.ObservedState = "maintenance"
	} else if op.Kind == "inspect-epochs" {
		v.Public.ObservedState = "inspecting"
	} else if op.Kind == "import-fake" || op.Kind == "import-pier" {
		v.Public.ObservedState = "importing"
	} else if op.Kind == "enroll-pier" || op.Kind == "recover-import" {
		v.Public.ObservedState = "enrolling"
	} else if op.Kind == "move-pier" || strings.HasSuffix(op.Kind, "-handoff") {
		v.Public.ObservedState = "moving"
	} else if op.Kind == "delete-pier" {
		v.Public.ObservedState = "deleting"
	} else {
		v.Public.ObservedState = "stopping"
	}
	v.Public.Health = "unknown"
	v.Public.Resources = nil
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
	if op.Kind == "recover-import" {
		if interrupted && cause == nil {
			cause = errors.New("recovery interrupted before completion publication")
		}
		if cause != nil {
			// Never publish a proposed completion with a failed/interrupted
			// receipt. The read-only worker has no resource changes to retain.
			state = v.Runtime
		}
	}
	now := time.Now().UTC()
	op.Status, op.FinishedAt = "succeeded", &now
	if routineKind(op.Kind) && cause != nil && state.Routine != nil {
		state.Routine.FinishedAt = &now
		state.Routine.Succeeded = false
		prefix := "failed: "
		if interrupted {
			prefix = "interrupted: "
		}
		state.Routine.Stage = prefix + state.Routine.Stage
	}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	v.Public.ActiveOperationID = ""
	applyObservation(&v, observation, now)
	if cause != nil {
		if strings.HasSuffix(op.Kind, "-handoff") && v.Runtime.Handoff != nil {
			v.Runtime.Handoff.Error = "Handoff paused: " + cause.Error()
			v.Public.Handoff = v.Runtime.Handoff
		}
		if routineKind(op.Kind) && observation.State == "stopped" {
			v.Public.DesiredState = "stopped"
		}
		op.Status, op.Message = "failed", "runtime action failed: "+cause.Error()
		if interrupted {
			op.Status = "interrupted"
		}
		if len(op.Message) > 1024 {
			op.Message = op.Message[:1024]
		}
		v.Public.ObservedState, v.Public.Health, v.Public.RecoveryRequired = "unknown", "unknown", true
		v.Public.Resources, v.Public.Epochs = nil, nil
	} else if op.Kind == "start" || op.Kind == "startup-resume" {
		v.Public.AppliedRevision = op.ConfigRevision
	}
	if cause != nil {
		v.Startup = nil
	} else {
		s.recordHealthyBoot(&v)
	}
	if op.Kind == "delete-pier" && cause == nil {
		if state.Deletion == nil || state.Deletion.Stage != "complete" {
			return errors.New("deletion has no completed filesystem receipt")
		}
		if _, err := tx.Exec("DELETE FROM maintenance_policies WHERE instance_id=?", op.InstanceID); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO deleted_instances VALUES (?,?,?) ON CONFLICT(id) DO UPDATE SET operation_id=excluded.operation_id,revision=excluded.revision", op.InstanceID, op.ID, op.ConfigRevision); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM instances WHERE id=?", op.InstanceID); err != nil {
			return err
		}
	} else if err := saveInstance(tx, v); err != nil {
		return err
	}
	if err := saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}

func applyObservation(v *storedInstance, observation runtimequalify.ManagedObservation, now time.Time) {
	v.Public.DeletePending = v.Runtime.Deletion != nil
	v.Public.Handoff = v.Runtime.Handoff
	v.Public.AmesPublished = v.Runtime.Boot != nil
	v.Public.IdentityPending = v.Runtime.Adoption != nil && !v.Runtime.Adoption.Verified
	if v.Runtime.Storage != nil {
		v.Public.PierPath = v.Runtime.Storage.Path
	}
	previousCPU := v.CPU
	if v.Public.Resources == nil || v.Public.Resources.Status != "fresh" || previousCPU == nil ||
		v.Public.Resources.ContainerID != previousCPU.ContainerID || v.Public.Resources.ObservedAt == nil || !v.Public.Resources.ObservedAt.Equal(previousCPU.ObservedAt) {
		previousCPU = nil
	}
	v.CPU = nil
	v.Public.PierName = "pier"
	if v.Runtime.Fake != nil {
		v.Public.Ship = v.Runtime.Fake.Ship
	}
	v.Public.Routine = v.Runtime.Routine
	if v.Runtime.Comet != nil {
		v.Public.Ship = v.Runtime.Comet.Ship
	}
	updateImportReport(v)
	v.Public.ObservedState, v.Public.Health, v.Public.Endpoint = observation.State, observation.Health, observation.Endpoint
	v.Public.RecoveryRequired, v.Public.ObservedAt = observation.RecoveryRequired, &now
	v.Public.RollNeedsRuntime = v.Runtime.RollNeedsRuntime
	v.Public.Epochs = nil
	if observation.State == "stopped" && !observation.RecoveryRequired {
		v.Public.Epochs = v.Runtime.Epochs
	}
	v.Public.Resources = nil
	if observation.State == "running" {
		v.Public.Resources = &protocol.ResourceMetrics{Status: "unavailable", Source: "docker-stats-v1.47"}
		if r := observation.Resources; r != nil {
			v.Public.Resources = &protocol.ResourceMetrics{Status: "fresh", Source: "docker-stats-v1.47", ContainerID: r.ContainerID, ObservedAt: &r.ObservedAt,
				MemoryCurrentBytes: &r.MemoryCurrentBytes, MemoryLimitBytes: &r.MemoryLimitBytes, PIDsCurrent: &r.PIDsCurrent}
			if cpu := r.CPU; cpu != nil && cpu.OnlineCPUs > 0 && cpu.OnlineCPUs <= 4096 && cpu.LimitCores == 2 {
				v.CPU = &cpuBaseline{ContainerID: r.ContainerID, ObservedAt: r.ObservedAt, Counters: *cpu}
				v.Public.Resources.CPULimitCores = &cpu.LimitCores
				v.Public.Resources.CPUUsagePercent = cpuUsage(previousCPU, v.CPU, now)
			}
		}
	}
	if m := v.Runtime.Maintenance; m != nil {
		v.Public.Maintenance = &protocol.MaintenanceReport{Kind: m.Kind, BeforePierBytes: m.BeforePierBytes, AfterPierBytes: m.AfterPierBytes, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, MemAvailableBytes: m.MemAvailableBytes, DiskAvailableBytes: m.DiskAvailableBytes, EpochsBefore: m.EpochsBefore, EpochsAfter: m.EpochsAfter, CreatedEpoch: m.CreatedEpoch}
	}
	if p := v.Runtime.Runtime.Prune; p != nil {
		report := &protocol.PruningReport{Cutoff: p.Cutoff, StartedAt: p.StartedAt, FinishedAt: p.FinishedAt, RemovedFiles: len(p.RemovedFiles), ExpectedFiles: len(p.Files), EpochRemoved: p.EpochRemoved, Uncertain: p.Uncertain}
		report.SelectedEpochs = []string{p.Cutoff}
		if p.EpochRemoved {
			report.RemovedEpochs = []string{p.Cutoff}
		}
		if p.Before != nil {
			report.InventoryDigest = p.Before.Digest
		}
		if p.After != nil {
			for _, e := range p.After.Entries {
				report.RetainedEpochs = append(report.RetainedEpochs, e.Name)
			}
		}
		v.Public.Pruning = report
	}
	if p := v.Runtime.PrefixPrune; p != nil {
		report := &protocol.PruningReport{Cutoff: p.Cutoff, StartedAt: p.StartedAt, FinishedAt: p.FinishedAt, Uncertain: p.Uncertain, EpochRemoved: len(p.Targets) > 0}
		if p.Before != nil {
			report.InventoryDigest = p.Before.Digest
		}
		for _, target := range p.Targets {
			report.SelectedEpochs = append(report.SelectedEpochs, target.Name)
			report.RemovedFiles += len(target.RemovedFiles)
			report.ExpectedFiles += len(target.Files)
			if target.Removed {
				report.RemovedEpochs = append(report.RemovedEpochs, target.Name)
			} else {
				report.EpochRemoved = false
			}
		}
		if p.After != nil {
			for _, e := range p.After.Entries {
				report.RetainedEpochs = append(report.RetainedEpochs, e.Name)
			}
		}
		v.Public.Pruning = report
	}
	if v.Runtime.Initialized && observation.State == "running" && observation.Health == "healthy" {
		v.Public.AppliedRevision = v.RuntimeConfigRevision
	}
}

func ageResources(r *protocol.ResourceMetrics, now time.Time) {
	if r == nil || r.Status != "fresh" {
		return
	}
	if r.ObservedAt == nil || r.ObservedAt.IsZero() || r.ContainerID == "" || r.MemoryCurrentBytes == nil || r.MemoryLimitBytes == nil || r.PIDsCurrent == nil {
		r.Status = "unavailable"
		r.ObservedAt, r.MemoryCurrentBytes, r.MemoryLimitBytes, r.PIDsCurrent = nil, nil, nil, nil
		r.CPUUsagePercent, r.CPULimitCores = nil, nil
		return
	}
	if now.Sub(*r.ObservedAt) > 30*time.Second || r.ObservedAt.After(now.Add(5*time.Second)) {
		r.Status = "stale"
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
		if strings.HasSuffix(op.Kind, "-handoff") && v.Runtime.Handoff != nil {
			v.Runtime.Handoff.Error = "Handoff interrupted; continue the same move to reconcile its recorded phase."
			v.Public.Handoff = v.Runtime.Handoff
		}
		if op.Kind == "enroll-pier" && v.Runtime.Adoption != nil {
			v.Runtime.Adoption.Uncertain = true
		}
		if op.Kind == "move-pier" && v.Runtime.Storage != nil {
			v.Runtime.Storage.Uncertain = true
		}
		if routineKind(op.Kind) && v.Runtime.Routine != nil {
			v.Runtime.Routine.FinishedAt = &now
			v.Runtime.Routine.Succeeded = false
			v.Runtime.Routine.Stage = "interrupted"
			v.Public.Routine = v.Runtime.Routine
		}
		if op.Kind == "start" && v.Runtime.Comet != nil && (v.Public.Ship == "" || !v.Runtime.Initialized || v.Public.AppliedRevision == 0) {
			v.Runtime.Comet.Uncertain = true
		}
		if op.Kind == "start" && v.Runtime.Storage != nil && v.Runtime.Prepared && v.Public.AppliedRevision == 0 {
			v.Runtime.Storage.Uncertain = true
		}
		if op.Kind == "import-fake" && v.Runtime.Import != nil {
			v.Runtime.Import.Uncertain = true
			updateImportReport(&v)
		}
		v.Public.ActiveOperationID, v.Public.ObservedState, v.Public.Health, v.Public.RecoveryRequired = "", "unknown", "unknown", true
		v.Startup = nil
		v.Public.Resources = nil
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
		var f *fault
		if errors.As(err, &f) && f.problem.Code == "instance_not_found" {
			return nil
		} // a completed deletion won the race
		return err
	}
	if current.Public.ActiveOperationID != "" || current.RuntimeRevision != v.RuntimeRevision || current.Runtime.ID != v.Runtime.ID {
		return nil
	}
	if state.ID != current.Runtime.ID {
		return errors.New("observed runtime identity mismatch")
	}
	current.Runtime, current.RuntimeRevision = state, current.RuntimeRevision+1
	applyObservation(&current, observation, time.Now().UTC())
	s.recordHealthyBoot(&current)
	if err := saveInstance(tx, current); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *store) close() error { return s.db.Close() }
