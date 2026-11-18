package projectflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type fixture struct {
	plan                                 projectnetwork.Plan
	operations                           map[string]protocol.Operation
	creates, lookups, downloads, actions int
	lose                                 string
	edit                                 func(*protocol.ProjectStatus)
}

// A real transport must release blocked reads on the wait's deadline, even
// before a first receipt can be returned. This wrapper models that boundary.
type blockedRead struct{ *fixture }

func (f blockedRead) RegisterProject(ctx context.Context, _ client.Identity, _ protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error) {
	<-ctx.Done()
	return protocol.ProjectRegistrationResult{}, ctx.Err()
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "bounded", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "member", Ship: "~zod", HTTPPort: 19081}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{plan: plan, operations: map[string]protocol.Operation{}}
}
func (f *fixture) RegisterProject(ctx context.Context, _ client.Identity, r protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error) {
	digest, _ := f.plan.Digest()
	if digest != r.ReviewDigest {
		return protocol.ProjectRegistrationResult{}, errors.New("changed registration")
	}
	if client.IsCreationReplayOnly(ctx) {
		f.lookups++
	}
	op, exists := f.operations[r.RequestID]
	if !exists {
		if client.IsCreationReplayOnly(ctx) {
			return protocol.ProjectRegistrationResult{}, errors.New("original absent")
		}
		f.creates++
		op = protocol.Operation{ID: r.RequestID, RequestID: r.RequestID, Actor: "actor", ProjectID: "project", ConfigRevision: 1, Kind: "register-project", Status: "succeeded"}
		f.operations[r.RequestID] = op
	}
	return protocol.ProjectRegistrationResult{Operation: op}, nil
}
func (f *fixture) PrepareProjectDependencies(context.Context, client.Identity) error {
	f.downloads++
	return nil
}
func (f *fixture) ProjectOperate(_ context.Context, _ client.Identity, _ string, r protocol.ProjectActionRequest) (protocol.ProjectActionResult, error) {
	f.actions++
	if prior, exists := f.operations[r.RequestID]; exists {
		return protocol.ProjectActionResult{Operation: prior}, nil
	}
	_, kind, _ := protocol.ProjectAction(r.Kind)
	op := protocol.Operation{ID: r.RequestID, RequestID: r.RequestID, Actor: "actor", ProjectID: "project", ConfigRevision: 1, Kind: kind, InstanceID: r.InstanceID, Status: "queued"}
	f.operations[r.RequestID] = op
	if f.lose == r.Kind {
		f.lose = ""
		return protocol.ProjectActionResult{}, &client.OutcomeUncertainError{}
	}
	return protocol.ProjectActionResult{Operation: op}, nil
}
func (f *fixture) ProjectStatus(context.Context, client.Identity, string) (protocol.ProjectStatus, error) {
	digest, _ := f.plan.Digest()
	s := protocol.ProjectStatus{Project: protocol.ProjectReservation{ID: "project", Revision: 1, ReviewDigest: digest}, ReadAt: time.Now().UTC()}
	for _, op := range f.operations {
		s.Operations = append(s.Operations, protocol.ProjectOperationSummary{ID: op.ID, RequestID: op.RequestID, Actor: op.Actor, ConfigRevision: op.ConfigRevision, Kind: op.Kind, InstanceID: op.InstanceID, Status: op.Status})
	}
	if f.edit != nil {
		f.edit(&s)
	}
	return s, nil
}
func (f *fixture) finish(id, status string) {
	op := f.operations[id]
	op.Status = status
	f.operations[id] = op
}

