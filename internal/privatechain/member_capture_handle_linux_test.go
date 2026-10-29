//go:build linux

package privatechain

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestMemberCapturedLaunchUsesOriginalHeldBirthHandle(t *testing.T) {
	f := newMemberBirthFixture(t)
	mustInput(t, f.v.Close())
	f.initial.Version = 3
	f.dispatched.Version = 3
	f.started.Version = 3
	var err error
	f.v, err = openMemberBirthCapture(context.Background(), f.directory, f.p, f.w, f.initial, func() time.Time { return f.now })
	mustInput(t, err)
	f.boot(t)
	before := workspaceFiles(t, f.directory)
	e, err := f.v.Capture(context.Background(), f.started, f.observe)
	mustInput(t, err)
	captured, err := f.started.Advance(f.p, f.w, "started", "captured", MemberLaunchResult{Capture: &e})
	mustInput(t, err)
	mustInput(t, captured.Validate(f.p, f.w))
	mustInput(t, f.v.Verify(context.Background()))
	if f.calls != 2 || !reflect.DeepEqual(before, workspaceFiles(t, f.directory)) {
		t.Fatal("capture changed filesystem or process reads")
	}
	if _, err = f.v.Capture(context.Background(), f.started, f.observe); err == nil {
		t.Fatal("capture repeated")
	}
	if next, err := OpenMemberBirthCapture(context.Background(), f.directory, f.p, f.w, captured); err == nil {
		next.Close()
		t.Fatal("reopened first-boot handle after capture")
	}
}
