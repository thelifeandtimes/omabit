package grubbery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func TestOwnershipRequiresObservedOriginalMarkersAndEveryPredecessor(t *testing.T) {
	e, c, p, _, b, _ := execution(t)
	ctx := context.Background()
	r, err := e.Lookup(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || reserved.SourceOrigin != nil || reserved.DeskOrigin != nil || reserved.Epoch != 0 || reserved.PendingRequestID != p.RequestID {
		t.Fatal("reservation invented native ownership", reserved, err)
	}
	for _, step := range p.Steps {
		c.StepID = step.ID
		r, err = e.Advance(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
	}
	first, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || first.SourceOrigin == nil || first.DeskOrigin == nil || first.Epoch != 1 || first.PendingRequestID != "" || first.SourceDigest != p.Source.Manifest.Digest {
		t.Fatal(first, err)
	}
	before := grubbery.Before{SnapshotDigest: strings.Repeat("f", 64), SourceState: "owned", SourceOwnerSHA256: first.OwnerSHA256, DeskState: "owned", DeskOwnerSHA256: first.OwnerSHA256, Epoch: first.Epoch, ReleaseRoot: first.ReleaseRoot}
	next, err := grubbery.NewPlan(p.Binding, p.StageID, strings.Repeat("2", 64), p.Source.Manifest, b.files, before)
	if err != nil {
		t.Fatal(err)
	}
	c.RequestID = next.RequestID
	c.PlanDigest = next.Digest
	c.StepID = next.Steps[0].ID
	r2, err := e.Begin(ctx, c, next)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := grubbery.DeriveOwnership([]grubbery.Record{r2}); err == nil {
		t.Fatal("missing original ownership/completion was accepted")
	}
	current, err := grubbery.DeriveOwnership([]grubbery.Record{r, r2})
	if err != nil || !reflect.DeepEqual(current.SourceOrigin, first.SourceOrigin) || !reflect.DeepEqual(current.DeskOrigin, first.DeskOrigin) || current.AppliedRequestID != p.RequestID || current.PendingRequestID != next.RequestID {
		t.Fatal("new reservation replaced original state/ownership", current, err)
	}
	for _, step := range next.Steps {
		c.StepID = step.ID
		r2, err = e.Advance(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
	}
	updated, err := grubbery.DeriveOwnership([]grubbery.Record{r, r2})
	if err != nil || updated.Epoch != 2 || updated.AppliedRequestID != next.RequestID || !reflect.DeepEqual(updated.SourceOrigin, first.SourceOrigin) || !reflect.DeepEqual(updated.DeskOrigin, first.DeskOrigin) {
		t.Fatal("update changed origin or lost version ancestry", updated, err)
	}
	if _, err := grubbery.DeriveOwnership([]grubbery.Record{r2, r}); err == nil {
		t.Fatal("reordered native ownership history accepted")
	}
	if _, err := grubbery.DeriveOwnership([]grubbery.Record{r, r}); err == nil {
		t.Fatal("duplicate app operation accepted")
	}
	if _, err := grubbery.DeriveOwnership(nil); !errors.Is(err, grubbery.ErrMissing) {
		t.Fatal("absence invented ownership")
	}
}

func TestUncertainMarkerNeverBecomesOwnership(t *testing.T) {
	e, c, p, _, b, _ := execution(t)
	ctx := context.Background()
	if _, err := e.Advance(ctx, c); err != nil {
		t.Fatal(err)
	}
	c.StepID = p.Steps[1].ID
	b.lost.Store(true)
	r, err := e.Advance(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || owned.SourceOrigin != nil {
		t.Fatal("uncertain marker became ownership", err)
	}
	r, err = e.Reconcile(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	owned, err = grubbery.DeriveOwnership([]grubbery.Record{r})
	if err != nil || owned.SourceOrigin == nil || owned.DeskOrigin != nil || owned.Epoch != 0 || owned.PendingRequestID != p.RequestID {
		t.Fatal("read-only marker recovery lost partial ancestry", owned, err)
	}
}
