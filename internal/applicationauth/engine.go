// Package applicationauth owns assisted-session issuance and cleanup. It never
// handles application data and is not needed by a connected plugin.
package applicationauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shipaccess"
)

var (
	ErrNotIssued = errors.New("application session was not issued; review the ship login or use standalone plugin sign-in")
	ErrRecovery  = errors.New("application connection needs reconciliation; delivery was not retried")
	ErrDelivery  = errors.New("application did not confirm the approved handoff")
	ErrCleanup   = errors.New("application session cleanup is pending; its credential reference was retained")
)

// Receiver must verify its immutable artifact before it receives private input.
// The trusted resolver supplies it, never an untrusted request's shell command.
// Deliver must be bounded, context-aware and return at most 4096 bytes. This
// internal interface is not yet an installed-artifact resolver or a public API.
type Receiver interface {
	Target() collection.ApplicationTarget
	Verify(context.Context) error
	Deliver(context.Context, []byte) ([]byte, error)
}

type Engine struct {
	registry         *collection.Registry
	expectedRevision uint64
	store            func(collection.ApplicationAttempt) (credentials.Store, error)
	issue            func(context.Context, collection.ApplicationAttempt) ([]byte, error)
	revoke           func(context.Context, collection.ApplicationAttempt, []byte) (string, error)
}

// New uses the connection's explicitly chosen credential store. There is no
// automatic plaintext fallback. No IO happens until an operation is requested.
func New(r *collection.Registry) *Engine {
	e := &Engine{registry: r}
	e.store = func(a collection.ApplicationAttempt) (credentials.Store, error) {
		if a.Credentials == "secret-service" {
			return credentials.NewSecretService(), nil
		}
		return credentials.NewFile(r.CredentialsDir())
	}
	e.issue = func(ctx context.Context, a collection.ApplicationAttempt) ([]byte, error) {
		ships, err := r.Ships(ctx)
		if err != nil {
			return nil, errors.Join(ErrNotIssued, err)
		}
		for _, s := range ships {
			if s.ID != a.ConnectionID {
				continue
			}
			if s.Revision != a.ConnectionRevision || s.Origin != a.Origin || s.Ship != a.Ship || s.Credentials != a.Credentials {
				return nil, errors.Join(ErrNotIssued, collection.ErrApplicationConflict)
			}
			parent, err := shipaccess.Open(ctx, r, s)
			if err != nil {
				return nil, errors.Join(ErrNotIssued, err)
			}
			defer parent.Close()
			child, err := parent.NewIndependentSession(ctx)
			if err != nil {
				if errors.Is(err, eyre.ErrIndependentSessionUnavailable) || errors.Is(err, eyre.ErrIndependentSessionNotIssued) {
					return nil, errors.Join(ErrNotIssued, err)
				}
				return nil, err
			}
			defer child.Close()
			return child.ExportSession()
		}
		return nil, errors.Join(ErrNotIssued, collection.ErrApplicationConflict)
	}
	e.revoke = func(ctx context.Context, a collection.ApplicationAttempt, data []byte) (string, error) {
		return eyre.RevokeStoredSession(ctx, a.Origin, a.Ship, data)
	}
	return e
}

// AtRevision returns an operation-scoped engine that checks the selected attempt
// revision while holding its transaction lock. Zero preserves the internal API.
func (e *Engine) AtRevision(revision uint64) *Engine {
	copy := *e
	copy.expectedRevision = revision
	return &copy
}

func credentialKey(a collection.ApplicationAttempt) string {
	// Domain separation from parent ship sessions, even with a malicious/corrupt
	// adapter response. This key can never address the owner's credential slot.
	return credentials.Scope("application:"+a.ID, a.Origin, a.SessionRef)
}

