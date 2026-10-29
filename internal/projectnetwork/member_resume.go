package projectnetwork

import (
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberResume creates a new container over the exact initialized, retired pier.
// The caller must bind PreviousGeneration and the new contract to accepted store
// history. Neither an absence observation nor a clean exit grants this intent.
// This model is not yet persisted or exposed through runner capabilities.
type MemberResume struct {
	Retirement         MemberRetirement          `json:"retirement"`
	PreviousGeneration uint64                    `json:"previous_generation"`
	OperationID        string                    `json:"operation_id"`
	Phase              string                    `json:"phase"`
	Container          ResourceIdentity          `json:"container"`
	Before             *MemberAbsenceObservation `json:"before,omitempty"`
	Created            *MemberResumeObservation  `json:"created,omitempty"`
	After              *MemberResumeObservation  `json:"after,omitempty"`
}
type MemberResumeObservation struct {
	Runtime    Observation        `json:"runtime"`
	Storage    pierstore.Location `json:"storage"`
	StartedAt  time.Time          `json:"started_at"`
	CapturedAt time.Time          `json:"captured_at"`
}

func cloneResumeObservation(o MemberResumeObservation) MemberResumeObservation {
	o.Runtime = cloneReadinessObservation(o.Runtime)
	return o
}
func cloneMemberResume(s MemberResume) MemberResume {
	s.Retirement = cloneMemberRetirement(s.Retirement)
	if s.Before != nil {
		v := *s.Before
		v.Runtime = cloneReadinessObservation(v.Runtime)
		s.Before = &v
	}
	if s.Created != nil {
		v := cloneResumeObservation(*s.Created)
		s.Created = &v
	}
	if s.After != nil {
		v := cloneResumeObservation(*s.After)
		s.After = &v
	}
	return s
}
func NewMemberResume(plan Plan, retired MemberRetirement, previous uint64, operation, contract string) (MemberResume, error) {
	s := cloneMemberResume(MemberResume{Retirement: retired, PreviousGeneration: previous, OperationID: operation, Phase: "planned", Container: ResourceIdentity{ContractDigest: contract}})
	return s, s.Validate(plan)
}
func (s MemberResume) Validate(plan Plan) error {
	fail := func() error { return errors.New("invalid retained-pier member resume") }
	r := s.Retirement
	old := r.Exit.Stop
	if r.Validate(plan) != nil || r.Phase != "absent" || s.PreviousGeneration == 0 || s.PreviousGeneration == math.MaxUint64 || !protocol.ValidID(s.OperationID) || !digestPattern.MatchString(s.Container.ContractDigest) {
		return fail()
	}
	for _, id := range []string{r.OperationID, r.Exit.OperationID, old.OperationID, old.LaunchOperationID, old.ReadinessOperationID} {
		if s.OperationID == id {
			return fail()
		}
	}
	phases := map[string]int{"planned": 0, "create-dispatched": 1, "created": 2, "start-dispatched": 3, "started": 4, "running": 5}
	n, ok := phases[s.Phase]
	if !ok {
		return fail()
	}
	if n == 0 {
		if s.Before != nil {
			return fail()
		}
	} else if s.Before == nil || s.checkBefore(*s.Before, s.Before.CapturedAt) != nil {
		return fail()
	}
	if n < 2 {
		if s.Container.ID != "" {
			return fail()
		}
	} else {
		if !s.Container.valid() || s.Container.ID == old.Binding.Bridge.ID || s.Container.ID == old.Binding.Anchor.ID {
			return fail()
		}
		for _, m := range old.Binding.Members {
			if s.Container.ID == m.Container.ID {
				return fail()
			}
		}
	}
	if n < 3 {
		if s.Created != nil {
			return fail()
		}
	} else if s.Created == nil || s.checkObservation(*s.Created, s.Created.CapturedAt, false) != nil {
		return fail()
	}
	if n < 5 {
		if s.After != nil {
			return fail()
		}
	} else if s.After == nil || s.checkObservation(*s.After, s.After.CapturedAt, true) != nil {
		return fail()
	}
	return nil
}
func (s MemberResume) checkBefore(o MemberAbsenceObservation, now time.Time) error {
	r := s.Retirement
	if r.After == nil || r.checkAbsent(o, now) != nil || o.CapturedAt.Before(r.After.CapturedAt) || o.Runtime.ObservedAt.Before(r.After.Runtime.ObservedAt) {
		return errors.New("resume lost fresh retirement absence")
	}
	expected := *r.After
	expected.Runtime = cloneReadinessObservation(expected.Runtime)
	expected.CapturedAt = o.CapturedAt
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	if len(o.Runtime.Members) == 0 {
		o.Runtime.Members = nil
	}
	if len(expected.Runtime.Members) == 0 {
		expected.Runtime.Members = nil
	}
	if !reflect.DeepEqual(expected, o) {
		return errors.New("resume project or storage changed")
	}
	return nil
}
func (s MemberResume) checkObservation(o MemberResumeObservation, now time.Time, running bool) error {
	fail := func() error { return errors.New("resume lacks exact fresh container, peer and pier observation") }
	if s.Before == nil || now.IsZero() || o.CapturedAt.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second || o.CapturedAt.Before(s.Before.CapturedAt) || o.Runtime.ObservedAt.Before(s.Before.Runtime.ObservedAt) || now.Sub(o.Runtime.ObservedAt) > 30*time.Second || o.Runtime.ObservedAt.Sub(now) > 5*time.Second || o.CapturedAt.Sub(o.Runtime.ObservedAt) > 30*time.Second || o.Runtime.ObservedAt.Sub(o.CapturedAt) > 5*time.Second || o.Storage != s.Retirement.Exit.Stop.Storage {
		return fail()
	}
	b := s.Retirement.Exit.Stop.Binding
	expected := cloneReadinessObservation(s.Before.Runtime)
	expected.ObservedAt = o.Runtime.ObservedAt
	target := MemberObservation{InstanceID: s.Retirement.Exit.Stop.InstanceID, ResourceObservation: ResourceObservation{Resource: s.Container, Present: true}, Running: running, AnchorID: b.Anchor.ID}
	if running {
		target.Namespace = b.Namespace
	}
	expected.Members = append(expected.Members, target)
	slices.SortFunc(expected.Members, func(a, b MemberObservation) int {
		if a.InstanceID < b.InstanceID {
			return -1
		}
		if a.InstanceID > b.InstanceID {
			return 1
		}
		return 0
	})
	if !reflect.DeepEqual(expected, o.Runtime) {
		return fail()
	}
	if !running {
		if !o.StartedAt.IsZero() {
			return fail()
		}
	} else {
		if s.Created == nil || o.StartedAt.IsZero() || o.StartedAt.Before(s.Created.CapturedAt.Add(-5*time.Second)) || o.StartedAt.After(o.CapturedAt.Add(5*time.Second)) || o.CapturedAt.Before(s.Created.CapturedAt) || o.Runtime.ObservedAt.Before(s.Created.Runtime.ObservedAt) {
			return fail()
		}
	}
	return nil
}

type MemberResumeResult struct {
	Before      *MemberAbsenceObservation
	ContainerID string
	Observation *MemberResumeObservation
}

func (s MemberResume) Advance(plan Plan, expected, next string, result MemberResumeResult) (MemberResume, error) {
	fail := func() (MemberResume, error) {
		return MemberResume{}, errors.New("stale or invalid member resume transition")
	}
	steps := map[string]string{"planned": "create-dispatched", "create-dispatched": "created", "created": "start-dispatched", "start-dispatched": "started", "started": "running"}
	if s.Validate(plan) != nil || s.Phase != expected || steps[expected] != next {
		return fail()
	}
	s = cloneMemberResume(s)
	switch next {
	case "create-dispatched":
		if result.Before == nil || result.ContainerID != "" || result.Observation != nil {
			return fail()
		}
		v := *result.Before
		v.Runtime = cloneReadinessObservation(v.Runtime)
		s.Before = &v
	case "created":
		if result.Before != nil || result.Observation != nil || !digestPattern.MatchString(result.ContainerID) {
			return fail()
		}
		s.Container.ID = result.ContainerID
	case "start-dispatched", "running":
		if result.Before != nil || result.ContainerID != "" || result.Observation == nil {
			return fail()
		}
		v := cloneResumeObservation(*result.Observation)
		if next == "running" {
			s.After = &v
		} else {
			s.Created = &v
		}
	case "started":
		if result.Before != nil || result.ContainerID != "" || result.Observation != nil {
			return fail()
		}
	}
	s.Phase = next
	if s.Validate(plan) != nil {
		return fail()
	}
	return s, nil
}

// Each inspection brackets complete engine inventory and unchanged peer processes
// with the exact initialized pier. Create and Start repeat freshness/storage/mount
// checks at dispatch. Create uses resume arguments (no -F, -B or -c), no pill,
// and generation PreviousGeneration+1. No method removes, boots, or retries.
type MemberResumeDriver interface {
	VerifyEngine(context.Context, string) error
	InspectAbsent(context.Context, MemberResume) (MemberAbsenceObservation, error)
	Create(context.Context, MemberResume) (string, error)
	Inspect(context.Context, MemberResume) (MemberResumeObservation, error)
	Start(context.Context, MemberResume) error
}
type MemberResumeCommit func(expected string, next MemberResume) error

func ExecuteMemberResume(ctx context.Context, plan Plan, s MemberResume, d MemberResumeDriver, commit MemberResumeCommit, now func() time.Time) (MemberResume, error) {
	if s.Validate(plan) != nil || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, errors.New("member resume requires fresh intent and durable dispatch")
	}
	s = cloneMemberResume(s)
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.VerifyEngine(ctx, s.Retirement.Exit.Stop.Binding.EngineID) != nil {
			return fail("resume engine verification failed")
		}
		return ctx.Err()
	}
	advance := func(next string, result MemberResumeResult) error {
		v, err := s.Advance(plan, s.Phase, next, result)
		if err != nil {
			return err
		}
		if err = commit(s.Phase, cloneMemberResume(v)); err != nil {
			return err
		}
		s = v
		return nil
	}
	if err := verify(); err != nil {
		return s, err
	}
	before, err := d.InspectAbsent(ctx, cloneMemberResume(s))
	before.Runtime = cloneReadinessObservation(before.Runtime)
	if err != nil {
		return s, fail("resume absence observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkBefore(before, now()); err != nil {
		return s, err
	}
	if err = advance("create-dispatched", MemberResumeResult{Before: &before}); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkBefore(*s.Before, now()); err != nil {
		return s, err
	}
	id, createErr := d.Create(ctx, cloneMemberResume(s))
	if id != "" {
		if err = advance("created", MemberResumeResult{ContainerID: id}); err != nil {
			return s, err
		}
	}
	if createErr != nil {
		return s, fail("resume Create outcome unresolved; retain its receipt")
	}
	if s.Phase != "created" {
		return s, fail("resume Create returned no exact ID")
	}
	if err = verify(); err != nil {
		return s, err
	}
	created, err := d.Inspect(ctx, cloneMemberResume(s))
	created = cloneResumeObservation(created)
	if err != nil {
		return s, fail("resume created-container observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(created, now(), false); err != nil {
		return s, err
	}
	if err = advance("start-dispatched", MemberResumeResult{Observation: &created}); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(*s.Created, now(), false); err != nil {
		return s, err
	}
	if err = d.Start(ctx, cloneMemberResume(s)); err != nil {
		return s, fail("resume Start outcome unresolved; retain its receipt")
	}
	if err = advance("started", MemberResumeResult{}); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	after, err := d.Inspect(ctx, cloneMemberResume(s))
	after = cloneResumeObservation(after)
	if err != nil {
		return s, fail("resume running-container observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(after, now(), true); err != nil {
		return s, err
	}
	if err = advance("running", MemberResumeResult{Observation: &after}); err != nil {
		return s, err
	}
	return s, nil
}
