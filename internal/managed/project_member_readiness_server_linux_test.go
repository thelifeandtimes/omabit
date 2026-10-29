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

type queuedReadinessBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectInitialReadinessSelection
	opID, instance string
}

func (b *queuedReadinessBackend) OpenInitialProjectReadiness(ctx context.Context, selection runtimequalify.ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Project.ProjectID, b.instance)
	if err != nil {
		b.t.Fatal(err)
	}
	if !reflect.DeepEqual(selection.Launches, launches) || selection.Project.RunnerID != b.s.runnerID || selection.Project.OperationID != launches[0].Anchor.Prepared.OperationID || selection.Project.Revision != p.Revision {
		b.t.Fatal("readiness selection lost launch binding")
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
	return &storedReadinessDriver{t: b.t, s: b.s, opID: b.opID, proof: memberReadinessProof(b.t, b.s, p.ID, b.instance)}, nil
}
func readinessQueueFixture(t *testing.T) (*Server, *queuedReadinessBackend, storedProject, projectMemberReadinessRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req := launchedReadinessProject(t, r.s)
	b := &queuedReadinessBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectInitialReadinessSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberReadinessQueueSharesAdmission(t *testing.T) {
	r, b, p, req := readinessQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectMemberReadiness("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberReadiness("actor", p.ID, b.instance, req)
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
	_, _, err = r.queueProjectMemberReadiness("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberReadinessQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := readinessQueueFixture(t)
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
			op, _, err := r.queueProjectMemberReadiness("actor", p.ID, b.instance, req)
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
			j, err := loadProjectMemberReadiness(r.s.db, op.ID)
			if err != nil || j.Phase != "planned" || j.Evidence != nil {
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
