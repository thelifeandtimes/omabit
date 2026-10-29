//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ProjectExpansion is an append-only membership boundary around the unchanged
// original anchor. Original operations remain historical and are fenced from
// further mutation once this ledger exists. No existing pier is moved or copied.
type ProjectExpansion struct {
	Version int                         `json:"version"`
	Project ProjectPreparationSelection `json:"project"`
	Anchor  projectnetwork.AnchorStart  `json:"anchor"`
	Members []ProjectExpandedMember     `json:"members"`
	History []ProjectExpandedMember     `json:"history,omitempty"`
}
type ProjectExpandedMember struct {
	Member      projectnetwork.Member           `json:"member"`
	Creator     projectnetwork.MemberGeneration `json:"creator"`
	Initialized pierstore.Location              `json:"initialized"`
	Added       bool                            `json:"added"`
	Phase       string                          `json:"phase"`
	Publication *ProjectPort                    `json:"publication,omitempty"`
}

func SeedProjectExpansion(s ProjectCurrentReadinessSelection) (ProjectExpansion, error) {
	out := ProjectExpansion{Version: 1, Project: s.Project}
	if s.RecoveryHistory != nil {
		return out, errors.New("expansion of a recovered anchor requires its successor membership adapter")
	}
	if len(s.Launches) == 0 {
		return out, errors.New("initial member history is required")
	}
	out.Anchor = s.Launches[0].Anchor
	current, err := projectnetwork.ResolveMemberGenerations(s.Project.Plan, s.Launches, s.Resumes)
	if err != nil {
		return out, err
	}
	for _, m := range s.Project.Plan.Members {
		for _, g := range current.Members {
			if g.InstanceID == m.InstanceID {
				out.Members = append(out.Members, ProjectExpandedMember{Member: m, Creator: g, Phase: "ready"})
			}
		}
	}
	return out, nil
}

func expansionEngineWithDependencies(ctx context.Context, s ProjectExpansion, d dependencies) (*projectCurrentReadinessEngine, error) {
	if s.Version != 1 || s.Anchor.Validate(s.Project.Plan) != nil || s.Anchor.Phase != "observed" {
		return nil, errors.New("invalid project expansion anchor")
	}
	raw, err := openProjectAnchorStart(ctx, s.Project, s.Anchor.Prepared, d)
	if err != nil {
		return nil, err
	}
	anchor := raw.(*projectAnchorEngine)
	e := &projectCurrentReadinessEngine{source: &projectInitialReadinessEngine{anchor: anchor}}
	ids, ships, ports, internals := map[string]bool{}, map[string]bool{}, map[uint16]bool{}, map[uint16]bool{}
	for _, m := range s.Members {
		if ids[m.Member.InstanceID] || ships[m.Member.Ship] || ports[m.Member.HTTPPort] || internals[m.Member.InternalHTTPPort] || m.Creator.InstanceID != m.Member.InstanceID {
			return nil, errors.New("conflicting expansion member")
		}
		ids[m.Member.InstanceID] = true
		ships[m.Member.Ship] = true
		ports[m.Member.HTTPPort] = true
		internals[m.Member.InternalHTTPPort] = true
		if m.Creator.Container.ID == "" {
			continue
		}
		c, err := expansionContract(anchor, s, m)
		if err != nil {
			return nil, err
		}
		if c.digest != m.Creator.Container.ContractDigest {
			return nil, errors.New("expansion creation contract changed")
		}
		if _, err = pierstore.CheckMount(c.home, m.Creator.Storage); err != nil {
			return nil, err
		}
		if m.Initialized.Identity.Inode != 0 {
			if _, err = pierstore.Check(c.home, m.Initialized); err != nil {
				return nil, err
			}
		}
		e.source.members = append(e.source.members, projectObservedMember{contract: c, id: m.Creator.Container.ID})
	}
	return e, nil
}
func expansionContract(a *projectAnchorEngine, s ProjectExpansion, m ProjectExpandedMember) (projectMemberContract, error) {
	g := m.Creator
	path, err := pierstore.Absolute(a.base.q.d.home, g.Storage.Path)
	if err != nil {
		return projectMemberContract{}, err
	}
	selection := projectMemberSelection{Project: s.Project, Anchor: s.Anchor, InstanceID: g.InstanceID, RuntimeID: g.RuntimeID, OperationID: g.OperationID, ContainerGeneration: g.Generation, Mode: g.Mode, Storage: g.Storage}
	if m.Added {
		copy := m.Member
		selection.AddedMember = &copy
	}
	return newProjectMemberContract(selection, a.base.q.d.home, filepath.Dir(path), a.base.contract.ImageID)
}
func expansionSnapshot(ctx context.Context, s ProjectExpansion) (*projectCurrentReadinessEngine, projectMemberSnapshot, error) {
	return expansionSnapshotWithDependencies(ctx, s, defaults())
}

