package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AnchorNamespace records a successor to the original namespace after all old
// members have cleanly exited. It does not rewrite AnchorStart or any creator,
// and does not make old containers members of the successor namespace. A future
// fenced recovery queue must persist this intent before connecting a driver.
type AnchorNamespace struct {
	OperationID       string                  `json:"operation_id"`
	Sequence          uint64                  `json:"sequence"`
	Drain             AnchorDrain             `json:"drain"`
	Phase             string                  `json:"phase"`
	StartAcknowledged bool                    `json:"start_acknowledged"`
	Before            *AnchorDrainObservation `json:"before,omitempty"`
	After             *AnchorDrainObservation `json:"after,omitempty"`
}

func (s AnchorNamespace) Clone() AnchorNamespace {
	s.Drain = s.Drain.Clone()
	if s.Before != nil {
		o := cloneAnchorDrainObservation(*s.Before)
		s.Before = &o
	}
	if s.After != nil {
		o := cloneAnchorDrainObservation(*s.After)
		s.After = &o
	}
	return s
}
func NewAnchorNamespace(h AnchorRecoveryHistory, drain AnchorDrain, operation string) (AnchorNamespace, error) {
	s := AnchorNamespace{OperationID: operation, Drain: drain.Clone(), Phase: "planned"}
	return s, s.Validate(h)
}
func freshNamespaceObservation(o AnchorDrainObservation, now, notBefore time.Time) bool {
	return !now.IsZero() && !o.CapturedAt.IsZero() && !o.Runtime.ObservedAt.IsZero() && now.Sub(o.CapturedAt) <= 30*time.Second && o.CapturedAt.Sub(now) <= 5*time.Second && now.Sub(o.Runtime.ObservedAt) <= 30*time.Second && o.Runtime.ObservedAt.Sub(now) <= 5*time.Second && !o.CapturedAt.Before(o.Runtime.ObservedAt) && !o.CapturedAt.Before(notBefore) && !o.Runtime.ObservedAt.Before(notBefore)
}
func sameNamespaceObservation(expected, actual AnchorDrainObservation) bool {
	expected = cloneAnchorDrainObservation(expected)
	actual = cloneAnchorDrainObservation(actual)
	expected.CapturedAt = actual.CapturedAt
	expected.Runtime.ObservedAt = actual.Runtime.ObservedAt
	sortRecoveryMembers(&expected.Runtime)
	sortRecoveryMembers(&actual.Runtime)
	less := func(a, b AnchorDrainProcess) int {
		if a.InstanceID < b.InstanceID {
			return -1
		}
		if a.InstanceID > b.InstanceID {
			return 1
		}
		return 0
	}
	slices.SortFunc(expected.Processes, less)
	slices.SortFunc(actual.Processes, less)
	return equalEvidence(expected, actual)
}
func (s AnchorNamespace) checkBefore(o AnchorDrainObservation, now time.Time) error {
	last := s.Drain.Members[len(s.Drain.Members)-1].After
	if last == nil || !freshNamespaceObservation(o, now, last.CapturedAt) || !sameNamespaceObservation(*last, o) {
		return errors.New("namespace succession lost the complete clean drain")
	}
	return nil
}
func (s AnchorNamespace) checkAfter(o AnchorDrainObservation, now time.Time) error {
	fail := func() error {
		return errors.New("namespace succession requires exact anchor and stopped-member evidence")
	}
	if s.Before == nil || !freshNamespaceObservation(o, now, s.Before.CapturedAt) {
		return fail()
	}
	old := s.Drain.Review.Context.Binding.Namespace
	ns := o.Runtime.Anchor.Namespace
	if !ns.valid() || ns == old || ns.BootID != old.BootID || !o.Runtime.Anchor.Running || !recoveryAnchorProcessValid(o.AnchorProcess, true, o.CapturedAt) {
		return fail()
	}
	expected := cloneAnchorDrainObservation(*s.Before)
	if !expected.Runtime.Anchor.Running {
		if !o.AnchorProcess.FinishedAt.IsZero() && !o.AnchorProcess.FinishedAt.Equal(expected.AnchorProcess.FinishedAt) {
			return fail()
		}

		if !o.AnchorProcess.StartedAt.After(expected.AnchorProcess.StartedAt) || o.AnchorProcess.StartedAt.Before(expected.CapturedAt.Add(-5*time.Second)) {
			return fail()
		}
		expected.Runtime.Anchor.Running = true
		expected.Runtime.Anchor.Namespace = ns
		expected.Runtime.Anchor.Active = slices.Clone(expected.Runtime.Anchor.Requested)
		expected.Runtime.BridgeAttachments = []string{expected.Runtime.Anchor.Resource.ID}
		expected.AnchorProcess = o.AnchorProcess
	}
	if !sameNamespaceObservation(expected, o) {
		return fail()
	}
	return nil
}
func (s AnchorNamespace) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid successor namespace intent") }
	if s.Drain.Validate(h) != nil || !s.Drain.Clean() || !protocol.ValidID(s.OperationID) || s.OperationID == s.Drain.OperationID || recoveryOperationExists(h, s.OperationID) {
		return fail()
	}
	for _, l := range h.Launches {
		for _, id := range []string{l.OperationID, l.Storage.OperationID, l.Anchor.OperationID, l.Anchor.Prepared.OperationID} {
			if s.OperationID == id {
				return fail()
			}
		}
	}
	for _, r := range h.Resumes {
		if s.OperationID == r.OperationID {
			return fail()
		}
	}
	if s.Phase == "planned" {
		if s.Sequence != 0 || s.Before != nil || s.After != nil || s.StartAcknowledged {
			return fail()
		}
		return nil
	}
	if s.Before == nil || s.checkBefore(*s.Before, s.Before.CapturedAt) != nil {
		return fail()
	}
	restarted := s.Before.Runtime.Anchor.Running
	switch s.Phase {
	case "observing":
		if !restarted || s.Sequence != 1 || s.After != nil || s.StartAcknowledged {
			return fail()
		}
	case "start-dispatched":
		if restarted || s.Sequence != 1 || s.After != nil || s.StartAcknowledged {
			return fail()
		}
	case "start-acknowledged":
		if restarted || s.Sequence != 2 || s.After != nil || !s.StartAcknowledged {
			return fail()
		}
	case "observed":
		sequence := uint64(2)
		if s.StartAcknowledged {
			sequence++
		}
		if s.Sequence != sequence || restarted && s.StartAcknowledged || s.After == nil || s.checkAfter(*s.After, s.After.CapturedAt) != nil {
			return fail()
		}
	default:
		return fail()
	}
	return nil
}
func (s AnchorNamespace) Advance(h AnchorRecoveryHistory, expected uint64, phase string, o *AnchorDrainObservation) (AnchorNamespace, error) {
	if s.Validate(h) != nil || s.Sequence != expected {
		return AnchorNamespace{}, errors.New("stale namespace succession")
	}
	next := s.Clone()
	next.Sequence++
	switch {
	case s.Phase == "planned" && (phase == "observing" || phase == "start-dispatched"):
		if o == nil {
			return AnchorNamespace{}, errors.New("namespace succession lacks preflight")
		}
		v := cloneAnchorDrainObservation(*o)
		next.Before = &v
	case s.Phase == "start-dispatched" && phase == "start-acknowledged":
		if o != nil {
			return AnchorNamespace{}, errors.New("acknowledgement cannot substitute observation")
		}
		next.StartAcknowledged = true
	case (s.Phase == "observing" || s.Phase == "start-dispatched" || s.Phase == "start-acknowledged") && phase == "observed":
		if o == nil {
			return AnchorNamespace{}, errors.New("namespace succession lacks observation")
		}
		v := cloneAnchorDrainObservation(*o)
		next.After = &v
	default:
		return AnchorNamespace{}, errors.New("namespace succession cannot skip or replay")
	}
	next.Phase = phase
	return next, next.Validate(h)
}

