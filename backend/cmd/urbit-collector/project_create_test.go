package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type createFixture struct {
	plan            projectnetwork.Plan
	operations      map[string]protocol.Operation
	calls           []protocol.ProjectActionRequest
	effects         int
	failKind        string
	loseKind        string
	foreign         bool
	resumeSeen      bool
	dependencies    int
	dependencyError bool
}

func (f *createFixture) PrepareProjectDependencies(context.Context, client.Identity) error {
	f.dependencies++
	if f.dependencyError {
		return errors.New("download failed")
	}
	return nil
}
func (f *createFixture) RegisterProject(ctx context.Context, _ client.Identity, r protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error) {
	f.resumeSeen = client.IsCreationReplayOnly(ctx)
	digest, _ := f.plan.Digest()
	if r.ReviewDigest != digest {
		return protocol.ProjectRegistrationResult{}, errors.New("changed plan")
	}
	return protocol.ProjectRegistrationResult{Operation: protocol.Operation{ProjectID: "project", ConfigRevision: 1, RequestID: r.RequestID}}, nil
}
func (f *createFixture) ProjectOperate(_ context.Context, _ client.Identity, _ string, r protocol.ProjectActionRequest) (protocol.ProjectActionResult, error) {
	f.calls = append(f.calls, r)
	if op, ok := f.operations[r.RequestID]; ok {
		return protocol.ProjectActionResult{Operation: op}, nil
	}
	f.effects++
	_, kind, _ := protocol.ProjectAction(r.Kind)
	op := protocol.Operation{ProjectID: "project", ConfigRevision: 1, RequestID: r.RequestID, ID: r.RequestID, Kind: kind, Actor: "actor", InstanceID: r.InstanceID, Status: "queued"}
	f.operations[r.RequestID] = op
	if r.Kind == f.loseKind {
		f.loseKind = ""
		return protocol.ProjectActionResult{}, errors.New("lost reply")
	}
	return protocol.ProjectActionResult{Operation: op}, nil
}
func (f *createFixture) ProjectStatus(_ context.Context, _ client.Identity, _ string) (protocol.ProjectStatus, error) {
	digest, _ := f.plan.Digest()
	s := protocol.ProjectStatus{Project: protocol.ProjectReservation{ID: "project", Revision: 1, ReviewDigest: digest}}
	for id, op := range f.operations {
		if op.Status == "queued" {
			op.Status = "succeeded"
			_, bad, _ := protocol.ProjectAction(f.failKind)
			if op.Kind == bad {
				op.Status = "interrupted"
			}
			f.operations[id] = op
		}
		s.Operations = append(s.Operations, protocol.ProjectOperationSummary{ID: op.ID, RequestID: op.RequestID, Actor: op.Actor, Kind: op.Kind, InstanceID: op.InstanceID, Status: op.Status})
	}
	if f.foreign {
		s.Operations = append(s.Operations, protocol.ProjectOperationSummary{RequestID: "other", Actor: "actor"})
	}
	return s, nil
}
func TestGuidedProjectCreation(t *testing.T) {
	for _, mode := range []string{"complete", "lost-reply", "interrupted", "foreign-history", "cancelled", "dependencies"} {
		t.Run(mode, func(t *testing.T) {
			plan, err := projectnetwork.Build(projectnetwork.Request{Label: "demo", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "zod", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "nec", Ship: "~nec", HTTPPort: 18382}}}, projectnetwork.Reservations{})
			if err != nil {
				t.Fatal(err)
			}
			f := &createFixture{plan: plan, operations: map[string]protocol.Operation{}}
			if mode == "dependencies" {
				f.dependencyError = true
			}
			if mode == "lost-reply" {
				f.loseKind = "launch"
			}
			if mode == "interrupted" {
				f.failKind = "launch"
			}
			if mode == "foreign-history" {
				f.foreign = true
			}
			wait := func(context.Context) error {
				if mode == "cancelled" {
					return context.Canceled
				}
				return nil
			}
			var progress []projectCreateProgress
			emit := func(p projectCreateProgress) error { progress = append(progress, p); return nil }
			err = createProject(context.Background(), f, client.Identity{}, "actor", plan, "parent", false, emit, wait)
			if mode == "complete" {
				if err != nil {
					t.Fatal(err)
				}
				if f.effects != 8 || progress[len(progress)-1].Phase != "ready" {
					t.Fatal(f.effects, progress)
				}
			} else if err == nil {
				t.Fatal("expected interruption")
			}
			if mode == "lost-reply" {
				previous := append([]protocol.ProjectActionRequest(nil), f.calls...)
				if len(previous) != 4 {
					t.Fatal("continued after lost mutation reply", previous)
				}
				if err = createProject(context.Background(), f, client.Identity{}, "actor", plan, "parent", true, emit, wait); err != nil {
					t.Fatal(err)
				}
				if f.effects != 8 || !f.resumeSeen {
					t.Fatal("replaced effects or reservation", f.effects, f.resumeSeen)
				}
				if !reflect.DeepEqual(previous, f.calls[len(previous):2*len(previous)]) {
					t.Fatal("resume changed original request inputs")
				}
			}
			if mode == "dependencies" && (f.effects != 0 || f.dependencies != 1) {
				t.Fatal("download failure reached project effects")
			}
			if mode == "lost-reply" && f.dependencies != 1 {
				t.Fatal("resume repeated dependency preparation after project effects")
			}
			if mode == "foreign-history" && (f.effects != 0 || f.dependencies != 0) {
				t.Fatal("acted on changed project")
			}
			if mode == "interrupted" {
				effects := f.effects
				if err = createProject(context.Background(), f, client.Identity{}, "actor", plan, "parent", true, emit, wait); err == nil || f.effects != effects {
					t.Fatal("repeated interrupted effect")
				}
			}
		})
	}
}

func TestGuidedProjectSingleShip(t *testing.T) {
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "demo", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	f := &createFixture{plan: plan, operations: map[string]protocol.Operation{}}
	var last projectCreateProgress
	err = createProject(context.Background(), f, client.Identity{}, "actor", plan, "one-ship", false, func(p projectCreateProgress) error { last = p; return nil }, func(context.Context) error { return nil })
	if err != nil || f.effects != 5 || last.Phase != "ready" {
		t.Fatal(err, f.effects, last)
	}
}

func TestGuidedEmptyProjectCreatesOnlyNetwork(t *testing.T) {
	p, err := projectnetwork.Build(projectnetwork.Request{Label: "empty", ConfirmSharedAdministration: true}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	f := &createFixture{plan: p, operations: map[string]protocol.Operation{}}
	err = createProject(context.Background(), f, client.Identity{}, "actor", p, "empty", false, func(projectCreateProgress) error { return nil }, func(context.Context) error { return nil })
	if err != nil || f.effects != 2 {
		t.Fatal("empty project failed", f.effects, err)
	}
}
