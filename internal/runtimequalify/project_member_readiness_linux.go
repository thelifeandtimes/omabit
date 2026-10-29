//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectInitialReadinessSelection struct {
	Project  ProjectPreparationSelection
	Launches []projectnetwork.MemberLaunch
}
type projectInitialReadinessEngine struct {
	anchor      *projectAnchorEngine
	binding     projectnetwork.RuntimeBinding
	launches    map[string]projectnetwork.MemberLaunch
	members     []projectObservedMember
	inspectedAt time.Time
}

// OpenInitialProjectReadiness is read-only. It requires complete immutable launch
// receipts, reconstructs exact contracts, and never starts, adopts or repairs a
// container or pier. The caller holds the common project resource/update fence.
func OpenInitialProjectReadiness(ctx context.Context, s ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error) {
	return openInitialProjectReadiness(ctx, s, defaults())
}
func openInitialProjectReadiness(ctx context.Context, s ProjectInitialReadinessSelection, d dependencies) (projectnetwork.InitialReadinessDriver, error) {
	b, err := projectnetwork.InitialRuntimeBinding(s.Project.Plan, s.Launches)
	if err != nil {
		return nil, err
	}
	started := s.Launches[0].Anchor
	raw, err := openProjectAnchorStart(ctx, s.Project, started.Prepared, d)
	if err != nil {
		return nil, err
	}
	anchor := raw.(*projectAnchorEngine)
	e := &projectInitialReadinessEngine{anchor: anchor, binding: b, launches: map[string]projectnetwork.MemberLaunch{}}
	for _, original := range s.Launches {
		launch := original
		launch.Peers = slices.Clone(original.Peers)
		ns := *original.Namespace
		launch.Namespace = &ns
		an := *original.Anchor.Namespace
		launch.Anchor.Namespace = &an
		path, err := pierstore.Absolute(d.home, launch.Storage.Location.Path)
		if err != nil {
			return nil, err
		}
		c, err := newProjectMemberContract(projectMemberSelection{Project: anchor.base.contract.Selection, Anchor: launch.Anchor, InstanceID: launch.Storage.InstanceID, RuntimeID: launch.Storage.RuntimeID, OperationID: launch.OperationID, ContainerGeneration: 1, Mode: "boot", Storage: launch.Storage.Location}, d.home, filepath.Dir(path), anchor.base.contract.ImageID)
		if err != nil || c.digest != launch.Container.ContractDigest {
			return nil, errors.New("readiness member contract differs from its original launch")
		}
		e.members = append(e.members, projectObservedMember{contract: c, id: launch.Container.ID})
		e.launches[launch.Storage.InstanceID] = launch
	}
	if err = e.parents(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func (e *projectInitialReadinessEngine) VerifyEngine(ctx context.Context, id string) error {
	if id != e.binding.EngineID {
		e.inspectedAt = time.Time{}
		return errors.New("readiness engine binding changed")
	}
	err := e.anchor.VerifyEngine(ctx, id)
	if err != nil {
		e.inspectedAt = time.Time{}
	}
	return err
}
func (e *projectInitialReadinessEngine) parents(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, m := range e.members {
		p, err := pierstore.CheckMount(m.contract.home, m.contract.selection.Storage)
		if err != nil || p != m.contract.mountSource {
			return errors.New("readiness member parent is missing, replaced or non-exclusive")
		}
		if m.contract.selection.Mode == "resume" {
			if _, err := pierstore.Check(m.contract.home, m.contract.selection.Storage); err != nil {
				return errors.New("resumed member initialized pier identity changed")
			}
		}
	}
	return ctx.Err()
}
func (e *projectInitialReadinessEngine) InspectProject(ctx context.Context) (projectnetwork.Observation, error) {
	e.inspectedAt = time.Time{}
	if err := e.parents(ctx); err != nil {
		return projectnetwork.Observation{}, err
	}
	o, err := e.anchor.inspectProjectMembers(ctx, e.members)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	if err = e.parents(ctx); err != nil {
		return projectnetwork.Observation{}, err
	}
	a, err := projectnetwork.AssessRuntime(e.anchor.base.contract.Selection.Plan, e.binding, o, time.Now())
	if err != nil || a.State != "ready" || a.RecoveryRequired || len(a.Running) != len(e.members) {
		return projectnetwork.Observation{}, projectnetwork.ErrRuntimeObservation
	}
	e.inspectedAt = o.ObservedAt
	return o, nil
}
func (e *projectInitialReadinessEngine) fresh(ctx context.Context) error {
	if ctx.Err() != nil || e.inspectedAt.IsZero() || time.Since(e.inspectedAt) > 30*time.Second {
		return errors.New("readiness requires fresh complete running-project evidence")
	}
	return nil
}
func (e *projectInitialReadinessEngine) InspectPier(ctx context.Context, s projectnetwork.MemberLaunch) (pierstore.Location, error) {
	refuse := func() (pierstore.Location, error) {
		return pierstore.Location{}, errors.New("readiness pier identity is missing or changed")
	}
	selected, ok := e.launches[s.Storage.InstanceID]
	if !ok || !reflect.DeepEqual(selected, s) || e.fresh(ctx) != nil || e.parents(ctx) != nil {
		return refuse()
	}
	location := s.Storage.Location
	path, err := pierstore.Absolute(e.anchor.base.q.d.home, location.Path)
	if err != nil {
		return refuse()
	}
	identity, err := pierstore.Identify(path)
	if err != nil {
		return refuse()
	}
	location.Identity = identity
	actual, err := pierstore.Check(e.anchor.base.q.d.home, location)
	if err != nil || actual != path {
		return refuse()
	}
	after, err := pierstore.Identify(path)
	if err != nil || after != identity || e.parents(ctx) != nil || e.fresh(ctx) != nil {
		return refuse()
	}
	return location, nil
}
func (e *projectInitialReadinessEngine) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if err := e.fresh(ctx); err != nil {
		return "", err
	}
	allowed := false
	for _, m := range e.anchor.base.contract.Selection.Plan.Members {
		if endpoint == fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", errors.New("readiness HTTP endpoint is not an inspected project publication")
	}
	// The default transport is credential-free and refuses proxies, redirects,
	// arbitrary URLs, oversized responses and compressed identity responses.
	return e.anchor.base.q.d.hostIdentity(ctx, endpoint)
}
