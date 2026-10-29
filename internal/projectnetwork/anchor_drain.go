package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AnchorRecoveryHistory is supplied from validated retained journals under the
// project fence. It is not an instruction to discover or adopt live resources.
type AnchorRecoveryHistory struct {
	Epochs    []AnchorRecoveryEpoch
	Plan      Plan
	Launches  []MemberLaunch
	Resumes   []MemberResume
	Originals []InitialMemberReadiness
	Current   []CurrentMemberReadiness
}

func (h AnchorRecoveryHistory) validateReview(r AnchorRecoveryReview) error {
	c, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return err
	}
	return r.validateContext(h.Plan, c)
}
func cloneAnchorRecoveryHistory(h AnchorRecoveryHistory) AnchorRecoveryHistory {
	h.Plan.Members = slices.Clone(h.Plan.Members)
	h.Epochs = slices.Clone(h.Epochs)
	for i := range h.Epochs {
		e := &h.Epochs[i]
		copy := cloneAnchorRecoveryHistory(AnchorRecoveryHistory{Resumes: e.Resumes, Current: e.Current})
		e.Resumes, e.Current = copy.Resumes, copy.Current
		e.Readiness = e.Readiness.Clone()
	}

	h.Launches = slices.Clone(h.Launches)
	for i := range h.Launches {
		m := &h.Launches[i]
		m.Peers = slices.Clone(m.Peers)
		if m.Namespace != nil {
			v := *m.Namespace
			m.Namespace = &v
		}
		if m.Anchor.Namespace != nil {
			v := *m.Anchor.Namespace
			m.Anchor.Namespace = &v
		}
	}
	h.Resumes = slices.Clone(h.Resumes)
	for i := range h.Resumes {
		h.Resumes[i] = cloneMemberResume(h.Resumes[i])
	}
	h.Originals = slices.Clone(h.Originals)
	for i := range h.Originals {
		m := &h.Originals[i]
		m.Binding.Members = slices.Clone(m.Binding.Members)
		m.Before = cloneReadinessObservation(m.Before)
		m.After = cloneReadinessObservation(m.After)
	}
	h.Current = slices.Clone(h.Current)
	for i := range h.Current {
		m := &h.Current[i]
		m.Binding.Members = slices.Clone(m.Binding.Members)
		m.Before = cloneMemberStopObservation(m.Before)
		m.After = cloneMemberStopObservation(m.After)
	}
	return h
}
func cloneAnchorRecoveryReview(r AnchorRecoveryReview) AnchorRecoveryReview {
	r.Context = cloneAnchorRecoveryContext(r.Context)
	r.Before = cloneAnchorRecoveryObservation(r.Before)
	r.After = cloneAnchorRecoveryObservation(r.After)
	r.Assessment.Running = slices.Clone(r.Assessment.Running)
	r.Assessment.Stopped = slices.Clone(r.Assessment.Stopped)
	return r
}

type RecoveryProcessState struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	ExitCode   int       `json:"exit_code"`
	OOMKilled  bool      `json:"oom_killed"`
}
type AnchorDrainProcess struct {
	InstanceID string               `json:"instance_id"`
	Storage    pierstore.Location   `json:"storage"`
	Process    RecoveryProcessState `json:"process"`
}
type AnchorDrainObservation struct {
	Runtime       Observation          `json:"runtime"`
	AnchorProcess RecoveryProcessState `json:"anchor_process"`
	Processes     []AnchorDrainProcess `json:"processes"`
	CapturedAt    time.Time            `json:"captured_at"`
}
type AnchorDrainMember struct {
	InstanceID         string                  `json:"instance_id"`
	Phase              string                  `json:"phase"`
	SignalAcknowledged bool                    `json:"signal_acknowledged"`
	Before             *AnchorDrainObservation `json:"before,omitempty"`
	After              *AnchorDrainObservation `json:"after,omitempty"`
}

