//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type queuedRetirementBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectMemberRetirementSelection
	opID, instance string
}

func (b *queuedRetirementBackend) OpenProjectMemberRetirement(ctx context.Context, selection runtimequalify.ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	job, err := loadProjectMemberRetirement(b.s.db, selection.OperationID)
	if err != nil {
		b.t.Fatal(err)
	}
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Stop.Project.ProjectID, b.instance)
	if err != nil {
		b.t.Fatal(err)
	}
	if !reflect.DeepEqual(selection.Stop.Launches, launches) || selection.Stop.Project.RunnerID != b.s.runnerID || selection.Stop.Project.Revision != p.Revision || !reflect.DeepEqual(selection.Exit, job.Retirement.Exit) || !reflect.DeepEqual(selection.Stop.Stop, job.Retirement.Exit.Stop) {
		b.t.Fatal("retirement lost original bindings")
	}
	b.selection <- selection
	select {
	case <-b.release:
	case <-ctx.Done():
		return projectnetwork.MemberRetirement{}, nil, ctx.Err()
	}
	if b.fail {
		return projectnetwork.MemberRetirement{}, nil, errors.New("synthetic open error")
	}
	before, after := retirementStoreObservations(b.t, b.s, job.OperationID)
	return job.Retirement, &storedRetirementDriver{t: b.t, s: b.s, id: job.OperationID, before: before, after: after}, nil
}
func retirementQueueFixture(t *testing.T) (*Server, *queuedRetirementBackend, storedProject, projectMemberRetirementRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req := readyRetirementProject(t, r.s)
	b := &queuedRetirementBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectMemberRetirementSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberRetirementQueueSharesAdmission(t *testing.T) {
	r, b, p, req := retirementQueueFixture(t)
	r.workGate <- struct{}{}
	original, err := loadOperation(r.s.db, req.ExitOperationID)
	if err != nil {
		t.Fatal(err)
	}
	op, hit, err := r.queueProjectMemberRetirement("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberRetirement("actor", p.ID, b.instance, req)
	if err != nil || !hit || again.ID != op.ID {
		t.Fatal(again, hit, err)
	}
	select {
	case <-b.selection:
		t.Fatal("factory bypassed work gate")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	<-r.workGate
	select {
	case <-b.selection:
	case <-time.After(2 * time.Second):
		t.Fatal("factory not reached")
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
		t.Fatal(w.Code)
	}
	close(b.release)
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil || actual.Status != "succeeded" {
		t.Fatal(actual, err)
	}
	retained, err := loadOperation(r.s.db, req.ExitOperationID)
	if err != nil || !reflect.DeepEqual(retained, original) || retained.Status != "succeeded" {
		t.Fatal("retirement changed exit observation", err)
	}
	select {
	case <-b.selection:
		t.Fatal("duplicate factory call")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "second-observation"
	_, _, err = r.queueProjectMemberRetirement("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberRetirementQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := retirementQueueFixture(t)
			r.workGate <- struct{}{}
			if mode == "retained-maintenance" {
				if _, _, err := r.s.register("actor", registration("legacy", "legacy", 18580)); err != nil {
					t.Fatal(err)
				}
				v, err := loadInstance(r.s.db, "legacy")
				if err != nil {
					t.Fatal(err)
				}
				v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{Kind: "pack", CreateUncertain: true})
				tx, err := r.s.db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if err = saveInstance(tx, v); err != nil {
					tx.Rollback()
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			op, _, err := r.queueProjectMemberRetirement("actor", p.ID, b.instance, req)
			if err != nil {
				t.Fatal(err)
			}
			b.opID = op.ID
			if mode == "cancel-waiting" {
				r.cancel()
			} else {
				<-r.workGate
			}
			if mode == "cancel-setup" || mode == "setup-failure" {
				select {
				case <-b.selection:
				case <-time.After(2 * time.Second):
					t.Fatal("factory not reached")
				}
				if mode == "cancel-setup" {
					r.cancel()
				} else {
					b.fail = true
					close(b.release)
				}
			}
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			if err != nil || actual.Status != "interrupted" {
				t.Fatal(actual, err)
			}
			j, err := loadProjectMemberRetirement(r.s.db, op.ID)
			if err != nil || j.Retirement.Phase != "planned" || j.Retirement.Before != nil || j.Retirement.After != nil {
				t.Fatal(j, err)
			}
			r.mu.Lock()
			workers := r.updateWorkers
			r.mu.Unlock()
			if workers != 0 || r.failed.Load() {
				t.Fatal("worker fence leaked")
			}
		})
	}
}
func TestProjectMemberRetirementFencesOnlyNewWorkInItsProject(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, req := readyRetirementProject(t, s)
	instance := p.Members[0].InstanceID
	op, _, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	j, err := loadProjectMemberRetirement(s.db, op.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range p.Members {
		_, _, err = s.acceptProjectMemberReadiness("actor", p.ID, member.InstanceID, projectMemberReadinessRequest{RequestID: "new-readiness-" + member.InstanceID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
		requireFault(t, err, "project_recovery_required")
	}
	_, _, err = s.acceptProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: "new-exit", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, StopOperationID: j.Retirement.Exit.Stop.OperationID})
	requireFault(t, err, "project_recovery_required")
	replay, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, projectMemberStopRequest{RequestID: "stop-member", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, ReadinessOperationID: j.Retirement.Exit.Stop.ReadinessOperationID})
	if err != nil || !hit || replay.ID != j.Retirement.Exit.Stop.OperationID || replay.Status != "interrupted" {
		t.Fatal("retirement hid retained Stop receipt", replay, hit, err)
	}
	if err = projectRetirementClear(s.db, "another-project", ""); err != nil {
		t.Fatal("unrelated project fenced", err)
	}
	if err = projectRetirementClear(s.db, p.ID, op.ID); err != nil {
		t.Fatal("own admitted worker fenced", err)
	}
	if err = s.interruptMemberRetirement(op.ID); err != nil {
		t.Fatal(err)
	}
	requireFault(t, projectRetirementClear(s.db, p.ID, ""), "project_recovery_required")
}

func TestProjectMemberRetirementWaitsForPreviouslyAcceptedProjectWork(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, req := readyRetirementProject(t, s)
	instance := p.Members[0].InstanceID
	_, _, err = s.acceptProjectMemberReadiness("actor", p.ID, instance, projectMemberReadinessRequest{RequestID: "already-accepted-read", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
	requireFault(t, err, "project_operation_pending")
}
