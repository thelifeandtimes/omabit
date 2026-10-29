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

type queuedContextReadinessBackend struct {
	*queuedProjectBackend
	selection      chan runtimequalify.ProjectCurrentReadinessSelection
	opID, instance string
	badBinding     bool
}

func (b *queuedContextReadinessBackend) OpenCurrentProjectReadiness(ctx context.Context, selection runtimequalify.ProjectCurrentReadinessSelection) (projectnetwork.CurrentReadinessDriver, error) {
	j, err := loadProjectMemberContextReadiness(b.s.db, b.opID)
	if err != nil {
		b.t.Fatal(err)
	}
	src, err := currentContextUnchanged(b.s.db, j)
	if err != nil {
		b.t.Fatal(err)
	}
	if selection.Project.RunnerID != b.s.runnerID || selection.Project.ProjectID != src.Project.ID || !reflect.DeepEqual(selection.Launches, src.Launches) || !reflect.DeepEqual(selection.Resumes, src.Resumes) || !reflect.DeepEqual(selection.Original, src.Original) {
		b.t.Fatal("current readiness lost retained history")
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
	proof := contextReadinessProof(b.t, b.s, b.opID)
	if b.badBinding {
		proof.Before.StartedAt = proof.Before.StartedAt.Add(time.Second)
	}
	return &contextReadinessStoreDriver{proof: proof}, nil
}
func contextReadinessQueueFixture(t *testing.T) (*Server, *queuedContextReadinessBackend, storedProject, projectMemberContextReadinessRequest) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, instance, req := contextReadinessProject(t, r.s, 0)
	b := &queuedContextReadinessBackend{queuedProjectBackend: original, selection: make(chan runtimequalify.ProjectCurrentReadinessSelection, 2), instance: instance}
	r.b = b
	return r, b, p, req
}
func TestProjectMemberContextReadinessQueueSharesAdmission(t *testing.T) {
	r, b, p, req := contextReadinessQueueFixture(t)
	r.workGate <- struct{}{}
	original, err := loadOperation(r.s.db, req.CreatorOperationID)
	if err != nil {
		t.Fatal(err)
	}
	op, hit, err := r.queueProjectMemberContextReadiness("actor", p.ID, b.instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	b.opID = op.ID
	again, hit, err := r.queueProjectMemberContextReadiness("actor", p.ID, b.instance, req)
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
	retained, err := loadOperation(r.s.db, req.CreatorOperationID)
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
	_, _, err = r.queueProjectMemberContextReadiness("actor", p.ID, b.instance, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberContextReadinessQueueRetainsFailures(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "changed-binding", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, p, req := contextReadinessQueueFixture(t)
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
			op, _, err := r.queueProjectMemberContextReadiness("actor", p.ID, b.instance, req)
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
			j, err := loadProjectMemberContextReadiness(r.s.db, op.ID)
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

// The queue uses inert observations only; no runtime is contacted.
type contextReadinessStoreDriver struct {
	proof projectnetwork.CurrentMemberReadiness
	reads int
}

func (d *contextReadinessStoreDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *contextReadinessStoreDriver) Inspect(_ context.Context, c projectnetwork.MemberContext) (projectnetwork.MemberStopObservation, error) {
	if !reflect.DeepEqual(c.Binding, d.proof.Binding) || c.Storage != d.proof.Storage {
		return projectnetwork.MemberStopObservation{}, errors.New("changed selection")
	}
	d.reads++
	if d.reads == 1 {
		return d.proof.Before, nil
	}
	return d.proof.After, nil
}
func (d *contextReadinessStoreDriver) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	if endpoint != d.proof.Endpoint {
		return "", errors.New("changed endpoint")
	}
	return d.proof.Ship, nil
}