// ValidateObservation is for a concrete driver's last check before its one Start.
func (s AnchorNamespace) ValidateObservation(h AnchorRecoveryHistory, o AnchorDrainObservation, after bool, now time.Time) error {
	if err := s.Validate(h); err != nil {
		return err
	}
	if after {
		return s.checkAfter(o, now)
	}
	return s.checkBefore(o, now)
}

// Namespace returns only the explicit successor identity. It grants no member
// replacement or readiness authority, and does not rebind any old receipt.
func (s AnchorNamespace) Namespace(h AnchorRecoveryHistory) (NamespaceIdentity, error) {
	if s.Validate(h) != nil || s.Phase != "observed" {
		return NamespaceIdentity{}, errors.New("namespace succession is incomplete")
	}
	return s.After.Runtime.Anchor.Namespace, nil
}

type AnchorNamespaceObserver interface {
	VerifyEngine(context.Context, string) error
	InspectNamespace(context.Context, AnchorNamespace) (AnchorDrainObservation, error)
}
type AnchorNamespaceDriver interface {
	AnchorNamespaceObserver
	// Start must freshly recheck every exact stopped process and the anchor before
	// one dispatch. Reopened or ambiguous intents must never resend this Start.
	StartRecoveryAnchor(context.Context, AnchorNamespace) error
}
type AnchorNamespaceCommit func(expected uint64, next AnchorNamespace) error

