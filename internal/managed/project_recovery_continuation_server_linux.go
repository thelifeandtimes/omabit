//go:build linux && amd64

package managed

import (
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type projectRecoveryContinuationBackend interface {
	projectAnchorRecoveryBackend
	projectCreatedAnchorReplacementBackend
}

// Shared public-command and qualification queue. The fresh request owns new continuation
// authority while the original parent and creator identities remain unchanged.
func (r *Server) queueProjectRecoveryContinuation(actor, project string, req projectRecoveryContinuationRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectRecoveryContinuationBackend)
	if !ok {
		return protocol.Operation{}, false, bad("explicit recovery continuation is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectRecoveryContinuation(actor, project, req)
	if err != nil || replayed {
		return op, replayed, err
	}
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { r.mu.Lock(); r.updateWorkers--; r.mu.Unlock() }()
		var runErr error
		select {
		case r.workGate <- struct{}{}:
			defer func() { <-r.workGate }()
			runErr = r.executeProjectRecoveryContinuation(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptRecoveryContinuation(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectRecoveryContinuation(b projectRecoveryContinuationBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := projectMaintenanceClear(r.s); err != nil {
		return err
	}
	if err := validateProjectAnchorRecoveries(r.s.db); err != nil {
		return err
	}
	if err := validateProjectRecoveryObservations(r.s.db); err != nil {
		return err
	}
	if err := validateProjectRecoveryContinuations(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	if actual.Kind != "continue-project-recovery" || actual.ProjectID != op.ProjectID || actual.Actor != op.Actor || actual.Status != "queued" {
		return errors.New("recovery continuation is not freshly queued")
	}
	j, err := r.s.beginProjectRecoveryContinuation(op.ID)
	if err != nil {
		return err
	}
	parent, err := loadProjectAnchorRecovery(r.s.db, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	if err = r.ctx.Err(); err != nil {
		return err
	}
	return r.driveProjectAnchorRecovery(b, b, parent, func(expected uint64, phase string, u projectnetwork.AnchorRecoveryJournal) (projectAnchorRecoveryJob, error) {
		return r.s.advanceProjectRecoveryContinuation(op.ID, expected, phase, u)
	})
}