func expansionSnapshotWithDependencies(ctx context.Context, s ProjectExpansion, d dependencies) (*projectCurrentReadinessEngine, projectMemberSnapshot, error) {
	e, err := expansionEngineWithDependencies(ctx, s, d)
	if err != nil {
		return nil, projectMemberSnapshot{}, err
	}
	snap, err := projectFleetInspect(ctx, e)
	if err == nil && snap.Observation.Anchor.Namespace != *s.Anchor.Namespace {
		err = errors.New("project namespace changed; explicit coordinated recovery is required")
	}
	return e, snap, err
}
func ReadExpandedProjectFleet(ctx context.Context, s ProjectExpansion) ([]protocol.Instance, error) {
	e, before, err := expansionSnapshotWithDependencies(ctx, s, fleetObservationDependencies(defaults()))
	if err != nil {
		return nil, err
	}
	selection := ProjectCurrentReadinessSelection{Project: s.Project}
	selection.Project.Plan.Members = nil
	for _, m := range s.Members {
		selection.Project.Plan.Members = append(selection.Project.Plan.Members, m.Member)
	}
	rows, err := projectFleetRows(ctx, selection, e, before)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		for _, m := range s.Members {
			if m.Member.InstanceID == row.ID && m.Added && (m.Publication == nil || e.source.anchor.base.q.verifyProjectPort(ctx, *m.Publication) != nil) {
				rows[i].Health = "unknown"
				rows[i].Endpoint = ""
			}
		}
	}
	return rows, nil
}
func ReadExpandedProjectCode(ctx context.Context, s ProjectExpansion, instance, containerID string) (string, error) {
	e, before, err := expansionSnapshot(ctx, s)
	if err != nil {
		return "", err
	}
	return projectCodeFromEngine(ctx, e, before, instance, containerID)
}

