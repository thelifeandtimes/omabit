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

type queuedResumedReadinessBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectResumedReadinessSelection
	opID, instance string
	badBinding     bool
}

func (b *queuedResumedReadinessBackend) OpenResumedProjectReadiness(ctx context.Context, selection runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	job, err := loadProjectMemberResume(b.s.db, selection.Resume.OperationID)
	if err != nil {
		b.t.Fatal(err)
	}
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Resume.Stop.Project.ProjectID, b.instance)
	if err != nil {
		b.t.Fatal(err)
	}
	if !reflect.DeepEqual(selection.Resume.Stop.Launches, launches) || selection.Resume.Stop.Project.RunnerID != b.s.runnerID || selection.Resume.Stop.Project.Revision != p.Revision || !reflect.DeepEqual(selection.Completed, *job.Resume) || !reflect.DeepEqual(selection.Resume.Retirement, job.Retirement) {
		b.t.Fatal("readiness lost original resume chain")
	}
	b.selection <- selection
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fail {
		return nil, errors.New("synthetic open error")
	}
	proof := resumedReadinessProof(b.t, b.s, p.ID, b.instance, job.OperationID)
	if b.badBinding {
		proof.Before.StartedAt = proof.Before.StartedAt.Add(time.Second)
	}
	return &resumedReadinessStoreDriver{proof: proof}, nil
}
func resumedReadinessQueueFixture(t *testing.T) (*Server, *queuedResumedReadinessBackend, storedProject, projectMemberResumedReadinessRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req := resumedReadinessProject(t, r.s)
	b := &queuedResumedReadinessBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectResumedReadinessSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberResumedReadinessQueueSharesAdmission(t *testing.T) {
	r, b, p, req := resumedReadinessQueueFixture(t)
	r.workGate <- struct{}{}
	original, err := loadOperation(r.s.db, req.ResumeOperationID)
	if err != nil {
		t.Fatal(err)
	}
	op, hit, err := r.queueProjectMemberResumedReadiness("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberResumedReadiness("actor", p.ID, b.instance, req)
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
	retained, err := loadOperation(r.s.db, req.ResumeOperationID)
	if err != nil || !reflect.DeepEqual(retained, original) || retained.Status != "succeeded" {
		t.Fatal("readiness changed original replacement", err)
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
	_, _, err = r.queueProjectMemberResumedReadiness("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberResumedReadinessQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "changed-binding", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := resumedReadinessQueueFixture(t)
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
			op, _, err := r.queueProjectMemberResumedReadiness("actor", p.ID, b.instance, req)
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
			j, err := loadProjectMemberResumedReadiness(r.s.db, op.ID)
			if err != nil || j.Evidence != nil {
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
