package projectnetwork

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberStop is the first explicit graceful Stop of a known initialized member.
// It preserves all containers, piers and project resources. It does not authorize
// recovery of uncertain launches, restart, removal or forced termination.
// Historical readiness supplies storage identity, never current runtime authority.
type MemberStop struct {
	OperationID          string                 `json:"operation_id"`
	LaunchOperationID    string                 `json:"launch_operation_id"`
	ReadinessOperationID string                 `json:"readiness_operation_id"`
	InstanceID           string                 `json:"instance_id"`
	Binding              RuntimeBinding         `json:"binding"`
	Storage              pierstore.Location     `json:"storage"`
	Phase                string                 `json:"phase"`
	Before               *MemberStopObservation `json:"before,omitempty"`
	After                *MemberStopObservation `json:"after,omitempty"`
}

// Process times come from the exact inspected container. StartedAt must remain
// unchanged across the stop; merely finding the same stopped name is insufficient.
type MemberStopObservation struct {
	Runtime    Observation        `json:"runtime"`
	Storage    pierstore.Location `json:"storage"`
	StartedAt  time.Time          `json:"started_at"`
	FinishedAt time.Time          `json:"finished_at"`
	ExitCode   int                `json:"exit_code"`
	OOMKilled  bool               `json:"oom_killed"`
	CapturedAt time.Time          `json:"captured_at"`
}

func NewInitialMemberStop(plan Plan, launches []MemberLaunch, ready InitialMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	if ready.Validate(plan, launches) != nil {
		return MemberStop{}, errors.New("member Stop requires complete original readiness evidence")
	}
	s := MemberStop{OperationID: operation, LaunchOperationID: ready.LaunchOperationID, ReadinessOperationID: readinessOperation, InstanceID: ready.InstanceID, Binding: ready.Binding, Storage: ready.Storage, Phase: "planned"}
	s.Binding.Members = slices.Clone(s.Binding.Members)
	return s, s.Validate(plan)
}
func cloneMemberStopObservation(o MemberStopObservation) MemberStopObservation {
	o.Runtime = cloneReadinessObservation(o.Runtime)
	return o
}
func cloneMemberStop(s MemberStop) MemberStop {
	s.Binding.Members = slices.Clone(s.Binding.Members)
	if s.Before != nil {
		v := cloneMemberStopObservation(*s.Before)
		s.Before = &v
	}
	if s.After != nil {
		v := cloneMemberStopObservation(*s.After)
		s.After = &v
	}
	return s
}
func (s MemberStop) Validate(plan Plan) error {
	fail := func() error { return errors.New("invalid initial member Stop binding") }
	if s.Binding.Validate(plan) != nil || !protocol.ValidID(s.OperationID) || !protocol.ValidID(s.LaunchOperationID) || !protocol.ValidID(s.ReadinessOperationID) || s.OperationID == s.LaunchOperationID || s.OperationID == s.ReadinessOperationID || s.ReadinessOperationID == s.LaunchOperationID {
		return fail()
	}
	found := false
	for _, m := range plan.Members {
		if m.InstanceID == s.InstanceID {
			found = true
			if s.Storage.Path != "fakeships/"+m.InstanceID+"/"+strings.TrimPrefix(m.Ship, "~") {
				return fail()
			}
		}
	}
	id := s.Storage.Identity
	parent := s.Storage.MountIdentity
	if !found || s.Storage.Uncertain || !validMemberDirectory(parent) || id.Device == 0 || id.Inode == 0 || id.UrbDevice == 0 || id.UrbInode == 0 || id.Birth == "" || id.UrbBirth == "" || id.Device == id.UrbDevice && id.Inode == id.UrbInode || id.Device == parent.Device && id.Inode == parent.Inode || id.UrbDevice == parent.Device && id.UrbInode == parent.Inode {
		return fail()
	}
	switch s.Phase {
	case "planned":
		if s.Before != nil || s.After != nil {
			return fail()
		}
	case "stop-dispatched", "stop-signalled":
		if s.Before == nil || s.After != nil || s.checkObservation(plan, *s.Before, false, s.Before.CapturedAt) != nil {
			return fail()
		}
	case "stopped":
		if s.Before == nil || s.After == nil || s.checkObservation(plan, *s.Before, false, s.Before.CapturedAt) != nil || s.checkObservation(plan, *s.After, true, s.After.CapturedAt) != nil {
			return fail()
		}
	default:
		return fail()
	}
	return nil
}
func (s MemberStop) checkObservation(plan Plan, o MemberStopObservation, stopped bool, now time.Time) error {
	fail := func() error { return errors.New("member Stop ownership, storage or process observation changed") }
	if o.CapturedAt.IsZero() || now.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second || o.Storage != s.Storage || o.StartedAt.IsZero() || o.StartedAt.After(o.CapturedAt.Add(5*time.Second)) {
		return fail()
	}
	a, err := AssessRuntime(plan, s.Binding, o.Runtime, now)
	if err != nil || a.RecoveryRequired || a.State != "ready" && a.State != "member-stopped" {
		return fail()
	}
	// Stored receipts must also be internally fresh at their own capture time.
	if _, err = AssessRuntime(plan, s.Binding, o.Runtime, o.CapturedAt); err != nil {
		return fail()
	}
	targetFound := false
	for _, m := range o.Runtime.Members {
		if m.InstanceID == s.InstanceID {
			targetFound = true
			if !m.Present || m.Running == stopped {
				return fail()
			}
		}
	}
	if !targetFound {
		return fail()
	}
	if !stopped {
		if !o.FinishedAt.IsZero() || o.ExitCode != 0 || o.OOMKilled {
			return fail()
		}
		return nil
	}
	if s.Before == nil || !o.StartedAt.Equal(s.Before.StartedAt) || o.FinishedAt.IsZero() || o.FinishedAt.Before(o.StartedAt) || o.FinishedAt.Before(s.Before.CapturedAt.Add(-5*time.Second)) || o.FinishedAt.After(o.CapturedAt.Add(5*time.Second)) || o.ExitCode < 0 || o.ExitCode > 255 || o.CapturedAt.Before(s.Before.CapturedAt) || o.Runtime.ObservedAt.Before(s.Before.Runtime.ObservedAt) {
		return fail()
	}
	expected := cloneReadinessObservation(s.Before.Runtime)
	expected.ObservedAt = o.Runtime.ObservedAt
	for i := range expected.Members {
		if expected.Members[i].InstanceID == s.InstanceID {
			expected.Members[i].Running = false
			expected.Members[i].Namespace = NamespaceIdentity{}
		}
	}
	if !reflect.DeepEqual(expected, o.Runtime) {
		return fail()
	}
	return nil
}
func (s MemberStop) Advance(plan Plan, expected, next string, o *MemberStopObservation) (MemberStop, error) {
	fail := func() (MemberStop, error) { return MemberStop{}, errors.New("stale or invalid member Stop transition") }
	steps := map[string]string{"planned": "stop-dispatched", "stop-dispatched": "stop-signalled", "stop-signalled": "stopped"}
	if s.Validate(plan) != nil || s.Phase != expected || steps[expected] != next {
		return fail()
	}
	s = cloneMemberStop(s)
	switch next {
	case "stop-dispatched":
		if o == nil {
			return fail()
		}
		v := cloneMemberStopObservation(*o)
		s.Before = &v
	case "stop-signalled":
		if o != nil {
			return fail()
		}
	case "stopped":
		if o == nil {
			return fail()
		}
		v := cloneMemberStopObservation(*o)
		s.After = &v
	}
	s.Phase = next
	if s.Validate(plan) != nil {
		return fail()
	}
	return s, nil
}

