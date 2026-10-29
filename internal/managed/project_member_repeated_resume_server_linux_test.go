//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type queuedRepeatedLifecycleBackend struct {
	*queuedProjectBackend
	generation int
	fault      string
	calls      chan string
}

func (b *queuedRepeatedLifecycleBackend) source(s runtimequalify.ProjectResumedMemberStopSelection) {
	b.t.Helper()
	if len(s.Resumes) != b.generation-1 || s.Readiness.ContainerGeneration != uint64(b.generation) || s.Readiness.ResumeOperationID != s.Stop.LaunchOperationID || len(s.StoppedPeers) != 0 {
		b.t.Fatal("wrong historical prefix or retired Stop included")
	}
	set, err := projectnetwork.ResolveMemberGenerations(s.Project.Plan, s.Launches, s.Resumes)
	if err != nil || !reflect.DeepEqual(set.Binding, s.Stop.Binding) {
		b.t.Fatal("wrong source binding", err)
	}
	for _, m := range set.Members {
		if m.InstanceID == s.Stop.InstanceID && (m.Generation != uint64(b.generation) || m.OperationID != s.Stop.LaunchOperationID || m.Storage != s.Stop.Storage) {
			b.t.Fatal("changed creation identity")
		}
	}
}
func (b *queuedRepeatedLifecycleBackend) OpenProjectMemberStopExit(context.Context, runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return nil, errors.New("initial Stop exit must not be used")
}
func (b *queuedRepeatedLifecycleBackend) OpenResumedProjectMemberStopExit(_ context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	b.source(s)
	b.calls <- "exit"
	var id string
	if err := b.s.db.QueryRow("SELECT operation_id FROM project_member_stop_exits WHERE stop_operation_id=?", s.Stop.OperationID).Scan(&id); err != nil {
		b.t.Fatal(err)
	}
	d := &storedStopExitDriver{t: b.t, s: b.s, id: id, proof: memberStopExitProof(b.t, b.s, id)}
	if b.fault == "exit" {
		d.fault = "read"
	}
	return d, nil
}
func (b *queuedRepeatedLifecycleBackend) OpenProjectMemberRetirement(context.Context, runtimequalify.ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return projectnetwork.MemberRetirement{}, nil, errors.New("initial retirement must not be used")
}
func (b *queuedRepeatedLifecycleBackend) OpenResumedProjectMemberRetirement(_ context.Context, s runtimequalify.ProjectResumedMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	b.source(s.Stop)
	b.calls <- "retire"
	p, err := loadProject(b.s.db, s.Stop.Project.ProjectID)
	if err != nil {
		b.t.Fatal(err)
	}
	initial, err := projectnetwork.NewMemberRetirement(p.Plan, s.Exit, s.OperationID)
	if err != nil {
		b.t.Fatal(err)
	}
	before, after := retirementStoreObservations(b.t, b.s, s.OperationID)
	d := &storedRetirementDriver{t: b.t, s: b.s, id: s.OperationID, before: before, after: after}
	if b.fault == "retire" {
		d.fault = "remove"
	}
	return initial, d, nil
}
func (b *queuedRepeatedLifecycleBackend) OpenProjectMemberResume(_ context.Context, s runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	if s.ResumedStop == nil || !reflect.DeepEqual(s.Stop, runtimequalify.ProjectMemberStopSelection{}) {
		b.t.Fatal("lost replacement source")
	}
	b.source(*s.ResumedStop)
	b.calls <- "resume"
	initial := initialResumeFixture(b.t, b.s, s.OperationID)
	if initial.PreviousGeneration != uint64(b.generation) {
		b.t.Fatal("wrong generation")
	}
	if b.fault == "bad-generation" {
		initial.PreviousGeneration++
	}
	d := &resumeStoreDriver{t: b.t, s: b.s, id: s.OperationID}
	if b.fault == "resume" {
		d.fault = "lost-start"
	}
	return initial, d, nil
}
func (b *queuedRepeatedLifecycleBackend) OpenResumedProjectReadiness(_ context.Context, s runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	if s.Resume.ResumedStop == nil {
		b.t.Fatal("lost historical creation source")
	}
	b.source(*s.Resume.ResumedStop)
	b.calls <- "ready"
	if s.Completed.PreviousGeneration != uint64(b.generation) || s.Resume.OperationID != s.Completed.OperationID || !reflect.DeepEqual(s.Resume.Retirement, s.Completed.Retirement) {
		b.t.Fatal("wrong completed creation")
	}
	p := s.Resume.ResumedStop.Project
	d := &resumedReadinessStoreDriver{proof: resumedReadinessProof(b.t, b.s, p.ProjectID, s.Completed.Retirement.Exit.Stop.InstanceID, s.Completed.OperationID)}
	if b.fault == "ready" {
		d.fault = "lost-observation"
	}
	return d, nil
}
func TestRepeatedMemberLifecycleQueueUsesExactPrefixesAndSharedGate(t *testing.T) {
	for _, fault := range []string{"pass", "exit", "retire", "resume", "ready", "bad-generation"} {
		t.Run(fault, func(t *testing.T) {
			r, original, _, _ := projectQueueFixture(t)
			p, stopReq, _ := readyCurrentStopProject(t, r.s)
			instance := p.Members[0].InstanceID
			b := &queuedRepeatedLifecycleBackend{queuedProjectBackend: original, fault: fault, calls: make(chan string, 8)}
			r.b = b
			for generation := 2; generation <= 3; generation++ {
				b.generation = generation
				stop := dispatchedCurrentStopFixture(t, r.s, p, stopReq, generation)
				history := currentStopHistory(t, r.s)
				var exitID, retiredID, resumeID string
				for _, stage := range []string{"exit", "retire", "resume", "ready"} {
					queue := func() (protocol.Operation, bool, error) {
						id := fmt.Sprintf("queued-%s-generation-%d", stage, generation)
						switch stage {
						case "exit":
							return r.queueProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: id, StopOperationID: stop.OperationID, ExpectedRevision: p.Revision, ReviewDigest: stopReq.ReviewDigest})
						case "retire":
							return r.queueProjectMemberRetirement("actor", p.ID, instance, projectMemberRetirementRequest{RequestID: id, ExitOperationID: exitID, ExpectedRevision: p.Revision, ReviewDigest: stopReq.ReviewDigest})
						case "resume":
							return r.queueProjectMemberResume("actor", p.ID, instance, projectMemberResumeRequest{RequestID: id, RetirementOperationID: retiredID, ExpectedRevision: p.Revision, ReviewDigest: stopReq.ReviewDigest})
						default:
							return r.queueProjectMemberResumedReadiness("actor", p.ID, instance, projectMemberResumedReadinessRequest{RequestID: id, ResumeOperationID: resumeID, ExpectedRevision: p.Revision, ReviewDigest: stopReq.ReviewDigest})
						}
					}
					r.workGate <- struct{}{}
					op, hit, err := queue()
					if err != nil || hit {
						t.Fatal(stage, op, hit, err)
					}
					replay, hit, err := queue()
					if err != nil || !hit || replay.ID != op.ID {
						t.Fatal("queue replay changed", stage, err)
					}
					select {
					case call := <-b.calls:
						t.Fatal("factory bypassed resource gate", call)
					default:
					}
					if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
						t.Fatal("worker bypassed update gate", w.Code)
					}
					<-r.workGate
					// Full historical validation under the race detector exceeds the
					// original one-generation helper's two-second deadline.
					began := time.Now()
					done := make(chan struct{})
					go func() { r.wg.Wait(); close(done) }()
					select {
					case <-done:
					case <-time.After(10 * time.Second):
						t.Fatal("repeated lifecycle worker did not finish", stage, generation)
					}
					t.Logf("generation%d %s worker completed in %s", generation, stage, time.Since(began))
					select {
					case call := <-b.calls:
						if call != stage {
							t.Fatal(call, stage)
						}
					default:
						t.Fatal("factory not called", stage)
					}
					got, err := loadOperation(r.s.db, op.ID)
					if err != nil {
						t.Fatal(err)
					}
					failed := fault == stage || fault == "bad-generation" && stage == "resume"
					want := "succeeded"
					if failed {
						want = "interrupted"
					}
					if got.Status != want {
						t.Fatal(stage, got.Status, want)
					}
					after := currentStopHistory(t, r.s)
					for id, body := range history {
						if after[id] != body {
							t.Fatal("rewrote older receipts", id)
						}
					}
					replay, hit, err = queue()
					if err != nil || !hit || replay.Status != want {
						t.Fatal("completed replay changed", err)
					}
					if failed {
						if stage == "resume" {
							j, e := loadProjectMemberResume(r.s.db, op.ID)
							if e != nil {
								t.Fatal(e)
							}
							if fault == "resume" && (j.Resume == nil || j.Resume.Phase != "start-dispatched") {
								t.Fatal("lost Start uncertainty")
							}
							if fault == "bad-generation" && j.Resume != nil {
								t.Fatal("accepted wrong generation")
							}
						}
						if err = validateProjectMemberResumedReadiness(r.s.db); err != nil {
							t.Fatal(err)
						}
						return
					}
					switch stage {
					case "exit":
						exitID = op.ID
					case "retire":
						retiredID = op.ID
					case "resume":
						resumeID = op.ID
					case "ready":
						stopReq.ReadinessOperationID = op.ID
					}
				}
			}
		})
	}
}

