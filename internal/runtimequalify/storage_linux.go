//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func (q *qualifier) pierPath() string {
	if q.d.managed {
		return filepath.Join(pierstore.Root(q.d.home), q.d.storage.Path)
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
	q.r = Result{Profile: q.d.profile, RunID: state.ID, Directory: dir, Image: candidate.Image, Platform: candidate.Platform, EngineVersion: rootless.EngineVersion, VereVersion: candidate.VereVersion, PillURL: candidate.PillURL, PillSHA256: d.pillSHA, PillBytes: d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Containers: []ContainerRecord{}}
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
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return state, unknown, err
	}
	now := time.Now().UTC()
	state.Initialized = true
	state.Runtime = q.r
	state.Epochs = inv
	state.Adoption.FinishedAt = &now
	state.Adoption.Uncertain = false
	if state.Comet != nil {
		state.Comet.StartedAt = &now
		state.Comet.FinishedAt = &now
		state.Comet.Uncertain = false
	}
	if err = persist(state); err != nil {
		state.Adoption.Uncertain = true
		return state, unknown, err
	}
	return state, ManagedObservation{State: "stopped", Health: "unknown"}, nil
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
