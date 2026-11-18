package grubbery

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// Store is authoritative, durable and transactional. Create atomically claims
// the derived app namespace until original completion or validated preactivation abandonment. It refuses a
// different unfinished owner, never changes an existing intent, and returns
// ErrConflict/ErrNamespaceBusy on collisions. Append commits one immutable event
// by exact revision CAS before success and releases only that operation's lease
// after its last observation or safe abandonment. Implementations must return independent snapshots.
// This interface is not evidence that a persistent adapter has been qualified.
type Store interface {
	Load(context.Context, string) (Record, error)
	Create(context.Context, Record) (Record, error)
	Append(context.Context, string, uint64, Event) (Record, error)
}

type Backend interface {
	// Admit and ObserveAfter are bounded read-only operations. No argument from
	// an external caller can substitute for their actual fresh evidence.
	Admit(context.Context, Plan, Step) (Review, error)
	// Dispatch makes exactly one typed native call. Virtual foundational source
	// does not establish a materialized directory or authorize adopting one.
	// It must not retry, redirect
	// or accept arbitrary URLs/commands. Source/cache identity is already checked.
	// ErrNativeRejected requires a complete, validated native pre-effect refusal
	// plus its response digest. Transport errors and5xx are never that evidence.
	Dispatch(context.Context, Plan, Step, Request) (string, error)
	// nil proof means not established yet. Repeated failed reads do not append
	// duplicate events or resubmit effects. Errors never become after-evidence.
	ObserveAfter(context.Context, Plan, Step) (*Proof, error)
}

// Call is assembled by the authenticated runner adapter. Actor/runner/mode come
// from its verified transport/server identity, not a model-supplied credential.
type Call struct {
	RunnerID, Mode, Actor                string
	Target                               apppolicy.Target
	AppID, RequestID, PlanDigest, StepID string
}

func (c Call) matches(r Record) bool {
	b := r.Plan.Binding
	return c.RunnerID == b.RunnerID && c.Mode == b.Mode && c.Actor == b.Actor && c.Target == b.Target && c.AppID == b.AppID && c.RequestID == r.Plan.RequestID && c.PlanDigest == r.Plan.Digest
}

type Execution struct {
	Store      Store
	Backend    Backend
	LoadSource func(context.Context, string) ([]deskwork.SourceFile, error)
	Now        func() time.Time
}

func (e Execution) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
func (e Execution) load(ctx context.Context, c Call) (Record, error) {
	if e.Store == nil || !digest.MatchString(c.RequestID) || !digest.MatchString(c.PlanDigest) {
		return Record{}, ErrConflict
	}
	r, err := e.Store.Load(ctx, c.RequestID)
	if err != nil {
		if errors.Is(err, ErrMissing) {
			return Record{}, ErrMissing
		}
		if errors.Is(err, ErrHistory) {
			return Record{}, ErrHistory
		}
		return Record{}, ErrJournal
	}
	if r.Validate() != nil {
		return Record{}, ErrHistory
	}
	if !c.matches(r) {
		return Record{}, ErrConflict
	}
	return r, nil
}

func (e Execution) Begin(ctx context.Context, c Call, p Plan) (Record, error) {
	if e.Store == nil {
		return Record{}, ErrJournal
	}
	if p.ValidateMetadata() != nil {
		return Record{}, ErrHistory
	}
	if !c.matches(Record{Plan: p}) {
		return Record{}, ErrConflict
	}
	if r, err := e.load(ctx, c); err == nil {
		return r, nil
	} else if !errors.Is(err, ErrMissing) {
		return Record{}, err
	}
	r, err := NewRecord(p, e.now())
	if err != nil {
		return Record{}, err
	}
	if !c.matches(r) {
		return Record{}, ErrConflict
	}
	if e.Backend == nil || e.LoadSource == nil {
		return Record{}, ErrAdmission
	}
	files, err := e.LoadSource(ctx, p.StageID)
	if err != nil || p.Validate(files) != nil {
		return Record{}, ErrSource
	}
	if ctx.Err() != nil {
		return Record{}, ErrAdmission
	}
	review, err := e.Backend.Admit(ctx, p, p.Steps[0])
	if err != nil || !review.matches(p, p.Steps[0].ID) {
		return Record{}, ErrAdmission
	}
	out, err := e.Store.Create(ctx, r)
	if err != nil {
		if errors.Is(err, ErrConflict) || errors.Is(err, ErrCAS) {
			if saved, readErr := e.load(ctx, c); readErr == nil {
				return saved, nil
			}
		}
		if errors.Is(err, ErrNamespaceBusy) {
			return Record{}, ErrNamespaceBusy
		}
		if errors.Is(err, ErrConflict) {
			return Record{}, ErrConflict
		}
		if errors.Is(err, ErrHistory) {
			return Record{}, ErrHistory
		}
		return Record{}, ErrJournal
	}
	if out.Validate() != nil || !c.matches(out) {
		return Record{}, ErrHistory
	}
	return out, nil
}

