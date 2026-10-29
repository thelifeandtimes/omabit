//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func captureDriverHarness(t *testing.T) *privateLaunchHarness {
	h := newPrivateLaunchHarness(t)
	h.d.initial.Version = 3
	h.saved = h.d.initial.Clone()
	return h
}

func runCaptureDriver(h *privateLaunchHarness, ctx context.Context, after func(privatechain.MemberLaunch) error) (privatechain.MemberLaunch, error) {
	session := &privateMemberSessionDriver{MemberLaunchDriver: h.d}
	return privatechain.ExecuteCapturedMemberLaunch(ctx, h.d.contract.selection.Plan, h.d.contract.selection.Workspace, h.d.initial, session, func(expected string, next privatechain.MemberLaunch) error {
		if expected != h.saved.Phase {
			h.t.Fatal("wrong journal ancestry")
		}
		if err := h.event("commit-" + next.Phase); err != nil {
			return err
		}
		h.saved = next.Clone()
		if after != nil {
			return after(next.Clone())
		}
		return nil
	}, h.d.now)
}

func countCaptureEvent(h *privateLaunchHarness, name string) int {
	n := 0
	for _, event := range h.events {
		if event == name {
			n++
		}
	}
	return n
}

func TestPrivateCapturedDriverCommitsBeforeGatewayAndForwardsSession(t *testing.T) {
	h := captureDriverHarness(t)
	gateway := h.d.ops.gateway
	h.d.ops.gateway = func(ctx context.Context, m []privateGatewayMember, p []privatechain.GatewayMemberPier) (privatechain.GatewayObservation, error) {
		if h.d.startAcknowledged && (h.saved.Phase != "captured" || h.saved.Capture == nil) {
			t.Fatal("post-Start gateway observation preceded durable capture")
		}
		return gateway(ctx, m, p)
	}
	got, err := runCaptureDriver(h, context.Background(), nil)
	if err != nil || got.Phase != "running" || got.Capture == nil || !reflect.DeepEqual(got.Capture, got.Running.Birth) || h.starts != 1 || h.creates != 1 || countCaptureEvent(h, "capture") != 1 || countCaptureEvent(h, "wait-pier") != 1 {
		t.Fatal("incomplete captured launch", err, got.Phase)
	}
	got.Capture.Pier.Inode++
	if reflect.DeepEqual(got.Capture, h.d.birthEvidence) || !reflect.DeepEqual(h.saved.Capture, h.d.birthEvidence) {
		t.Fatal("returned capture aliases retained identity")
	}
}

func TestPrivateCapturedDriverFailureDoesNotRepeatCaptureOrStart(t *testing.T) {
	for _, at := range []string{"wait-pier", "capture", "commit-captured", "birth-held", "commit-running"} {
		t.Run(at, func(t *testing.T) {
			h := captureDriverHarness(t)
			h.fail = at
			got, err := runCaptureDriver(h, context.Background(), nil)
			phase := "started"
			if at == "birth-held" || at == "commit-running" {
				phase = "captured"
			}
			if err == nil || strings.Contains(err.Error(), "PRIVATE") || got.Phase != phase || h.saved.Phase != phase || h.starts != 1 || h.creates != 1 {
				t.Fatal("wrong uncertainty boundary", err, got.Phase, h.saved.Phase)
			}
			before := len(h.events)
			if _, err = h.d.CaptureBirth(context.Background(), got); err == nil {
				t.Fatal("capture repeated")
			}
			if _, err = privatechain.ExecuteCapturedMemberLaunch(context.Background(), h.d.contract.selection.Plan, h.d.contract.selection.Workspace, got, h.d, func(string, privatechain.MemberLaunch) error { t.Fatal("interrupted launch committed"); return nil }, h.d.now); err == nil {
				t.Fatal("first boot repeated")
			}
			if len(h.events) != before {
				t.Fatal("refusal touched runtime")
			}
		})
	}
}

func TestPrivateCapturedDriverLostCommitReplyKeepsDurableIdentity(t *testing.T) {
	h := captureDriverHarness(t)
	got, err := runCaptureDriver(h, context.Background(), func(s privatechain.MemberLaunch) error {
		if s.Phase == "captured" {
			return errors.New("lost commit reply")
		}
		return nil
	})
	if err == nil || got.Phase != "started" || h.saved.Phase != "captured" || h.saved.Capture == nil || h.d.observeAttempted || h.starts != 1 {
		t.Fatal("lost reply promoted or capture lost")
	}
	before := len(h.events)
	for _, kind := range []string{"missing", "pier", "process", "version", "closed"} {
		t.Run(kind, func(t *testing.T) {
			s := h.saved.Clone()
			switch kind {
			case "missing":
				s.Phase = "started"
				s.Capture = nil
			case "pier":
				s.Capture.Pier.Inode += 100
			case "process":
				s.Capture.Before.StartedAt = s.Capture.Before.StartedAt.Add(time.Millisecond)
				s.Capture.After.StartedAt = s.Capture.Before.StartedAt
			case "version":
				s.Version = 2
				s.Phase = "started"
				s.Capture = nil
			case "closed":
				h.d.Close()
			}
			if _, err := h.d.ObserveRunning(context.Background(), s); err == nil {
				t.Fatal("substituted capture observed")
			}
		})
	}
	if len(h.events) != before {
		t.Fatal("invalid capture reached runtime")
	}
}

func TestPrivateCapturedDriverRetainsCaptureOnLaterFailure(t *testing.T) {
	for _, kind := range []string{"gateway", "process", "held-pier", "cancel", "closed"} {
		t.Run(kind, func(t *testing.T) {
			h := captureDriverHarness(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			got, err := runCaptureDriver(h, ctx, func(s privatechain.MemberLaunch) error {
				if s.Phase != "captured" {
					return nil
				}
				switch kind {
				case "gateway":
					h.fail = "gateway"
				case "process":
					h.mutateProcess = func(p *privatechain.MemberBirthProcess) { p.StartedAt = p.StartedAt.Add(time.Millisecond) }
				case "held-pier":
					h.fail = "birth-held"
				case "cancel":
					cancel()
				case "closed":
					h.d.Close()
				}
				return nil
			})
			if err == nil || got.Phase != "captured" || got.Capture == nil || got.Running != nil || !reflect.DeepEqual(got, h.saved) || countCaptureEvent(h, "capture") != 1 || h.starts != 1 {
				t.Fatal("later failure lost capture", err, got.Phase)
			}
		})
	}
}

type captureHiddenDriver struct {
	privatechain.MemberLaunchDriver
}

func TestPrivateCapturedSessionRefusesMissingCapability(t *testing.T) {
	h := captureDriverHarness(t)
	session := &privateMemberSessionDriver{MemberLaunchDriver: &captureHiddenDriver{MemberLaunchDriver: h.d}}
	if _, err := session.CaptureBirth(context.Background(), h.saved); err == nil || len(h.events) != 0 {
		t.Fatal("missing capability was invoked")
	}
}
