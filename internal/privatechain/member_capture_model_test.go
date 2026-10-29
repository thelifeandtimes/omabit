package privatechain

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type memberCaptureHarness struct {
	*memberLaunchHarness
	captures      int
	changeCapture func(*MemberBirthEvidence)
}

func newMemberCaptureHarness(t *testing.T) (*memberCaptureHarness, MemberLaunch) {
	h, old := newMemberLaunchHarness(t, false)
	s, err := NewCapturedMemberLaunch(h.p, h.w, old.Gateway, old.InstanceID, old.RuntimeID, old.OperationID, old.ContractDigest, old.Peers)
	if err != nil {
		t.Fatal(err)
	}
	h.saved = s.Clone()
	return &memberCaptureHarness{memberLaunchHarness: h}, s
}
func (h *memberCaptureHarness) CaptureBirth(_ context.Context, s MemberLaunch) (MemberBirthEvidence, error) {
	h.selected(s, "started")
	h.captures++
	process := MemberBirthProcess{ObservedAt: h.now(), StartedAt: workspaceNow.Add(3 * time.Second), ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, Namespace: s.Gateway.Observation.Namespace, CompleteContract: true, Running: true}
	e := MemberBirthEvidence{InstanceID: s.InstanceID, OperationID: s.OperationID, RuntimeID: s.RuntimeID, ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, EmptyObservedAt: s.Key.CheckedAt, CapturedAt: h.now(), Before: process, After: process, Pier: pierstore.Identity{Device: 12, Inode: 300, UrbDevice: 12, UrbInode: 301, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}
	if h.changeCapture != nil {
		h.changeCapture(&e)
	}
	return e, h.event("capture")
}
func (h *memberCaptureHarness) ObserveRunning(_ context.Context, s MemberLaunch) (MemberRunningObservation, error) {
	h.selected(s, "captured")
	if s.Capture == nil || h.saved.Capture == nil {
		h.t.Fatal("observation preceded durable capture")
	}
	g := s.Gateway.Observation.Clone()
	g.ObservedAt = h.now()
	b := *s.Capture
	o := MemberRunningObservation{ObservedAt: h.now(), ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, StartedAt: b.After.StartedAt, Namespace: g.Namespace, Gateway: g, Peers: slices.Clone(s.Peers), CompleteContract: true, Running: true, Birth: &b}
	if h.runningChange != nil {
		h.runningChange(&o)
	}
	return o, h.event("observe")
}
func (h *memberCaptureHarness) executeCaptured(ctx context.Context, s MemberLaunch) (MemberLaunch, error) {
	return ExecuteCapturedMemberLaunch(ctx, h.p, h.w, s, h, h.commit, h.now)
}

func TestMemberCaptureModelCommitBeforeLaterObservation(t *testing.T) {
	h, s := newMemberCaptureHarness(t)
	got, err := h.executeCaptured(context.Background(), s)
	if err != nil || got.Version != 3 || got.Phase != "running" || got.Capture == nil || !reflect.DeepEqual(got.Capture, got.Running.Birth) || got.Validate(h.p, h.w) != nil {
		t.Fatal("capture not retained", err)
	}
	if h.artifacts != 1 || h.creates != 1 || h.starts != 1 || h.captures != 1 {
		t.Fatal("repeated effect")
	}
	i := slices.Index(h.events, "commit-captured")
	j := slices.Index(h.events, "observe")
	if i < 0 || j != i+1 {
		t.Fatal("capture commit did not precede observer", h.events)
	}
	raw, err := json.Marshal(got)
	var decoded MemberLaunch
	if err != nil || json.Unmarshal(raw, &decoded) != nil || decoded.Validate(h.p, h.w) != nil || !reflect.DeepEqual(got, decoded) {
		t.Fatal("serialization lost capture")
	}
	copy := got.Clone()
	copy.Capture.Pier.Inode++
	copy.Running.Birth.Pier.UrbInode++
	if got.Capture.Pier.Inode == copy.Capture.Pier.Inode || got.Running.Birth.Pier.UrbInode == copy.Running.Birth.Pier.UrbInode {
		t.Fatal("capture clone aliases")
	}
	if s.Capture != nil {
		t.Fatal("caller mutated")
	}
}

func TestMemberCaptureModelInterruptionAndCommitLoss(t *testing.T) {
	for _, kind := range []string{"capture-error", "capture-cancel", "commit-before", "commit-reply-loss", "commit-cancel", "observation-error", "observation-cancel", "running-commit-loss"} {
		t.Run(kind, func(t *testing.T) {
			h, s := newMemberCaptureHarness(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h.cancel = cancel
			phase := "captured"
			switch kind {
			case "capture-error":
				h.fail = "capture"
				phase = "started"
			case "capture-cancel":
				h.cancelAt = "capture"
				phase = "started"
			case "commit-before":
				h.fail = "commit-captured"
				phase = "started"
			case "commit-reply-loss":
				h.fail = "commit-captured"
				h.lostCommit = true
			case "commit-cancel":
				h.cancelAt = "commit-captured"
			case "observation-error":
				h.fail = "observe"
			case "observation-cancel":
				h.cancelAt = "observe"
			case "running-commit-loss":
				h.fail = "commit-running"
				h.lostCommit = true
				phase = "running"
			}
			_, err := h.executeCaptured(ctx, s)
			if err == nil || strings.Contains(err.Error(), "PRIVATE") || h.saved.Phase != phase || h.saved.Validate(h.p, h.w) != nil {
				t.Fatal("wrong retained outcome", h.saved.Phase, phase, err)
			}
			if phase == "started" {
				if h.saved.Capture != nil {
					t.Fatal("uncommitted capture invented")
				}
			} else if h.saved.Capture == nil {
				t.Fatal("durable capture lost")
			}
			if !strings.HasPrefix(kind, "observation") && kind != "running-commit-loss" && slices.Contains(h.events, "observe") {
				t.Fatal("observed after uncertain capture commit")
			}
			n := len(h.events)
			if _, e := h.executeCaptured(context.Background(), h.saved); e == nil || len(h.events) != n {
				t.Fatal("interrupted launch replayed")
			}
			if h.starts != 1 || h.captures != 1 {
				t.Fatal("duplicate Start/capture")
			}
		})
	}
}

func TestMemberCaptureModelRejectsInvalidCaptureBeforeCommit(t *testing.T) {
	for _, kind := range []string{"instance", "container", "process-change", "namespace", "pier-alias", "stale", "future", "before-start", "incomplete"} {
		t.Run(kind, func(t *testing.T) {
			h, s := newMemberCaptureHarness(t)
			h.changeCapture = func(b *MemberBirthEvidence) {
				switch kind {
				case "instance":
					b.InstanceID = "other"
				case "container":
					b.ContainerID = strings.Repeat("f", 64)
				case "process-change":
					b.After.StartedAt = b.After.StartedAt.Add(time.Second)
				case "namespace":
					b.After.Namespace.Inode++
				case "pier-alias":
					b.Pier.Inode = h.saved.Artifact.Inode
					b.Pier.Birth = h.saved.Artifact.Birth
				case "stale":
					b.CapturedAt = h.now().Add(-31 * time.Second)
				case "future":
					b.CapturedAt = h.now().Add(6 * time.Second)
				case "before-start":
					b.Before.StartedAt = b.EmptyObservedAt.Add(-time.Second)
				case "incomplete":
					b.After.CompleteContract = false
				}
			}
			if _, err := h.executeCaptured(context.Background(), s); err == nil || h.saved.Phase != "started" || h.saved.Capture != nil || slices.Contains(h.events, "commit-captured") || slices.Contains(h.events, "observe") {
				t.Fatal("invalid capture accepted", err)
			}
		})
	}
}

func TestMemberCaptureModelRefusesSubstitutionAndPhaseSkipping(t *testing.T) {
	h, s := newMemberCaptureHarness(t)
	h.fail = "observe"
	if _, err := h.executeCaptured(context.Background(), s); err == nil {
		t.Fatal("fixture should stop after capture")
	}
	captured := h.saved.Clone()
	h.fail = ""
	running, err := h.ObserveRunning(context.Background(), captured)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"missing", "early", "legacy1", "legacy2", "wrong-process", "different-running-pier", "skip-capture"} {
		t.Run(kind, func(t *testing.T) {
			v := captured.Clone()
			switch kind {
			case "missing":
				v.Capture = nil
			case "early":
				v.Phase = "started"
			case "legacy1":
				v.Version = 1
			case "legacy2":
				v.Version = 2
			case "wrong-process":
				v.Capture.After.StartedAt = v.Capture.After.StartedAt.Add(time.Second)
			case "different-running-pier":
				changed := running
				birth := *running.Birth
				birth.Pier.Inode++
				changed.Birth = &birth
				if _, e := v.Advance(h.p, h.w, "captured", "running", MemberLaunchResult{Running: &changed}); e == nil {
					t.Fatal("capture substituted")
				}
				return
			case "skip-capture":
				v.Phase = "started"
				v.Capture = nil
				if _, e := v.Advance(h.p, h.w, "started", "running", MemberLaunchResult{Running: &running}); e == nil {
					t.Fatal("capture skipped")
				}
				return
			}
			if v.Validate(h.p, h.w) == nil {
				t.Fatal("invalid captured state accepted")
			}
		})
	}
	// A callback mutating its own commit argument must not alter the controller
	// or the independently retained journal.
	h2, s2 := newMemberCaptureHarness(t)
	commit := func(expected string, next MemberLaunch) error {
		err := h2.commit(expected, next)
		if next.Capture != nil {
			next.Capture.Pier.Inode++
		}
		return err
	}
	if _, err := ExecuteCapturedMemberLaunch(context.Background(), h2.p, h2.w, s2, h2, commit, h2.now); err != nil {
		t.Fatal("callback alias escaped", err)
	}
}

func TestMemberCaptureModelKeepsControllerVersionsSeparate(t *testing.T) {
	h, s := newMemberCaptureHarness(t)
	if _, err := ExecuteMemberLaunch(context.Background(), h.p, h.w, s, h, h.commit, h.now); !errors.Is(err, ErrMemberLaunch) || len(h.events) != 0 {
		t.Fatal("legacy controller accepted successor")
	}
	s.Version = 2
	h.saved = s.Clone()
	if _, err := h.executeCaptured(context.Background(), s); !errors.Is(err, ErrMemberLaunch) || len(h.events) != 0 {
		t.Fatal("successor controller executed legacy intent")
	}
	raw, _ := json.Marshal(s)
	if strings.Contains(string(raw), `"capture"`) {
		t.Fatal("legacy serialization gained capture")
	}
}
