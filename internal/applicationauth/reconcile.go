package applicationauth

import (
	"context"
	"encoding/json"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

type Reconciler interface {
	Receiver
	Reconcile(context.Context, []byte) ([]byte, error)
}

// Reconcile removes only the receiver state associated with this attempt, and
// only after remote cleanup and local issuer credential deletion are complete.
// A lost reply leaves the fence in place; the receiver operation is idempotent.
func (e *Engine) Reconcile(ctx context.Context, id string, receiver Reconciler) (collection.ApplicationAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
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
		if a.Phase != "closed" {
			return ErrRecovery
		}
		if !a.ReceiverReconciliation {
			return nil
		}
		if a.Outcome != "revoked" && a.Outcome != "rejected" {
			return ErrRecovery
		}
		if err := receiver.Verify(ctx); err != nil {
			return ErrArtifact
		}
		request, _ := json.Marshal(map[string]any{"version": 1, "action": "reconcile", "request_id": a.ID,
			"connection_id": a.ConnectionID, "plugin_id": a.Target.PluginID, "profile": a.Target.Profile,
			"origin": a.Origin, "ship": a.Ship})
		reply, err := receiver.Reconcile(ctx, request)
		defer clear(reply)
		if err != nil {
			return ErrRecovery
		}
		for _, outcome := range []string{"removed", "absent", "superseded"} {
			if boundReply(reply, a, "reconciled", outcome) {
				return tx.Reconciled(ctx, outcome)
			}
		}
		return ErrRecovery
	})
	return result, err
}
