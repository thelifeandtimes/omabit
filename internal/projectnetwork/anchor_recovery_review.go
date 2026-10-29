package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// AnchorRecoveryContext selects current ownership and supplied process evidence.
// It grants no mutation authority and cannot adopt a new anchor or namespace.
// Callers must supply validated receipts, including known process observations,
// and exclude pending/uncertain work first. A new healthy observation of every
// peer immediately before an anchor fault is not required.
type AnchorRecoveryContext struct {
	Binding RuntimeBinding         `json:"binding"`
	Members []AnchorRecoveryMember `json:"members"`
}
type AnchorRecoveryMember struct {
	Context   MemberContext `json:"context"`
	StartedAt time.Time     `json:"started_at"`
	NotBefore time.Time     `json:"not_before"`
}
type AnchorRecoveryProcess struct {
	InstanceID string             `json:"instance_id"`
	Storage    pierstore.Location `json:"storage"`
	StartedAt  time.Time          `json:"started_at"`
}

// This initial review requires every exact member still running in its old
// namespace. Missing/stopped members need a separate recovery contract. Original
// creators may lack a recorded start time; then the two fresh reads establish
// only their own process continuity, never uninterrupted prior uptime.
type AnchorRecoveryObservation struct {
	Runtime    Observation             `json:"runtime"`
	Processes  []AnchorRecoveryProcess `json:"processes"`
	CapturedAt time.Time               `json:"captured_at"`
}
type AnchorRecoveryReview struct {
	Context    AnchorRecoveryContext     `json:"context"`
	Before     AnchorRecoveryObservation `json:"before"`
	After      AnchorRecoveryObservation `json:"after"`
	Assessment Assessment                `json:"assessment"`
	CapturedAt time.Time                 `json:"captured_at"`
}