func TestBoundedHandlesResumeAfterLostReplyWithoutResubmission(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pin := client.Identity{}
	if _, err := Inspect(ctx, f, pin, "actor", f.plan, "parent"); err == nil || f.creates != 0 || f.actions != 0 {
		t.Fatal("lookup created absent reservation")
	}
	h, err := Begin(ctx, f, pin, "actor", f.plan, "parent")
	if err != nil || h.Phase != "reserved" || f.creates != 1 || f.downloads != 0 || f.actions != 0 {
		t.Fatal(h, err)
	}
	for _, step := range Steps(f.plan) {
		f.lose = step.Kind
		_, err := Advance(ctx, f, pin, "actor", f.plan, "parent", RequestID("parent", step))
		var uncertain *client.OutcomeUncertainError
		if !errors.As(err, &uncertain) {
			t.Fatal("lost phase reply was not uncertain", step, err)
		}
		before := f.actions
		// Only the server's saved operations survive: a replacement coordinator
		// has no in-memory handle and cannot automatically replay this phase.
		body, err := json.Marshal(f.operations)
		if err != nil {
			t.Fatal(err)
		}
		restored := map[string]protocol.Operation{}
		if err := json.Unmarshal(body, &restored); err != nil {
			t.Fatal(err)
		}
		f.operations = restored
		for i := 0; i < 2; i++ {
			h, err = Advance(ctx, f, pin, "actor", f.plan, "parent", RequestID("parent", step))
			if err != nil || h.Phase != "waiting" || h.NativeRequestID != RequestID("parent", step) || h.OperationID != RequestID("parent", step) || f.actions != before {
				t.Fatal("pending phase resubmitted or rebound", h, err)
			}
		}
		f.finish(h.NativeRequestID, "succeeded")
	}
	h, err = Inspect(ctx, f, pin, "actor", f.plan, "parent")
	if err != nil || h.Phase != "ready" || f.creates != 1 || f.actions != 5 || f.downloads != 1 {
		t.Fatal(h, err, f.creates, f.actions, f.downloads)
	}
	if _, err := Advance(ctx, f, pin, "actor", f.plan, "parent", RequestID("parent", Steps(f.plan)[0])); err != nil || f.actions != 5 {
		t.Fatal("complete handle dispatched again", err)
	}
}

func TestInterruptedOrChangedHistoryNeverDispatches(t *testing.T) {
	for _, scenario := range []string{"interrupted", "failed", "actor", "kind", "revision", "duplicate", "reordered", "truncated"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			pin := client.Identity{}
			if _, err := Begin(ctx, f, pin, "actor", f.plan, "parent"); err != nil {
				t.Fatal(err)
			}
			native := RequestID("parent", Steps(f.plan)[0])
			h, err := Advance(ctx, f, pin, "actor", f.plan, "parent", native)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "interrupted" || scenario == "failed" {
				f.finish(h.NativeRequestID, scenario)
				next, err := Advance(ctx, f, pin, "actor", f.plan, "parent", native)
				if err != nil || next.Phase != "needs-recovery" || next.OperationID != h.OperationID || f.actions != 1 {
					t.Fatal(next, err)
				}
				return
			}
			f.finish(h.NativeRequestID, "succeeded")
			f.edit = func(s *protocol.ProjectStatus) {
				switch scenario {
				case "actor":
					s.Operations[0].Actor = "other"
				case "kind":
					s.Operations[0].Kind = "stop-project-ship"
				case "revision":
					s.Operations[0].ConfigRevision++
				case "duplicate":
					s.Operations = append(s.Operations, s.Operations[0])
				case "truncated":
					s.HasOlderOperations = true
				case "reordered":
					step := Steps(f.plan)[2]
					_, kind, _ := protocol.ProjectAction(step.Kind)
					s.Operations = append(s.Operations, protocol.ProjectOperationSummary{ID: "out-of-order", RequestID: RequestID("parent", step), Actor: "actor", ConfigRevision: 1, Kind: kind, InstanceID: step.Member, Status: "succeeded"})
				}
			}
			if _, err := Advance(ctx, f, pin, "actor", f.plan, "parent", native); !errors.Is(err, ErrHistory) || f.actions != 1 || f.downloads != 1 {
				t.Fatal("changed history reached a new effect", err, f.actions)
			}
		})
	}
}

