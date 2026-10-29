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

type queuedStopExitBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectMemberStopSelection
	opID, instance string
}

func (b *queuedStopExitBackend) OpenProjectMemberStopExit(ctx context.Context, selection runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(b.s.db, selection.Project.ProjectID, b.instance)
	if err != nil {
		b.t.Fatal(err)
	}
	if !reflect.DeepEqual(selection.Launches, launches) || selection.Project.RunnerID != b.s.runnerID || selection.Project.OperationID != launches[0].Anchor.Prepared.OperationID || selection.Project.Revision != p.Revision {
		b.t.Fatal("Stop exit selection lost original launch binding")
	}
	var id string
	if err = b.s.db.QueryRow("SELECT j.operation_id FROM project_member_stop_exits j JOIN operations o ON j.operation_id=o.id WHERE j.stop_operation_id=? AND json_extract(o.body,'$.status')='queued'", selection.Stop.OperationID).Scan(&id); err != nil {
		b.t.Fatal(err)
	}
	job, err := loadProjectMemberStopExit(b.s.db, id)
	if err != nil || !reflect.DeepEqual(job.Stop, selection.Stop) {
		b.t.Fatal("Stop exit selection changed retained process", err)
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
	proof := memberStopExitProof(b.t, b.s, id)
	if selection.Stop.After != nil {
		proof.Observation = *selection.Stop.After
		proof.Observation.CapturedAt = time.Now().UTC()
		proof.Observation.Runtime.ObservedAt = proof.Observation.CapturedAt
	}
	return &storedStopExitDriver{t: b.t, s: b.s, id: id, proof: proof}, nil
}
func stopExitQueueFixture(t *testing.T) (*Server, *queuedStopExitBackend, storedProject, projectMemberStopExitRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, req := interruptedStopProject(t, r.s)
	b := &queuedStopExitBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectMemberStopSelection, 2), instance: p.Members[0].InstanceID}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberStopExitQueueSharesAdmission(t *testing.T) {
	r, b, p, req := stopExitQueueFixture(t)
	r.workGate <- struct{}{}
	original, err := loadOperation(r.s.db, req.StopOperationID)
	if err != nil {
		t.Fatal(err)
	}
	op, hit, err := r.queueProjectMemberStopExit("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberStopExit("actor", p.ID, b.instance, req)
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
	retained, err := loadOperation(r.s.db, req.StopOperationID)
	if err != nil || !reflect.DeepEqual(retained, original) || retained.Status != "interrupted" {
		t.Fatal("exit observation changed original Stop", err)
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
	_, _, err = r.queueProjectMemberStopExit("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberStopExitQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := stopExitQueueFixture(t)
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
			op, _, err := r.queueProjectMemberStopExit("actor", p.ID, b.instance, req)
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
			j, err := loadProjectMemberStopExit(r.s.db, op.ID)
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
func TestProjectMemberStopExitQueueRetainsPeerExitAndRefusesUnresolvedPeer(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "unresolved", true: "complete"}[complete], func(t *testing.T) {
			r, b, p, req := stopQueueFixture(t)
			first, _, err := r.queueProjectMemberStop("actor", p.ID, b.instance, req)
			if err != nil {
				t.Fatal(err)
			}
			close(b.release)
			waitProjectWorker(t, r)
			firstOp, err := loadOperation(r.s.db, first.ID)
			if err != nil || firstOp.Status != "succeeded" {
				t.Fatal(firstOp, err)
			}
			secondInstance := p.Members[1].InstanceID
			req.RequestID = "stop-peer"
			if err = r.s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE instance_id=?", secondInstance).Scan(&req.ReadinessOperationID); err != nil {
				t.Fatal(err)
			}
			second, _, err := r.s.acceptProjectMemberStop("actor", p.ID, secondInstance, req)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.s.beginProjectMemberStop(second.ID); err != nil {
				t.Fatal(err)
			}
			before, after := storeStopObservations(t, r.s, p.ID, secondInstance)
			for _, o := range []*projectnetwork.MemberStopObservation{&before, &after} {
				for i := range o.Runtime.Members {
					if o.Runtime.Members[i].InstanceID == b.instance {
						o.Runtime.Members[i].Running = false
						o.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
					}
				}
			}
			advanceStopFixture(t, r.s, p, second.ID, "stop-dispatched", &before)
			if complete {
				advanceStopFixture(t, r.s, p, second.ID, "stop-signalled", nil)
				advanceStopFixture(t, r.s, p, second.ID, "stopped", &after)
				if err = r.s.finishProjectMemberStop(second.ID); err != nil {
					t.Fatal(err)
				}
			} else if err = r.s.interruptMemberStop(second.ID); err != nil {
				t.Fatal(err)
			}
			peerID, targetID, targetInstance := second.ID, first.ID, b.instance
			if complete {
				peerID, targetID, targetInstance = first.ID, second.ID, secondInstance
			}
			peer, err := loadProjectMemberStop(r.s.db, peerID)
			if err != nil {
				t.Fatal(err)
			}
			exitBackend := &queuedStopExitBackend{queuedProjectBackend: b.queuedProjectBackend, selection: make(chan runtimequalify.ProjectMemberStopSelection, 2), instance: targetInstance}
			r.b = exitBackend
			request := projectMemberStopExitRequest{RequestID: "observe-first", StopOperationID: targetID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
			observed, _, err := r.queueProjectMemberStopExit("actor", p.ID, targetInstance, request)
			if err != nil {
				t.Fatal(err)
			}
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, observed.ID)
			if err != nil {
				t.Fatal(err)
			}
			if complete {
				if actual.Status != "succeeded" {
					t.Fatal(actual)
				}
				select {
				case selected := <-exitBackend.selection:
					if len(selected.StoppedPeers) != 1 || !reflect.DeepEqual(selected.StoppedPeers[0], peer.Stop) {
						t.Fatal("lost prior peer process exit")
					}
				default:
					t.Fatal("complete peer never reached factory")
				}
			} else {
				if actual.Status != "interrupted" {
					t.Fatal(actual)
				}
				select {
				case <-exitBackend.selection:
					t.Fatal("unresolved peer reached factory")
				default:
				}
			}
			unchanged, err := loadOperation(r.s.db, first.ID)
			if err != nil || !reflect.DeepEqual(unchanged, firstOp) {
				t.Fatal("observation rewrote original Stop", err)
			}
		})
	}
}
