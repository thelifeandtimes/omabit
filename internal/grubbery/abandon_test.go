package grubbery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

type abandonBackend struct {
	*phaseBackend
	checks  atomic.Int32
	changed bool
}

func (b *abandonBackend) ObserveAbandon(_ context.Context, r grubbery.Record) (*grubbery.AbandonProof, error) {
	b.checks.Add(1)
	previous := r.HeaderDigest
	if len(r.Events) > 0 {
		previous = r.Events[len(r.Events)-1].Digest
	}
	proof := &grubbery.AbandonProof{Version: 1, PlanDigest: r.Plan.Digest, BindingDigest: hashValue(r.Plan.Binding), Revision: r.Revision, PreviousDigest: previous, Epoch: r.Plan.Before.Epoch, ReleaseRoot: r.Plan.Before.ReleaseRoot, NativeEvidenceDigest: hashValue("unchanged-native-baseline")}
	if b.changed {
		proof.Epoch++
	}
	return proof, nil
}
func TestAbandonReservedIntentNeverAcquiresRuntimeOrSourceAuthority(t *testing.T) {
	e, c, p, j, b, loads := execution(t)
	ctx := context.Background()
	e.Backend = nil
	e.LoadSource = nil
	calls := b.admissions.Load()
	reads := loads.Load()
	r, err := e.Abandon(ctx, c)
	if err != nil || !r.Abandoned() {
		t.Fatal(r, err)
	}
	if terminal, err := r.Terminal(); !terminal || err != nil {
		t.Fatal("abandoned lease is not terminal", err)
	}
	if _, done, err := r.Current(); done || !errors.Is(err, grubbery.ErrAbandoned) {
		t.Fatal("abandonment became source-followed completion", err)
	}
	if b.admissions.Load() != calls || b.dispatches.Load() != 0 || loads.Load() != reads {
		t.Fatal("no-effect cancellation probed/dispatched")
	}
	for _, f := range []func(context.Context, grubbery.Call) (grubbery.Record, error){e.Abandon, e.Advance, e.Reconcile} {
		got, err := f(ctx, c)
		if err != nil || !reflect.DeepEqual(got, r) {
			t.Fatal("terminal replay changed", err)
		}
	}
	got, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || got.Epoch != 0 || got.PendingRequestID != "" || got.SourceOrigin != nil {
		t.Fatal(got, err)
	}
	next := p
	next.RequestID = strings.Repeat("3", 64) // Rebuild the immutable digest; never edit a saved plan.
	next, err = grubbery.NewPlan(p.Binding, p.StageID, next.RequestID, p.Source.Manifest, b.files, p.Before)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := grubbery.NewRecord(next, r.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = j.Create(ctx, replacement); err != nil {
		t.Fatal("safe abandonment retained namespace lease", err)
	}
}
func TestFailedCompileAbandonsOnlyAfterAllOriginalWritesAndMarkersObserved(t *testing.T) {
	e, c, p, _, b, _ := execution(t)
	ctx := context.Background()
	backend := &abandonBackend{phaseBackend: b}
	e.Backend = backend
	var r grubbery.Record
	for _, step := range p.Steps {
		c.StepID = step.ID
		if step.Kind == "observe-code" {
			b.hide.Store(true)
			var err error
			r, err = e.Advance(ctx, c)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		var err error
		r, err = e.Advance(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
	}
	if state, _ := r.State(c.StepID); state != "uncertain" {
		t.Fatal("compile failure not retained")
	}
	before := copyRecord(r)
	dispatches := b.dispatches.Load()
	e.LoadSource = nil
	backend.changed = true
	if _, err := e.Abandon(ctx, c); !errors.Is(err, grubbery.ErrAdmission) {
		t.Fatal("changed live source accepted", err)
	}
	backend.changed = false
	r, err := e.Abandon(ctx, c)
	if err != nil || !r.Abandoned() {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Events, r.Events[:len(before.Events)]) || b.dispatches.Load() != dispatches {
		t.Fatal("abandonment rewrote history or sent native effect")
	}
	own, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || own.SourceOrigin == nil || own.DeskOrigin != nil || own.PendingRequestID != "" || own.Epoch != 0 || own.ReleaseRoot != "" {
		t.Fatal("abandonment changed live app or lost original marker", own, err)
	}
	calls := backend.checks.Load()
	e.Backend = nil
	if _, err := e.Abandon(ctx, c); err != nil || backend.checks.Load() != calls {
		t.Fatal("replay required live evidence", err)
	}
	// The next attempt uses the retained marker and unchanged live epoch, with a
	// new immutable request/release. Failed code is preserved outside live data.
	before2 := p.Before
	before2.SourceState = "owned"
	before2.SourceOwnerSHA256 = own.OwnerSHA256
	next, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("4", 64), p.Source.Manifest, b.files, before2)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := grubbery.NewRecord(next, r.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := grubbery.DeriveOwnership([]grubbery.Record{r, r2}); err != nil {
		t.Fatal("failed compile prevented safe retry", err)
	}
}
func TestAbandonNeverResolvesUnknownWriteOrDispatchedActivation(t *testing.T) {
	for _, kind := range []string{"first-write", "marker", "activation"} {
		t.Run(kind, func(t *testing.T) {
			e, c, p, j, b, _ := execution(t)
			ctx := context.Background()
			backend := &abandonBackend{phaseBackend: b}
			e.Backend = backend
			for _, step := range p.Steps {
				c.StepID = step.ID
				stop := kind == "first-write" || kind == "marker" && step.Kind == "make-file" && strings.HasSuffix(step.Path, "/omabit-owner.json") || kind == "activation" && step.Kind == "poke-source"
				if stop {
					b.lost.Store(true)
				}
				if _, err := e.Advance(ctx, c); err != nil {
					t.Fatal(err)
				}
				if stop {
					break
				}
			}
			before, _ := j.Load(ctx, c.RequestID)
			calls := b.dispatches.Load()
			if _, err := e.Abandon(ctx, c); !errors.Is(err, grubbery.ErrAdmission) {
				t.Fatal("unsafe abandonment admitted", err)
			}
			after, _ := j.Load(ctx, c.RequestID)
			if !reflect.DeepEqual(before, after) || b.dispatches.Load() != calls || backend.checks.Load() != 0 {
				t.Fatal("unsafe abandonment changed/probed state")
			}
		})
	}
}
func TestAbandonCASPreventsStaleWorkerDispatch(t *testing.T) {
	e, c, _, _, b, _ := execution(t)
	ctx := context.Background()
	b.before = func() {
		b.before = nil
		if _, err := e.Abandon(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	r, err := e.Advance(ctx, c)
	if err != nil || !r.Abandoned() || b.dispatches.Load() != 0 {
		t.Fatal("stale admission dispatched after abandonment", err)
	}
}
