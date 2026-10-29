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

type queuedCurrentStopBackend struct {
	*queuedStopBackend
	selection chan runtimequalify.ProjectResumedMemberStopSelection
}

func (b *queuedCurrentStopBackend) OpenResumedProjectMemberStop(ctx context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, s.Project.ProjectID, s.Stop.InstanceID)
	if err != nil {
		b.t.Fatal(err)
	}
	resumes, err := completedProjectMemberResumes(b.s.db, p.ID)
	if err != nil {
		b.t.Fatal(err)
	}
	expected, err := projectnetwork.NewResumedMemberStop(p.Plan, launches, resumes, s.Readiness, s.Stop.ReadinessOperationID, s.Stop.OperationID)
	if err != nil || !reflect.DeepEqual(expected, s.Stop) || !reflect.DeepEqual(launches, s.Launches) || !reflect.DeepEqual(resumes, s.Resumes) || s.Project.RunnerID != b.s.runnerID || s.Project.Revision != p.Revision || len(s.StoppedPeers) != 0 {
		b.t.Fatal("lost current selection or included retired Stop", err)
	}
	b.selection <- s
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fail {
		return nil, errors.New("synthetic setup failure")
	}
	before, after := currentStopObservations(b.t, b.s, p.ID, s.Stop.InstanceID)
	return &storedStopDriver{t: b.t, s: b.s, id: s.Stop.OperationID, before: before, after: after}, nil
}
func currentStopQueueFixture(t *testing.T) (*Server, *queuedCurrentStopBackend, storedProject, projectMemberStopRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req, _ := readyCurrentStopProject(t, r.s)
	b := &queuedCurrentStopBackend{queuedStopBackend: &queuedStopBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectMemberStopSelection, 2), instance: p.Members[0].InstanceID}, selection: make(chan runtimequalify.ProjectResumedMemberStopSelection, 2)}
	r.b = b
	return r, b, p, req
}
func TestCurrentProjectMemberStopQueueRequiresBackendBeforeAdmission(t *testing.T) {
	r, b, p, req := currentStopQueueFixture(t)
	r.b = b.queuedStopBackend
	before := currentStopHistory(t, r.s)
	if _, _, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req); err == nil {
		t.Fatal("missing capability admitted")
	}
	if !reflect.DeepEqual(before, currentStopHistory(t, r.s)) {
		t.Fatal("unavailable backend created work")
	}
}

func TestCurrentProjectMemberStopQueueSharesAdmission(t *testing.T) {
	r, b, p, req := currentStopQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req)
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
	select {
	case <-b.selection:
		t.Fatal("duplicate factory call")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "second-observation"
	_, _, err = r.queueProjectMemberStop("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestCurrentProjectMemberStopQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := currentStopQueueFixture(t)
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
			op, _, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req)
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
			j, err := loadProjectMemberStop(r.s.db, op.ID)
			if err != nil || j.Stop.Phase != "planned" || j.Stop.Before != nil || j.Stop.After != nil {
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
