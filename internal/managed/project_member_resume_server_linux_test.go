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

type queuedResumeBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectMemberResumeSelection
	opID, instance string
	badBinding     bool
}

func (b *queuedResumeBackend) OpenProjectMemberResume(ctx context.Context, selection runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	job, err := loadProjectMemberResume(b.s.db, selection.OperationID)
	if err != nil {
		b.t.Fatal(err)
	}
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Stop.Project.ProjectID, b.instance)
	if err != nil {
		b.t.Fatal(err)
	}
	if !reflect.DeepEqual(selection.Stop.Launches, launches) || selection.Stop.Project.RunnerID != b.s.runnerID || selection.Stop.Project.Revision != p.Revision || !reflect.DeepEqual(selection.Retirement, job.Retirement) || !reflect.DeepEqual(selection.Stop.Stop, job.Retirement.Exit.Stop) {
		b.t.Fatal("resume lost original bindings")
	}
	b.selection <- selection
	select {
	case <-b.release:
	case <-ctx.Done():
		return projectnetwork.MemberResume{}, nil, ctx.Err()
	}
	if b.fail {
		return projectnetwork.MemberResume{}, nil, errors.New("synthetic open error")
	}
	initial := initialResumeFixture(b.t, b.s, job.OperationID)
	if b.badBinding {
		initial.PreviousGeneration++
	}
	return initial, &resumeStoreDriver{t: b.t, s: b.s, id: job.OperationID}, nil
}
func resumeQueueFixture(t *testing.T) (*Server, *queuedResumeBackend, storedProject, projectMemberResumeRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req := readyResumeProject(t, r.s)
	b := &queuedResumeBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectMemberResumeSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberResumeQueueSharesAdmission(t *testing.T) {
	r, b, p, req := resumeQueueFixture(t)
	r.workGate <- struct{}{}
	original, err := loadOperation(r.s.db, req.RetirementOperationID)
	if err != nil {
		t.Fatal(err)
	}
	op, hit, err := r.queueProjectMemberResume("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberResume("actor", p.ID, b.instance, req)
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
	retained, err := loadOperation(r.s.db, req.RetirementOperationID)
	if err != nil || !reflect.DeepEqual(retained, original) || retained.Status != "succeeded" {
		t.Fatal("resume changed original retirement", err)
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
	_, _, err = r.queueProjectMemberResume("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberResumeQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "changed-binding", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := resumeQueueFixture(t)
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
			op, _, err := r.queueProjectMemberResume("actor", p.ID, b.instance, req)
			if err != nil {
				t.Fatal(err)
			}
			b.opID = op.ID
			if mode == "cancel-waiting" {
				r.cancel()
			} else {
				<-r.workGate
			}
			if mode == "cancel-setup" || mode == "setup-failure" || mode == "changed-binding" {
				select {
				case <-b.selection:
				case <-time.After(2 * time.Second):
					t.Fatal("factory not reached")
				}
				if mode == "cancel-setup" {
					r.cancel()
				} else {
					if mode == "changed-binding" {
						b.badBinding = true
					} else {
						b.fail = true
					}
					close(b.release)
				}
			}
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			if err != nil || actual.Status != "interrupted" {
				t.Fatal(actual, err)
			}
			j, err := loadProjectMemberResume(r.s.db, op.ID)
			if err != nil || j.Resume != nil {
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
