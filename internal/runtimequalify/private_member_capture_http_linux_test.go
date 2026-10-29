//go:build linux && amd64

package runtimequalify

import (
	"context"
	"testing"
)

func TestPrivateCapturedHTTPDriverPreservesLaunchBinding(t *testing.T) {
	h := privateHTTPFixture(t)
	h.d.initial.Version = 3
	b := *h.d.initial.Running.Birth
	h.d.initial.Capture = &b
	r, err := h.collect(context.Background())
	if err != nil || h.httpCalls != 1 || r.Validate(h.p, h.w, h.d.initial) != nil {
		t.Fatal("captured launch adapter refused", err)
	}
	h.d.initial.Capture.Pier.Inode++
	if r.Validate(h.p, h.w, h.d.initial) == nil {
		t.Fatal("readiness accepted changed capture")
	}
}
