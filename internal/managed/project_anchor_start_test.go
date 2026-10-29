package managed

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func completeProjectPreparation(t *testing.T, s *store, op protocol.Operation) {
	t.Helper()
	prepared := projectPreparationFixture(t, s, op.ProjectID, op.ID)
	engine := &storedPreparationEngine{t: t, s: s, id: op.ProjectID}
	if err := s.runProjectPreparation(context.Background(), op.ProjectID, op.ID, prepared, engine); err != nil {
		t.Fatal(err)
	}
}
func acceptAnchorStartFixture(t *testing.T, s *store) (protocol.Operation, projectAnchorStartRequest) {
	t.Helper()
	prep, _ := acceptProjectFixture(t, s)
	completeProjectPreparation(t, s, prep)
	p, err := loadProject(s.db, prep.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	req := projectAnchorStartRequest{RequestID: "start-anchor", ExpectedRevision: 1, ReviewDigest: digest}
	op, hit, err := s.acceptProjectAnchorStart("actor", p.ID, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	return op, req
}
func anchorNamespaceFixture() *projectnetwork.NamespaceIdentity {
	return &projectnetwork.NamespaceIdentity{BootID: "00000000-0000-0000-0000-000000000001", Device: 4, Inode: 1001}
}
func TestProjectAnchorStartSurvivesEveryCommitBoundary(t *testing.T) {
	for boundary := 0; boundary <= 4; boundary++ {
		t.Run(fmt.Sprint(boundary), func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			op, req := acceptAnchorStartFixture(t, s)
			if boundary >= 1 {
				if _, err = s.beginProjectAnchorStart(op.ProjectID, op.ID); err != nil {
					t.Fatal(err)
				}
			}
			phases := []string{"planned", "dispatched", "started", "observed"}
			for n := 0; n < boundary-1; n++ {
				var ns *projectnetwork.NamespaceIdentity
				if n == 2 {
					ns = anchorNamespaceFixture()
				}
				if err = s.advanceProjectAnchorStart(op.ProjectID, op.ID, phases[n], phases[n+1], ns); err != nil {
					t.Fatal(err)
				}
			}
			before, err := loadProjectAnchorStart(s.db, op.ProjectID)
			if err != nil {
				t.Fatal(err)
			}
			prepBefore, err := loadProjectJob(s.db, op.ProjectID)
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
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			after, err := loadProjectAnchorStart(s.db, op.ProjectID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("journal changed on restart", after, err)
			}
			prepAfter, err := loadProjectJob(s.db, op.ProjectID)
			if err != nil || !reflect.DeepEqual(prepBefore, prepAfter) {
				t.Fatal("creation evidence changed", err)
			}
			receipt, hit, err := s.acceptProjectAnchorStart("actor", op.ProjectID, req)
			want := "interrupted"
			if boundary == 4 {
				want = "succeeded"
			}
			if err != nil || !hit || receipt.ID != op.ID || receipt.Status != want {
				t.Fatal(receipt, hit, err)
			}
			if _, err = s.beginProjectAnchorStart(op.ProjectID, op.ID); err == nil {
				t.Fatal("restarted worker reacquired journal")
			}
			if err = s.advanceProjectAnchorStart(op.ProjectID, op.ID, "planned", "dispatched", nil); err == nil {
				t.Fatal("late callback advanced terminal operation")
			}
			req.RequestID = "second-attempt"
			_, _, err = s.acceptProjectAnchorStart("actor", op.ProjectID, req)
			requireFault(t, err, "project_recovery_required")
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			again, err := loadOperation(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(again, receipt) {
				t.Fatal("second reconciliation changed receipt", err)
			}
		})
	}
}

func TestProjectAnchorStartRequiresPreparationAndReview(t *testing.T) {
	s := testStore(t)
	prep, preq := acceptProjectFixture(t, s)
	req := projectAnchorStartRequest{RequestID: "start", ExpectedRevision: 1, ReviewDigest: preq.ReviewDigest}
	_, _, err := s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	requireFault(t, err, "project_not_prepared")
	completeProjectPreparation(t, s, prep)
	req.ExpectedRevision++
	_, _, err = s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	requireFault(t, err, "revision_conflict")
	req.ExpectedRevision--
	req.ReviewDigest = strings.Repeat("f", 64)
	if _, _, err = s.acceptProjectAnchorStart("actor", prep.ProjectID, req); err == nil {
		t.Fatal("unreviewed plan accepted")
	}
	req.ReviewDigest = preq.ReviewDigest
	req.ReplayOnly = true
	_, _, err = s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	requireFault(t, err, "replay_not_found")
	req.ReplayOnly = false
	op, hit, err := s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	if err != nil || hit {
		t.Fatal(err)
	}
	req.ReplayOnly = true
	got, hit, err := s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	if err != nil || !hit || got.ID != op.ID {
		t.Fatal(got, hit, err)
	}
	_, _, err = s.acceptProjectAnchorStart("another-actor", prep.ProjectID, req)
	requireFault(t, err, "replay_not_found")
	req.ExpectedRevision++
	_, _, err = s.acceptProjectAnchorStart("actor", prep.ProjectID, req)
	requireFault(t, err, "request_conflict")
}

func TestProjectAnchorStartAtomicMirrorAndCorruption(t *testing.T) {
	for _, query := range []string{
		"DELETE FROM project_anchor_starts",
		"UPDATE operations SET runtime='{}' WHERE json_extract(body,'$.kind')='start-project-anchor'",
		"UPDATE project_anchor_starts SET body='{}'",
		"UPDATE operations SET body=json_set(body,'$.status','running') WHERE json_extract(body,'$.kind')='prepare-project'",
	} {
		t.Run(query, func(t *testing.T) {
			s := testStore(t)
			acceptAnchorStartFixture(t, s)
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if err := validateProjectAnchorStarts(s.db); err == nil {
				t.Fatal("corrupt journal accepted")
			}
			dir := s.directory
			s.close()
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt store reopened")
			}
		})
	}
	s := testStore(t)
	op, _ := acceptAnchorStartFixture(t, s)
	if _, err := s.beginProjectAnchorStart(op.ProjectID, op.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := loadProjectAnchorStart(s.db, op.ProjectID)
	if _, err := s.db.Exec(`CREATE TRIGGER fail_anchor_mirror BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.advanceProjectAnchorStart(op.ProjectID, op.ID, "planned", "dispatched", nil); err == nil {
		t.Fatal("partial dispatch committed")
	}
	after, err := loadProjectAnchorStart(s.db, op.ProjectID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("mirror failure changed journal", err)
	}
}

func TestProjectAnchorStartConcurrentClaims(t *testing.T) {
	dir := privateDir(t)
	a, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	op, _ := acceptAnchorStartFixture(t, a)
	b, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, s := range []*store{a, b} {
		wg.Add(1)
		go func(s *store) {
			defer wg.Done()
			_, err := s.beginProjectAnchorStart(op.ProjectID, op.ID)
			results <- err
		}(s)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else {
			requireFault(t, err, "stale_project_operation")
		}
	}
	if success != 1 {
		t.Fatal("multiple workers acquired Start", success)
	}
	var instances int
	if err = a.db.QueryRow("SELECT count(*) FROM instances").Scan(&instances); err != nil || instances != 0 {
		t.Fatal("anchor Start created member instances", instances, err)
	}
}

type storedAnchorEngine struct {
	t             *testing.T
	s             *store
	id            string
	calls         []string
	running       bool
	lostReply     bool
	badInspection bool
}

func (e *storedAnchorEngine) VerifyEngine(context.Context, string) error { return nil }
func (e *storedAnchorEngine) StartAnchor(_ context.Context, p projectnetwork.Preparation) error {
	state, err := loadProjectAnchorStart(e.s.db, e.id)
	if err != nil || state.Phase != "dispatched" || !reflect.DeepEqual(state.Prepared, p) {
		e.t.Fatal("Start before durable dispatch", state, err)
	}
	e.calls = append(e.calls, "start")
	e.running = true
	if e.lostReply {
		return errors.New("synthetic lost reply")
	}
	return nil
}
func (e *storedAnchorEngine) InspectAnchor(_ context.Context, p projectnetwork.Preparation) (projectnetwork.Observation, error) {
	state, err := loadProjectAnchorStart(e.s.db, e.id)
	if err != nil {
		e.t.Fatal(err)
	}
	phase := "planned"
	if e.running {
		phase = "started"
	}
	if state.Phase != phase {
		e.t.Fatal("inspection before prior commit", state.Phase)
	}
	e.calls = append(e.calls, "inspect-"+phase)
	if e.badInspection && e.running {
		return projectnetwork.Observation{}, errors.New("synthetic inspection failed")
	}
	project, err := loadProject(e.s.db, e.id)
	if err != nil {
		e.t.Fatal(err)
	}
	o := projectnetwork.Observation{ProjectID: p.ProjectID, Revision: p.ProjectRevision, Generation: p.Generation, EngineID: p.EngineID, ObservedAt: time.Now().UTC(), Complete: true, Bridge: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Bridge.ID, ContractDigest: p.Bridge.ContractDigest}, Present: true}, Anchor: projectnetwork.AnchorObservation{ResourceObservation: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Anchor.ID, ContractDigest: p.Anchor.ContractDigest}, Present: true}, BridgeID: p.Bridge.ID, Running: e.running}}
	for _, m := range project.Plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: project.Plan.BindAddress, HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	if e.running {
		o.Anchor.Namespace = *anchorNamespaceFixture()
		o.Anchor.Active = append([]projectnetwork.Publication(nil), o.Anchor.Requested...)
		o.BridgeAttachments = []string{p.Anchor.ID}
	}
	return o, nil
}
func TestProjectAnchorStartWorkerUsesDurableEvidence(t *testing.T) {
	for _, mode := range []string{"success", "lost-reply", "inspection-failed"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			op, req := acceptAnchorStartFixture(t, s)
			e := &storedAnchorEngine{t: t, s: s, id: op.ProjectID, lostReply: mode == "lost-reply", badInspection: mode == "inspection-failed"}
			err := s.runProjectAnchorStart(context.Background(), op.ProjectID, op.ID, e)
			if (err == nil) != (mode == "success") {
				t.Fatal(err)
			}
			state, err := loadProjectAnchorStart(s.db, op.ProjectID)
			if err != nil {
				t.Fatal(err)
			}
			phase, status, calls := "observed", "succeeded", 3
			if mode == "lost-reply" {
				phase, status, calls = "dispatched", "interrupted", 2
			}
			if mode == "inspection-failed" {
				phase, status = "started", "interrupted"
			}
			receipt, hit, err := s.acceptProjectAnchorStart("actor", op.ProjectID, req)
			if err != nil || !hit || receipt.Status != status || state.Phase != phase || len(e.calls) != calls {
				t.Fatal(receipt, state, e.calls, err)
			}
			if err = s.runProjectAnchorStart(context.Background(), op.ProjectID, op.ID, e); err == nil || len(e.calls) != calls {
				t.Fatal("worker replayed Start", err, e.calls)
			}
			if err = validateProjectAnchorStarts(s.db); err != nil {
				t.Fatal(err)
			}
		})
	}
}