// Connect consumes an already durable approval once. A reopened partial attempt
// is handled by Recover; it is never implicitly issued or delivered again.
func (e *Engine) Connect(ctx context.Context, id string, receiver Receiver) (collection.ApplicationAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	var result collection.ApplicationAttempt
	err := e.registry.WithApplication(ctx, id, func(tx *collection.ApplicationTransaction) error {
		defer func() { result = tx.Current() }()
		a := tx.Current()
		if e.expectedRevision != 0 && a.Revision != e.expectedRevision {
			return collection.ErrApplicationConflict
		}
		if receiver == nil || receiver.Target() != a.Target {
			return collection.ErrApplicationConflict
		}
		if a.Phase == "connected" {
			return nil
		} // exact duplicate, no network/receiver IO
		if a.Phase != "approved" {
			return ErrRecovery
		}
		if err := receiver.Verify(ctx); err != nil {
			return ErrDelivery
		}
		store, err := e.store(a)
		if err != nil {
			return credentials.ErrStore
		}
		if err := tx.Advance(ctx, "issuing", ""); err != nil {
			return err
		}
		data, issueErr := e.issue(ctx, a)
		defer clear(data)
		if len(data) == 0 || issueErr != nil {
			// A lost login response can leave a server-side session whose token we
			// never received. Keep the attempt unresolved, including on cancellation.
			recoveryCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer stop()
			if len(data) == 0 && errors.Is(issueErr, ErrNotIssued) {
				return errors.Join(ErrNotIssued, tx.Advance(recoveryCtx, "closed", "not-issued"))
			}
			if len(data) > 0 {
				return errors.Join(ErrRecovery, e.cleanup(recoveryCtx, tx, store, data))
			}
			return errors.Join(ErrRecovery, tx.Advance(recoveryCtx, "unresolved", ""))
		}
		// Persist and read back before allowing the receiver to see the token.
		saveErr := store.Save(ctx, credentialKey(a), data)
		if saveErr == nil {
			var saved []byte
			saved, saveErr = store.Load(ctx, credentialKey(a))
			if saveErr == nil && !bytes.Equal(saved, data) {
				saveErr = credentials.ErrStore
			}
			clear(saved)
		}
		if saveErr != nil {
			recoveryCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer stop()
			return errors.Join(credentials.ErrStore, e.cleanup(recoveryCtx, tx, store, data))
		}
		if err := tx.Advance(ctx, "saved", ""); err != nil {
			return err
		}
		if err := receiver.Verify(ctx); err != nil {
			return e.failedDelivery(ctx, tx, store, data)
		}
		if err := tx.Advance(ctx, "delivering", ""); err != nil {
			return err
		}
		offer, err := json.Marshal(struct {
			Version      int             `json:"version"`
			RequestID    string          `json:"request_id"`
			ConnectionID string          `json:"connection_id"`
			PluginID     string          `json:"plugin_id"`
			Profile      string          `json:"profile"`
			Session      json.RawMessage `json:"session"`
		}{1, a.ID, a.ConnectionID, a.Target.PluginID, a.Target.Profile, json.RawMessage(data)})
		defer clear(offer)
		if err != nil || len(offer) > 16384 {
			return e.failedDelivery(ctx, tx, store, data)
		}
		deliveryCtx, stop := context.WithTimeout(ctx, 25*time.Second)
		reply, deliveryErr := receiver.Deliver(deliveryCtx, offer)
		stop()
		defer clear(reply)
		if deliveryErr != nil || !accepted(reply, a) {
			return e.failedDelivery(ctx, tx, store, data)
		}
		return tx.Advance(ctx, "connected", "")
	})
	return result, err
}

func (e *Engine) failedDelivery(ctx context.Context, tx *collection.ApplicationTransaction, store credentials.Store, data []byte) error {
	cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer stop()
	return errors.Join(ErrDelivery, e.cleanup(cleanupCtx, tx, store, data))
}

// Recover never delivers and never revokes a durably connected attempt. An
// explicit Disconnect is required for that transition. Both calls are safe to
// repeat after a restart, including remote logout succeeding before a crash.
func (e *Engine) Recover(ctx context.Context, id string) (collection.ApplicationAttempt, error) {
	return e.recover(ctx, id, false)
}

