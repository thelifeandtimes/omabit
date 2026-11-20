package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCatalogTimeoutsCoverLongRPCAndIdentityHandshakeBounds(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]time.Duration{"app.advance-source.v1": protocol.GrubberyAppTimeout + 15*time.Second, "app.review-source.v1": protocol.GrubberyAppTimeout + 15*time.Second, "desk.stage.v1": protocol.DeskStageTimeout + 15*time.Second, "desk.inspect.v1": protocol.DeskReadTimeout + 15*time.Second, "project.create-advance.v1": 105 * time.Second}
	want["app.review-runtime-recovery.v1"] = protocol.GrubberyAppTimeout + 15*time.Second
	want["app.apply-runtime-recovery.v1"] = protocol.GrubberyAppTimeout + 15*time.Second
	want["project.allocation.v1"] = protocol.ProjectAllocationTimeout + 15*time.Second
	for _, d := range s.Catalog() {
		if d.TimeoutMillis < 1 || d.TimeoutMillis > (16*time.Minute).Milliseconds() {
			t.Fatal("unbounded operation", d.ID)
		}
		if expected, ok := want[d.ID]; ok {
			if d.TimeoutMillis != expected.Milliseconds() {
				t.Fatal("adapter would time out before its bounded RPC", d.ID, d.TimeoutMillis)
			}
			delete(want, d.ID)
		}
	}
	if len(want) != 0 {
		t.Fatal("expected bounded operation missing", want)
	}
}

func TestCoreCallEnforcesCatalogTimeoutAndRetainsUncertaintyClass(t *testing.T) {
	for _, mutates := range []bool{false, true} {
		s, err := New(Options{EnableMutations: true})
		if err != nil {
			t.Fatal(err)
		}
		d := Descriptor{ID: "test.deadline.v1", TimeoutMillis: 20, Effects: []string{"offline-read"}}
		if mutates {
			d.Effects = []string{"journal-write"}
		}
		err = bind(s, d, func(ctx context.Context, _ *Service, _ Empty) (Empty, error) {
			<-ctx.Done()
			return Empty{}, ctx.Err()
		})
		if err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		out := s.Call(t.Context(), d.ID, json.RawMessage(`{}`))
		want := "read-failed"
		if mutates {
			want = "outcome-uncertain"
		}
		if out.Error == nil || out.Error.Dispatch != want || time.Since(started) > time.Second {
			t.Fatal("catalog deadline was ignored or cancellation became safe redispatch", out)
		}
	}
}

func TestCoreLongDeadlinePreservesEarlierCallerDeadlineAndRejectsOverrides(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	err = bind(s, Descriptor{ID: "test.parent.v1", TimeoutMillis: 255000}, func(ctx context.Context, _ *Service, _ Empty) (int64, error) {
		called++
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("core did not supply its declared bound")
		}
		return time.Until(deadline).Milliseconds(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	out := s.Call(ctx, "test.parent.v1", json.RawMessage(`{}`))
	if out.Error != nil || out.Data.(int64) > 1000 || out.Data.(int64) < 1 {
		t.Fatal("declared bound extended caller lifetime", out)
	}
	out = s.Call(t.Context(), "test.parent.v1", json.RawMessage(`{"timeout_ms":999999}`))
	if out.Error == nil || out.Error.Code != "invalid_input" || called != 1 {
		t.Fatal("tool arguments changed the trusted operation bound", out)
	}
	err = bind(s, Descriptor{ID: "test.unbounded.v1", TimeoutMillis: (17 * time.Minute).Milliseconds()}, func(context.Context, *Service, Empty) (Empty, error) { return Empty{}, nil })
	if err == nil {
		t.Fatal("unbounded catalog descriptor accepted")
	}
}
