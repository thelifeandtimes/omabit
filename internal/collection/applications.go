package collection

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

var (
	ErrApplicationConflict = errors.New("application approval changed or an existing connection needs reconciliation")
	ErrApplicationLimit    = errors.New("application attempt limit reached; archive fully closed connections before approving another")
	ErrApplicationBusy     = errors.New("application connection operation is already running")
	ErrApplicationJournal  = errors.New("application connection journal is invalid or unavailable; state was preserved")
)

// ApplicationTarget identifies a verified immutable receiver generation. It is
// evidence supplied by the trusted artifact resolver, never a command to run.
type ApplicationTarget struct {
	PluginID string `json:"plugin_id"`
	Profile  string `json:"profile"`
	SHA256   string `json:"artifact_sha256"`
}

var applicationID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
var applicationProfile = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

func (t ApplicationTarget) Valid() bool {
	digest, err := hex.DecodeString(t.SHA256)
	return applicationID.MatchString(t.PluginID) && applicationProfile.MatchString(t.Profile) && err == nil && len(digest) == 32 && t.SHA256 == strings.ToLower(t.SHA256)
}

// ApplicationAttempt contains only approval and lifecycle evidence. Credentials
// live behind SessionRef in the explicitly selected credential adapter. These
// records survive removal of the parent ship enrollment so cleanup stays possible.
type ApplicationAttempt struct {
	ID                 string            `json:"request_id"`
	ConnectionID       string            `json:"connection_id"`
	ConnectionRevision uint64            `json:"connection_revision"`
	Origin             string            `json:"origin"`
	Ship               string            `json:"ship"`
	Credentials        string            `json:"credentials"`
	SessionRef         string            `json:"session_ref"`
	Target             ApplicationTarget `json:"target"`
	Authority          string            `json:"authority"`
	Phase              string            `json:"phase"`
	Revision           uint64            `json:"revision"`
	// Sticky once delivery might have happened. Cleanup cannot infer that the
	// receiver removed its local account merely because its session was revoked.
	ReceiverOutcome        string    `json:"receiver_outcome,omitempty"`
	ReceiverReconciliation bool      `json:"receiver_reconciliation"`
	Outcome                string    `json:"outcome,omitempty"`
	ApprovedAt             time.Time `json:"approved_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type applicationsFile struct {
	Version  int                  `json:"version"`
	Attempts []ApplicationAttempt `json:"attempts"`
}

func (a ApplicationAttempt) valid() bool {
	origin, err := eyre.Origin(a.Origin)
	if err != nil || origin != a.Origin || !eyre.ValidShip(a.Ship) || !validRef(a.ID) || !validRef(a.ConnectionID) || !validRef(a.SessionRef) || a.ConnectionRevision == 0 || !a.Target.Valid() || (a.Credentials != "secret-service" && a.Credentials != "file") || a.Authority != "eyre-owner-session" || a.Revision == 0 || a.ApprovedAt.IsZero() || a.UpdatedAt.Before(a.ApprovedAt) {
		return false
	}
	if a.ReceiverOutcome != "" && (a.Phase != "closed" || a.ReceiverReconciliation ||
		(a.ReceiverOutcome != "removed" && a.ReceiverOutcome != "absent" && a.ReceiverOutcome != "superseded")) {
		return false
	}
	switch a.Phase {
	case "approved", "issuing", "saved":
		return !a.ReceiverReconciliation && a.Outcome == ""
	case "delivering", "connected":
		return a.ReceiverReconciliation && a.Outcome == ""
	case "unresolved", "revoking":
		return a.Outcome == ""
	case "revoked":
		return a.Outcome == "revoked" || a.Outcome == "rejected"
	case "closed":
		return a.Outcome == "revoked" || a.Outcome == "rejected" || a.Outcome == "not-issued" && !a.ReceiverReconciliation
	}
	return false
}

func (r *Registry) readApplications() (applicationsFile, error) {
	state := applicationsFile{Version: 1, Attempts: []ApplicationAttempt{}}
	f, err := privateFile(filepath.Join(r.dir, "applications.json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, ErrApplicationJournal
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRegistryBytes+1))
	if err != nil || len(data) > maxRegistryBytes {
		return state, ErrApplicationJournal
	}
	if !uniqueApplicationJSON(data) {
		return state, ErrApplicationJournal
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || (state.Version != 1 && state.Version != 2) || state.Attempts == nil || len(state.Attempts) > 64 {
		return state, ErrApplicationJournal
	}
	ids, refs, active := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, a := range state.Attempts {
		key := a.Origin + "\x00" + a.Target.PluginID + "\x00" + a.Target.Profile
		if !a.valid() || ids[a.ID] || refs[a.SessionRef] || a.blocksApproval() && active[key] {
			return state, ErrApplicationJournal
		}
		ids[a.ID], refs[a.SessionRef] = true, true
		if a.blocksApproval() {
			active[key] = true
		}
	}
	return state, nil
}

func (a ApplicationAttempt) blocksApproval() bool {
	return a.Phase != "closed" || a.ReceiverReconciliation
}

func (r *Registry) Applications(ctx context.Context) ([]ApplicationAttempt, error) {
	var state applicationsFile
	err := r.locked(ctx, func() (err error) { state, err = r.readApplications(); return err })
	if err != nil {
		return nil, err
	}
	return state.Attempts, nil
}

// ApproveApplication is an internal trusted-UI boundary, not an approval prompt.
// Its caller must already have shown this exact target, connection and broad
// owner authority to the user. Replaying the same ID never issues another session.
func (r *Registry) ApproveApplication(ctx context.Context, expected Ship, id string, target ApplicationTarget) (ApplicationAttempt, error) {
	if !validRef(id) || !target.Valid() {
		return ApplicationAttempt{}, ErrApplicationConflict
	}
	var result ApplicationAttempt
	err := r.locked(ctx, func() error {
		if _, archived, err := r.readApplicationArchive(id); err != nil {
			return err
		} else if archived {
			return ErrApplicationArchived
		}

		state, err := r.readApplications()
		if err != nil {
			return err
		}
		for _, a := range state.Attempts {
			if a.ID == id {
				if a.ConnectionID != expected.ID || a.ConnectionRevision != expected.Revision || a.Origin != expected.Origin || a.Ship != expected.Ship || a.Credentials != expected.Credentials || a.Target != target {
					return ErrApplicationConflict
				}
				result = a
				return nil
			}
			if a.blocksApproval() && a.Origin == expected.Origin && a.Target.PluginID == target.PluginID && a.Target.Profile == target.Profile {
				return ErrApplicationConflict
			}
		}
		ships, err := r.readShips()
		if err != nil {
			return err
		}
		found := false
		for _, s := range ships.Ships {
			if s.ID == expected.ID && s.Revision == expected.Revision && s.Origin == expected.Origin && s.Ship == expected.Ship && s.Credentials == expected.Credentials && s.SessionRef != "" {
				found = true
			}
		}
		if !found {
			return ErrApplicationConflict
		}
		if len(state.Attempts) >= 64 {
			return ErrApplicationLimit
		}
		now := time.Now().UTC()
		result = ApplicationAttempt{ID: id, ConnectionID: expected.ID, ConnectionRevision: expected.Revision,
			Origin: expected.Origin, Ship: expected.Ship, Credentials: expected.Credentials, SessionRef: NewSessionRef(), Target: target,
			Authority: "eyre-owner-session", Phase: "approved", Revision: 1, ApprovedAt: now, UpdatedAt: now}
		state.Attempts = append(state.Attempts, result)
		return r.writeFile("applications.json", state)
	})
	if err != nil {
		return ApplicationAttempt{}, err
	}
	return result, nil
}

// ApplicationTransaction holds a process-released lock across bounded session
// issuance, storage and delivery. Separate attempts never hold the registry lock
// during HTTP/receiver IO; another process cannot recover an in-flight attempt.
type ApplicationTransaction struct {
	r       *Registry
	current ApplicationAttempt
	active  bool
}

func (tx *ApplicationTransaction) Current() ApplicationAttempt { return tx.current }

func (r *Registry) WithApplication(ctx context.Context, id string, fn func(*ApplicationTransaction) error) error {
	if !validRef(id) || fn == nil {
		return ErrApplicationConflict
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Check existence before creating any per-attempt file.
	all, err := r.Applications(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, a := range all {
		if a.ID == id {
			found = true
		}
	}
	if !found {
		return ErrApplicationConflict
	}
	f, err := privateFile(filepath.Join(r.dir, "application-"+id+".lock"), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return ErrApplicationJournal
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrApplicationBusy
		}
		return ErrApplicationJournal
	}
	all, err = r.Applications(ctx)
	if err != nil {
		return err
	}
	for _, a := range all {
		if a.ID == id {
			tx := &ApplicationTransaction{r: r, current: a, active: true}
			defer func() { tx.active = false }()
			return fn(tx)
		}
	}
	return ErrApplicationConflict
}

// Advance permits only forward lifecycle transitions. Unknown outcomes can be
// cleaned up, never sent again; delivery's reconciliation flag is sticky.
func (tx *ApplicationTransaction) Advance(ctx context.Context, phase, outcome string) error {
	if !tx.active {
		return ErrApplicationConflict
	}
	old, next := tx.current, tx.current
	allowed := false
	switch old.Phase {
	case "approved":
		allowed = phase == "issuing" || phase == "closed" && outcome == "not-issued"
	case "issuing":
		allowed = phase == "saved" || phase == "unresolved" || phase == "revoking" || phase == "closed" && outcome == "not-issued"
	case "saved":
		allowed = phase == "delivering" || phase == "revoking" || phase == "unresolved"
	case "delivering":
		allowed = phase == "connected" || phase == "revoking" || phase == "unresolved"
	case "connected":
		allowed = phase == "revoking"
	case "unresolved":
		allowed = phase == "revoking"
	case "revoking":
		allowed = phase == "revoked" || phase == "unresolved"
	case "revoked":
		allowed = phase == "closed" && outcome == old.Outcome
	}
	if !allowed || old.Revision == ^uint64(0) {
		return ErrApplicationConflict
	}
	next.Phase, next.Outcome, next.Revision, next.UpdatedAt = phase, outcome, old.Revision+1, time.Now().UTC()
	next.ReceiverReconciliation = old.ReceiverReconciliation || phase == "delivering"
	if !next.valid() {
		return ErrApplicationConflict
	}
	return tx.replace(ctx, old, next)
}

// Reconciled clears the sticky fence only for remotely closed attempts. The
// engine calls it after a matching acknowledgement from the approved artifact.
func (tx *ApplicationTransaction) Reconciled(ctx context.Context, outcome string) error {
	old, next := tx.current, tx.current
	if !tx.active || old.Phase != "closed" || !old.ReceiverReconciliation ||
		(old.Outcome != "revoked" && old.Outcome != "rejected") || old.Revision == ^uint64(0) {
		return ErrApplicationConflict
	}
	next.ReceiverOutcome, next.ReceiverReconciliation = outcome, false
	next.Revision++
	next.UpdatedAt = time.Now().UTC()
	if outcome == "" || !next.valid() {
		return ErrApplicationConflict
	}
	return tx.replace(ctx, old, next)
}

func (tx *ApplicationTransaction) replace(ctx context.Context, old, next ApplicationAttempt) error {
	err := tx.r.locked(ctx, func() error {
		state, err := tx.r.readApplications()
		if err != nil {
			return err
		}
		for i, a := range state.Attempts {
			if a.ID != old.ID {
				continue
			}
			if a != old {
				return ErrApplicationConflict
			}
			state.Attempts[i] = next
			return tx.r.writeFile("applications.json", state)
		}
		return ErrApplicationConflict
	})
	if err == nil {
		tx.current = next
	}
	return err
}

// Keep corruption or duplicate JSON keys from changing approval evidence while
// it is decoded. The byte bound above and this depth bound limit parser work.
func uniqueApplicationJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 8 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delim, container := token.(json.Delim)
		if !container {
			return true
		}
		if delim != '{' && delim != '[' {
			return false
		}
		keys := map[string]bool{}
		for d.More() {
			if delim == '{' {
				key, err := d.Token()
				name, ok := key.(string)
				// encoding/json accepts case-insensitive matches to struct tags.
				name = strings.ToLower(name)
				if err != nil || !ok || keys[name] {
					return false
				}
				keys[name] = true
			}
			if !value(depth + 1) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && (delim == '{' && end == json.Delim('}') || delim == '[' && end == json.Delim(']'))
	}
	return value(0) && d.Decode(new(any)) == io.EOF
}

// ApplicationCacheDir holds retained nonsecret receiver generations.
func (r *Registry) ApplicationCacheDir() string { return filepath.Join(r.dir, "application-receivers") }
