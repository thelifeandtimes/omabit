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

type queuedStopBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectMemberStopSelection
	opID, instance string
}

func (b *queuedStopBackend) OpenProjectMemberStop(ctx context.Context, selection runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Project.ProjectID, selection.Stop.InstanceID)
	if err != nil {
		b.t.Fatal(err)
	}
	j, err := loadProjectMemberStop(b.s.db, selection.Stop.OperationID)
	if err != nil || !reflect.DeepEqual(j.Stop, selection.Stop) || !reflect.DeepEqual(selection.Launches, launches) || selection.Project.RunnerID != b.s.runnerID || selection.Project.OperationID != launches[0].Anchor.Prepared.OperationID || selection.Project.Revision != p.Revision {
		b.t.Fatal("Stop selection lost accepted binding", err)
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
	before, after := storeStopObservations(b.t, b.s, p.ID, selection.Stop.InstanceID)
	for _, peer := range selection.StoppedPeers {
		for _, o := range []*projectnetwork.MemberStopObservation{&before, &after} {
			for i := range o.Runtime.Members {
				if o.Runtime.Members[i].InstanceID == peer.InstanceID {
					o.Runtime.Members[i].Running = false
					o.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
				}
			}
		}
	}
	return &storedStopDriver{t: b.t, s: b.s, id: selection.Stop.OperationID, before: before, after: after}, nil
}
func stopQueueFixture(t *testing.T) (*Server, *queuedStopBackend, storedProject, projectMemberStopRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req, _ := readyStopProject(t, r.s)
	b := &queuedStopBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectMemberStopSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberStopQueueSharesAdmission(t *testing.T) {
	r, b, p, req := stopQueueFixture(t)
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
func TestProjectMemberStopQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := stopQueueFixture(t)
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

func TestProjectMemberStopQueuePassesCompletePeerExitAndPreservesReplay(t *testing.T) {
	r, b, p, req := stopQueueFixture(t)
	first, hit, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(first, hit, err)
	}
	select {
	case selected := <-b.selection:
		if len(selected.StoppedPeers) != 0 {
			t.Fatal("invented peer exit")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first factory not reached")
	}
	close(b.release)
	waitProjectWorker(t, r)
	original, err := loadProjectMemberStop(r.s.db, first.ID)
	if err != nil || !original.Stop.Clean() {
		t.Fatal(original, err)
	}
	instance := p.Members[1].InstanceID
	req2 := req
	req2.RequestID = "stop-second"
	if err = r.s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE instance_id=?", instance).Scan(&req2.ReadinessOperationID); err != nil {
		t.Fatal(err)
	}
	second, hit, err := r.queueProjectMemberStop("actor", p.ID, instance, req2)
	if err != nil || hit {
		t.Fatal(second, hit, err)
	}
	select {
	case selected := <-b.selection:
		if selected.Stop.InstanceID != instance || len(selected.StoppedPeers) != 1 || !reflect.DeepEqual(selected.StoppedPeers[0], original.Stop) {
			t.Fatal("second Stop lost exact prior exit")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second factory not reached")
	}
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, second.ID)
	if err != nil || actual.Status != "succeeded" {
		t.Fatal(actual, err)
	}
	retained, err := loadProjectMemberStop(r.s.db, first.ID)
	if err != nil || !reflect.DeepEqual(retained, original) {
		t.Fatal("peer receipt changed", err)
	}
	replay, hit, err := r.queueProjectMemberStop("actor", p.ID, instance, req2)
	if err != nil || !hit || replay.ID != second.ID {
		t.Fatal(replay, hit, err)
	}
	select {
	case <-b.selection:
		t.Fatal("replay scheduled new Stop")
	default:
	}
}
