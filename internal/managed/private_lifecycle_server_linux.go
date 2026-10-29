//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type privateGatewayDriver interface {
	privatechain.GatewayDriver
	io.Closer
}
type privateWorkspaceObserver interface {
	privatechain.WorkspaceObserver
	io.Closer
}
type privateGatewayObserver interface {
	privatechain.GatewayObserver
	io.Closer
}

type privateGatewayBackend interface {
	// Inputs/engine are backend-selected after replay lookup. The directory is
	// the runner's own state root, not a client-supplied runtime path.
	InspectPrivateGatewayInputs(context.Context, string, privatechain.Plan, privatechain.Workspace, string) (string, privatechain.RuntimeInputInspection, error)
	OpenPrivateGateway(context.Context, string, privatechain.Plan, privatechain.Workspace, privatechain.Gateway) (privateGatewayDriver, error)
}
type privateWorkspaceObservationBackend interface {
	OpenPrivateWorkspaceObserver(context.Context, string, privatechain.Plan, privatechain.Workspace, string) (privateWorkspaceObserver, error)
}
type privateGatewayObservationBackend interface {
	OpenPrivateGatewayObserver(context.Context, string, privatechain.Plan, privatechain.Workspace, privatechain.Gateway, string) (privateGatewayObserver, error)
}

func (realBackend) OpenPrivateWorkspaceObserver(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, _ string) (privateWorkspaceObserver, error) {
	d, err := privatechain.OpenWorkspaceFilesystem(ctx, directory, w.Parent, p, w)
	if err != nil {
		return nil, err
	}
	// Do not expose creation methods through the observation factory.
	return struct {
		privatechain.WorkspaceObserver
		io.Closer
	}{d, d}, nil
}

func (r *Server) privateLifecycleAdmissionLocked() error {
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	return nil
}
func privateLifecycleReplayDone(hit bool, err error, replayOnly bool) bool {
	var problem *fault
	return err == nil || hit || replayOnly || !errors.As(err, &problem) || problem.problem.Code != "replay_not_found"
}

// Caller holds r.mu and has committed the accepted receipt. These operations
// share the existing host work gate and management-update worker count. Startup
// only interrupts their receipts; it never calls this scheduler.
func (r *Server) startPrivateLifecycleWorker(execute func() error, interrupt func() error) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { r.mu.Lock(); r.updateWorkers--; r.mu.Unlock() }()
		var runErr error
		select {
		case r.workGate <- struct{}{}:
			defer func() { <-r.workGate }()
			runErr = execute()
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := interrupt(); err != nil {
				r.failed.Store(true)
			}
		}
	}()
}