func ExecuteAnchorNamespace(ctx context.Context, h AnchorRecoveryHistory, s AnchorNamespace, d AnchorNamespaceDriver, commit AnchorNamespaceCommit, now func() time.Time) (AnchorNamespace, error) {
	if s.Validate(h) != nil || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, errors.New("namespace succession requires a fresh retained intent")
	}
	h = h.Clone()
	s = s.Clone()
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	advance := func(phase string, o *AnchorDrainObservation) error {
		next, err := s.Advance(h, s.Sequence, phase, o)
		if err != nil {
			return err
		}
		if err = commit(s.Sequence, next.Clone()); err != nil {
			return err
		}
		s = next
		return nil
	}
	if !verify() {
		return s, fail("namespace succession engine unavailable")
	}
	before, err := d.InspectNamespace(ctx, s.Clone())
	before = cloneAnchorDrainObservation(before)
	if err != nil || !verify() || s.checkBefore(before, now()) != nil {
		return s, fail("namespace succession preflight refused")
	}
	phase := "observing"
	if !before.Runtime.Anchor.Running {
		phase = "start-dispatched"
	}
	if err = advance(phase, &before); err != nil {
		return s, err
	}
	if !verify() || s.checkBefore(*s.Before, now()) != nil {
		return s, fail("namespace succession dispatch evidence expired or changed")
	}
	if s.Phase == "start-dispatched" {
		if err = d.StartRecoveryAnchor(ctx, s.Clone()); err != nil {
			return s, fail("anchor Start outcome unresolved; do not resend")
		}
		if err = advance("start-acknowledged", nil); err != nil {
			return s, err
		}
	}
	return ObserveAnchorNamespace(ctx, h, s, d, commit, now)
}

// ObserveAnchorNamespace completes pending evidence without issuing a Start.
// A lost acknowledgement remains false even if the exact running anchor is seen.
func ObserveAnchorNamespace(ctx context.Context, h AnchorRecoveryHistory, s AnchorNamespace, d AnchorNamespaceObserver, commit AnchorNamespaceCommit, now func() time.Time) (AnchorNamespace, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil || (s.Phase != "observing" && s.Phase != "start-dispatched" && s.Phase != "start-acknowledged") {
		return s, errors.New("namespace observation requires retained intent")
	}
	h = h.Clone()
	s = s.Clone()
	fail := func() (AnchorNamespace, error) {
		return s, errors.Join(errors.New("namespace succession observation unresolved"), ctx.Err())
	}
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return fail()
	}
	after, err := d.InspectNamespace(ctx, s.Clone())
	after = cloneAnchorDrainObservation(after)
	if err != nil || !verify() || s.checkAfter(after, now()) != nil {
		return fail()
	}
	next, err := s.Advance(h, s.Sequence, "observed", &after)
	if err != nil {
		return fail()
	}
	if err = commit(s.Sequence, next.Clone()); err != nil {
		return s, err
	}
	return next, nil
}