func SelectAnchorRecoveryContext(plan Plan, launches []MemberLaunch, resumes []MemberResume, originals []InitialMemberReadiness, ready []CurrentMemberReadiness) (AnchorRecoveryContext, error) {
	fail := func() (AnchorRecoveryContext, error) {
		return AnchorRecoveryContext{}, errors.New("anchor recovery review requires complete retained member ownership")
	}
	if len(originals) != len(plan.Members) || len(ready) > len(plan.Members) {
		return fail()
	}
	// Capture only the requested historical bindings during one complete causal
	// traversal. Snapshots are private to this call and are discarded on any error.
	type prefix struct {
		generations MemberGenerationSet
		resumes     []MemberResume
		found       bool
	}
	prefixes := make([]prefix, len(ready))
	current, err := resolveMemberGenerations(plan, launches, resumes, func(g MemberGenerationSet, through []MemberResume) {
		for i, r := range ready {
			if equalEvidence(g.Binding, r.Binding) {
				v := g
				v.Binding.Members = slices.Clone(g.Binding.Members)
				v.Members = slices.Clone(g.Members)
				prefixes[i] = prefix{generations: v, resumes: slices.Clone(through), found: true}
			}
		}
	})
	if err != nil || len(originals) != len(plan.Members) || len(ready) > len(plan.Members) {
		return fail()
	}
	result := AnchorRecoveryContext{Binding: current.Binding}
	result.Binding.Members = slices.Clone(current.Binding.Members)
	for _, member := range current.Members {
		var original *InitialMemberReadiness
		var fresh *CurrentMemberReadiness
		freshIndex := -1
		for i := range originals {
			if originals[i].InstanceID == member.InstanceID {
				if original != nil {
					return fail()
				}
				original = &originals[i]
			}
		}
		for i := range ready {
			if ready[i].InstanceID == member.InstanceID {
				if fresh != nil {
					return fail()
				}
				fresh = &ready[i]
				freshIndex = i
			}
		}
		if original == nil {
			return fail()
		}
		if fresh != nil {
			selected := prefixes[freshIndex]
			if !selected.found || fresh.CreatorOperationID != member.OperationID || fresh.ContainerGeneration != member.Generation {
				return fail()
			}
			historical, e := selectMemberContextFromValidatedGenerations(plan, launches, selected.resumes, *original, selected.generations)
			if e != nil || fresh.validateContext(plan, historical) != nil {
				return fail()
			}
		}
		c, err := selectMemberContextFromValidatedGenerations(plan, launches, resumes, *original, current)
		if err != nil {
			return fail()
		}
		started, notBefore := c.KnownStartedAt, c.NotBefore
		if fresh != nil {
			if !started.IsZero() && !started.Equal(fresh.After.StartedAt) {
				return fail()
			}
			started = fresh.After.StartedAt
			if fresh.CapturedAt.After(notBefore) {
				notBefore = fresh.CapturedAt
			}
		}
		result.Members = append(result.Members, AnchorRecoveryMember{Context: c, StartedAt: started, NotBefore: notBefore})
	}
	for _, proof := range ready {
		found := false
		for _, member := range current.Members {
			if proof.InstanceID == member.InstanceID {
				found = true
			}
		}
		if !found {
			return fail()
		}
	}
	return result, nil
}
func cloneAnchorRecoveryContext(c AnchorRecoveryContext) AnchorRecoveryContext {
	c.Binding.Members = slices.Clone(c.Binding.Members)
	c.Members = slices.Clone(c.Members)
	for i := range c.Members {
		c.Members[i].Context = cloneMemberContext(c.Members[i].Context)
	}
	return c
}
func cloneAnchorRecoveryObservation(o AnchorRecoveryObservation) AnchorRecoveryObservation {
	o.Runtime = cloneReadinessObservation(o.Runtime)
	o.Processes = slices.Clone(o.Processes)
	return o
}
func checkAnchorRecoveryObservation(plan Plan, c AnchorRecoveryContext, o AnchorRecoveryObservation, now time.Time) (Assessment, error) {
	fail := func() (Assessment, error) {
		return Assessment{}, errors.New("anchor recovery review cannot establish unchanged owned processes")
	}
	if now.IsZero() || o.CapturedAt.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second || o.CapturedAt.Before(o.Runtime.ObservedAt) || len(o.Processes) != len(c.Members) {
		return fail()
	}
	a, err := AssessRuntime(plan, c.Binding, o.Runtime, now)
	if err != nil || !a.RecoveryRequired || (a.State != "anchor-unavailable" && a.State != "namespace-changed") || !o.Runtime.Bridge.Present || !o.Runtime.Anchor.Present || len(a.Running) != len(c.Members) || len(a.Stopped) != 0 {
		return fail()
	}
	if o.Runtime.Anchor.Running && (!samePublications(plan, o.Runtime.Anchor.Active) || o.Runtime.Anchor.Namespace.BootID != c.Binding.Namespace.BootID) {
		return fail()
	}
	// No mixed or silently migrated member namespace is a recoverable old set.
	for _, m := range o.Runtime.Members {
		if !m.Present || !m.Running || m.Namespace != c.Binding.Namespace {
			return fail()
		}
	}
	for _, m := range c.Members {
		count := 0
		for _, v := range o.Processes {
			if v.InstanceID == m.Context.Member.InstanceID {
				count++
				if v.Storage != m.Context.Storage || v.StartedAt.IsZero() || (!m.StartedAt.IsZero() && !v.StartedAt.Equal(m.StartedAt)) || v.StartedAt.After(o.CapturedAt) || o.CapturedAt.Before(m.NotBefore) || o.Runtime.ObservedAt.Before(m.NotBefore) {
					return fail()
				}
			}
		}
		if count != 1 {
			return fail()
		}
	}
	return a, nil
}
func (r AnchorRecoveryReview) Validate(plan Plan, launches []MemberLaunch, resumes []MemberResume, originals []InitialMemberReadiness, ready []CurrentMemberReadiness) error {
	c, err := SelectAnchorRecoveryContext(plan, launches, resumes, originals, ready)
	if err != nil {
		return err
	}
	return r.validateContext(plan, c)
}
func (r AnchorRecoveryReview) validateContext(plan Plan, c AnchorRecoveryContext) error {
	fail := func() error { return errors.New("invalid anchor recovery review") }
	if !equalEvidence(c, r.Context) {
		return fail()
	}

	before, err := checkAnchorRecoveryObservation(plan, c, r.Before, r.CapturedAt)
	if err != nil {
		return fail()
	}
	after, err := checkAnchorRecoveryObservation(plan, c, r.After, r.CapturedAt)
	if err != nil || !equalEvidence(before, after) || !equalEvidence(after, r.Assessment) {
		return fail()
	}
	if r.After.CapturedAt.Before(r.Before.CapturedAt) || r.After.Runtime.ObservedAt.Before(r.Before.Runtime.ObservedAt) {
		return fail()
	}
	expected := cloneAnchorRecoveryObservation(r.Before)
	expected.CapturedAt = r.After.CapturedAt
	expected.Runtime.ObservedAt = r.After.Runtime.ObservedAt
	// Canonicalize complete inspections, preserving the source snapshots themselves.
	actual := cloneAnchorRecoveryObservation(r.After)
	sortObservation := func(o *AnchorRecoveryObservation) {
		slices.SortFunc(o.Processes, func(a, b AnchorRecoveryProcess) int {
			if a.InstanceID < b.InstanceID {
				return -1
			}
			if a.InstanceID > b.InstanceID {
				return 1
			}
			return 0
		})
		slices.SortFunc(o.Runtime.Members, func(a, b MemberObservation) int {
			if a.InstanceID < b.InstanceID {
				return -1
			}
			if a.InstanceID > b.InstanceID {
				return 1
			}
			return 0
		})
	}
	sortObservation(&expected)
	sortObservation(&actual)
	if !equalEvidence(expected, actual) {
		return fail()
	}
	return nil
}