func TestCreationWaitIsBoundedAndCannotAdvanceOrCancelWork(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pin := client.Identity{}
	if _, err := Begin(ctx, f, pin, "actor", f.plan, "parent"); err != nil {
		t.Fatal(err)
	}
	h, err := Advance(ctx, f, pin, "actor", f.plan, "parent", RequestID("parent", Steps(f.plan)[0]))
	if err != nil {
		t.Fatal(err)
	}
	waited, err := Wait(ctx, f, pin, "actor", f.plan, "parent", 0)
	if err != nil || !waited.WaitExpired || waited.Handle.OperationID != h.OperationID || f.actions != 1 {
		t.Fatal(waited, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	f.edit = func(*protocol.ProjectStatus) { cancel() }
	if _, err := Wait(cancelled, f, pin, "actor", f.plan, "parent", time.Second); !errors.Is(err, context.Canceled) || f.actions != 1 || f.operations[h.NativeRequestID].Status != "queued" {
		t.Fatal("wait cancellation changed native work", err)
	}
	f.edit = nil
	f.finish(h.NativeRequestID, "succeeded")
	waited, err = Wait(ctx, f, pin, "actor", f.plan, "parent", time.Second)
	if err != nil || waited.WaitExpired || waited.Handle.Phase != "next-phase" || f.actions != 1 {
		t.Fatal("wait advanced instead of returning the saved boundary", waited, err)
	}
	if _, err := Wait(ctx, f, pin, "actor", f.plan, "parent", 31*time.Second); err == nil || f.actions != 1 {
		t.Fatal("unbounded wait admitted")
	}
	if _, err := Wait(ctx, blockedRead{f}, pin, "actor", f.plan, "parent", time.Millisecond); !errors.Is(err, context.DeadlineExceeded) || f.actions != 1 {
		t.Fatal("blocked first receipt read escaped wait deadline", err)
	}
}

func TestCompletedPhaseRetryCannotAdvanceAfterLostReply(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pin := client.Identity{}
	begin, err := Begin(ctx, f, pin, "actor", f.plan, "parent")
	if err != nil {
		t.Fatal(err)
	}
	first := RequestID("parent", Steps(f.plan)[0])
	if begin.NativeRequestID != first {
		t.Fatal("begin omitted initial exact phase binding", begin)
	}
	f.lose = "prepare"
	if _, err := Advance(ctx, f, pin, "actor", f.plan, "parent", first); err == nil {
		t.Fatal("lost reply not uncertain")
	}
	// The background phase completes BEFORE the caller can read or retry. The
	// unchanged call must keep its old intent, not submit the next missing step.
	f.finish(first, "succeeded")
	for i := 0; i < 2; i++ {
		h, err := Advance(ctx, f, pin, "actor", f.plan, "parent", first)
		if err != nil || h.Phase != "completed" || h.NativeRequestID != first || h.OperationID != first || f.actions != 1 || f.downloads != 1 {
			t.Fatal("old phase call advanced after lost reply", h, err, f.actions)
		}
	}
	future := RequestID("parent", Steps(f.plan)[2])
	if _, err := Advance(ctx, f, pin, "actor", f.plan, "parent", future); !errors.Is(err, ErrPhase) || f.actions != 1 {
		t.Fatal("future phase bypassed missing predecessor", err)
	}
	before := f.lookups
	if _, err := Advance(ctx, f, pin, "actor", f.plan, "parent", "unknown"); !errors.Is(err, ErrPhase) || f.lookups != before || f.actions != 1 {
		t.Fatal("unknown phase reached runner", err)
	}
	next, err := Inspect(ctx, f, pin, "actor", f.plan, "parent")
	if err != nil {
		t.Fatal(err)
	}
	if h, err := Advance(ctx, f, pin, "actor", f.plan, "parent", next.NativeRequestID); err != nil || h.Step != "start-anchor" || f.actions != 2 {
		t.Fatal("explicit next phase did not advance", h, err)
	}
}