// AnchorDrain is a distinct recovery authority. Ordinary MemberStop remains
// limited to healthy anchor bindings. This intent may only drain the old exact
// processes; it grants no anchor Start, retirement, replacement or adoption.
// Persistence and concrete dispatch are not enabled by this pure model.
type AnchorDrain struct {
	OperationID string               `json:"operation_id"`
	Sequence    uint64               `json:"sequence"`
	Review      AnchorRecoveryReview `json:"review"`
	Members     []AnchorDrainMember  `json:"members"`
}

func cloneAnchorDrainObservation(o AnchorDrainObservation) AnchorDrainObservation {
	o.Runtime = cloneReadinessObservation(o.Runtime)
	o.Processes = slices.Clone(o.Processes)
	return o
}
func cloneAnchorDrain(s AnchorDrain) AnchorDrain {
	s.Review = cloneAnchorRecoveryReview(s.Review)
	s.Members = slices.Clone(s.Members)
	for i := range s.Members {
		m := &s.Members[i]
		if m.Before != nil {
			v := cloneAnchorDrainObservation(*m.Before)
			m.Before = &v
		}
		if m.After != nil {
			v := cloneAnchorDrainObservation(*m.After)
			m.After = &v
		}
	}
	return s
}
func NewAnchorDrain(h AnchorRecoveryHistory, r AnchorRecoveryReview, operation string) (AnchorDrain, error) {
	s := AnchorDrain{OperationID: operation, Review: cloneAnchorRecoveryReview(r)}
	for _, m := range r.Context.Binding.Members {
		s.Members = append(s.Members, AnchorDrainMember{InstanceID: m.InstanceID, Phase: "planned"})
	}
	return s, s.Validate(h)
}
func recoveryProcessValid(p RecoveryProcessState, running bool, at time.Time) bool {
	if p.StartedAt.IsZero() || p.StartedAt.After(at) || p.ExitCode < 0 || p.ExitCode > 255 {
		return false
	}
	if running {
		return p.FinishedAt.IsZero() && p.ExitCode == 0 && !p.OOMKilled
	}
	return !p.FinishedAt.IsZero() && !p.FinishedAt.Before(p.StartedAt) && !p.FinishedAt.After(at.Add(5*time.Second))
}