func (e *Engine) Disconnect(ctx context.Context, id string) (collection.ApplicationAttempt, error) {
	return e.recover(ctx, id, true)
}

func (e *Engine) recover(ctx context.Context, id string, disconnect bool) (collection.ApplicationAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var result collection.ApplicationAttempt
	err := e.registry.WithApplication(ctx, id, func(tx *collection.ApplicationTransaction) error {
		defer func() { result = tx.Current() }()
		a := tx.Current()
		if e.expectedRevision != 0 && a.Revision != e.expectedRevision {
			return collection.ErrApplicationConflict
		}
		if a.Phase == "closed" || a.Phase == "connected" && !disconnect {
			return nil
		}
		if a.Phase == "approved" {
			return tx.Advance(ctx, "closed", "not-issued")
		}
		store, err := e.store(a)
		if err != nil {
			return ErrCleanup
		}
		if a.Phase == "revoked" {
			return e.finishCleanup(ctx, tx, store)
		}
		if a.Phase == "connected" {
			if err := tx.Advance(ctx, "revoking", ""); err != nil {
				return err
			}
		}
		data, err := store.Load(ctx, credentialKey(a))
		defer clear(data)
		if err != nil {
			if errors.Is(err, credentials.ErrMissing) && tx.Current().Phase != "unresolved" {
				return errors.Join(ErrCleanup, tx.Advance(ctx, "unresolved", ""))
			}
			return ErrCleanup // unavailable keyring != absent session != revoked
		}
		return e.cleanup(ctx, tx, store, data)
	})
	return result, err
}

func (e *Engine) cleanup(ctx context.Context, tx *collection.ApplicationTransaction, store credentials.Store, data []byte) error {
	a := tx.Current()
	if a.Phase != "revoking" {
		if err := tx.Advance(ctx, "revoking", ""); err != nil {
			return err
		}
	}
	outcome, err := e.revoke(ctx, a, data)
	if err != nil || (outcome != "revoked" && outcome != "rejected") {
		return ErrCleanup
	}
	// Record remote evidence BEFORE removing the token. A restart at this point
	// can retry local deletion without needing the already-deleted credential.
	if err := tx.Advance(ctx, "revoked", outcome); err != nil {
		return err
	}
	return e.finishCleanup(ctx, tx, store)
}

func (e *Engine) finishCleanup(ctx context.Context, tx *collection.ApplicationTransaction, store credentials.Store) error {
	a := tx.Current()
	if err := store.Delete(ctx, credentialKey(a)); err != nil {
		return ErrCleanup
	}
	return tx.Advance(ctx, "closed", a.Outcome)
}

func accepted(data []byte, a collection.ApplicationAttempt) bool {
	return boundReply(data, a, "accepted", "")
}

func boundReply(data []byte, a collection.ApplicationAttempt, status, outcome string) bool {
	expectedFields := 6
	if outcome != "" {
		expectedFields++
	}
	if len(data) > 4096 {
		return false
	}
	// Decode tokens first so duplicate keys cannot change the apparent binding.
	d := json.NewDecoder(bytes.NewReader(data))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return false
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		key, err := d.Token()
		name, ok := key.(string)
		if err != nil || !ok || fields[name] != nil {
			return false
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return false
		}
		fields[name] = value
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') || d.Decode(new(any)) != io.EOF || len(fields) != expectedFields {
		return false
	}
	expected := map[string]string{"request_id": a.ID, "connection_id": a.ConnectionID, "plugin_id": a.Target.PluginID, "profile": a.Target.Profile, "status": status}
	if outcome != "" {
		expected["outcome"] = outcome
	}
	var version int
	if json.Unmarshal(fields["version"], &version) != nil || version != 1 {
		return false
	}
	for key, want := range expected {
		var got string
		if json.Unmarshal(fields[key], &got) != nil || got != want {
			return false
		}
	}
	return true
}