func TestRepeatedMemberLifecycleRequiresCapabilitiesBeforeAdmission(t *testing.T) {
	for _, stage := range []string{"exit", "retire"} {
		t.Run(stage, func(t *testing.T) {
			r, original, _, _ := projectQueueFixture(t)
			p, req, _ := readyCurrentStopProject(t, r.s)
			instance := p.Members[0].InstanceID
			stop := dispatchedCurrentStopFixture(t, r.s, p, req, 2)
			var exitID string
			if stage == "retire" {
				op, _, err := r.s.acceptProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: "completed-exit", StopOperationID: stop.OperationID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = r.s.beginProjectMemberStopExit(op.ID); err != nil {
					t.Fatal(err)
				}
				if err = r.s.completeProjectMemberStopExit(op.ID, memberStopExitProof(t, r.s, op.ID)); err != nil {
					t.Fatal(err)
				}
				exitID = op.ID
			}
			before := currentStopHistory(t, r.s)
			var err error
			if stage == "exit" {
				r.b = &queuedStopExitBackend{queuedProjectBackend: original}
				_, _, err = r.queueProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: "unsupported-exit", StopOperationID: stop.OperationID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
			} else {
				r.b = &queuedRetirementBackend{queuedProjectBackend: original}
				_, _, err = r.queueProjectMemberRetirement("actor", p.ID, instance, projectMemberRetirementRequest{RequestID: "unsupported-retirement", ExitOperationID: exitID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
			}
			if err == nil || !reflect.DeepEqual(before, currentStopHistory(t, r.s)) {
				t.Fatal("unavailable backend admitted work", err)
			}
		})
	}
}
