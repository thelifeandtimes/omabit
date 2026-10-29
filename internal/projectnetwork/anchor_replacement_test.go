package projectnetwork

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

type anchorReplacementFixture struct {
	t               *testing.T
	h               AnchorRecoveryHistory
	saved           AnchorReplacement
	state           AnchorRetirementObservation
	now             time.Time
	creates, starts int
	idBase          int
	mode            string
	mutate          func(string, *AnchorRetirementObservation)
	cancel          context.CancelFunc
}

func newAnchorReplacementFixture(t *testing.T, stopped bool) *anchorReplacementFixture {
	t.Helper()
	r := newAnchorRetirementFixture(t, stopped)
	retired := r.saved.Clone()
	for !retired.Complete() {
		var err error
		retired, err = ExecuteNextAnchorRetirement(context.Background(), r.history, retired, r, r.commit, r.clock)
		if err != nil {
			t.Fatal(err)
		}
	}
	var members []RecoveredMember
	for i, m := range retired.Members {
		c, err := SelectRecoveryMemberCreation(r.history, retired, m.InstanceID, "replace-"+m.InstanceID)
		if err != nil {
			t.Fatal(err)
		}
		members = append(members, RecoveredMember{Creation: c, Container: ResourceIdentity{ContractDigest: strings.Repeat(string(rune('a'+i)), 64)}, Phase: "planned"})
	}
	s, err := NewAnchorReplacement(r.history, retired, "replace-members", members)
	if err != nil {
		t.Fatal(err)
	}
	return &anchorReplacementFixture{t: t, h: r.history, saved: s, state: cloneAnchorRetirementObservation(*retired.Members[len(retired.Members)-1].After), now: r.now}
}
func (f *anchorReplacementFixture) clock() time.Time { return f.now }
func (f *anchorReplacementFixture) VerifyEngine(ctx context.Context, id string) error {
	if id != f.saved.Retirement.Namespace.Drain.Review.Context.Binding.EngineID {
		f.t.Fatal("wrong engine")
	}
	if f.mode == "engine" || f.mode == "late-engine" && f.starts > 0 {
		return errors.New("private engine failure")
	}
	return ctx.Err()
}
func (f *anchorReplacementFixture) InspectReplacement(_ context.Context, s AnchorReplacement, index int, stage string) (AnchorRetirementObservation, error) {
	if !reflect.DeepEqual(s, f.saved) || index != s.next() {
		f.t.Fatal("changed inspection selection")
	}
	f.now = f.now.Add(time.Second)
	o := cloneAnchorRetirementObservation(f.state)
	o.CapturedAt = f.now
	o.Runtime.ObservedAt = f.now
	if f.mutate != nil {
		f.mutate(stage, &o)
	}
	if f.mode == "driver-alias" {
		s.Members[0].Creation.Storage.Path = "changed"
		s.Retirement.Namespace.Drain.Members[0].InstanceID = "changed"
	}
	if f.mode == "inspect-"+stage {
		return o, errors.New("private inspection error")
	}
	return o, nil
}
func (f *anchorReplacementFixture) CreateReplacement(ctx context.Context, s AnchorReplacement, index int) (string, error) {
	if !reflect.DeepEqual(s, f.saved) || s.Members[index].Phase != "create-dispatched" {
		f.t.Fatal("Create before exact intent")
	}
	o, err := f.InspectReplacement(ctx, s.Clone(), index, "absent")
	if err != nil || s.ValidateObservation(f.h, index, o, "absent", f.now) != nil {
		return "", errors.New("Create recheck refused")
	}
	f.creates++
	if f.mode == "create-error" {
		return "", errors.New("private Create failure")
	}
	base := f.idBase
	if base == 0 {
		base = 100
	}
	id := fmt.Sprintf("%064x", base+index)
	m := s.Members[index]
	container := ResourceIdentity{ID: id, ContractDigest: m.Container.ContractDigest}
	f.state.Runtime.Members = append(f.state.Runtime.Members, MemberObservation{InstanceID: m.Creation.InstanceID, ResourceObservation: ResourceObservation{Resource: container, Present: true}, AnchorID: f.state.Runtime.Anchor.Resource.ID})
	f.state.Processes = append(f.state.Processes, AnchorDrainProcess{InstanceID: m.Creation.InstanceID, Storage: m.Creation.Storage})
	switch f.mode {
	case "create-lost-id":
		return "", errors.New("private lost reply")
	case "create-returned-id-error":
		return id, errors.New("private error after returned ID")
	case "create-old-id":
		return s.Retirement.Namespace.Drain.Review.Context.Binding.Members[0].Container.ID, nil
	}
	return id, nil
}
func (f *anchorReplacementFixture) StartReplacement(ctx context.Context, s AnchorReplacement, index int) error {
	if !reflect.DeepEqual(s, f.saved) || s.Members[index].Phase != "start-dispatched" {
		f.t.Fatal("Start before exact intent")
	}
	o, err := f.InspectReplacement(ctx, s.Clone(), index, "created")
	if err != nil || s.ValidateObservation(f.h, index, o, "created", f.now) != nil {
		return errors.New("Start recheck refused")
	}
	f.starts++
	if f.mode == "start-error" {
		return errors.New("private Start failure")
	}
	if f.mode != "false-start-ack" {
		m := s.Members[index]
		for i, p := range f.state.Processes {
			if p.InstanceID == m.Creation.InstanceID {
				f.state.Processes[i].Process.StartedAt = f.now
			}
		}
		for i, p := range f.state.Runtime.Members {
			if p.InstanceID == m.Creation.InstanceID {
				f.state.Runtime.Members[i].Running = true
				f.state.Runtime.Members[i].Namespace = m.Creation.Namespace
			}
		}
	}
	if f.mode == "start-lost-reply" {
		return errors.New("private lost reply")
	}
	if f.mode == "cancel-start" {
		f.cancel()
	}
	return nil
}
func (f *anchorReplacementFixture) commit(seq uint64, next AnchorReplacement) error {
	if seq != f.saved.Sequence || next.Sequence != seq+1 || next.Validate(f.h) != nil {
		f.t.Fatal("stale or invalid commit")
	}
	index := f.saved.next()
	phase := next.Members[index].Phase
	if f.mode == "commit-"+phase {
		return errors.New("commit failed")
	}
	f.saved = next.Clone()
	if f.mode == "commit-alias" {
		next.Members[0].Creation.Storage.Path = "changed"
		next.Retirement.Namespace.Drain.Members[0].InstanceID = "changed"
	}
	if phase == "create-dispatched" && f.mode == "stale-create" || phase == "start-dispatched" && f.mode == "stale-start" {
		f.now = f.now.Add(time.Minute)
	}
	return nil
}
func TestAnchorReplacementSerialPreservesHistoryAndPiers(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, mode := range []string{"", "driver-alias", "commit-alias"} {
			f := newAnchorReplacementFixture(t, stopped)
			f.mode = mode
			s := f.saved.Clone()
			original := s.Retirement.Clone()
			history := f.h.Clone()
			for !s.Running() {
				var err error
				s, err = ExecuteNextAnchorReplacement(context.Background(), f.h, s, f, f.commit, f.clock)
				if err != nil {
					t.Fatal("replacement failed", err)
				}
			}
			if s.Validate(f.h) != nil || f.creates != len(s.Members) || f.starts != len(s.Members) || !reflect.DeepEqual(s.Retirement, original) || !reflect.DeepEqual(f.h, history) {
				t.Fatal("changed authority or incorrect dispatch count")
			}
			last := s.Members[len(s.Members)-1].After
			if !reflect.DeepEqual(last.Piers, original.Members[len(original.Members)-1].After.Piers) || len(last.Processes) != len(s.Members) {
				t.Fatal("replacement lost pier/process evidence")
			}
			if _, err := ExecuteNextAnchorReplacement(context.Background(), f.h, s, f, f.commit, f.clock); err == nil {
				t.Fatal("completed creation replayed")
			}
			if _, err := ObserveAnchorReplacementRunning(context.Background(), f.h, s, f, f.commit, f.clock); err == nil {
				t.Fatal("completed intent re-observed")
			}
			s.Members[0].After.Piers[0].Storage.Path = "changed"
			if f.saved.Validate(f.h) != nil {
				t.Fatal("result aliases committed history")
			}
		}
	}
}
func TestAnchorReplacementUncertaintyNeverReplays(t *testing.T) {
	for _, mode := range []string{"engine", "inspect-absent", "stale-create", "create-error", "create-lost-id", "create-returned-id-error", "create-old-id", "inspect-created", "stale-start", "start-error", "false-start-ack", "start-lost-reply", "cancel-start", "inspect-running", "late-engine", "commit-create-dispatched", "commit-created", "commit-start-dispatched", "commit-start-acknowledged", "commit-running"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorReplacementFixture(t, false)
			f.mode = mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			got, err := ExecuteNextAnchorReplacement(ctx, f.h, f.saved, f, f.commit, f.clock)
			if err == nil || got.Validate(f.h) != nil || !reflect.DeepEqual(got, f.saved) {
				t.Fatal("failure not retained", err)
			}
			creates, starts := f.creates, f.starts
			wantPhase := map[string]string{
				"engine": "planned", "inspect-absent": "planned", "commit-create-dispatched": "planned",
				"stale-create": "create-dispatched", "create-error": "create-dispatched", "create-lost-id": "create-dispatched", "create-returned-id-error": "created", "create-old-id": "create-dispatched", "commit-created": "create-dispatched",
				"inspect-created": "created", "commit-start-dispatched": "created", "stale-start": "start-dispatched", "start-error": "start-dispatched", "false-start-ack": "start-acknowledged", "start-lost-reply": "start-dispatched", "cancel-start": "start-dispatched", "inspect-running": "start-acknowledged", "late-engine": "start-acknowledged", "commit-start-acknowledged": "start-dispatched", "commit-running": "start-acknowledged",
			}[mode]
			if got.Members[0].Phase != wantPhase {
				t.Fatal("failure did not reach intended boundary", mode, got.Members[0].Phase, wantPhase)
			}

			if got.Members[0].Phase != "planned" {
				if _, err := ExecuteNextAnchorReplacement(context.Background(), f.h, got, f, f.commit, f.clock); err == nil || f.creates != creates || f.starts != starts {
					t.Fatal("pending effects replayed")
				}
			}
			phase := got.Members[0].Phase
			f.mode = ""
			observed, err := ObserveAnchorReplacementRunning(context.Background(), f.h, got, f, f.commit, f.clock)
			running := len(f.state.Runtime.Members) > 0 && f.state.Runtime.Members[0].Running
			if (phase == "start-dispatched" || phase == "start-acknowledged") && running {
				if err != nil || observed.Members[0].Phase != "running" || observed.Members[0].StartAcknowledged != got.Members[0].StartAcknowledged {
					t.Fatal("read-only recovery failed or forged acknowledgement", err)
				}
			} else if err == nil {
				t.Fatal("invented completed Start")
			}
			if f.creates != creates || f.starts != starts {
				t.Fatal("observation replayed mutation")
			}
			if mode == "create-returned-id-error" && (got.Members[0].Phase != "created" || got.Members[0].Container.ID == "" || starts != 0) {
				t.Fatal("returned Create ID lost or followed by Start")
			}
		})
	}
}
func TestAnchorReplacementRefusesChangedEvidenceAndAuthority(t *testing.T) {
	for _, stage := range []string{"absent", "created", "running"} {
		for _, fault := range []string{"pier", "anchor", "namespace", "foreign", "process", "stale"} {
			if stage == "absent" && fault == "process" {
				continue
			}
			t.Run(stage+"/"+fault, func(t *testing.T) {
				f := newAnchorReplacementFixture(t, false)
				f.mutate = func(got string, o *AnchorRetirementObservation) {
					if got != stage {
						return
					}
					switch fault {
					case "pier":
						o.Piers[0].Storage.Identity.Inode++
					case "anchor":
						o.AnchorProcess.StartedAt = o.CapturedAt
					case "namespace":
						o.Runtime.Anchor.Namespace.Inode++
					case "foreign":
						o.Processes = append(o.Processes, AnchorDrainProcess{InstanceID: "foreign"})
					case "process":
						o.Processes[0].Process.FinishedAt = o.CapturedAt
					case "stale":
						o.CapturedAt = o.CapturedAt.Add(-time.Minute)
					}
				}
				if _, err := ExecuteNextAnchorReplacement(context.Background(), f.h, f.saved, f, f.commit, f.clock); err == nil {
					t.Fatal("changed evidence accepted")
				}
			})
		}
	}
	f := newAnchorReplacementFixture(t, false)
	for _, mutate := range []func(*AnchorReplacement){
		func(s *AnchorReplacement) { s.Sequence++ }, func(s *AnchorReplacement) { s.OperationID = s.Retirement.OperationID },
		func(s *AnchorReplacement) { s.Members[1].Creation.OperationID = s.Members[0].Creation.OperationID }, func(s *AnchorReplacement) { s.Members[0].Creation.Generation++ },
		func(s *AnchorReplacement) { s.Members[0].StartAcknowledged = true }, func(s *AnchorReplacement) { s.Members[0], s.Members[1] = s.Members[1], s.Members[0] },
		func(s *AnchorReplacement) { s.Retirement.Members[0].After.Piers = nil },
	} {
		s := f.saved.Clone()
		mutate(&s)
		if s.Validate(f.h) == nil {
			t.Fatal("forged authority accepted")
		}
	}
	if _, err := f.saved.Advance(f.h, 1, 0, "create-dispatched", AnchorReplacementResult{Observation: &f.state}); err == nil {
		t.Fatal("skipped first member")
	}
}

func TestAnchorReplacementPinsEarlierReplacementWhileCreatingNext(t *testing.T) {
	f := newAnchorReplacementFixture(t, false)
	s, err := ExecuteNextAnchorReplacement(context.Background(), f.h, f.saved, f, f.commit, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	f.mutate = func(stage string, o *AnchorRetirementObservation) {
		if stage == "absent" {
			o.Processes[0].Process.StartedAt = o.CapturedAt
		}
	}
	if _, err = ExecuteNextAnchorReplacement(context.Background(), f.h, s, f, f.commit, f.clock); err == nil || f.creates != 1 || f.starts != 1 {
		t.Fatal("changed completed peer allowed next Create")
	}
}
