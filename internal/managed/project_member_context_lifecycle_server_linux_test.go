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

type queuedContextLifecycleBackend struct {
	*queuedRepeatedLifecycleBackend
}

func (b *queuedContextLifecycleBackend) checkCurrent(s runtimequalify.ProjectCurrentMemberStopSelection) {
	b.t.Helper()
	src := s.ReadinessSource
	if src.Project.RunnerID != b.s.runnerID || len(s.StoppedPeers) != 0 || projectnetwork.ValidateCurrentMemberStop(src.Project.Plan, src.Launches, src.Resumes, src.Original, s.Readiness, s.Stop) != nil {
		b.t.Fatal("current source or history changed")
	}
	set, err := projectnetwork.ResolveMemberGenerations(src.Project.Plan, src.Launches, src.Resumes)
	g02Check(b.t, err)
	if !reflect.DeepEqual(set.Binding, s.Stop.Binding) {
		b.t.Fatal("current source has wrong historical prefix")
	}
}
func (b *queuedContextLifecycleBackend) OpenProjectMemberStop(context.Context, runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return nil, errors.New("wrong initial Stop factory")
}
func (b *queuedContextLifecycleBackend) OpenCurrentProjectMemberStop(_ context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	b.checkCurrent(s)
	b.calls <- "stop"
	before, after := contextStopObservations(b.t, b.s, s.Stop.ReadinessOperationID)
	d := &storedStopDriver{t: b.t, s: b.s, id: s.Stop.OperationID, before: before, after: after}
	if b.fault == "lost-stop" {
		d.fault = "signal"
	}
	return d, nil
}
func (b *queuedContextLifecycleBackend) OpenCurrentProjectMemberStopExit(_ context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	b.checkCurrent(s)
	b.calls <- "exit"
	var id string
	g02Check(b.t, b.s.db.QueryRow("SELECT operation_id FROM project_member_stop_exits WHERE stop_operation_id=?", s.Stop.OperationID).Scan(&id))
	return &storedStopExitDriver{t: b.t, s: b.s, id: id, proof: memberStopExitProof(b.t, b.s, id)}, nil
}
func (b *queuedContextLifecycleBackend) OpenCurrentProjectMemberRetirement(_ context.Context, s runtimequalify.ProjectCurrentMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	b.checkCurrent(s.Stop)
	b.calls <- "retire"
	initial, err := projectnetwork.NewMemberRetirement(s.Stop.ReadinessSource.Project.Plan, s.Exit, s.OperationID)
	g02Check(b.t, err)
	before, after := retirementStoreObservations(b.t, b.s, s.OperationID)
	return initial, &storedRetirementDriver{t: b.t, s: b.s, id: s.OperationID, before: before, after: after}, nil
}

type contextResumeStoreDriver struct{ *resumeStoreDriver }

func (d *contextResumeStoreDriver) Create(context.Context, projectnetwork.MemberResume) (string, error) {
	d.check("create-dispatched")
	d.creates++
	return strings.Repeat("7", 64), nil
}
func (b *queuedContextLifecycleBackend) OpenProjectMemberResume(_ context.Context, s runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	if s.CurrentStop == nil || s.ResumedStop != nil || !reflect.DeepEqual(s.Stop, runtimequalify.ProjectMemberStopSelection{}) {
		b.t.Fatal("lost current Stop source")
	}
	b.checkCurrent(*s.CurrentStop)
	b.calls <- "resume"
	initial := initialResumeFixture(b.t, b.s, s.OperationID)
	if initial.PreviousGeneration != s.CurrentStop.Readiness.ContainerGeneration {
		b.t.Fatal("wrong generation")
	}
	d := &contextResumeStoreDriver{&resumeStoreDriver{t: b.t, s: b.s, id: s.OperationID}}
	if b.fault == "resume-failure" {
		d.fault = "lost-start"
	}
	return initial, d, nil
}
func (b *queuedContextLifecycleBackend) OpenResumedProjectReadiness(_ context.Context, s runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	if s.Resume.CurrentStop == nil {
		b.t.Fatal("readiness lost current Stop creation source")
	}
	b.checkCurrent(*s.Resume.CurrentStop)
	b.calls <- "ready"
	p := s.Resume.CurrentStop.ReadinessSource.Project
	return &resumedReadinessStoreDriver{proof: resumedReadinessProof(b.t, b.s, p.ProjectID, s.Completed.Retirement.Exit.Stop.InstanceID, s.Completed.OperationID)}, nil
}
func reopenContextQueueHistory(t *testing.T, r *Server) {
	t.Helper()
	r.cancel()
	r.wg.Wait()
	history := currentStopHistory(t, r.s)
	dir := r.s.directory
	g02Check(t, r.s.close())
	reopened, err := testOpenStore(dir)
	g02Check(t, err)
	defer reopened.close()
	g02Check(t, reopened.reconcileInterrupted())
	assertContextHistory(t, reopened, history)
	g02Check(t, validateProjectMemberContextReadiness(reopened.db))
}