// RunProjectAddition is single dispatch. Each filesystem/engine effect is
// preceded by a durable phase. A lost outcome is retained, never blindly retried.
func RunProjectAddition(ctx context.Context, s *ProjectExpansion, id string, save func(ProjectExpansion) error) error {
	index := -1
	for i, m := range s.Members {
		if m.Member.InstanceID == id {
			index = i
		}
	}
	if index < 0 || s.Members[index].Phase != "reserved" {
		return errors.New("addition is not a fresh reserved intent")
	}
	phase := func(value string) error { s.Members[index].Phase = value; return save(*s) }
	e, before, err := expansionSnapshot(ctx, *s)
	if err != nil {
		return err
	}
	q := e.source.anchor.base.q
	if err = q.checkProjectPorts(ctx); err != nil {
		return err
	}
	m := &s.Members[index]
	// Preserve initialized identities for the existing ships at the transition.
	for i, old := range s.Members {
		if old.Creator.Container.ID == "" {
			continue
		}
		path, err := pierstore.Absolute(q.d.home, old.Creator.Storage.Path)
		if err != nil {
			return err
		}
		identity, err := pierstore.Identify(path)
		if err != nil {
			return err
		}
		if old.Initialized.Identity.Inode != 0 && old.Initialized.Identity != identity {
			return errors.New("existing project pier changed")
		}
		s.Members[i].Initialized = old.Creator.Storage
		s.Members[i].Initialized.Identity = identity
	}
	if err = phase("mount-dispatched"); err != nil {
		return err
	}
	loc := m.Creator.Storage
	mount, err := pierstore.PrepareMount(q.d.home, &loc)
	m.Creator.Storage = loc
	if err != nil {
		_ = save(*s)
		return err
	}
	if err = phase("mounted"); err != nil {
		return err
	}
	c, err := expansionContract(e.source.anchor, *s, *m)
	if err != nil {
		return err
	}
	if err = q.admitStart(mount, false, "new project ship"); err != nil {
		return err
	}
	if err = c.checkStorageBeforeCreate(ctx); err != nil {
		return err
	}
	if err = e.source.anchor.rejectConflictingProjectMounts(ctx, c, ""); err != nil {
		return err
	}
	meta := protocol.BootPill{SHA256: q.d.pillSHA, Bytes: q.d.pillBytes}
	if err = bootpill.Verify(ctx, bootpill.CachePath(q.base, meta), meta); err != nil {
		return err
	}
	if err = phase("workspace-dispatched"); err != nil {
		return err
	}
	if err = os.Mkdir(filepath.Dir(c.pillPath), 0700); err != nil {
		return err
	}
	source, err := bootpill.Open(bootpill.CachePath(q.base, meta))
	if err != nil {
		return err
	}
	err = bootpill.Publish(ctx, source, meta, c.pillPath)
	err = errors.Join(err, source.Close())
	if err != nil {
		return err
	}
	if err = phase("workspace-ready"); err != nil {
		return err
	}
	if _, _, err = expansionSnapshot(ctx, *s); err != nil {
		return err
	}
	m.Creator.Container.ContractDigest = c.digest
	if err = phase("create-dispatched"); err != nil {
		return err
	}
	containerID, err := q.dockerCreateID(ctx, 30*time.Second, c.createArgs()...)
	containerID = strings.TrimSpace(containerID)
	if containerIDPattern.MatchString(containerID) {
		m.Creator.Container.ID = containerID
		if saved := save(*s); saved != nil {
			return saved
		}
	}
	if err != nil {
		return err
	}
	if !containerIDPattern.MatchString(containerID) {
		return errors.New("creation did not return a full container ID")
	}
	if err = phase("created"); err != nil {
		return err
	}
	if _, _, err = expansionSnapshot(ctx, *s); err != nil {
		return err
	}
	if err = phase("start-dispatched"); err != nil {
		return err
	}
	if _, err = q.docker(ctx, 30*time.Second, "container", "start", containerID); err != nil {
		return err
	}
	if err = phase("started"); err != nil {
		return err
	}
	_, snap, err := expansionSnapshot(ctx, *s)
	if err != nil {
		return err
	}
	// Only the inspected anchor's bridge address can be a forward target.
	anchor := snap.Containers[s.Anchor.Prepared.Anchor.ID]
	var address string
	for _, raw := range anchor.NetworkSettings.Networks {
		var n struct{ IPAddress string }
		if json.Unmarshal(raw, &n) != nil {
			return errors.New("invalid anchor network address")
		}
		address = n.IPAddress
	}
	spec := ProjectPortSpec{Proto: "tcp4", ParentIP: "127.0.0.1", ParentPort: int(m.Member.HTTPPort), ChildIP: address, ChildPort: int(m.Member.InternalHTTPPort)}
	if err = phase("publication-dispatched"); err != nil {
		return err
	}
	publication, err := q.addProjectPort(ctx, spec)
	if err != nil {
		return err
	}
	m.Publication = &publication
	if err = phase("published"); err != nil {
		return err
	}
	deadline, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	for {
		path, err := pierstore.Absolute(q.d.home, m.Creator.Storage.Path)
		if err != nil {
			return err
		}
		identity, err := pierstore.Identify(path)
		if err == nil {
			m.Initialized = m.Creator.Storage
			m.Initialized.Identity = identity
			rows, observeErr := ReadExpandedProjectFleet(deadline, *s)
			if observeErr == nil {
				for _, row := range rows {
					if row.ID == id && row.Health == "healthy" {
						// Existing member process identities must survive the addition unchanged.
						_, after, err := expansionSnapshot(deadline, *s)
						if err != nil {
							return err
						}
						for cid, old := range before.Containers {
							if !reflect.DeepEqual(old, after.Containers[cid]) {
								return errors.New("existing project process changed during member addition")
							}
						}
						return phase("ready")
					}
				}
			}
		}
		select {
		case <-deadline.Done():
			return errors.New("member started but readiness is unresolved; retained for review")
		case <-time.After(time.Second):
		}
	}
}

