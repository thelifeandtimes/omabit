//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// ProjectResumedMemberStopSelection contains validated retained history, supplied
// under the common resource/update fence. Only completed replacements belong in
// Resumes; an unresolved dispatch must be fenced before opening this driver.
// StoppedPeers must refer to each peer's current container, although their Stop
// bindings may precede another member's replacement.
type ProjectResumedMemberStopSelection struct {
	Project      ProjectPreparationSelection
	Launches     []projectnetwork.MemberLaunch
	Resumes      []projectnetwork.MemberResume
	Readiness    projectnetwork.ResumedMemberReadiness
	Stop         projectnetwork.MemberStop
	StoppedPeers []projectnetwork.MemberStop
}

// OpenResumedProjectMemberStop reconstructs current contracts without mutation.
// The returned driver uses the same fresh-inspection/committed-dispatch/single-TERM
// protocol as initial Stop; it does not replay, adopt or remove any resource.
func OpenResumedProjectMemberStop(ctx context.Context, s ProjectResumedMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return openResumedProjectMemberStop(ctx, s, defaults())
}
func openResumedProjectMemberStop(ctx context.Context, s ProjectResumedMemberStopSelection, d dependencies) (projectnetwork.MemberStopDriver, error) {
	expected, err := projectnetwork.NewResumedMemberStop(s.Project.Plan, s.Launches, s.Resumes, s.Readiness, s.Stop.ReadinessOperationID, s.Stop.OperationID)
	if err != nil || !reflect.DeepEqual(expected, s.Stop) {
		return nil, errors.New("replacement Stop requires fresh current readiness-bound intent")
	}
	current, err := projectnetwork.ResolveMemberGenerations(s.Project.Plan, s.Launches, s.Resumes)
	if err != nil {
		return nil, err
	}
	source, err := openProjectMemberGenerations(ctx, s.Project, s.Launches, current, d)
	if err != nil {
		return nil, err
	}
	e := &projectMemberStopEngine{source: source, initial: expected, exits: map[string]projectMemberExit{}}
	e.exits, err = currentMemberPeerExits(s.Project.Plan, s.Launches, s.Resumes, current, expected, s.StoppedPeers)
	if err != nil {
		return nil, err
	}
	for _, m := range source.members {
		if m.contract.selection.InstanceID == expected.InstanceID {
			e.target = m
		}
	}
	if e.target.id == "" {
		return nil, errors.New("replacement Stop target is missing")
	}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

// Rebuild current contracts using original launch/anchor receipts and causally
// selected member creations. This is not an adoption path: every digest must
// reproduce the retained contract, and every original parent must still match.
func openProjectMemberGenerations(ctx context.Context, project ProjectPreparationSelection, launches []projectnetwork.MemberLaunch, current projectnetwork.MemberGenerationSet, d dependencies) (*projectInitialReadinessEngine, error) {
	raw, err := openInitialProjectReadiness(ctx, ProjectInitialReadinessSelection{Project: project, Launches: launches}, d)
	if err != nil {
		return nil, err
	}
	source := raw.(*projectInitialReadinessEngine)
	source.members = nil
	source.binding = current.Binding
	for _, m := range current.Members {
		launch, ok := source.launches[m.InstanceID]
		if !ok {
			return nil, errors.New("current member lost original launch")
		}
		path, err := pierstore.Absolute(d.home, m.Storage.Path)
		if err != nil {
			return nil, err
		}
		c, err := newProjectMemberContract(projectMemberSelection{Project: source.anchor.base.contract.Selection, Anchor: launch.Anchor, InstanceID: m.InstanceID, RuntimeID: m.RuntimeID, OperationID: m.OperationID, ContainerGeneration: m.Generation, Mode: m.Mode, Storage: m.Storage}, d.home, filepath.Dir(path), source.anchor.base.contract.ImageID)
		if err != nil || c.digest != m.Container.ContractDigest {
			return nil, errors.New("current member contract differs from retained creation")
		}
		source.members = append(source.members, projectObservedMember{contract: c, id: m.Container.ID})
	}
	if err = source.parents(ctx); err != nil {
		return nil, err
	}
	return source, nil
}

// Peer exits are admitted only for their exact current creation contract.
func currentMemberPeerExits(plan projectnetwork.Plan, launches []projectnetwork.MemberLaunch, resumes []projectnetwork.MemberResume, current projectnetwork.MemberGenerationSet, expected projectnetwork.MemberStop, peers []projectnetwork.MemberStop) (map[string]projectMemberExit, error) {
	exits := map[string]projectMemberExit{}
	for _, peer := range peers {
		if peer.Validate(plan) != nil || peer.Phase != "stopped" || peer.InstanceID == expected.InstanceID || peer.OperationID == expected.OperationID {
			return nil, errors.New("member Stop peer exit is invalid")
		}
		if _, exists := exits[peer.InstanceID]; exists {
			return nil, errors.New("duplicate member Stop peer exit")
		}
		prefix, err := projectnetwork.MemberGenerationsAtBinding(plan, launches, resumes, peer.Binding)
		if err != nil {
			return nil, err
		}
		var historical, latest projectnetwork.MemberGeneration
		for _, m := range prefix.Members {
			if m.InstanceID == peer.InstanceID {
				historical = m
			}
		}
		for _, m := range current.Members {
			if m.InstanceID == peer.InstanceID {
				latest = m
			}
		}
		if historical.InstanceID == "" || historical != latest || latest.OperationID != peer.LaunchOperationID || latest.Storage.Path != peer.Storage.Path || latest.Storage.MountIdentity != peer.Storage.MountIdentity || latest.Mode == "resume" && latest.Storage != peer.Storage {
			return nil, errors.New("member Stop peer exit does not identify its current generation")
		}
		o := peer.After
		exits[peer.InstanceID] = projectMemberExit{o.StartedAt, o.FinishedAt, o.ExitCode, o.OOMKilled, peer.Storage}
	}
	return exits, nil
}
