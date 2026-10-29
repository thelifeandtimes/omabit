//go:build linux && amd64

package managed

import (
	"context"
	"reflect"
	"testing"
)

func TestPrivateCaptureQueuePersistsBeforeLaterObservation(t *testing.T) {
	for _, failure := range []string{"", "capture-observe"} {
		t.Run("later-failure="+failure, func(t *testing.T) {
			r, b, admit := privateLifecycleQueueFixture(t, "member-launch")
			b.fail = failure
			close(b.release)
			op, hit, err := admit("capture-actor", "captured-queue", false)
			g02Check(t, err)
			if hit {
				t.Fatal("new launch replayed")
			}
			waitProjectWorker(t, r)
			j, err := loadPrivateMemberLaunch(r.s.db, op.InstanceID)
			g02Check(t, err)
			got, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			phase, status := "running", "succeeded"
			if failure != "" {
				phase, status = "captured", "interrupted"
			}
			if j.Version != 3 || j.Phase != phase || j.Capture == nil || got.Status != status || b.closed.Load() != 1 {
				t.Fatal("wrong capture outcome", j.Phase, got.Status)
			}
			r.b = &fakeBackend{}
			replay, hit, err := admit("capture-actor", "captured-queue", true)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(replay, got) {
				t.Fatal("lost actor receipt without backend")
			}
		})
	}
}

func TestPrivateCaptureQueueReplaysLegacyWithoutUpgradingLaunch(t *testing.T) {
	s, p, instance, req := privateMemberStoreFixture(t)
	op, _, err := s.acceptPrivateMemberLaunch("legacy-owner", p.Binding.ProjectID, instance, req, privateMemberStoreContract)
	g02Check(t, err)
	g02Check(t, s.runPrivateMemberLaunch(context.Background(), p.Binding.ProjectID, instance, op.ID, &privateMemberStoreDriver{t: t, s: s}))
	before, err := loadPrivateMemberLaunch(s.db, instance)
	g02Check(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &Server{s: s, b: &fakeBackend{}, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1)}
	got, hit, err := r.queuePrivateMemberLaunch("legacy-owner", p.Binding.ProjectID, instance, req)
	g02Check(t, err)
	after, err := loadPrivateMemberLaunch(s.db, instance)
	g02Check(t, err)
	if !hit || got.ID != op.ID || got.Status != "succeeded" || after.Version != 2 || after.Capture != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("historical replay required backend or rewrote capture")
	}
}