func (s ProjectExpansion) NewMember(member projectnetwork.MemberRequest, runtimeID, operationID string) (ProjectExpansion, error) {
	if !protocol.ValidPierName(member.InstanceID) || len(member.Ship) != 4 || !protocol.ValidFakeShip(member.Ship) || member.HTTPPort < 1024 || !projectRuntimeID.MatchString(runtimeID) || !protocol.ValidID(operationID) {
		return s, errors.New("invalid fresh project member")
	}
	port := uint16(8080)
	for _, m := range s.Members {
		if m.Member.InstanceID == member.InstanceID || m.Member.Ship == member.Ship || m.Member.HTTPPort == member.HTTPPort {
			return s, errors.New("instance, ship identity or port already belongs to this project")
		}
		if m.Member.InternalHTTPPort >= port {
			port = m.Member.InternalHTTPPort + 1
		}
	}
	if port > 8335 {
		return s, errors.New("project already contains 256 fake galaxies")
	}
	s.Members = append(s.Members, ProjectExpandedMember{Member: projectnetwork.Member{MemberRequest: member, InternalHTTPPort: port}, Creator: projectnetwork.MemberGeneration{InstanceID: member.InstanceID, RuntimeID: runtimeID, OperationID: operationID, Generation: 1, Mode: "boot", Storage: pierstore.Location{Path: "fakeships/" + member.InstanceID + "/" + strings.TrimPrefix(member.Ship, "~")}}, Added: true, Phase: "reserved"})
	return s, nil
}

// Lifecycle uses the selected full container ID and an explicit request. A
// successor is created only after a confirmed clean exit and exact pier check.
func RunProjectShipAction(ctx context.Context, s *ProjectExpansion, id, action, operation string, save func(ProjectExpansion) error) error {
	index := -1
	for i, m := range s.Members {
		if m.Member.InstanceID == id {
			index = i
		}
	}
	if index < 0 {
		return errors.New("project member is missing")
	}
	e, snap, err := expansionSnapshot(ctx, *s)
	if err != nil {
		return err
	}
	q := e.source.anchor.base.q
	m := &s.Members[index]
	if action == "continue-start" {
		if m.Phase != "retired" || m.Creator.Mode != "resume" || m.Creator.Container.ID != "" || m.Initialized.Identity.Inode == 0 {
			return errors.New("start continuation requires retired-before-create evidence")
		}
		return createExpandedSuccessor(ctx, s, index, e, save)
	}
	old := snap.Containers[m.Creator.Container.ID]
	phase := func(value string) error { m.Phase = value; return save(*s) }
	if m.Initialized.Identity.Inode == 0 {
		path, err := pierstore.Absolute(q.d.home, m.Creator.Storage.Path)
		if err != nil {
			return err
		}
		identity, err := pierstore.Identify(path)
		if err != nil {
			return err
		}
		m.Initialized = m.Creator.Storage
		m.Initialized.Identity = identity
	}
	if action == "stop" {
		if !*old.State.Running {
			return errors.New("selected project member is already stopped")
		}
		if err = phase("stop-dispatched"); err != nil {
			return err
		}
		if _, err = q.docker(ctx, 30*time.Second, "container", "kill", "--signal", "SIGTERM", m.Creator.Container.ID); err != nil {
			return err
		}
		deadline, cancel := context.WithTimeout(ctx, 300*time.Second)
		defer cancel()
		var after projectMemberSnapshot
		for {
			_, after, err = expansionSnapshot(deadline, *s)
			if err != nil {
				return err
			}
			if !*after.Containers[m.Creator.Container.ID].State.Running {
				break
			}
			select {
			case <-deadline.Done():
				return errors.New("graceful stop timed out; project runtime retained without force-kill")
			case <-time.After(time.Second):
			}
		}
		if err != nil {
			return err
		}
		v := after.Containers[m.Creator.Container.ID]
		if *v.State.Running || *v.State.ExitCode != 0 || *v.State.OOMKilled {
			return errors.New("project member did not exit cleanly")
		}
		return phase("stopped")
	}
	if action != "start" || *old.State.Running || *old.State.ExitCode != 0 || *old.State.OOMKilled {
		return errors.New("start requires the selected cleanly stopped project member")
	}
	if _, err = pierstore.Check(q.d.home, m.Initialized); err != nil {
		return err
	}
	absolute, err := pierstore.Absolute(q.d.home, m.Initialized.Path)
	if err != nil {
		return err
	}
	if err = q.admitStart(filepath.Dir(absolute), true, "existing project ship"); err != nil {
		return err
	}
	if err = phase("retire-dispatched"); err != nil {
		return err
	}
	if _, err = q.docker(ctx, 30*time.Second, "container", "rm", m.Creator.Container.ID); err != nil {
		return err
	}
	s.History = append(s.History, *m)
	m.Creator.Generation++
	m.Creator.OperationID = operation
	m.Creator.Mode = "resume"
	m.Creator.Storage = m.Initialized
	m.Creator.Container = projectnetwork.ResourceIdentity{}
	if err = phase("retired"); err != nil {
		return err
	}
	return createExpandedSuccessor(ctx, s, index, e, save)
}

