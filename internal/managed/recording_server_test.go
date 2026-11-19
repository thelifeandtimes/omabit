package managed

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type recordingSelectionBackend struct {
	fakeBackend
	configured atomic.Bool
	tooEarly   atomic.Bool
	selected   chan string
}

func (b *recordingSelectionBackend) Observe(ctx context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if !b.configured.Load() {
		b.tooEarly.Store(true)
	}
	return b.fakeBackend.Observe(ctx, state)
}

func (b *recordingSelectionBackend) ApplyWithRecording(ctx context.Context, action string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error, lifecycle runtimequalify.ManagedRecordingLifecycle) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if ctx.Err() != nil || lifecycle.BeforeStart == nil || lifecycle.BeforeRemove == nil {
		return state, runtimequalify.ManagedObservation{}, errors.New("invalid service-owned recording selection")
	}
	b.selected <- action
	return b.fakeBackend.Apply(ctx, action, state, persist)
}

func TestRecordingServerSelectsActualJournalOperationBeforeWorkers(t *testing.T) {
	b := &recordingSelectionBackend{selected: make(chan string, 2)}
	actualOps := make(chan protocol.Operation, 2)
	r, err := newServerConfigured(privateDir(t), b, time.Hour, true, "", func(r *Server) error {
		r.recording = func(op protocol.Operation) runtimequalify.ManagedRecordingLifecycle {
			actualOps <- op
			return runtimequalify.ManagedRecordingLifecycle{
				BeforeStart: func(context.Context, runtimequalify.ManagedState, string) (func(context.Context) error, error) {
					return func(context.Context) error { return nil }, nil
				},
				BeforeRemove: func(context.Context, runtimequalify.ManagedState, string) error { return nil },
			}
		}
		b.configured.Store(true)
		return nil
	})
	g02Check(t, err)
	defer r.Close()
	_, _, err = r.s.register("original-actor", registration("one", "register", 18087))
	g02Check(t, err)
	for _, action := range []string{"start", "stop"} {
		op, _, err := r.s.accept("original-actor", "one", protocol.OperationRequest{Kind: action, RequestID: action, ExpectedRevision: 1})
		g02Check(t, err)
		queued := op
		queued.Actor, queued.Kind, queued.RequestID = "untrusted-caller-copy", "pack", "not-the-journal"
		r.mu.Lock()
		r.launch(queued)
		r.mu.Unlock()
		select {
		case selected := <-actualOps:
			if selected.ID != op.ID || selected.Actor != op.Actor || selected.RequestID != op.RequestID || selected.Kind != action || selected.Status != "running" {
				t.Fatal("recording selected a caller/queued copy", selected)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("actual recording operation was not selected")
		}
		select {
		case got := <-b.selected:
			if got != action {
				t.Fatal("wrong recording backend action", got)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("recording backend was bypassed")
		}
		deadline := time.Now().Add(3 * time.Second)
		for {
			saved, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if saved.Status == "succeeded" {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("selected operation did not finish", saved)
			}
			time.Sleep(time.Millisecond)
		}
	}
	if b.tooEarly.Load() {
		t.Fatal("observation ran before recording configuration")
	}
	req := httptest.NewRequest("GET", "/hello", nil)
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "runtime-recording-pilot-v1") {
		t.Fatal("selected pilot is not distinguishable", w.Body)
	}
}

func TestRecordingServerConfigurationFailureStartsNoWorkersAndClosesStore(t *testing.T) {
	dir := privateDir(t)
	b := &recordingSelectionBackend{}
	sentinel := errors.New("pilot configuration refused")
	r, err := newServerConfigured(dir, b, time.Hour, true, "", func(*Server) error { return sentinel })
	if r != nil || err != sentinel || b.tooEarly.Load() {
		t.Fatal("failed configuration started service", err)
	}
	s, err := openStore(dir)
	g02Check(t, err)
	g02Check(t, s.close())
}