func TestContextStopQueueRequiresCurrentBackend(t *testing.T) {
	r, original, _, _ := projectQueueFixture(t)
	p, instance, req := readyContextStopProject(t, r.s, 1)
	r.b = &queuedStopBackend{queuedProjectBackend: original}
	before := currentStopHistory(t, r.s)
	if _, _, err := r.queueProjectMemberStop("actor", p.ID, instance, req); err == nil {
		t.Fatal("missing current backend admitted Stop")
	}
	assertContextHistory(t, r.s, before)
}

func TestContextLifecycleQueuePreservesHistoricalSourceAndSharedGate(t *testing.T) {
	for target := 0; target < 2; target++ {
		for _, fault := range []string{"pass", "lost-stop", "resume-failure"} {
			t.Run(fmt.Sprintf("target%d/%s", target, fault), func(t *testing.T) {
				r, original, _, _ := projectQueueFixture(t)
				p, instance, req := readyContextStopProject(t, r.s, target)
				b := &queuedContextLifecycleBackend{&queuedRepeatedLifecycleBackend{queuedProjectBackend: original, fault: fault, calls: make(chan string, 8)}}
				r.b = b
				var stopID, exitID, retiredID, resumeID string
				for _, stage := range []string{"stop", "exit", "retire", "resume", "ready"} {
					queue := func() (protocol.Operation, bool, error) {
						id := "context-queue-" + stage
						switch stage {
						case "stop":
							return r.queueProjectMemberStop("actor", p.ID, instance, req)
						case "exit":
							return r.queueProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: id, StopOperationID: stopID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
						case "retire":
							return r.queueProjectMemberRetirement("actor", p.ID, instance, projectMemberRetirementRequest{RequestID: id, ExitOperationID: exitID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
						case "resume":
							return r.queueProjectMemberResume("actor", p.ID, instance, projectMemberResumeRequest{RequestID: id, RetirementOperationID: retiredID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
						default:
							return r.queueProjectMemberResumedReadiness("actor", p.ID, instance, projectMemberResumedReadinessRequest{RequestID: id, ResumeOperationID: resumeID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
						}
					}
					history := currentStopHistory(t, r.s)
					r.workGate <- struct{}{}
					op, hit, err := queue()
					g02Check(t, err)
					if hit {
						t.Fatal("unexpected initial replay")
					}
					replay, hit, err := queue()
					g02Check(t, err)
					if !hit || replay.ID != op.ID {
						t.Fatal("queued replay changed")
					}
					select {
					case call := <-b.calls:
						t.Fatal("factory bypassed gate", call)
					default:
					}
					if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
						t.Fatal("update gate bypassed", w.Code)
					}
					<-r.workGate
					done := make(chan struct{})
					go func() { r.wg.Wait(); close(done) }()
					select {
					case <-done:
					case <-time.After(15 * time.Second):
						t.Fatal("context worker deadline", stage)
					}
					select {
					case call := <-b.calls:
						if call != stage {
							t.Fatal("wrong factory", stage, call)
						}
					default:
						t.Fatal("missing factory", stage)
					}
					got, err := loadOperation(r.s.db, op.ID)
					g02Check(t, err)
					want := "succeeded"
					if stage == "stop" && fault == "lost-stop" || stage == "resume" && fault == "resume-failure" {
						want = "interrupted"
					}
					if got.Status != want {
						t.Fatal(stage, got.Status, want)
					}
					assertContextHistory(t, r.s, history)
					replay, hit, err = queue()
					g02Check(t, err)
					if !hit || !reflect.DeepEqual(replay, got) {
						t.Fatal("terminal replay changed")
					}
					g02Check(t, validateProjectMemberContextReadiness(r.s.db))
					switch stage {
					case "stop":
						stopID = op.ID
					case "exit":
						exitID = op.ID
					case "retire":
						retiredID = op.ID
					case "resume":
						resumeID = op.ID
					}
					if stage == "resume" && fault == "resume-failure" {
						j, err := loadProjectMemberResume(r.s.db, op.ID)
						g02Check(t, err)
						if j.Resume == nil || j.Resume.Phase != "start-dispatched" {
							t.Fatal("lost Start uncertainty erased")
						}
						reopenContextQueueHistory(t, r)
						return
					}
				}
				// A new resume changes the peer context; the earlier current-readiness receipt
				// remains historical and cannot be reused to authorize another Stop.
				stale := req
				stale.RequestID = "stale-context-stop"
				if _, _, err := r.s.acceptProjectMemberStop("actor", p.ID, instance, stale); err == nil {
					t.Fatal("old current readiness became new authority")
				}
				reopenContextQueueHistory(t, r)
			})
		}
	}
}