func createExpandedSuccessor(ctx context.Context, s *ProjectExpansion, index int, e *projectCurrentReadinessEngine, save func(ProjectExpansion) error) error {
	q := e.source.anchor.base.q
	m := &s.Members[index]
	id := m.Member.InstanceID
	phase := func(value string) error { m.Phase = value; return save(*s) }
	if _, err := pierstore.Check(q.d.home, m.Initialized); err != nil {
		return err
	}
	absolute, err := pierstore.Absolute(q.d.home, m.Initialized.Path)
	if err != nil {
		return err
	}
	if err = q.admitStart(filepath.Dir(absolute), true, "existing project ship"); err != nil {
		return err
	}
	c, err := expansionContract(e.source.anchor, *s, *m)
	if err != nil {
		return err
	}
	if err = c.checkStorageBeforeCreate(ctx); err != nil {
		return err
	}
	if err = e.source.anchor.rejectConflictingProjectMounts(ctx, c, ""); err != nil {
		return err
	}
	m.Creator.Container.ContractDigest = c.digest
	if err = phase("create-dispatched"); err != nil {
		return err
	}
	cid, err := q.dockerCreateID(ctx, 30*time.Second, c.createArgs()...)
	cid = strings.TrimSpace(cid)
	if containerIDPattern.MatchString(cid) {
		m.Creator.Container.ID = cid
		if saved := save(*s); saved != nil {
			return saved
		}
	}
	if err != nil {
		return err
	}
	if !containerIDPattern.MatchString(cid) {
		return errors.New("successor creation outcome is unknown")
	}
	if err = phase("created"); err != nil {
		return err
	}
	if _, _, err = expansionSnapshot(ctx, *s); err != nil {
		return err
	}
	if err = phase("start-dispatched"); err != nil {
		return err
	}
	if _, err = q.docker(ctx, 30*time.Second, "container", "start", cid); err != nil {
		return err
	}
	if err = phase("started"); err != nil {
		return err
	}
	deadline, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	for {
		rows, err := ReadExpandedProjectFleet(deadline, *s)
		if err == nil {
			for _, row := range rows {
				if row.ID == id && row.Health == "healthy" {
					return phase("ready")
				}
			}
		}
		select {
		case <-deadline.Done():
			return errors.New("resumed project member readiness is unresolved")
		case <-time.After(time.Second):
		}
	}
}
