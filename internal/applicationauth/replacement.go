package applicationauth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

// ConnectReplacement consumes one durable lineage intent. Only this path may
// prepare a receiver and replace its existing account; ordinary Connect cannot
// turn a replacement into an initial offer. A partial attempt is never replayed.
func (e *Engine) ConnectReplacement(ctx context.Context, id string, receiver ReplacementReceiver) (collection.ApplicationAttempt, error) {
	return e.connect(ctx, id, receiver, true)
}

func replacementBinding(a collection.ApplicationAttempt) map[string]any {
	return map[string]any{"version": 1, "request_id": a.ID, "connection_id": a.ConnectionID,
		"plugin_id": a.Target.PluginID, "profile": a.Target.Profile, "origin": a.Origin, "ship": a.Ship}
}

func (e *Engine) prepareReplacement(ctx context.Context, tx *collection.ApplicationTransaction, receiver ReplacementReceiver, fingerprint *[32]byte) (collection.ApplicationAttempt, error) {
	a := tx.Current()
	previous, archived, err := e.registry.Application(ctx, a.Replacement.PreviousID)
	if err != nil || archived || previous.Phase != "retiring" || previous.Replacement.SuccessorID != "" || previous.Target != a.Target || previous.Fleet.Request.Binding != a.Fleet.Request.Binding {
		return previous, collection.ErrApplicationConflict
	}
	if err := e.registry.CheckFleetApplicationPermission(ctx, a); err != nil {
		return previous, err
	}
	store, err := e.store(previous)
	if err != nil {
		return previous, credentials.ErrStore
	}
	data, err := store.Load(ctx, credentialKey(previous))
	defer clear(data)
	if err != nil {
		return previous, credentials.ErrStore
	}
	if err := eyre.ValidateStoredSession(previous.Origin, previous.Ship, append([]byte(nil), data...)); err != nil {
		return previous, err
	}
	*fingerprint = replacementCredentialFingerprint(data)
	if err := tx.Advance(ctx, "preparing", ""); err != nil {
		return previous, err
	}
	request := replacementBinding(previous)
	request["action"], request["next_request_id"] = "prepare-replacement", a.ID
	raw, _ := json.Marshal(request)
	reply, err := receiver.PrepareReplacement(ctx, raw)
	defer clear(reply)
	if err != nil || !boundReplyWith(reply, previous, "prepared", map[string]string{"next_request_id": a.ID}) {
		// Preparation has no login or account-install effects. The old receiver
		// may be paused; preserve that parent and never restore it by inference.
		recovery, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		return previous, errors.Join(ErrNotIssued, ErrDelivery, tx.Advance(recovery, "closed", "not-issued"))
	}
	return previous, tx.Advance(ctx, "prepared", "")
}

// RecoverReplacement observes a lost delivery acknowledgement using its exact
// retained helper. It never prepares, logs in, or delivers another credential.
// Positive acceptance is recorded before either continuing or honoring revoke.
func (e *Engine) RecoverReplacement(ctx context.Context, id string, receiver ReplacementReceiver) (collection.ApplicationAttempt, error) {
	ctx, stop := context.WithTimeout(ctx, 45*time.Second)
	defer stop()
	var result collection.ApplicationAttempt
	err := e.registry.WithApplication(ctx, id, func(tx *collection.ApplicationTransaction) error {
		defer func() { result = tx.Current() }()
		a := tx.Current()
		if a.Replacement.PreviousID == "" || receiver == nil || receiver.Target() != a.Target || e.expectedRevision != 0 && e.expectedRevision != a.Revision {
			return collection.ErrApplicationConflict
		}
		if !receiver.SupportsReplacement() {
			return ErrArtifact
		}
		if a.Phase == "connected" {
			return nil
		}
		if a.Phase != "delivering" {
			return ErrRecovery
		}
		return e.observeReplacement(ctx, tx, receiver)
	})
	return result, err
}

func (e *Engine) observeReplacement(ctx context.Context, tx *collection.ApplicationTransaction, receiver ReplacementReceiver) error {
	a := tx.Current()
	if err := receiver.Verify(ctx); err != nil {
		return ErrArtifact
	}
	request := replacementBinding(a)
	request["action"] = "replacement-status"
	raw, _ := json.Marshal(request)
	reply, err := receiver.ReplacementStatus(ctx, raw)
	defer clear(reply)
	if err != nil {
		return ErrRecovery
	}
	status := ""
	for _, candidate := range []string{"accepted", "pending", "absent", "superseded"} {
		if boundReply(reply, a, candidate, "") {
			status = candidate
			break
		}
	}
	if status == "" {
		return ErrRecovery
	}
	store, err := e.store(a)
	if err != nil {
		return ErrCleanup
	}
	data, err := store.Load(ctx, credentialKey(a))
	defer clear(data)
	if err != nil {
		return ErrCleanup
	}
	// Validation consumes its buffer; preserve the original for exact cleanup.
	copy := append([]byte(nil), data...)
	if err := eyre.ValidateStoredSession(a.Origin, a.Ship, copy); err != nil {
		return err
	}
	if status != "accepted" {
		return errors.Join(ErrDelivery, e.cleanup(ctx, tx, store, data))
	}
	if err := tx.Advance(ctx, "connected", ""); err != nil {
		return err
	}
	return e.replacementPermissionAfterAcceptance(ctx, tx, store, data)
}

func (e *Engine) replacementPermissionAfterAcceptance(ctx context.Context, tx *collection.ApplicationTransaction, store credentials.Store, data []byte) error {
	err := e.registry.CheckFleetApplicationGrant(ctx, tx.Current())
	if err == nil {
		return nil
	}
	if !errors.Is(err, collection.ErrApplicationPermission) {
		return err
	}
	cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer stop()
	return errors.Join(err, e.cleanup(cleanup, tx, store, data))
}

// Compare only ship/cookie authority, excluding origin and expiry: a fresh
// login must not reuse the predecessor token even through a second URL. Never
// revoke such a response as though it were an independent successor session.
func replacementCredentialFingerprint(data []byte) [32]byte {
	var s struct {
		Ship  string `json:"ship"`
		Name  string `json:"cookie_name"`
		Value string `json:"cookie_value"`
	}
	if json.Unmarshal(data, &s) != nil || s.Ship == "" || s.Name == "" || s.Value == "" {
		return [32]byte{}
	}
	raw, _ := json.Marshal(s)
	defer clear(raw)
	return sha256.Sum256(raw)
}
