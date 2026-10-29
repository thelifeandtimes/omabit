//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type projectMemberStorageDriver struct {
	q          *qualifier
	plan       projectnetwork.Plan
	initial    projectnetwork.MemberStorage
	dispatched bool
	created    pierstore.DirectoryIdentity
}

// OpenProjectMemberStorage grants only the accepted fresh storage operation.
// The caller holds the runner resource/update fence, and must persist dispatch
// before invoking CreateMount. No engine socket or ship process is accessed.
func OpenProjectMemberStorage(ctx context.Context, plan projectnetwork.Plan, state projectnetwork.MemberStorage) (projectnetwork.MemberStorageDriver, error) {
	return openProjectMemberStorage(ctx, plan, state, defaults())
}
func openProjectMemberStorage(ctx context.Context, plan projectnetwork.Plan, state projectnetwork.MemberStorage, d dependencies) (projectnetwork.MemberStorageDriver, error) {
	if state.Validate(plan) != nil || state.Phase != "planned" {
		return nil, errors.New("storage driver requires fresh accepted member intent")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := &qualifier{d: d}
	if err := q.accountBoundary(false); err != nil {
		return nil, err
	}
	plan.Members = slices.Clone(plan.Members)
	return &projectMemberStorageDriver{q: q, plan: plan, initial: state}, nil
}
func (d *projectMemberStorageDriver) selected(ctx context.Context, s projectnetwork.MemberStorage, phase string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Validate(d.plan) != nil || s.Phase != phase {
		return errors.New("member storage selection phase changed")
	}
	base := s
	base.Phase = "planned"
	base.Location.MountIdentity = pierstore.DirectoryIdentity{}
	if !reflect.DeepEqual(base, d.initial) {
		return errors.New("member storage selection changed")
	}
	return d.q.accountBoundary(false)
}
func (d *projectMemberStorageDriver) CreateMount(ctx context.Context, s projectnetwork.MemberStorage) (pierstore.DirectoryIdentity, error) {
	if err := d.selected(ctx, s, "dispatched"); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if d.dispatched {
		return pierstore.DirectoryIdentity{}, errors.New("member storage creation cannot be retried")
	}
	d.dispatched = true
	loc := s.Location
	_, err := pierstore.PrepareMount(d.q.d.home, &loc)
	d.created = loc.MountIdentity
	return d.created, errors.Join(err, ctx.Err())
}
func (d *projectMemberStorageDriver) InspectMount(ctx context.Context, s projectnetwork.MemberStorage) error {
	if err := d.selected(ctx, s, "created"); err != nil {
		return err
	}
	if !d.dispatched || d.created != (s.Location.MountIdentity) {
		return errors.New("member mount was not created by this dispatch")
	}
	p, err := pierstore.CheckMount(d.q.d.home, s.Location)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return errors.New("fresh member mount contains unexpected data")
	}
	if err = pierstore.SyncDestinationParents(d.q.d.home, s.Location, nil); err != nil {
		return err
	}
	return ctx.Err()
}
