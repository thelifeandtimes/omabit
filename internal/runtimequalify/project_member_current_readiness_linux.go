//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectCurrentReadinessSelection struct {
	RecoveryHistory *projectnetwork.AnchorRecoveryHistory
	Project         ProjectPreparationSelection
	Launches        []projectnetwork.MemberLaunch
	Resumes         []projectnetwork.MemberResume
	Original        projectnetwork.InitialMemberReadiness
}
type projectCurrentReadinessEngine struct {
	recovery       *projectAnchorRecoveryReviewEngine
	source         *projectInitialReadinessEngine
	expected       projectnetwork.MemberContext
	ship, endpoint string
	inspectedAt    time.Time
	first          map[string]container
}

// OpenCurrentProjectReadiness observes current contracts after peer replacement.
// This first adapter requires every peer running. It has no mutation methods and
// cannot reconcile a prior Stop. Callers must supply validated journals under the
// project fence; persistence and public scheduling are separate work.
func OpenCurrentProjectReadiness(ctx context.Context, s ProjectCurrentReadinessSelection) (projectnetwork.CurrentReadinessDriver, error) {
	return openCurrentProjectReadiness(ctx, s, defaults())
}
func openCurrentProjectReadiness(ctx context.Context, s ProjectCurrentReadinessSelection, d dependencies) (projectnetwork.CurrentReadinessDriver, error) {
	if s.RecoveryHistory != nil {
		return openRecoveredCurrentProjectReadiness(ctx, s, d)
	}
	c, err := projectnetwork.SelectMemberContext(s.Project.Plan, s.Launches, s.Resumes, s.Original)
	if err != nil {
		return nil, err
	}
	current, err := projectnetwork.ResolveMemberGenerations(s.Project.Plan, s.Launches, s.Resumes)
	if err != nil {
		return nil, err
	}
	source, err := openProjectMemberGenerations(ctx, s.Project, s.Launches, current, d)
	if err != nil {
		return nil, err
	}
	for i, m := range source.members {
		for _, resume := range s.Resumes {
			if resume.OperationID == m.contract.selection.OperationID {
				source.members[i].processStartedAt = resume.After.StartedAt
			}
		}
	}
	c.Binding.Members = slices.Clone(c.Binding.Members)
	e := &projectCurrentReadinessEngine{source: source, expected: c}
	for _, m := range source.anchor.base.contract.Selection.Plan.Members {
		if m.InstanceID == c.Member.InstanceID {
			e.ship = m.Ship
			e.endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func (e *projectCurrentReadinessEngine) storage(ctx context.Context) error {
	if err := e.source.parents(ctx); err != nil {
		return err
	}
	if _, err := pierstore.Check(e.source.anchor.base.q.d.home, e.expected.Storage); err != nil {
		return errors.New("current readiness initialized pier changed")
	}
	return ctx.Err()
}
func (e *projectCurrentReadinessEngine) VerifyEngine(ctx context.Context, id string) error {
	err := e.source.VerifyEngine(ctx, id)
	if err != nil {
		e.inspectedAt = time.Time{}
	}
	return err
}
func (e *projectCurrentReadinessEngine) Inspect(ctx context.Context, c projectnetwork.MemberContext) (projectnetwork.MemberStopObservation, error) {
	e.inspectedAt = time.Time{}
	refuse := func() (projectnetwork.MemberStopObservation, error) {
		return projectnetwork.MemberStopObservation{}, errors.New("current readiness ownership or process changed")
	}
	if ctx.Err() != nil || !reflect.DeepEqual(c, e.expected) || e.storage(ctx) != nil {
		return refuse()
	}
	var snapshot projectMemberSnapshot
	var err error
	if e.recovery != nil {
		snapshot, err = e.recovery.inspectMembers(ctx, e.source.members, nil)
	} else {
		snapshot, err = e.source.anchor.inspectProjectMemberSnapshot(ctx, e.source.members)
	}
	if err != nil || e.storage(ctx) != nil {
		return refuse()
	}
	now := time.Now().UTC()
	a, err := projectnetwork.AssessRuntime(e.source.anchor.base.contract.Selection.Plan, e.expected.Binding, snapshot.Observation, now)
	if err != nil || a.RecoveryRequired || a.State != "ready" || len(a.Running) != len(e.source.members) || snapshot.Observation.ObservedAt.Before(c.NotBefore) {
		return refuse()
	}
	if e.first != nil && !reflect.DeepEqual(e.first, snapshot.Containers) {
		return refuse()
	}
	v := snapshot.Containers[c.Member.Container.ID]
	started, se := time.Parse(time.RFC3339Nano, v.State.StartedAt)
	finished, fe := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
	if se != nil || fe != nil || started.IsZero() || !finished.IsZero() || *v.State.OOMKilled || *v.State.ExitCode != 0 || !c.KnownStartedAt.IsZero() && !started.Equal(c.KnownStartedAt) {
		return refuse()
	}
	if ctx.Err() != nil {
		return refuse()
	}
	e.first = snapshot.Containers
	e.inspectedAt = snapshot.Observation.ObservedAt
	return projectnetwork.MemberStopObservation{Runtime: snapshot.Observation, Storage: c.Storage, StartedAt: started, CapturedAt: now}, nil
}
func (e *projectCurrentReadinessEngine) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if ctx.Err() != nil || endpoint != e.endpoint || e.inspectedAt.IsZero() || time.Since(e.inspectedAt) > 30*time.Second || time.Until(e.inspectedAt) > 5*time.Second {
		return "", errors.New("current readiness HTTP requires fresh exact publication evidence")
	}
	return e.source.anchor.base.q.d.hostIdentity(ctx, endpoint)
}
