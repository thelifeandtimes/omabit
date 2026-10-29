//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"io"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type privateWorkspaceDriver interface {
	privatechain.WorkspaceDriver
	io.Closer
}

type privateWorkspaceBackend interface {
	InspectPrivateWorkspaceParent(context.Context, string) (pierstore.DirectoryIdentity, error)
	OpenPrivateWorkspace(context.Context, string, privatechain.Plan, privatechain.Workspace) (privateWorkspaceDriver, error)
}

func (realBackend) InspectPrivateWorkspaceParent(ctx context.Context, directory string) (pierstore.DirectoryIdentity, error) {
	return privatechain.InspectWorkspaceParent(ctx, directory)
}
func (realBackend) OpenPrivateWorkspace(ctx context.Context, directory string, plan privatechain.Plan, w privatechain.Workspace) (privateWorkspaceDriver, error) {
	driver, err := privatechain.OpenWorkspaceFilesystem(ctx, directory, w.Parent, plan, w)
	if err != nil {
		return nil, err
	}
	return driver, nil
}

// Internal scheduler only. There is no public route or startup auto-dispatch.
// A first empty shared parent is provisioned only after fresh request review;
// retained workspace claims always require their original existing parent.
// All paths come from the runner's own opened store, never a client request.
func (r *Server) queuePrivateWorkspace(actor, project string, req privateWorkspaceRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.privateLifecycleAdmissionLocked(); err != nil {
		return protocol.Operation{}, false, err
	}
	// Replays cannot touch the parent, open a driver or enqueue a second worker.
	lookup := req
	lookup.ReplayOnly = true
	op, hit, err := r.s.acceptPrivateWorkspace(actor, project, lookup, pierstore.DirectoryIdentity{})
	if privateLifecycleReplayDone(hit, err, req.ReplayOnly) {
		return op, hit, err
	}
	if _, err = reviewPrivateWorkspace(r.s.db, project, req); err != nil {
		return protocol.Operation{}, false, err
	}
	b, ok := r.b.(privateWorkspaceBackend)
	if !ok {
		return protocol.Operation{}, false, bad("private workspace preparation is unavailable on this runner")
	}
	parent, err := r.privateWorkspaceParent(b)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op, hit, err = r.s.acceptPrivateWorkspace(actor, project, req, parent)
	if err != nil || hit {
		return op, hit, err
	}
	r.startPrivateLifecycleWorker(func() error { return r.executePrivateWorkspace(b, op) }, func() error { return r.s.interruptPrivateWorkspace(op.ProjectID, op.ID) })
	return op, false, nil
}

func (r *Server) executePrivateWorkspace(b privateWorkspaceBackend, op protocol.Operation) error {
	if err := r.privateLifecycleQueued(op, privateWorkspaceKind); err != nil {
		return err
	}
	w, err := loadPrivateWorkspace(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if op.ProjectID != p.Binding.ProjectID || op.ConfigRevision != p.Revision || w.OperationID != op.ID || w.Phase != "planned" {
		return errors.New("private workspace is no longer freshly queued")
	}
	driver, err := b.OpenPrivateWorkspace(r.ctx, filepath.Join(r.s.directory, "private-projects"), p.Plan, w)
	if err = privateLifecycleOpenError(driver, err); err != nil {
		return err
	}
	runErr := r.s.runPrivateWorkspace(r.ctx, op.ProjectID, op.ID, driver)
	return errors.Join(runErr, driver.Close())
}