// An anchor may retain a previous exit timestamp across Docker Start. Preserve
// it in evidence, while distinguishing it from an exit of the current process.
func recoveryAnchorProcessValid(p RecoveryProcessState, running bool, at time.Time) bool {
	if running && !p.FinishedAt.IsZero() && p.FinishedAt.Before(p.StartedAt) {
		p.FinishedAt = time.Time{}
	}
	return recoveryProcessValid(p, running, at)
}
func drainProcess(o AnchorDrainObservation, instance string) (AnchorDrainProcess, bool) {
	var result AnchorDrainProcess
	count := 0
	for _, p := range o.Processes {
		if p.InstanceID == instance {
			result = p
			count++
		}
	}
	return result, count == 1
}
func sortRecoveryMembers(o *Observation) {
	slices.SortFunc(o.Members, func(a, b MemberObservation) int {
		if a.InstanceID < b.InstanceID {
			return -1
		}
		if a.InstanceID > b.InstanceID {
			return 1
		}
		return 0
	})
}
func (s AnchorDrain) checkObservation(h AnchorRecoveryHistory, index int, o AnchorDrainObservation, exited bool, now time.Time) error {
	fail := func() error {
		return errors.New("anchor drain requires unchanged exact process, pier and fault evidence")
	}
	if index < 0 || index >= len(s.Members) || now.IsZero() || o.CapturedAt.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second || o.CapturedAt.Before(o.Runtime.ObservedAt) || o.CapturedAt.Before(s.Review.CapturedAt) || o.Runtime.ObservedAt.Before(s.Review.CapturedAt) || len(o.Processes) != len(s.Members) {
		return fail()
	}
	assessment, err := AssessRuntime(h.Plan, s.Review.Context.Binding, o.Runtime, now)
	if err != nil || assessment.State != s.Review.Assessment.State || !assessment.RecoveryRequired {
		return fail()
	}
	expected := cloneReadinessObservation(s.Review.After.Runtime)
	expected.ObservedAt = o.Runtime.ObservedAt
	for i, m := range s.Members {
		for k := range expected.Members {
			if expected.Members[k].InstanceID == m.InstanceID && (i < index || i == index && exited) {
				expected.Members[k].Running = false
				expected.Members[k].Namespace = NamespaceIdentity{}
			}
		}
	}
	actual := cloneReadinessObservation(o.Runtime)
	sortRecoveryMembers(&expected)
	sortRecoveryMembers(&actual)
	if !equalEvidence(expected, actual) || !recoveryAnchorProcessValid(o.AnchorProcess, o.Runtime.Anchor.Running, o.CapturedAt) {
		return fail()
	}
	if s.Members[0].Before != nil && !equalEvidence(o.AnchorProcess, s.Members[0].Before.AnchorProcess) {
		return fail()
	}
	if index > 0 && (s.Members[index-1].After == nil || o.CapturedAt.Before(s.Members[index-1].After.CapturedAt) || o.Runtime.ObservedAt.Before(s.Members[index-1].After.Runtime.ObservedAt)) {
		return fail()
	}
	for i, m := range s.Members {
		got, ok := drainProcess(o, m.InstanceID)
		if !ok {
			return fail()
		}
		var original AnchorRecoveryProcess
		for _, p := range s.Review.After.Processes {
			if p.InstanceID == m.InstanceID {
				original = p
			}
		}
		if original.InstanceID == "" || got.Storage != original.Storage || !got.Process.StartedAt.Equal(original.StartedAt) {
			return fail()
		}
		switch {
		case i < index:
			if m.After == nil {
				return fail()
			}
			old, ok := drainProcess(*m.After, m.InstanceID)
			if !ok || !equalEvidence(got, old) || !recoveryProcessValid(got.Process, false, o.CapturedAt) {
				return fail()
			}
		case i == index && exited:
			if m.Before == nil || !recoveryProcessValid(got.Process, false, o.CapturedAt) || got.Process.FinishedAt.Before(m.Before.CapturedAt.Add(-5*time.Second)) || o.CapturedAt.Before(m.Before.CapturedAt) || o.Runtime.ObservedAt.Before(m.Before.Runtime.ObservedAt) {
				return fail()
			}
		default:
			if !recoveryProcessValid(got.Process, true, o.CapturedAt) {
				return fail()
			}
		}
	}
	return nil
}
func (s AnchorDrain) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid coordinated anchor drain") }
	if !protocol.ValidID(s.OperationID) || recoveryOperationExists(h, s.OperationID) || h.validateReview(s.Review) != nil || len(s.Members) != len(s.Review.Context.Binding.Members) {
		return fail()
	}
	for _, m := range h.Launches {
		if s.OperationID == m.OperationID || s.OperationID == m.Anchor.OperationID || s.OperationID == m.Anchor.Prepared.OperationID || s.OperationID == m.Storage.OperationID {
			return fail()
		}
	}
	for _, m := range h.Resumes {
		if s.OperationID == m.OperationID {
			return fail()
		}
	}
	var sequence uint64
	pending := false
	unclean := false
	for i, m := range s.Members {
		if m.InstanceID != s.Review.Context.Binding.Members[i].InstanceID {
			return fail()
		}
		if m.Phase == "planned" {
			if m.Before != nil || m.After != nil || m.SignalAcknowledged {
				return fail()
			}
			pending = true
			continue
		}
		if pending || unclean || m.Before == nil || s.checkObservation(h, i, *m.Before, false, m.Before.CapturedAt) != nil {
			return fail()
		}
		switch m.Phase {
		case "stop-dispatched":
			if m.After != nil || m.SignalAcknowledged {
				return fail()
			}
			sequence++
			pending = true
		case "stop-signalled":
			if m.After != nil || !m.SignalAcknowledged {
				return fail()
			}
			sequence += 2
			pending = true
		case "exited":
			if m.After == nil || s.checkObservation(h, i, *m.After, true, m.After.CapturedAt) != nil {
				return fail()
			}
			sequence += 2
			if m.SignalAcknowledged {
				sequence++
			}
			p, ok := drainProcess(*m.After, m.InstanceID)
			if !ok {
				return fail()
			}
			unclean = p.Process.ExitCode != 0 || p.Process.OOMKilled
		default:
			return fail()
		}
	}
	if sequence != s.Sequence {
		return fail()
	}
	return nil
}

