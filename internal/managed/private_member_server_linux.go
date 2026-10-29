//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type privateMemberLaunchDriver interface {
	privatechain.MemberLaunchDriver
	io.Closer
}
type privateMemberLaunchBackend interface {
	// Pure: no filesystem, engine, dependency or service work in the transaction.
	PrivateMemberLaunchContract(string, privatechain.Plan, privatechain.Workspace, privatechain.Gateway, string, string, string) (string, error)
	OpenPrivateMemberLaunch(context.Context, string, privatechain.Plan, privatechain.Workspace, privatechain.MemberLaunch) (privateMemberLaunchDriver, error)
}

func (realBackend) PrivateMemberLaunchContract(directory string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, instance, runtime, operation string) (string, error) {
	return runtimequalify.PrivateMemberLaunchContract(directory, p, w, g, instance, runtime, operation)
}
func (realBackend) OpenPrivateMemberLaunch(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch) (privateMemberLaunchDriver, error) {
	return runtimequalify.OpenPrivateMemberLaunch(ctx, directory, p, w, s)
}

// Internal admission only. Replay precedes backend lookup, and the durable
// acceptance precedes scheduling. No public route or restart dispatcher is added.
func (r *Server) queuePrivateMemberLaunch(actor, project, instance string, req privateMemberLaunchRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.privateLifecycleAdmissionLocked(); err != nil {
		return protocol.Operation{}, false, err
	}
	lookup := req
	lookup.ReplayOnly = true
	op, hit, err := r.s.acceptCapturedPrivateMemberLaunch(actor, project, instance, lookup, nil)
	if privateLifecycleReplayDone(hit, err, req.ReplayOnly) {
		return op, hit, err
	}
	b, ok := r.b.(privateMemberLaunchBackend)
	if !ok {
		return protocol.Operation{}, false, bad("private member launch is unavailable on this runner")
	}
	contract := func(p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, instance, runtime, operation string) (string, error) {
		return b.PrivateMemberLaunchContract(r.s.directory, p, w, g, instance, runtime, operation)
	}
	op, hit, err = r.s.acceptCapturedPrivateMemberLaunch(actor, project, instance, req, contract)
	if err != nil || hit {
		return op, hit, err
	}
	r.startPrivateLifecycleWorker(func() error { return r.executePrivateMemberLaunch(b, op) }, func() error { return r.s.interruptPrivateMemberLaunch(op.ProjectID, op.InstanceID, op.ID) })
	return op, false, nil
}
func (r *Server) executePrivateMemberLaunch(b privateMemberLaunchBackend, op protocol.Operation) error {
	if err := r.privateLifecycleQueued(op, privateMemberLaunchKind); err != nil {
		return err
	}
	j, err := loadPrivateMemberLaunch(r.s.db, op.InstanceID)
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
	if j.OperationID != op.ID || j.InstanceID != op.InstanceID || j.Gateway.Selection.Binding.ProjectID != op.ProjectID || j.Version != 3 || j.Phase != "planned" {
		return errors.New("private member launch selection changed")
	}
	d, err := b.OpenPrivateMemberLaunch(r.ctx, r.s.directory, p.Plan, w, j)
	if err = privateLifecycleOpenError(d, err); err != nil {
		return err
	}
	return errors.Join(r.s.runPrivateMemberLaunch(r.ctx, op.ProjectID, op.InstanceID, op.ID, d), d.Close())
}

var _ privateMemberLaunchBackend = realBackend{}
