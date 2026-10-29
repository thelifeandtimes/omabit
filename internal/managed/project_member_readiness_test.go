package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func launchedReadinessProject(t *testing.T, s *store) (storedProject, projectMemberReadinessRequest) {
	t.Helper()
	p, req := preparedMemberLaunchProject(t, s)
	for i, m := range p.Members {
		req.RequestID = "launch-" + m.InstanceID
		op, _, err := s.acceptProjectMemberLaunch("actor", p.ID, m.InstanceID, req)
		if err != nil {
			t.Fatal(err)
		}
		initial := memberLaunchSelection(t, s, op)
		if _, err = s.beginProjectMemberLaunch(m.InstanceID, op.ID, initial); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < len(memberLaunchPhases)-1; n++ {
			advanceMemberLaunchFixture(t, s, op, n, fmt.Sprintf("%064x", 100+i))
		}
	}
	return p, projectMemberReadinessRequest{RequestID: "readiness", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
}
func memberReadinessProof(t *testing.T, s *store, project, instance string) projectnetwork.InitialMemberReadiness {
	t.Helper()
	p, launches, b, target, err := projectReadinessPrerequisites(s.db, project, instance)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	o := projectnetwork.Observation{ProjectID: b.ProjectID, Revision: b.Revision, Generation: b.Generation, EngineID: b.EngineID, ObservedAt: now, Complete: true, Bridge: projectnetwork.ResourceObservation{Resource: b.Bridge, Present: true}, BridgeAttachments: []string{b.Anchor.ID}, Anchor: projectnetwork.AnchorObservation{ResourceObservation: projectnetwork.ResourceObservation{Resource: b.Anchor, Present: true}, Running: true, Namespace: b.Namespace, BridgeID: b.Bridge.ID}}
	var location pierstore.Location
	var ship, endpoint string
	for _, l := range launches {
		if l.Storage.InstanceID == instance {
			location = l.Storage.Location
		}
	}
	location.Identity = pierstore.Identity{Device: 7, Inode: 800, UrbDevice: 7, UrbInode: 801, Birth: "1001:1", UrbBirth: "1001:2"}
	for _, m := range p.Plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: p.Plan.BindAddress, HostPort: m.HTTPPort, Protocol: "tcp"})
		if m.InstanceID == instance {
			ship = m.Ship
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	o.Anchor.Active = append([]projectnetwork.Publication(nil), o.Anchor.Requested...)
	for _, m := range b.Members {
		o.Members = append(o.Members, projectnetwork.MemberObservation{InstanceID: m.InstanceID, ResourceObservation: projectnetwork.ResourceObservation{Resource: m.Container, Present: true}, Running: true, Namespace: b.Namespace, AnchorID: b.Anchor.ID})
	}
	proof := projectnetwork.InitialMemberReadiness{InstanceID: instance, LaunchOperationID: target, Binding: b, Storage: location, Ship: ship, Endpoint: endpoint, Before: o, After: o, CapturedAt: now}
	if err = proof.Validate(p.Plan, launches); err != nil {
		t.Fatal(err)
	}
	return proof
}
func TestProjectMemberReadinessCommitRestartReplay(t *testing.T) {
	for _, phase := range []string{"planned", "observing", "observed"} {
		t.Run(phase, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := launchedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			op, hit, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			if phase != "planned" {
				if _, err = s.beginProjectMemberReadiness(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "observed" {
				if err = s.completeProjectMemberReadiness(op.ID, memberReadinessProof(t, s, p.ID, instance)); err != nil {
					t.Fatal(err)
				}
			}
			before, err := loadProjectMemberReadiness(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			originalLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			for i := 0; i < 2; i++ {
				if err = s.reconcileInterrupted(); err != nil {
					t.Fatal(err)
				}
			}
			after, _ := loadProjectMemberReadiness(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("restart rewrote historical evidence")
			}
			retained, hit, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
			expected := "interrupted"
			if phase == "observed" {
				expected = "succeeded"
			}
			if err != nil || !hit || retained.ID != op.ID || retained.Status != expected {
				t.Fatal(retained, hit, err)
			}
			if _, err = s.beginProjectMemberReadiness(op.ID); err == nil {
				t.Fatal("restarted old attempt")
			}
			if err = s.completeProjectMemberReadiness(op.ID, memberReadinessProof(t, s, p.ID, instance)); err == nil {
				t.Fatal("rewrote old attempt")
			}
			req.RequestID = "new-observation"
			next, hit, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
			if err != nil || hit || next.ID == op.ID {
				t.Fatal(next, hit, err)
			}
			currentLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			if !reflect.DeepEqual(originalLaunch, currentLaunch) {
				t.Fatal("observation changed launch")
			}
			jobs, _ := projectMemberReadinessJobs(s.db)
			if len(jobs) != 2 {
				t.Fatal("lost attempt history")
			}
		})
	}
}
func TestProjectMemberReadinessRejectsInvalidCompletion(t *testing.T) {
	for _, fault := range []string{"not-begun", "member", "launch", "binding", "ship", "endpoint", "storage", "namespace", "incomplete", "stale", "future"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := launchedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			if fault != "not-begun" {
				if _, err = s.beginProjectMemberReadiness(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			proof := memberReadinessProof(t, s, p.ID, instance)
			switch fault {
			case "member":
				proof.InstanceID = p.Members[1].InstanceID
			case "launch":
				proof.LaunchOperationID = "other"
			case "binding":
				proof.Binding.Namespace.Inode++
			case "ship":
				proof.Ship = "~bud"
			case "endpoint":
				proof.Endpoint = "http://127.0.0.1:1/~/host"
			case "storage":
				proof.Storage.MountIdentity.Inode++
			case "namespace":
				proof.After.Anchor.Namespace.Inode++
			case "incomplete":
				proof.Before.Complete = false
			case "stale":
				proof.CapturedAt = time.Now().Add(-31 * time.Second)
				proof.Before.ObservedAt = proof.CapturedAt
				proof.After.ObservedAt = proof.CapturedAt
			case "future":
				proof.CapturedAt = time.Now().Add(6 * time.Second)
				proof.Before.ObservedAt = proof.CapturedAt
				proof.After.ObservedAt = proof.CapturedAt
			}
			before, _ := loadProjectMemberReadiness(s.db, op.ID)
			if err = s.completeProjectMemberReadiness(op.ID, proof); err == nil {
				t.Fatal("accepted invalid proof")
			}
			after, _ := loadProjectMemberReadiness(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("partial evidence committed")
			}
		})
	}
}
func TestProjectMemberReadinessAdmissionAndConcurrency(t *testing.T) {
	s := testStore(t)
	p, req := launchedReadinessProject(t, s)
	instance := p.Members[0].InstanceID
	var wg sync.WaitGroup
	success := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.RequestID = fmt.Sprintf("observe-%d", i)
			op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, r)
			if err == nil {
				success <- op.ID
			}
		}(i)
	}
	wg.Wait()
	close(success)
	if len(success) != 1 {
		t.Fatal("overlapping observation admitted", len(success))
	}
	id := <-success
	if err := s.interruptMemberReadiness(id); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"actor", "revision", "review", "member", "replay-only"} {
		r := req
		a := "actor"
		m := instance
		switch fault {
		case "actor":
			a = ""
		case "revision":
			r.ExpectedRevision++
		case "review":
			r.ReviewDigest = "wrong"
		case "member":
			m = "other-member"
		case "replay-only":
			r.ReplayOnly = true
		}
		if _, _, err := s.acceptProjectMemberReadiness(a, p.ID, m, r); err == nil {
			t.Fatal(fault)
		}
	}
	op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	changed := req
	changed.ExpectedRevision++
	if _, _, err = s.acceptProjectMemberReadiness("actor", p.ID, instance, changed); err == nil {
		t.Fatal("changed actor/request fingerprint accepted")
	}
	req.ReplayOnly = true
	again, hit, err := s.acceptProjectMemberReadiness("actor", p.ID, instance, req)
	if err != nil || !hit || again.ID != op.ID {
		t.Fatal(again, hit, err)
	}
}
func TestProjectMemberReadinessRequiresCompleteLaunchSet(t *testing.T) {
	s := testStore(t)
	p, launchReq := preparedMemberLaunchProject(t, s)
	req := projectMemberReadinessRequest{RequestID: "observe", ExpectedRevision: 1, ReviewDigest: launchReq.ReviewDigest}
	if _, _, err := s.acceptProjectMemberReadiness("actor", p.ID, p.Members[0].InstanceID, req); err == nil {
		t.Fatal("observed unlaunched project")
	}
	jobs, _ := projectMemberReadinessJobs(s.db)
	if len(jobs) != 0 {
		t.Fatal("partial admission")
	}
}
func TestProjectMemberReadinessCorruptionRefusesOpen(t *testing.T) {
	for _, fault := range []string{"mirror", "orphan", "phase", "binding", "ownership"} {
		t.Run(fault, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := launchedReadinessProject(t, s)
			op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			j, _ := loadProjectMemberReadiness(s.db, op.ID)
			switch fault {
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_readiness WHERE operation_id=?", op.ID)
			default:
				switch fault {
				case "phase":
					j.Phase = "observed"
				case "binding":
					j.Binding.Namespace.Inode++
				case "ownership":
					j.LaunchOperationID = "other"
				}
				b, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_member_readiness SET body=? WHERE operation_id=?", b, op.ID)
				if err == nil {
					_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", b, op.ID)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			s.close()
			reopened, err := testOpenStore(dir)
			if err == nil {
				reopened.close()
				t.Fatal("corrupt evidence accepted")
			}
		})
	}
}

type storedReadinessDriver struct {
	t      *testing.T
	s      *store
	opID   string
	proof  projectnetwork.InitialMemberReadiness
	cancel context.CancelFunc
	reads  int
}

func (d *storedReadinessDriver) check() {
	d.t.Helper()
	j, err := loadProjectMemberReadiness(d.s.db, d.opID)
	if err != nil || j.Phase != "observing" {
		d.t.Fatal("observation before committed phase", err)
	}
	if err = validateProjectMemberReadiness(d.s.db); err != nil {
		d.t.Fatal(err)
	}
}
func (d *storedReadinessDriver) VerifyEngine(context.Context, string) error { d.check(); return nil }
func (d *storedReadinessDriver) InspectProject(context.Context) (projectnetwork.Observation, error) {
	d.check()
	return d.proof.Before, nil
}
func (d *storedReadinessDriver) InspectPier(context.Context, projectnetwork.MemberLaunch) (pierstore.Location, error) {
	d.check()
	return d.proof.Storage, nil
}
func (d *storedReadinessDriver) ReadHostIdentity(context.Context, string) (string, error) {
	d.check()
	d.reads++
	if d.cancel != nil {
		d.cancel()
	}
	return d.proof.Ship, nil
}
func TestProjectMemberReadinessWorkerCommittedEvidence(t *testing.T) {
	for _, fault := range []string{"pass", "wrong-ship", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := launchedReadinessProject(t, s)
			op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &storedReadinessDriver{t: t, s: s, opID: op.ID, proof: memberReadinessProof(t, s, p.ID, op.InstanceID)}
			if fault == "wrong-ship" {
				d.proof.Ship = "~bud"
			}
			if fault == "cancel" {
				d.cancel = cancel
			}
			err = s.runProjectMemberReadiness(ctx, op.ID, d)
			after, _ := loadOperation(s.db, op.ID)
			j, _ := loadProjectMemberReadiness(s.db, op.ID)
			if fault == "pass" {
				if err != nil || after.Status != "succeeded" || j.Evidence == nil {
					t.Fatal(after, err)
				}
			} else if err == nil || after.Status != "interrupted" || j.Evidence != nil {
				t.Fatal(after, err)
			}
			if d.reads != 1 {
				t.Fatal("unexpected identity read count")
			}
			if err = validateProjectMemberReadiness(s.db); err != nil {
				t.Fatal(err)
			}
		})
	}
}