// Clean classifies all retained exits only; it grants no later effect authority.
func (s AnchorDrain) Clean() bool {
	if len(s.Members) == 0 {
		return false
	}
	for _, m := range s.Members {
		if m.Phase != "exited" || m.After == nil {
			return false
		}
		p, ok := drainProcess(*m.After, m.InstanceID)
		if !ok || p.Process.ExitCode != 0 || p.Process.OOMKilled || p.Process.FinishedAt.IsZero() {
			return false
		}
	}
	return true
}
func (s AnchorDrain) next() int {
	for i, m := range s.Members {
		if m.Phase != "exited" {
			return i
		}
		p, ok := drainProcess(*m.After, m.InstanceID)
		if !ok || p.Process.ExitCode != 0 || p.Process.OOMKilled {
			return -1
		}
	}
	return -1
}
func (s AnchorDrain) Advance(h AnchorRecoveryHistory, index int, expected uint64, next string, o *AnchorDrainObservation) (AnchorDrain, error) {
	fail := func() (AnchorDrain, error) {
		return AnchorDrain{}, errors.New("stale or invalid anchor drain transition")
	}
	if s.Validate(h) != nil || expected != s.Sequence || index != s.next() || index < 0 {
		return fail()
	}
	s = cloneAnchorDrain(s)
	m := &s.Members[index]
	switch next {
	case "stop-dispatched":
		if m.Phase != "planned" || o == nil {
			return fail()
		}
		v := cloneAnchorDrainObservation(*o)
		m.Before = &v
	case "stop-signalled":
		if m.Phase != "stop-dispatched" || o != nil {
			return fail()
		}
		m.SignalAcknowledged = true
	case "exited":
		if (m.Phase != "stop-dispatched" && m.Phase != "stop-signalled") || o == nil {
			return fail()
		}
		v := cloneAnchorDrainObservation(*o)
		m.After = &v
	default:
		return fail()
	}
	m.Phase = next
	s.Sequence++
	if s.Validate(h) != nil {
		return fail()
	}
	return s, nil
}

type AnchorDrainObserver interface {
	VerifyEngine(context.Context, string) error
	InspectDrain(context.Context, AnchorDrain) (AnchorDrainObservation, error)
}

// SignalDrainMember must recheck complete fresh ownership after the committed
// dispatch. It may issue one graceful TERM to the selected old member only.
// AwaitDrainExit is bounded and read-only. Neither method may Start or remove.
type AnchorDrainDriver interface {
	AnchorDrainObserver
	SignalDrainMember(context.Context, AnchorDrain, int) error
	AwaitDrainExit(context.Context, AnchorDrain, int) (AnchorDrainObservation, error)
}
type AnchorDrainCommit func(expected uint64, next AnchorDrain) error