// This interface intentionally contains no Start, Stop, remove, adopt or HTTP
// operation. A review is diagnostic evidence; coordinated recovery remains a
// separate write-ahead protocol with its own admission and dispatch.
type AnchorRecoveryObserver interface {
	VerifyEngine(context.Context, string) error
	InspectAnchorRecovery(context.Context, AnchorRecoveryContext) (AnchorRecoveryObservation, error)
}

func CollectAnchorRecoveryReview(ctx context.Context, plan Plan, launches []MemberLaunch, resumes []MemberResume, originals []InitialMemberReadiness, ready []CurrentMemberReadiness, d AnchorRecoveryObserver, now func() time.Time) (AnchorRecoveryReview, error) {
	c, err := SelectAnchorRecoveryContext(plan, launches, resumes, originals, ready)
	if err != nil {
		return AnchorRecoveryReview{}, errors.New("anchor recovery observation failed; no recovery action was performed")
	}
	return collectAnchorRecoveryReviewContext(ctx, plan, c, d, now, func(r AnchorRecoveryReview) error { return r.Validate(plan, launches, resumes, originals, ready) })
}
func collectAnchorRecoveryReviewContext(ctx context.Context, plan Plan, c AnchorRecoveryContext, d AnchorRecoveryObserver, now func() time.Time, validate func(AnchorRecoveryReview) error) (AnchorRecoveryReview, error) {
	fail := func() (AnchorRecoveryReview, error) {
		return AnchorRecoveryReview{}, errors.New("anchor recovery observation failed; no recovery action was performed")
	}
	if d == nil || now == nil {
		return fail()
	}

	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, c.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return fail()
	}
	before, err := d.InspectAnchorRecovery(ctx, cloneAnchorRecoveryContext(c))
	before = cloneAnchorRecoveryObservation(before)
	if err != nil || !verify() {
		return fail()
	}
	if _, err = checkAnchorRecoveryObservation(plan, c, before, now()); err != nil {
		return fail()
	}
	after, err := d.InspectAnchorRecovery(ctx, cloneAnchorRecoveryContext(c))
	after = cloneAnchorRecoveryObservation(after)
	if err != nil || !verify() {
		return fail()
	}
	captured := now()
	a, err := checkAnchorRecoveryObservation(plan, c, after, captured)
	if err != nil {
		return fail()
	}
	result := AnchorRecoveryReview{Context: c, Before: before, After: after, Assessment: a, CapturedAt: captured}
	if ctx.Err() != nil || validate(result) != nil {
		return fail()
	}
	return result, nil
}
