//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type expansionFixtureBackend struct {
	*queuedProjectBackend
	calls     int
	preflight bool
}

func (b *expansionFixtureBackend) AddProjectMember(ctx context.Context, s *runtimequalify.ProjectExpansion, id string, save func(runtimequalify.ProjectExpansion) error) error {
	b.calls++
	if b.preflight {
		return errors.New("synthetic preflight refusal")
	}
	for i := range s.Members {
		if s.Members[i].Member.InstanceID == id {
			s.Members[i].Phase = "create-dispatched"
		}
	}
	if err := save(*s); err != nil {
		return err
	}
	return errors.New("synthetic lost creation response")
}
func (*expansionFixtureBackend) ExpandedProjectFleet(context.Context, runtimequalify.ProjectExpansion) ([]protocol.Instance, error) {
	return nil, nil
}
func (*expansionFixtureBackend) ExpandedProjectCode(context.Context, runtimequalify.ProjectExpansion, string, string) (string, error) {
	return "", nil
}
func (*expansionFixtureBackend) ProjectShipAction(context.Context, *runtimequalify.ProjectExpansion, string, string, string, func(runtimequalify.ProjectExpansion) error) error {
	return nil
}

func TestProjectExpansionAdmissionReplayAndUnknownOutcome(t *testing.T) {
	r, original, _, _ := projectQueueFixture(t)
	p, req, _ := readyStopProject(t, r.s)
	before, err := r.s.projectFleetSelection(p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	b := &expansionFixtureBackend{queuedProjectBackend: original}
	r.b = b
	add := protocol.ProjectActionRequest{Kind: "add-member", InstanceID: "new-nec", Ship: "~nec", HTTPPort: 29081, RequestID: "add-nec", ExpectedRevision: 1, ReviewDigest: req.ReviewDigest, ConfirmAction: true}
	// Fixture's existing identities are zod/nec. Use bus for the new member.
	for _, m := range p.Plan.Members {
		if m.Ship == add.Ship {
			add.Ship = "~bus"
		}
	}
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectAddition("actor", p.ID, add, false)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	same, hit, err := r.queueProjectAddition("actor", p.ID, add, false)
	if err != nil || !hit || same.ID != op.ID {
		t.Fatal("replay", same, hit, err)
	}
	changed := add
	changed.HTTPPort++
	if _, _, err = r.queueProjectAddition("actor", p.ID, changed, false); err == nil {
		t.Fatal("changed replay accepted")
	}
	if _, err := projectReservations(r.s.db); err != nil {
		t.Fatal(err)
	}
	if err := reservePierPath(r.s.db, "intruder", "fakeships/new-nec/nec"); err == nil {
		t.Fatal("added pier not reserved")
	}
	// Legacy lifecycle cannot race this newer ledger.
	if _, _, err = r.s.acceptProjectMemberStop("actor", p.ID, p.Members[0].InstanceID, req); err == nil {
		t.Fatal("legacy lifecycle accepted expanded project")
	}
	<-r.workGate
	waitProjectWorker(t, r)
	after, err := loadProjectExpansion(r.s.db, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Members[len(after.Members)-1].Phase != "create-dispatched" || b.calls != 1 {
		t.Fatal("lost dispatch boundary")
	}
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil || receipt.Status != "interrupted" {
		t.Fatal(receipt, err)
	}
	if err = r.s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	got, hit, err := r.queueProjectAddition("actor", p.ID, add, true)
	if err != nil || !hit || got.ID != op.ID || b.calls != 1 {
		t.Fatal("replay redispatched")
	}
	fresh := add
	fresh.RequestID = "another"
	fresh.InstanceID = "other"
	fresh.HTTPPort++
	if _, _, err = r.queueProjectAddition("actor", p.ID, fresh, false); err == nil {
		t.Fatal("unresolved member permitted another dispatch")
	}
	old, err := r.s.projectFleetSelection(p.ID, "")
	if err != nil || !reflect.DeepEqual(before, old) {
		t.Fatal("original project history changed", err)
	}
	// The reservation table must agree with the durable member intent.
	if _, err = r.s.db.Exec("DELETE FROM project_expansion_members WHERE instance_id=?", add.InstanceID); err != nil {
		t.Fatal(err)
	}
	if _, err = projectReservations(r.s.db); err == nil {
		t.Fatal("missing claim accepted")
	}
}
func TestProjectExpansionPlanAllowsSameIdentityInSeparateProjects(t *testing.T) {
	a := runtimequalify.ProjectExpansion{Version: 1}
	b := runtimequalify.ProjectExpansion{Version: 1}
	// Pure planning has no engine or host side effects.
	member := func(id string) runtimequalify.ProjectExpansion {
		s, err := (runtimequalify.ProjectExpansion{Version: 1}).NewMember(projectnetwork.MemberRequest{InstanceID: id, Ship: "~zod", HTTPPort: 29082}, strings.Repeat("a", 32), "op-new")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	a = member("one-zod")
	b = member("two-zod")
	if a.Members[0].Member.Ship != b.Members[0].Member.Ship {
		t.Fatal("identity changed")
	}
	if _, err := a.NewMember(projectnetwork.MemberRequest{InstanceID: "duplicate-zod", Ship: "~zod", HTTPPort: 29083}, strings.Repeat("b", 32), "op-next"); err == nil {
		t.Fatal("duplicate galaxy within one namespace")
	}
}

func TestProjectExpansionExplicitPreflightContinuation(t *testing.T) {
	r, base, _, _ := projectQueueFixture(t)
	p, request, _ := readyStopProject(t, r.s)
	b := &expansionFixtureBackend{queuedProjectBackend: base, preflight: true}
	r.b = b
	add := protocol.ProjectActionRequest{Kind: "add-member", InstanceID: "next-bus", Ship: "~bus", HTTPPort: 29181, RequestID: "add-bus", ExpectedRevision: 1, ReviewDigest: request.ReviewDigest, ConfirmAction: true}
	op, _, err := r.queueProjectAddition("actor", p.ID, add, false)
	if err != nil {
		t.Fatal(err)
	}
	waitProjectWorker(t, r)
	state, err := loadProjectExpansion(r.s.db, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := state.Members[len(state.Members)-1]
	next := protocol.ProjectActionRequest{Kind: "member-continue", InstanceID: add.InstanceID, AfterOperationID: op.ID, RequestID: "continue-bus", ExpectedRevision: 1, ReviewDigest: request.ReviewDigest, ConfirmAction: true}
	if !protocol.ValidProjectActionRequest(next) {
		t.Fatal("invalid continuation")
	}
	b.preflight = false
	second, _, err := r.queueProjectAddition("actor", p.ID, next, false)
	if err != nil {
		t.Fatal(err)
	}
	waitProjectWorker(t, r)
	old, err := loadOperation(r.s.db, op.ID)
	if err != nil || old.Status != "failed" {
		t.Fatal(old, err)
	}
	state, err = loadProjectExpansion(r.s.db, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	after := state.Members[len(state.Members)-1]
	if before.Creator.RuntimeID != after.Creator.RuntimeID || after.Creator.OperationID != second.ID || len(state.History) != 1 || !reflect.DeepEqual(state.History[0], before) {
		t.Fatal("preflight history or reservation lost")
	}
	// Now the second attempt has an unknown dispatched outcome: continuation must refuse.
	next.AfterOperationID = second.ID
	next.RequestID = "unsafe-continue"
	if _, _, err = r.queueProjectAddition("actor", p.ID, next, false); err == nil {
		t.Fatal("dispatched outcome repeated")
	}
}
