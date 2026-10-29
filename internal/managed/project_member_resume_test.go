package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func readyResumeProject(t *testing.T, s *store) (storedProject, projectMemberResumeRequest) {
	t.Helper()
	p, req := readyRetirementProject(t, s)
	instance := p.Members[0].InstanceID
	op, hit, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	if _, err = s.beginProjectMemberRetirement(op.ID); err != nil {
		t.Fatal(err)
	}
	before, after := retirementStoreObservations(t, s, op.ID)
	advanceRetirementFixture(t, s, p, op.ID, "remove-dispatched", &before, nil)
	advanceRetirementFixture(t, s, p, op.ID, "removed", nil, nil)
	advanceRetirementFixture(t, s, p, op.ID, "absent", nil, &after)
	if err = s.finishProjectMemberRetirement(op.ID); err != nil {
		t.Fatal(err)
	}
	return p, projectMemberResumeRequest{RequestID: "resume-member", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, RetirementOperationID: op.ID}
}
func initialResumeFixture(t *testing.T, s *store, id string) projectnetwork.MemberResume {
	t.Helper()
	j, err := loadProjectMemberResume(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	p, err := loadProject(s.db, j.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := memberResumeGeneration(s.db, j.Retirement, false)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := projectnetwork.NewMemberResume(p.Plan, j.Retirement, previous, id, strings.Repeat("9", 64))
	if err != nil {
		t.Fatal(err)
	}
	return initial
}
func nextResumeFixture(t *testing.T, s *store, id, phase string) projectnetwork.MemberResume {
	t.Helper()
	j, err := loadProjectMemberResume(s.db, id)
	if err != nil || j.Resume == nil {
		t.Fatal(j, err)
	}
	p, err := loadProject(s.db, j.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	state := j.Resume
	var result projectnetwork.MemberResumeResult
	if phase == "create-dispatched" {
		raw, _ := json.Marshal(state.Retirement.After)
		var o projectnetwork.MemberAbsenceObservation
		if err = json.Unmarshal(raw, &o); err != nil {
			t.Fatal(err)
		}
		o.CapturedAt = time.Now().UTC()
		o.Runtime.ObservedAt = o.CapturedAt
		result.Before = &o
	}
	if phase == "created" {
		result.ContainerID = strings.Repeat("8", 64)
		if state.PreviousGeneration > 1 {
			result.ContainerID = fmt.Sprintf("%064x", state.PreviousGeneration)
		}
	}
	if phase == "start-dispatched" || phase == "running" {
		raw, _ := json.Marshal(state.Before.Runtime)
		var runtime projectnetwork.Observation
		if err = json.Unmarshal(raw, &runtime); err != nil {
			t.Fatal(err)
		}
		running := phase == "running"
		member := projectnetwork.MemberObservation{InstanceID: j.InstanceID, ResourceObservation: projectnetwork.ResourceObservation{Resource: state.Container, Present: true}, AnchorID: state.Retirement.Exit.Stop.Binding.Anchor.ID, Running: running}
		if running {
			member.Namespace = state.Retirement.Exit.Stop.Binding.Namespace
		}
		runtime.Members = append(runtime.Members, member)
		slices.SortFunc(runtime.Members, func(a, b projectnetwork.MemberObservation) int { return strings.Compare(a.InstanceID, b.InstanceID) })
		o := projectnetwork.MemberResumeObservation{Runtime: runtime, Storage: state.Retirement.Exit.Stop.Storage, CapturedAt: time.Now().UTC()}
		o.Runtime.ObservedAt = o.CapturedAt
		if running {
			o.StartedAt = o.CapturedAt
		}
		result.Observation = &o
	}
	next, err := state.Advance(p.Plan, state.Phase, phase, result)
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func TestProjectMemberResumeRestartRetainsEveryBoundary(t *testing.T) {
	cuts := []string{"accepted", "begun", "create-dispatched", "created", "start-dispatched", "started", "running", "finished"}
	for index, cut := range cuts {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := readyResumeProject(t, s)
			instance := p.Members[0].InstanceID
			retired, err := loadProjectMemberRetirement(s.db, req.RetirementOperationID)
			if err != nil {
				t.Fatal(err)
			}
			originalStop, err := loadOperation(s.db, retired.Retirement.Exit.Stop.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			op, hit, err := s.acceptProjectMemberResume("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			if index >= 1 {
				if _, err = s.beginProjectMemberResume(op.ID, initialResumeFixture(t, s, op.ID)); err != nil {
					t.Fatal(err)
				}
			}
			for n, phase := range cuts[2:7] {
				if index < n+2 {
					break
				}
				j, _ := loadProjectMemberResume(s.db, op.ID)
				next := nextResumeFixture(t, s, op.ID, phase)
				if err = s.advanceProjectMemberResume(op.ID, j.Resume.Phase, next); err != nil {
					t.Fatal(err)
				}
			}
			if cut == "finished" {
				if err = s.finishProjectMemberResume(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			retained, err := loadProjectMemberResume(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			for range 2 {
				if err = s.reconcileInterrupted(); err != nil {
					t.Fatal(err)
				}
			}
			got, err := loadProjectMemberResume(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(got, retained) {
				t.Fatal("restart changed resume journal", err)
			}
			actualRetirement, err := loadProjectMemberRetirement(s.db, req.RetirementOperationID)
			if err != nil || !reflect.DeepEqual(actualRetirement, retired) {
				t.Fatal("restart changed original retirement", err)
			}
			actualStop, err := loadOperation(s.db, retired.Retirement.Exit.Stop.OperationID)
			if err != nil || !reflect.DeepEqual(actualStop, originalStop) {
				t.Fatal("resume rewrote original Stop uncertainty", err)
			}
			replay, hit, err := s.acceptProjectMemberResume("actor", p.ID, instance, req)
			want := "interrupted"
			if cut == "finished" {
				want = "succeeded"
			}
			if err != nil || !hit || replay.ID != op.ID || replay.Status != want {
				t.Fatal(replay, hit, err)
			}
			if _, err = s.beginProjectMemberResume(op.ID, initialResumeFixture(t, s, op.ID)); err == nil {
				t.Fatal("restart replayed old intent")
			}
			req.RequestID = "another-resume"
			if _, _, err = s.acceptProjectMemberResume("actor", p.ID, instance, req); err == nil {
				t.Fatal("consumed retirement twice")
			}
			var mirror []byte
			if err = s.db.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
				t.Fatal(err)
			}
			body, _ := json.Marshal(got)
			if string(body) != string(mirror) {
				t.Fatal("resume mirror changed")
			}
		})
	}
}
func TestProjectMemberResumeRejectsChangedTransitionAtomically(t *testing.T) {
	for _, mode := range []string{"generation", "retirement", "operation", "digest", "stale-capture", "stale-runtime", "skipped", "wrong-expected"} {
		t.Run(mode, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, req := readyResumeProject(t, s)
			op, _, err := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.beginProjectMemberResume(op.ID, initialResumeFixture(t, s, op.ID)); err != nil {
				t.Fatal(err)
			}
			before, _ := loadProjectMemberResume(s.db, op.ID)
			next := nextResumeFixture(t, s, op.ID, "create-dispatched")
			expected := "planned"
			switch mode {
			case "generation":
				next.PreviousGeneration++
			case "retirement":
				next.Retirement.OperationID = "other-retirement"
			case "operation":
				next.OperationID = "other-resume"
			case "digest":
				next.Container.ContractDigest = strings.Repeat("a", 64)
			case "stale-capture":
				next.Before.CapturedAt = next.Before.CapturedAt.Add(-time.Minute)
			case "stale-runtime":
				next.Before.Runtime.ObservedAt = next.Before.Runtime.ObservedAt.Add(-time.Minute)
			case "skipped":
				next.Phase = "created"
				next.Container.ID = strings.Repeat("8", 64)
			case "wrong-expected":
				expected = "created"
			}
			if err = s.advanceProjectMemberResume(op.ID, expected, next); err == nil {
				t.Fatal("invalid transition accepted")
			}
			after, _ := loadProjectMemberResume(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("refused transition changed journal")
			}
			if err = validateProjectMemberResumes(s.db); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type resumeStoreDriver struct {
	t               *testing.T
	s               *store
	id, fault       string
	creates, starts int
}

func (d *resumeStoreDriver) check(phase string) {
	d.t.Helper()
	j, err := loadProjectMemberResume(d.s.db, d.id)
	if err != nil || j.Resume == nil || j.Resume.Phase != phase {
		d.t.Fatal("effect preceded resume journal", phase, err)
	}
	op, err := loadOperation(d.s.db, d.id)
	if err != nil || op.Status != "running" {
		d.t.Fatal("effect lacks running receipt", err)
	}
	if err = validateProjectMemberResumes(d.s.db); err != nil {
		d.t.Fatal(err)
	}
}
func (d *resumeStoreDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *resumeStoreDriver) InspectAbsent(context.Context, projectnetwork.MemberResume) (projectnetwork.MemberAbsenceObservation, error) {
	d.check("planned")
	return *nextResumeFixture(d.t, d.s, d.id, "create-dispatched").Before, nil
}
func (d *resumeStoreDriver) Create(_ context.Context, state projectnetwork.MemberResume) (string, error) {
	d.check("create-dispatched")
	d.creates++
	id := strings.Repeat("8", 64)
	if state.PreviousGeneration > 1 {
		id = fmt.Sprintf("%064x", state.PreviousGeneration)
	}
	if d.fault == "lost-create" {
		return "", errors.New("lost reply")
	}
	if d.fault == "late-create" {
		return id, errors.New("late error")
	}
	return id, nil
}
func (d *resumeStoreDriver) Inspect(_ context.Context, state projectnetwork.MemberResume) (projectnetwork.MemberResumeObservation, error) {
	d.check(state.Phase)
	if state.Phase == "created" {
		return *nextResumeFixture(d.t, d.s, d.id, "start-dispatched").Created, nil
	}
	return *nextResumeFixture(d.t, d.s, d.id, "running").After, nil
}
func (d *resumeStoreDriver) Start(context.Context, projectnetwork.MemberResume) error {
	d.check("start-dispatched")
	d.starts++
	if d.fault == "lost-start" {
		return errors.New("lost reply")
	}
	return nil
}
func TestProjectMemberResumeWorkerPreservesUncertainty(t *testing.T) {
	for _, c := range []struct {
		fault, phase    string
		creates, starts int
	}{{"", "running", 1, 1}, {"cancel-before", "planned", 0, 0}, {"lost-create", "create-dispatched", 1, 0}, {"late-create", "created", 1, 0}, {"lost-start", "start-dispatched", 1, 1}} {
		t.Run(c.fault, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, req := readyResumeProject(t, s)
			op, _, err := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			d := &resumeStoreDriver{t: t, s: s, id: op.ID, fault: c.fault}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if c.fault == "cancel-before" {
				cancel()
			}
			err = s.runProjectMemberResume(ctx, op.ID, initialResumeFixture(t, s, op.ID), d)
			if (err == nil) != (c.fault == "") {
				t.Fatal(err)
			}
			j, err := loadProjectMemberResume(s.db, op.ID)
			if err != nil || j.Resume == nil || j.Resume.Phase != c.phase || d.creates != c.creates || d.starts != c.starts {
				t.Fatal(j, err, d.creates, d.starts)
			}
			receipt, _ := loadOperation(s.db, op.ID)
			want := "interrupted"
			if c.fault == "" {
				want = "succeeded"
			}
			if receipt.Status != want {
				t.Fatal(receipt)
			}
		})
	}
}
func TestProjectMemberResumeRefusesCorruptMirrorsAndMissingJournal(t *testing.T) {
	for _, mode := range []string{"mirror", "body", "orphan"} {
		t.Run(mode, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := readyResumeProject(t, s)
			op, _, err := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", []byte(`{}`), op.ID)
			case "body":
				_, err = s.db.Exec("UPDATE project_member_resumes SET body=? WHERE operation_id=?", []byte(`{}`), op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_resumes WHERE operation_id=?", op.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := testOpenStore(dir)
			if err == nil {
				reopened.close()
				t.Fatal("opened corrupt resume journal")
			}
		})
	}
}
func TestProjectMemberResumeAdmissionSerializesAndFences(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, req := readyResumeProject(t, s)
	instance := p.Members[0].InstanceID
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.RequestID = fmt.Sprintf("resume-%d", i)
			_, hit, err := s.acceptProjectMemberResume("actor", p.ID, instance, r)
			if err == nil && !hit {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if accepted != 1 {
		t.Fatal("overlapping resume intents", accepted)
	}
	readyReq := projectMemberReadinessRequest{RequestID: "observe-after-resume-queued", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
	if _, _, err = s.acceptProjectMemberReadiness("actor", p.ID, instance, readyReq); err == nil {
		t.Fatal("unresolved resume allowed new same-project observation")
	}
	jobs, err := projectMemberResumeJobs(s.db)
	if err != nil || len(jobs) != 1 {
		t.Fatal(jobs, err)
	}
	if err = s.interruptMemberResume(jobs[0].OperationID); err != nil {
		t.Fatal(err)
	}
	if err = projectRetirementClear(s.db, p.ID, ""); err == nil {
		t.Fatal("interrupted resume lost project fence")
	}
	if err = projectRetirementClear(s.db, "different-project", ""); err != nil {
		t.Fatal("resume fenced unrelated project", err)
	}
}
