package privatechain

import (
	"context"
	"testing"
)

func TestMemberCapturedHTTPReadinessBindsCommittedCapture(t *testing.T) {
	f := newMemberHTTPFixture(t)
	f.s.Version = 3
	b := *f.s.Running.Birth
	f.s.Capture = &b
	r, err := f.collect(context.Background())
	if err != nil || r.Validate(f.p, f.w, f.s) != nil || f.calls != 7 {
		t.Fatal("captured launch readiness refused", err)
	}
	f.s.Capture.Pier.Inode++
	if r.Validate(f.p, f.w, f.s) == nil {
		t.Fatal("readiness accepted substituted durable capture")
	}
}