// Clean classifies the retained exit only. Callers must validate the complete
// state; it is not evidence of current runtime or permission to resume.
func (s MemberStop) Clean() bool {
	return s.Phase == "stopped" && s.After != nil && s.After.ExitCode == 0 && !s.After.OOMKilled
}

// The adapter must recheck fresh exact ownership at signal dispatch. SignalStop
// requests graceful termination once; it must not escalate to SIGKILL or remove
// anything. AwaitStopped is read-only and bounded by ctx. Unknown/failed exits
// remain explicit evidence, not automatic permission to resume a pier.
type MemberStopDriver interface {
	VerifyEngine(context.Context, string) error
	Inspect(context.Context, MemberStop) (MemberStopObservation, error)
	SignalStop(context.Context, MemberStop) error
	AwaitStopped(context.Context, MemberStop) (MemberStopObservation, error)
}
type MemberStopCommit func(expected string, next MemberStop) error

func ExecuteMemberStop(ctx context.Context, plan Plan, s MemberStop, d MemberStopDriver, commit MemberStopCommit, now func() time.Time) (MemberStop, error) {
	if s.Validate(plan) != nil || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, errors.New("member Stop requires fresh intent and durable dispatch")
	}
	s = cloneMemberStop(s)
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.VerifyEngine(ctx, s.Binding.EngineID) != nil {
			return fail("member Stop engine could not be verified")
		}
		return ctx.Err()
	}
	advance := func(phase string, o *MemberStopObservation) error {
		next, err := s.Advance(plan, s.Phase, phase, o)
		if err != nil {
			return err
		}
		if err = commit(s.Phase, cloneMemberStop(next)); err != nil {
			return err
		}
		s = next
		return nil
	}
	if err := verify(); err != nil {
		return s, err
	}
	before, err := d.Inspect(ctx, cloneMemberStop(s))
	before = cloneMemberStopObservation(before)
	if err != nil {
		return s, fail("member Stop initial observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(plan, before, false, now()); err != nil {
		return s, err
	}
	if err = advance("stop-dispatched", &before); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(plan, *s.Before, false, now()); err != nil {
		return s, err
	}
	if err = d.SignalStop(ctx, cloneMemberStop(s)); err != nil {
		return s, fail("member Stop signal outcome is unresolved; retain dispatch intent")
	}
	if err = advance("stop-signalled", nil); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	after, waitErr := d.AwaitStopped(ctx, cloneMemberStop(s))
	after = cloneMemberStopObservation(after)
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkObservation(plan, after, true, now()); err != nil {
		return s, fail("member Stop has no complete unchanged exit evidence")
	}
	if err = advance("stopped", &after); err != nil {
		return s, err
	}
	if waitErr != nil {
		return s, fail("member Stop retained exit evidence with an unresolved observation result")
	}
	if !s.Clean() {
		return s, fmt.Errorf("member stopped with exit %d or OOM; preserve its pier for review", s.After.ExitCode)
	}
	return s, nil
}