func (r *Server) queuePrivateGateway(actor, project string, req privateGatewayRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.privateLifecycleAdmissionLocked(); err != nil {
		return protocol.Operation{}, false, err
	}
	lookup := req
	lookup.ReplayOnly = true
	op, hit, err := r.s.acceptPrivateGateway(actor, project, lookup, "", privatechain.RuntimeInputInspection{})
	if privateLifecycleReplayDone(hit, err, req.ReplayOnly) {
		return op, hit, err
	}
	b, ok := r.b.(privateGatewayBackend)
	if !ok {
		return protocol.Operation{}, false, bad("private gateway initialization is unavailable on this runner")
	}
	p, err := loadPrivateProject(r.s.db, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	w, err := readyPrivateWorkspace(r.s.db, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	engine, inputs, err := b.InspectPrivateGatewayInputs(r.ctx, r.s.directory, p.Plan, w, req.InputManifestSHA256)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op, hit, err = r.s.acceptPrivateGateway(actor, project, req, engine, inputs)
	if err != nil || hit {
		return op, hit, err
	}
	r.startPrivateLifecycleWorker(func() error { return r.executePrivateGateway(b, op) }, func() error { return r.s.interruptPrivateGateway(op.ProjectID, op.ID) })
	return op, false, nil
}
func (r *Server) queuePrivateWorkspaceObservation(actor, project string, req privateWorkspaceObservationRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.privateLifecycleAdmissionLocked(); err != nil {
		return protocol.Operation{}, false, err
	}
	lookup := req
	lookup.ReplayOnly = true
	op, hit, err := r.s.acceptPrivateWorkspaceObservation(actor, project, lookup)
	if privateLifecycleReplayDone(hit, err, req.ReplayOnly) {
		return op, hit, err
	}
	b, ok := r.b.(privateWorkspaceObservationBackend)
	if !ok {
		return protocol.Operation{}, false, bad("private workspace observation is unavailable on this runner")
	}
	op, hit, err = r.s.acceptPrivateWorkspaceObservation(actor, project, req)
	if err != nil || hit {
		return op, hit, err
	}
	r.startPrivateLifecycleWorker(func() error { return r.executePrivateWorkspaceObservation(b, op) }, func() error { return r.s.interruptPrivateWorkspaceObservation(op.ID) })
	return op, false, nil
}
func (r *Server) queuePrivateGatewayObservation(actor, project string, req privateGatewayObservationRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.privateLifecycleAdmissionLocked(); err != nil {
		return protocol.Operation{}, false, err
	}
	lookup := req
	lookup.ReplayOnly = true
	op, hit, err := r.s.acceptPrivateGatewayObservation(actor, project, lookup)
	if privateLifecycleReplayDone(hit, err, req.ReplayOnly) {
		return op, hit, err
	}
	b, ok := r.b.(privateGatewayObservationBackend)
	if !ok {
		return protocol.Operation{}, false, bad("private gateway observation is unavailable on this runner")
	}
	op, hit, err = r.s.acceptPrivateGatewayObservation(actor, project, req)
	if err != nil || hit {
		return op, hit, err
	}
	r.startPrivateLifecycleWorker(func() error { return r.executePrivateGatewayObservation(b, op) }, func() error { return r.s.interruptPrivateGatewayObservation(op.ID) })
	return op, false, nil
}

func (r *Server) privateLifecycleQueued(op protocol.Operation, kind string) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if _, err := projectReservations(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	if actual.Kind != kind || actual.ProjectID != op.ProjectID || actual.ID != op.ID || actual.Actor != op.Actor || actual.RequestID != op.RequestID || actual.ConfigRevision != op.ConfigRevision || actual.Status != "queued" {
		return errors.New("private lifecycle receipt is no longer freshly queued")
	}
	return projectMaintenanceClear(r.s)
}
func missingPrivateLifecycleHandle(handle any) bool {
	if handle == nil {
		return true
	}
	value := reflect.ValueOf(handle)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

// Open failures may still return an owned handle. Close that handle exactly once;
// typed-nil handles are absent authority and must not be called.
func privateLifecycleOpenError(handle io.Closer, err error) error {
	if missingPrivateLifecycleHandle(handle) {
		return errors.Join(err, errors.New("private lifecycle driver is unavailable"))
	}
	if err != nil {
		return errors.Join(err, handle.Close())
	}
	return nil
}
func (r *Server) executePrivateGateway(b privateGatewayBackend, op protocol.Operation) error {
	if err := r.privateLifecycleQueued(op, privateGatewayKind); err != nil {
		return err
	}
	g, err := loadPrivateGateway(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	w, err := readyPrivateWorkspace(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if g.Selection.OperationID != op.ID || g.Phase != "planned" {
		return errors.New("private gateway selection changed")
	}
	d, err := b.OpenPrivateGateway(r.ctx, r.s.directory, p.Plan, w, g)
	if err = privateLifecycleOpenError(d, err); err != nil {
		return err
	}
	return errors.Join(r.s.runPrivateGateway(r.ctx, op.ProjectID, op.ID, d), d.Close())
}
func (r *Server) executePrivateWorkspaceObservation(b privateWorkspaceObservationBackend, op protocol.Operation) error {
	if err := r.privateLifecycleQueued(op, privateWorkspaceObservationKind); err != nil {
		return err
	}
	j, err := loadPrivateWorkspaceObservation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	w, err := loadPrivateWorkspace(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if j.Phase != "planned" || j.ProjectID != op.ProjectID || j.SourceOperationID != w.OperationID {
		return errors.New("private workspace observation selection changed")
	}
	d, err := b.OpenPrivateWorkspaceObserver(r.ctx, filepath.Join(r.s.directory, "private-projects"), p.Plan, w, op.ID)
	if err = privateLifecycleOpenError(d, err); err != nil {
		return err
	}
	return errors.Join(r.s.runPrivateWorkspaceObservation(r.ctx, op.ID, d), d.Close())
}
func (r *Server) executePrivateGatewayObservation(b privateGatewayObservationBackend, op protocol.Operation) error {
	if err := r.privateLifecycleQueued(op, privateGatewayObservationKind); err != nil {
		return err
	}
	j, err := loadPrivateGatewayObservation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	w, err := readyPrivateWorkspace(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	g, err := loadPrivateGateway(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if j.Phase != "planned" || j.ProjectID != op.ProjectID || j.SourceOperationID != g.Selection.OperationID {
		return errors.New("private gateway observation selection changed")
	}
	d, err := b.OpenPrivateGatewayObserver(r.ctx, r.s.directory, p.Plan, w, g, op.ID)
	if err = privateLifecycleOpenError(d, err); err != nil {
		return err
	}
	return errors.Join(r.s.runPrivateGatewayObservation(r.ctx, op.ID, d), d.Close())
}