// ExecuteNextAnchorDrain stops at most one member. A dispatched/signalled step
// is never re-sent; use ObserveAnchorDrainExit to resolve a retained uncertainty.
func ExecuteNextAnchorDrain(ctx context.Context, h AnchorRecoveryHistory, s AnchorDrain, d AnchorDrainDriver, commit AnchorDrainCommit, now func() time.Time) (AnchorDrain, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("anchor drain requires retained review and durable dispatch")
	}
	h = cloneAnchorRecoveryHistory(h)
	s = cloneAnchorDrain(s)
	index := s.next()
	if index < 0 || s.Members[index].Phase != "planned" {
		return s, errors.New("anchor drain has no fresh next signal; retain uncertainty")
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	advance := func(phase string, o *AnchorDrainObservation) error {
		next, err := s.Advance(h, index, s.Sequence, phase, o)
		if err != nil {
			return err
		}
		if err = commit(s.Sequence, cloneAnchorDrain(next)); err != nil {
			return err
		}
		s = next
		return nil
	}
	if !verify() {
		return s, fail("anchor drain engine unavailable")
	}
	before, err := d.InspectDrain(ctx, cloneAnchorDrain(s))
	before = cloneAnchorDrainObservation(before)
	if err != nil || !verify() || s.checkObservation(h, index, before, false, now()) != nil {
		return s, fail("anchor drain initial observation refused")
	}
	if err = advance("stop-dispatched", &before); err != nil {
		return s, err
	}
	if !verify() || s.checkObservation(h, index, *s.Members[index].Before, false, now()) != nil {
		return s, fail("anchor drain dispatch evidence expired or changed")
	}
	if err = d.SignalDrainMember(ctx, cloneAnchorDrain(s), index); err != nil {
		return s, fail("anchor drain signal outcome unresolved; do not resend")
	}
	if err = advance("stop-signalled", nil); err != nil {
		return s, err
	}
	if !verify() {
		return s, fail("anchor drain engine changed after signal")
	}
	after, waitErr := d.AwaitDrainExit(ctx, cloneAnchorDrain(s), index)
	after = cloneAnchorDrainObservation(after)
	if !verify() || waitErr != nil || s.checkObservation(h, index, after, true, now()) != nil {
		return s, fail("anchor drain exit observation refused")
	}
	if err = advance("exited", &after); err != nil {
		return s, err
	}
	p, _ := drainProcess(after, s.Members[index].InstanceID)
	if p.Process.ExitCode != 0 || p.Process.OOMKilled {
		return s, fail("anchor drain retained an exit requiring review")
	}
	return s, nil
}

func ObserveAnchorDrainExit(ctx context.Context, h AnchorRecoveryHistory, s AnchorDrain, d AnchorDrainObserver, commit AnchorDrainCommit, now func() time.Time) (AnchorDrain, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("anchor drain exit requires retained dispatch")
	}
	h = cloneAnchorRecoveryHistory(h)
	s = cloneAnchorDrain(s)
	index := s.next()
	if index < 0 || (s.Members[index].Phase != "stop-dispatched" && s.Members[index].Phase != "stop-signalled") {
		return s, errors.New("anchor drain exit has no unresolved dispatch")
	}
	fail := func() (AnchorDrain, error) {
		return s, errors.Join(errors.New("anchor drain exit has no exact fresh evidence"), ctx.Err())
	}
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return fail()
	}
	o, err := d.InspectDrain(ctx, cloneAnchorDrain(s))
	o = cloneAnchorDrainObservation(o)
	if err != nil || !verify() || s.checkObservation(h, index, o, true, now()) != nil {
		return fail()
	}
	next, err := s.Advance(h, index, s.Sequence, "exited", &o)
	if err != nil {
		return fail()
	}
	if err = commit(s.Sequence, cloneAnchorDrain(next)); err != nil {
		return s, err
	}
	p, _ := drainProcess(o, s.Members[index].InstanceID)
	if p.Process.ExitCode != 0 || p.Process.OOMKilled {
		return next, errors.New("anchor drain observed an exit requiring review")
	}
	return next, nil
}

// ValidateObservation lets a concrete adapter recheck a retained intent at the
// last boundary before its one signal. It cannot relax historical validation.
func (s AnchorDrain) ValidateObservation(h AnchorRecoveryHistory, index int, o AnchorDrainObservation, exited bool, now time.Time) error {
	if err := s.Validate(h); err != nil {
		return err
	}
	return s.checkObservation(h, index, o, exited, now)
}
