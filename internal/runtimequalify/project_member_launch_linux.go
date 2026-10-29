//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This selection comes only from an accepted runner operation while holding the
// common resource/update fence. No paths, images or commands are RPC inputs.
type ProjectMemberLaunchSelection struct {
	Project     ProjectPreparationSelection
	Anchor      projectnetwork.AnchorStart
	Storage     projectnetwork.MemberStorage
	OperationID string
	Peers       []projectnetwork.MemberLaunch
}
type projectMemberLaunchEngine struct {
	anchor                                                 *projectAnchorEngine
	contract                                               projectMemberContract
	initial                                                projectnetwork.MemberLaunch
	peers                                                  []projectObservedMember
	workspace                                              pierstore.DirectoryIdentity
	verifiedPill                                           projectPillSnapshot
	workspaceDispatched, createDispatched, startDispatched bool
	createdID                                              string
	preflightAt, inspectedAt                               time.Time
	inspectedPhase                                         string
}

// OpenProjectMemberLaunch is read-only. It requires the pinned image and a
// separately staged pinned pill. Dependency transfer is a separate visible flow;
// this driver never downloads/pulls or modifies shared caches.
func OpenProjectMemberLaunch(ctx context.Context, s ProjectMemberLaunchSelection) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
	return openProjectMemberLaunch(ctx, s, defaults())
}
func openProjectMemberLaunch(ctx context.Context, s ProjectMemberLaunchSelection, d dependencies) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
	fail := func(err error) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
		return projectnetwork.MemberLaunch{}, nil, err
	}
	if s.Storage.Validate(s.Project.Plan) != nil || s.Storage.Phase != "observed" || s.Anchor.Validate(s.Project.Plan) != nil || s.Anchor.Phase != "observed" {
		return fail(errors.New("member launch needs completed storage and anchor receipts"))
	}
	// Derive the mount path from the validated storage receipt. No caller path is used.
	path, err := pierstore.Absolute(d.home, s.Storage.Location.Path)
	if err != nil {
		return fail(err)
	}
	selectContract := func(storage projectnetwork.MemberStorage, anchor projectnetwork.AnchorStart, operation, image string) (projectMemberContract, error) {
		p, err := pierstore.Absolute(d.home, storage.Location.Path)
		if err != nil {
			return projectMemberContract{}, err
		}
		return newProjectMemberContract(projectMemberSelection{Project: s.Project, Anchor: anchor, InstanceID: storage.InstanceID, RuntimeID: storage.RuntimeID, OperationID: operation, ContainerGeneration: 1, Mode: "boot", Storage: storage.Location}, d.home, filepath.Dir(p), image)
	}
	raw, err := openProjectAnchorStart(ctx, s.Project, s.Anchor.Prepared, d)
	if err != nil {
		return fail(err)
	}
	anchor := raw.(*projectAnchorEngine)
	c, err := selectContract(s.Storage, s.Anchor, s.OperationID, anchor.base.contract.ImageID)
	if err != nil {
		return fail(err)
	}
	if c.mountSource != filepath.Dir(path) {
		return fail(errors.New("member mount selection changed"))
	}
	var known []projectObservedMember
	var peers []projectnetwork.RuntimeMember
	for _, peer := range s.Peers {
		if peer.Validate(s.Project.Plan) != nil || peer.Phase != "running" || !reflect.DeepEqual(peer.Anchor, s.Anchor) {
			return fail(errors.New("member peer lacks its completed original launch"))
		}
		pc, err := selectContract(peer.Storage, peer.Anchor, peer.OperationID, c.imageID)
		if err != nil || pc.digest != peer.Container.ContractDigest {
			return fail(errors.New("member peer contract differs from its retained launch"))
		}
		known = append(known, projectObservedMember{contract: pc, id: peer.Container.ID})
		peers = append(peers, projectnetwork.RuntimeMember{InstanceID: peer.Storage.InstanceID, RuntimeID: peer.Storage.RuntimeID, Container: peer.Container})
	}
	slices.SortFunc(peers, func(a, b projectnetwork.RuntimeMember) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	initial, err := projectnetwork.NewMemberLaunch(s.Project.Plan, s.Anchor, s.Storage, s.OperationID, c.digest, peers)
	if err != nil {
		return fail(err)
	}
	e := &projectMemberLaunchEngine{anchor: anchor, contract: c, initial: initial, peers: known}
	if err = c.checkStorageBeforeCreate(ctx); err != nil {
		return fail(err)
	}
	if _, err = os.Lstat(filepath.Dir(c.pillPath)); !errors.Is(err, os.ErrNotExist) {
		return fail(errors.New("member workspace must be fresh; retain existing evidence for recovery"))
	}
	if err = e.checkPillCache(ctx); err != nil {
		return fail(err)
	}
	if err = anchor.base.q.admitStart(c.mountSource, false, "new project ship"); err != nil {
		return fail(err)
	}
	return initial, e, nil
}
func (e *projectMemberLaunchEngine) selected(ctx context.Context, s projectnetwork.MemberLaunch, phase string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Validate(e.contract.selection.Project.Plan) != nil || s.Phase != phase {
		return errors.New("member launch phase changed")
	}
	base := s
	base.Phase = "planned"
	base.Workspace = pierstore.DirectoryIdentity{}
	base.Container.ID = ""
	base.Namespace = nil
	if !reflect.DeepEqual(base, e.initial) {
		return errors.New("member launch binding changed")
	}
	if s.Workspace != (pierstore.DirectoryIdentity{}) && s.Workspace != e.workspace {
		return errors.New("member workspace did not originate in this dispatch")
	}
	if s.Container.ID != "" && s.Container.ID != e.createdID {
		return errors.New("member container did not originate in this dispatch")
	}
	return nil
}
func (e *projectMemberLaunchEngine) VerifyEngine(ctx context.Context, id string) error {
	err := e.anchor.VerifyEngine(ctx, id)
	if err != nil {
		e.preflightAt = time.Time{}
		e.inspectedAt = time.Time{}
	}
	return err
}
func (e *projectMemberLaunchEngine) pillMeta() protocol.BootPill {
	// Dependencies are injectable only through the private test constructor. The
	// public constructor uses the exact runtimecandidate pins from defaults().
	d := e.anchor.base.q.d
	return protocol.BootPill{SHA256: d.pillSHA, Bytes: d.pillBytes}
}
func (e *projectMemberLaunchEngine) checkPillCache(ctx context.Context) error {
	q := e.anchor.base.q
	p := e.pillMeta()
	path := bootpill.CachePath(q.base, p)
	if err := q.owned(filepath.Dir(path), os.ModeDir, true); err != nil {
		return errors.New("stage the pinned project boot pill before launching a member")
	}
	if err := q.owned(path, 0, true); err != nil {
		return errors.New("staged project boot pill ownership is unresolved")
	}
	return bootpill.Verify(ctx, path, p)
}
func (e *projectMemberLaunchEngine) workspaceQualifier() *qualifier {
	q := *e.anchor.base.q
	q.r.Directory = filepath.Dir(e.contract.pillPath)
	q.r.PillPath = e.contract.pillPath
	return &q
}
func (e *projectMemberLaunchEngine) PrepareWorkspace(ctx context.Context, s projectnetwork.MemberLaunch) (pierstore.DirectoryIdentity, error) {
	if err := e.selected(ctx, s, "workspace-dispatched"); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if e.workspaceDispatched {
		return pierstore.DirectoryIdentity{}, errors.New("member workspace preparation cannot replay")
	}
	e.workspaceDispatched = true
	if err := e.VerifyEngine(ctx, s.Anchor.Prepared.EngineID); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if err := e.contract.checkStorageBeforeCreate(ctx); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if err := e.checkPillCache(ctx); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	q := e.workspaceQualifier()
	if err := os.Mkdir(q.r.Directory, 0700); err != nil {
		return pierstore.DirectoryIdentity{}, errors.New("member workspace must be fresh; retain existing evidence")
	}
	id, err := pierstore.IdentifyMount(q.r.Directory)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	e.workspace = id
	// Preserve identity on every subsequent error. The controller commits this
	// evidence before returning uncertainty; no cleanup removes the directory.
	finish := func(err error) (pierstore.DirectoryIdentity, error) { return id, errors.Join(err, ctx.Err()) }
	if err = q.owned(q.r.Directory, os.ModeDir, true); err != nil {
		return finish(err)
	}
	parent, err := os.Open(q.base)
	if err != nil {
		return finish(err)
	}
	err = errors.Join(parent.Sync(), parent.Close())
	if err != nil {
		return finish(err)
	}
	source, err := bootpill.Open(bootpill.CachePath(q.base, e.pillMeta()))
	if err != nil {
		return finish(err)
	}
	err = bootpill.Publish(ctx, source, e.pillMeta(), q.r.PillPath)
	closeErr := source.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return finish(err)
	}
	return finish(e.checkWorkspace(ctx))
}
func (e *projectMemberLaunchEngine) Preflight(ctx context.Context, s projectnetwork.MemberLaunch) error {
	e.preflightAt = time.Time{}
	if s.Phase != "workspace-created" && s.Phase != "created" {
		return errors.New("member preflight requires acknowledged workspace or container")
	}
	if err := e.selected(ctx, s, s.Phase); err != nil {
		return err
	}
	if err := e.checkWorkspace(ctx); err != nil {
		return fmt.Errorf("member workspace preflight: %w", err)
	}
	if err := e.anchor.rejectConflictingProjectMounts(ctx, e.contract, s.Container.ID); err != nil {
		return fmt.Errorf("member mount preflight: %w", err)
	}
	e.preflightAt = time.Now()
	return nil
}
func (e *projectMemberLaunchEngine) Inspect(ctx context.Context, s projectnetwork.MemberLaunch) (projectnetwork.Observation, error) {
	e.inspectedAt = time.Time{}
	if s.Phase != "planned" && s.Phase != "workspace-created" && s.Phase != "created" && s.Phase != "started" {
		return projectnetwork.Observation{}, errors.New("unsupported member observation phase")
	}
	if err := e.selected(ctx, s, s.Phase); err != nil {
		return projectnetwork.Observation{}, err
	}
	known := slices.Clone(e.peers)
	if s.Container.ID != "" {
		known = append(known, projectObservedMember{contract: e.contract, id: s.Container.ID})
	}
	// Observe every recorded exclusive parent before and after Docker collection.
	checkParents := func() error {
		for _, member := range known {
			c := member.contract
			path, err := pierstore.CheckMount(c.home, c.selection.Storage)
			if err != nil || path != c.mountSource {
				return errors.New("member mount parent is missing, replaced or non-exclusive")
			}
		}
		return ctx.Err()
	}
	if err := checkParents(); err != nil {
		return projectnetwork.Observation{}, err
	}
	o, err := e.anchor.inspectProjectMembers(ctx, known)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	if err = checkParents(); err != nil {
		return projectnetwork.Observation{}, err
	}
	if o.Anchor.Namespace != *s.Anchor.Namespace {
		return projectnetwork.Observation{}, projectnetwork.ErrRuntimeObservation
	}
	for _, member := range o.Members {
		if member.Running && member.Namespace != *s.Anchor.Namespace {
			return projectnetwork.Observation{}, projectnetwork.ErrRuntimeObservation
		}
		if member.InstanceID == s.Storage.InstanceID && member.Running != (s.Phase == "started") {
			return projectnetwork.Observation{}, projectnetwork.ErrRuntimeObservation
		}
	}
	e.inspectedAt = o.ObservedAt
	e.inspectedPhase = s.Phase
	return o, nil
}
func (e *projectMemberLaunchEngine) fresh(phase string) error {
	if e.preflightAt.IsZero() || e.inspectedAt.IsZero() || time.Since(e.preflightAt) > 30*time.Second || time.Since(e.inspectedAt) > 30*time.Second || e.inspectedPhase != phase {
		age := func(t time.Time) string {
			if t.IsZero() {
				return "missing"
			}
			return time.Since(t).Round(time.Millisecond).String()
		}
		return fmt.Errorf("member dispatch requires fresh preflight and complete ownership observation (preflight=%s, inspection=%s, phase=%s, expected=%s)", age(e.preflightAt), age(e.inspectedAt), e.inspectedPhase, phase)
	}
	return nil
}
func (e *projectMemberLaunchEngine) Create(ctx context.Context, s projectnetwork.MemberLaunch) (string, error) {
	if err := e.selected(ctx, s, "create-dispatched"); err != nil {
		return "", err
	}
	if e.createDispatched {
		return "", errors.New("member Create cannot replay")
	}
	if err := e.fresh("workspace-created"); err != nil {
		return "", err
	}
	if err := e.VerifyEngine(ctx, s.Anchor.Prepared.EngineID); err != nil {
		return "", err
	}
	if err := e.contract.checkStorageBeforeCreate(ctx); err != nil {
		return "", err
	}
	if err := e.checkVerifiedWorkspace(ctx); err != nil {
		return "", err
	}
	if err := e.fresh("workspace-created"); err != nil {
		return "", err
	}
	e.createDispatched = true
	e.preflightAt = time.Time{}
	e.inspectedAt = time.Time{}
	q := e.anchor.base.q
	if err := q.admitStart(e.contract.mountSource, false, "new project ship create"); err != nil {
		return "", err
	}
	out, err := q.dockerCreateID(ctx, q.d.times.command, e.contract.createArgs()...)
	id := strings.TrimSpace(out)
	if !containerIDPattern.MatchString(id) {
		return "", errors.New("member Create returned no exact container identity")
	}
	e.createdID = id
	return id, err
}
func (e *projectMemberLaunchEngine) Start(ctx context.Context, s projectnetwork.MemberLaunch) error {
	if err := e.selected(ctx, s, "start-dispatched"); err != nil {
		return err
	}
	if e.startDispatched {
		return errors.New("member Start cannot replay")
	}
	if err := e.fresh("created"); err != nil {
		return err
	}
	if err := e.VerifyEngine(ctx, s.Anchor.Prepared.EngineID); err != nil {
		return err
	}
	if err := e.contract.checkStorageBeforeCreate(ctx); err != nil {
		return err
	}
	if err := e.checkVerifiedWorkspace(ctx); err != nil {
		return err
	}
	if err := e.fresh("created"); err != nil {
		return err
	}
	e.startDispatched = true
	e.preflightAt = time.Time{}
	e.inspectedAt = time.Time{}
	q := e.anchor.base.q
	if err := q.admitStart(e.contract.mountSource, false, "new project ship start"); err != nil {
		return err
	}
	out, err := q.docker(ctx, q.d.times.command, "container", "start", s.Container.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != s.Container.ID {
		return errors.New("member Start acknowledgement did not identify the selected container")
	}
	return nil
}
