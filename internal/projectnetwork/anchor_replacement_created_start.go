package projectnetwork

import (
	"context"
	"errors"
	"time"
)

// StartCreatedAnchorReplacement requires separate explicit continuation authority
// when reopening a retained Create reply. Its driver has no Create method. The
// exact ID must still be unstarted with unchanged pier, peer and namespace state;
// an uncertain or already dispatched Start must use observation instead.
func StartCreatedAnchorReplacement(ctx context.Context, h AnchorRecoveryHistory, s AnchorReplacement, d AnchorReplacementStartDriver, commit AnchorReplacementCommit, now func() time.Time) (AnchorReplacement, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("created replacement Start requires retained authority and durable dispatch")
	}
	h = h.Clone()
	s = s.Clone()
	index := s.next()
	if index < 0 || s.Members[index].Phase != "created" {
		return s, errors.New("replacement lacks an exact retained Create reply")
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Retirement.Namespace.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	advance := func(phase string, result AnchorReplacementResult) error {
		next, err := s.Advance(h, index, s.Sequence, phase, result)
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
		return s, fail("created replacement engine unavailable")
	}
	created, err := d.InspectReplacement(ctx, s.Clone(), index, "created")
	created = cloneAnchorRetirementObservation(created)
	if err != nil || !verify() || s.checkObservation(index, created, "created", now()) != nil {
		return s, fail("created replacement observation refused")
	}
	if err = advance("start-dispatched", AnchorReplacementResult{Observation: &created}); err != nil {
		return s, err
	}
	if !verify() || s.checkObservation(index, *s.Members[index].Created, "created", now()) != nil {
		return s, fail("replacement Start preflight expired")
	}
	if err = d.StartReplacement(ctx, s.Clone(), index); err != nil || ctx.Err() != nil {
		return s, fail("replacement Start unresolved; do not resend")
	}
	if err = advance("start-acknowledged", AnchorReplacementResult{}); err != nil {
		return s, err
	}
	return ObserveAnchorReplacementRunning(ctx, h, s, d, commit, now)
}
