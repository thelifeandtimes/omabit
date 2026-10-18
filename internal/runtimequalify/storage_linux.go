//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func (q *qualifier) pierPath() string {
	if q.d.managed {
		p, _ := pierstore.Absolute(q.d.home, q.d.storage.Path) // managedSession/Check validate the root before use
		return p
	}
	return filepath.Join(q.r.Directory, "data", q.epochPierName())
}
func (q *qualifier) dataMount() mount {
	if q.d.managed {
		// The image declares VOLUME /urbit. Bind the pier at that exact path so
		// Docker does not create an additional anonymous parent volume.
		return mount{"bind", filepath.Dir(q.pierPath()), "/urbit", true, "rprivate"}
	}
	return mount{"bind", filepath.Join(q.r.Directory, "data"), "/urbit", true, "rprivate"}
}

func ListPierSources(ctx context.Context) ([]pierstore.Candidate, error) {
	q := &qualifier{d: defaults()}
	if err := q.accountBoundary(false); err != nil {
		return nil, err
	}
	return pierstore.Scan(ctx, q.d.home)
}

func applyPierEnrollment(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || state.Prepared || state.Storage == nil || state.Adoption == nil || !state.Adoption.Uncertain {
		return state, unknown, errors.New("enrollment requires its fresh durable storage intent")
	}
	p, err := pierstore.Check(d.home, *state.Storage)
	if err != nil {
		return state, unknown, err
	}
	if pierstore.Busy(p) || state.Adoption.Fingerprint != pierstore.Fingerprint(state.Storage.Identity) {
		return state, unknown, errors.New("source changed or still has a Vere lock")
	}
	if err := pierstore.CheckStateFiles(ctx, p); err != nil {
		return state, unknown, err
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err = q.accountBoundary(false); err != nil {
		return state, unknown, err
	}
	dir := filepath.Join(q.base, state.ID)
	if _, err = os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		return state, unknown, errors.New("enrollment working directory already exists")
	}
	state.Prepared = true
	q.r = q.enrollmentRuntime(state.ID)
	state.Runtime = q.r
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	for _, p := range []string{dir, q.config} {
		if err = os.Mkdir(p, 0700); err != nil {
			return state, unknown, err
		}
	}
	// Validate the pinned epoch layout without opening LMDB or starting Vere.
	inv, err := q.inspectImportEpochs(ctx)
	if err != nil {
		return state, unknown, err
	}
	// Publish initialized/adoption/import completion as one state. A failed
	// callback must leave the caller's retained enrollment intent recoverable.
	completed := state
	adoption := *state.Adoption
	completed.Adoption = &adoption
	if state.Comet != nil {
		comet := *state.Comet
		completed.Comet = &comet
	}
	if state.ExternalImport != nil {
		intent := *state.ExternalImport
		completed.ExternalImport = &intent
		intent.Events = append([]protocol.ImportEvent(nil), intent.Events...)
		intent.Stage = "complete"
		intent.Events = append(intent.Events, protocol.ImportEvent{Stage: "complete", At: time.Now().UTC()})
		if len(intent.Events) > 16 {
			intent.Events = intent.Events[len(intent.Events)-16:]
		}
	}
	now := time.Now().UTC()
	completed.Initialized = true
	completed.Runtime = q.r
	completed.Epochs = inv
	completed.Adoption.FinishedAt = &now
	completed.Adoption.Uncertain = false
	if completed.Comet != nil {
		completed.Comet.StartedAt = &now
		completed.Comet.FinishedAt = &now
		completed.Comet.Uncertain = false
	}
	if err = persist(completed); err != nil {
		return state, unknown, err
	}
	return completed, ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

// This is management workspace provenance only. It does not create or adopt
// storage, provision a pill/key, or dispatch a runtime.
func (q *qualifier) enrollmentRuntime(id string) Result {
	dir := filepath.Join(q.base, id)
	return Result{Profile: q.d.profile, RunID: id, Directory: dir, Image: candidate.Image, Platform: candidate.Platform, EngineVersion: rootless.EngineVersion, VereVersion: candidate.VereVersion, PillURL: q.pillSource(), PillSHA256: q.d.pillSHA, PillBytes: q.d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Containers: []ContainerRecord{}}
}

// applyLegacyImportCompletionRecovery handles the historical split write:
// enrollment finished, but its enclosing import still says enrolling. Reuse the
// normal recovery's read-only source, mount, child, lock and epoch validation;
// never re-enroll or reconstruct a pier. Admission and execution use the same
// normalization of the historical finished evidence.
func applyLegacyImportCompletionRecovery(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	candidate, eligible := state.legacyImportCompletionCandidate()
	if persist == nil || !eligible {
		return state, unknown, errors.New("legacy completion requires finished enrollment evidence")
	}
	// applyImportRecovery checks the full unchanged eligibility predicate on
	// this normalized proposal, and never persists it as an intermediate state.
	completed := state
	_, obs, err := applyImportRecovery(ctx, candidate, func(proposed ManagedState) error {
		finished := *state.Adoption.FinishedAt
		proposed.Adoption.FinishedAt = &finished
		if proposed.Comet != nil {
			started, finished := *state.Comet.StartedAt, *state.Comet.FinishedAt
			proposed.Comet.StartedAt, proposed.Comet.FinishedAt = &started, &finished
		}
		if err := persist(proposed); err != nil {
			return err
		}
		completed = proposed
		return nil
	}, d)
	if err != nil {
		return state, unknown, err
	}
	return completed, obs, nil
}

func applyPierMove(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || !pierstore.ValidPath(state.MoveTarget) || state.UncertainMaintenance() || state.UncertainProvisioning() {
		return state, unknown, errors.New("pier move requires a reviewed clean stopped registration")
	}
	for _, c := range state.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return state, unknown, errors.New("stop and remove owned containers before moving a pier")
		}
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err = q.accountBoundary(false); err != nil {
		return state, unknown, err
	}
	target, err := pierstore.Parents(d.home, state.MoveTarget)
	if err != nil {
		return state, unknown, err
	}
	if _, err = os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return state, unknown, errors.New("destination already exists; discovery enrollment is a separate action")
	}
	if !state.Prepared {
		state.Storage = &pierstore.Location{Path: state.MoveTarget}
		state.MoveTarget = ""
		return state, ManagedObservation{State: "uninitialized", Health: "unknown"}, persist(state)
	}
	if !state.Initialized {
		return state, unknown, errors.New("finish or review bootstrap before moving its storage")
	}
	if err = q.pier(); err != nil {
		return state, unknown, err
	}
	source := q.pierPath()
	identity, err := pierstore.Identify(source)
	if err != nil {
		return state, unknown, err
	}
	if pierstore.Busy(source) {
		return state, unknown, errors.New("stop the pier before moving its storage")
	}
	// Journal uncertainty before the no-replacement rename. A lost outcome must
	// never be interpreted as permission to repeat a move or recreate storage.
	state.Storage = &pierstore.Location{Path: state.MoveTarget, Identity: identity, MountIdentity: state.Storage.MountIdentity, Uncertain: true}
	state.Runtime.Stage = "moving-pier"
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	if err = pierstore.Move(source, target); err != nil {
		return state, unknown, err
	}
	state.Storage.Uncertain = false
	state.MoveTarget = ""
	state.Epochs = nil
	if err = persist(state); err != nil {
		state.Storage.Uncertain = true
		return state, unknown, err
	}
	return state, ManagedObservation{State: "stopped", Health: "unknown"}, nil
}
