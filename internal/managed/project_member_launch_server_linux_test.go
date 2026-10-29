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

type queuedMemberLaunchBackend struct {
	*queuedProjectBackend
	memberOpened chan runtimequalify.ProjectMemberLaunchSelection
}

func (b *queuedMemberLaunchBackend) OpenProjectMemberLaunch(ctx context.Context, selection runtimequalify.ProjectMemberLaunchSelection) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
	op, err := loadOperation(b.s.db, selection.OperationID)
	if err != nil {
		b.t.Fatal(err)
	}
	initial := memberLaunchSelection(b.t, b.s, op)
	if !reflect.DeepEqual(initial.Anchor, selection.Anchor) || initial.Storage != selection.Storage || selection.Project.RunnerID != b.s.runnerID || selection.Project.OperationID != initial.Anchor.Prepared.OperationID {
		b.t.Fatal("setup lost accepted receipt binding")
	}
	b.memberOpened <- selection
	select {
	case <-b.release:
	case <-ctx.Done():
		return projectnetwork.MemberLaunch{}, nil, ctx.Err()
	}
	if b.fail {
		return projectnetwork.MemberLaunch{}, nil, errors.New("synthetic setup failure")
	}
	return initial, &storedMemberLaunchDriver{t: b.t, s: b.s, instance: op.InstanceID, fault: "success"}, nil
}
func memberLaunchQueueFixture(t *testing.T) (*Server, *queuedMemberLaunchBackend, storedProject, projectMemberLaunchRequest) {
	t.Helper()
	r, b, _, _ := projectQueueFixture(t)
	p, req := preparedMemberLaunchProject(t, r.s)
	backend := &queuedMemberLaunchBackend{queuedProjectBackend: b, memberOpened: make(chan runtimequalify.ProjectMemberLaunchSelection, 2)}
	r.b = backend
	return r, backend, p, req
}
func TestProjectMemberLaunchQueueUsesResourceAndUpdateFences(t *testing.T) {
	r, b, p, req := memberLaunchQueueFixture(t)
	instance := p.Members[0].InstanceID
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectMemberLaunch("actor", p.ID, instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	again, hit, err := r.queueProjectMemberLaunch("actor", p.ID, instance, req)
	if err != nil || !hit || again.ID != op.ID {
		t.Fatal(again, hit, err)
	}
	select {
	case <-b.memberOpened:
		t.Fatal("factory bypassed work gate")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 || !strings.Contains(w.Body.String(), "update_busy") {
		t.Fatal(w.Code, w.Body.String())
	}
	<-r.workGate
	select {
	case s := <-b.memberOpened:
		if s.OperationID != op.ID || s.Storage.InstanceID != instance {
			t.Fatal(s)
		}
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
	case <-b.memberOpened:
		t.Fatal("duplicate factory call")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "second-member"
	_, _, err = r.queueProjectMemberLaunch("actor", p.ID, p.Members[1].InstanceID, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberLaunchQueueFailuresKeepUnboundReceipt(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := memberLaunchQueueFixture(t)
			instance := p.Members[0].InstanceID
			if mode == "cancel-waiting" {
				r.workGate <- struct{}{}
			}
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
			op, _, err := r.queueProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "cancel-waiting":
				r.cancel()
			case "cancel-setup", "setup-failure":
				select {
				case <-b.memberOpened:
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
			j, err := loadProjectMemberLaunch(r.s.db, instance)
			if err != nil || j.Launch != nil {
				t.Fatal("setup failure acquired launch authority", j, err)
			}
			again, hit, err := r.s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil || !hit || again.ID != op.ID || again.Status != "interrupted" {
				t.Fatal(again, hit, err)
			}
			r.mu.Lock()
			workers := r.updateWorkers
			r.mu.Unlock()
			if workers != 0 || r.failed.Load() {
				t.Fatal("worker/update fence leaked", workers)
			}
		})
	}
}