func (e Execution) Lookup(ctx context.Context, c Call) (Record, error) { return e.load(ctx, c) }

// Advance admits only a selected original phase. Repeating an observed phase
// returns its history even if later phases exist. A dispatched/uncertain phase
// returns saved progress, with no source read, probe, or native redispatch.
func (e Execution) Advance(ctx context.Context, c Call) (Record, error) {
	r, err := e.load(ctx, c)
	if err != nil {
		return Record{}, err
	}
	state, err := r.State(c.StepID)
	if err != nil {
		return Record{}, err
	}
	if state != "pending" || r.Abandoned() {
		return r, nil
	}
	if r.Plan.Version != 2 {
		return r, ErrLegacyPlan
	}
	step, complete, err := r.Current()
	if err != nil {
		return Record{}, err
	}
	if complete || step.ID != c.StepID {
		return Record{}, ErrPhase
	}
	if e.Backend == nil || e.LoadSource == nil {
		return r, ErrAdmission
	}
	files, err := e.LoadSource(ctx, r.Plan.StageID)
	if err != nil || r.Plan.Validate(files) != nil {
		return r, ErrSource
	}
	request, err := r.Plan.Request(step.ID, files)
	if err != nil {
		return r, ErrHistory
	}
	if ctx.Err() != nil {
		return r, ErrAdmission
	}
	review, err := e.Backend.Admit(ctx, r.Plan, step)
	if err != nil || !review.matches(r.Plan, step.ID) {
		return r, ErrAdmission
	}
	var claimed bool
	r, claimed, err = e.claim(ctx, c, r, Event{StepID: step.ID, State: "dispatched", At: e.now(), Review: &review})
	if err != nil {
		return r, err
	}
	// Another caller may have won the dispatch CAS. Its saved handle never
	// becomes permission for this worker to perform an additional native call.
	if !claimed {
		return r, nil
	}
	response, sendErr := e.Backend.Dispatch(ctx, r.Plan, step, request)
	if errors.Is(sendErr, ErrNativeRejected) && digest.MatchString(response) {
		return e.append(ctx, c, r, Event{StepID: step.ID, State: "refused", At: e.now(), ResponseDigest: response, Reason: "native-rejected"})
	}
	if sendErr != nil || !digest.MatchString(response) {
		return e.append(ctx, c, r, Event{StepID: step.ID, State: "uncertain", At: e.now(), Reason: "transport-outcome-unknown"})
	}
	r, err = e.append(ctx, c, r, Event{StepID: step.ID, State: "acknowledged", At: e.now(), ResponseDigest: response})
	if err != nil {
		return r, err
	}
	state, _ = r.State(step.ID)
	if state != "acknowledged" {
		return r, nil
	}
	proof, readErr := e.Backend.ObserveAfter(ctx, r.Plan, step)
	if readErr != nil || proof == nil || !proof.matches(r.Plan, step.ID) {
		return e.append(ctx, c, r, Event{StepID: step.ID, State: "uncertain", At: e.now(), Reason: "observation-unavailable"})
	}
	return e.append(ctx, c, r, Event{StepID: step.ID, State: "observed", At: e.now(), Proof: proof})
}

// Reconcile performs one bounded after-read for the exact dispatched phase. It
// never loads source bytes or calls Dispatch. Original dispatch/uncertainty stays
// in the chain; no repeated read failure appends another durable state row.
func (e Execution) Reconcile(ctx context.Context, c Call) (Record, error) {
	r, err := e.load(ctx, c)
	if err != nil {
		return Record{}, err
	}
	state, err := r.State(c.StepID)
	if err != nil {
		return Record{}, err
	}
	if r.Abandoned() || state == "refused" || state == "observed" || state == "pending" {
		return r, nil
	}
	step, complete, err := r.Current()
	if err != nil {
		return r, err
	}
	if complete || step.ID != c.StepID {
		return r, ErrPhase
	}
	if e.Backend == nil {
		return r, ErrAdmission
	}
	proof, readErr := e.Backend.ObserveAfter(ctx, r.Plan, step)
	if readErr != nil || proof == nil || !proof.matches(r.Plan, step.ID) {
		return r, nil
	}
	return e.append(ctx, c, r, Event{StepID: step.ID, State: "observed", At: e.now(), Proof: proof})
}

func (e Execution) append(ctx context.Context, c Call, r Record, event Event) (Record, error) {
	out, _, err := e.claim(ctx, c, r, event)
	return out, err
}

func (e Execution) claim(ctx context.Context, c Call, r Record, event Event) (Record, bool, error) {
	next, err := r.WithEvent(event)
	if err != nil {
		return r, false, err
	}
	out, err := e.Store.Append(ctx, c.RequestID, r.Revision, next.Events[len(next.Events)-1])
	if err != nil {
		if errors.Is(err, ErrCAS) {
			if current, readErr := e.load(ctx, c); readErr == nil {
				return current, false, nil
			}
		}
		return r, false, ErrJournal
	}
	if out.Validate() != nil || !c.matches(out) {
		return r, false, ErrHistory
	}
	return out, true, nil
}
