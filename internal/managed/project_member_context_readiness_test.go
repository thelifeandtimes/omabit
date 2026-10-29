package managed

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func contextReadinessProject(t *testing.T, s *store, target int) (storedProject, string, projectMemberContextReadinessRequest) {
	t.Helper()
	p, _ := resumedReadinessProject(t, s)
	instance := p.Members[target].InstanceID
	var original string
	if err := s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE project_id=? AND instance_id=? ORDER BY operation_id LIMIT 1", p.ID, instance).Scan(&original); err != nil {
		t.Fatal(err)
	}
	src, err := projectContextReadinessSource(s.db, p.ID, instance, original, nil)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return p, instance, projectMemberContextReadinessRequest{OriginalReadinessOperationID: original, CreatorOperationID: src.Context.Member.OperationID, RequestID: "context-readiness", ExpectedRevision: p.Revision, ReviewDigest: digest}
}
func contextReadinessProof(t *testing.T, s *store, id string) projectnetwork.CurrentMemberReadiness {
	t.Helper()
	j, err := loadProjectMemberContextReadiness(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	src, err := currentContextUnchanged(s.db, j)
	if err != nil {
		t.Fatal(err)
	}
	baseline := src.Original.After
	if src.RecoveryHistory != nil {
		baseline = src.RecoveryHistory.Epochs[len(src.RecoveryHistory.Epochs)-1].Readiness.After.Runtime
	}
	raw, _ := json.Marshal(baseline)
	var runtime projectnetwork.Observation
	if err = json.Unmarshal(raw, &runtime); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	runtime.ObservedAt = now
	for i, m := range runtime.Members {
		for _, current := range src.Context.Binding.Members {
			if current.InstanceID == m.InstanceID {
				runtime.Members[i].Resource = current.Container
				runtime.Members[i].Namespace = src.Context.Binding.Namespace
			}
		}
	}
	started := src.Context.KnownStartedAt
	if started.IsZero() {
		started = now.Add(-time.Minute)
	}
	o := projectnetwork.MemberStopObservation{Runtime: runtime, Storage: src.Context.Storage, StartedAt: started, CapturedAt: now}
	r := projectnetwork.CurrentMemberReadiness{InstanceID: j.InstanceID, CreatorOperationID: src.Context.Member.OperationID, ContainerGeneration: src.Context.Member.Generation, Binding: src.Context.Binding, Storage: src.Context.Storage, Ship: src.Original.Ship, Endpoint: src.Original.Endpoint, Before: o, After: o, CapturedAt: now}
	if err = contextReadinessProofValid(src, r); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestProjectMemberContextReadinessRestartReplay(t *testing.T) {
	for target := 0; target < 2; target++ {
		for _, phase := range []string{"planned", "observing", "observed"} {
			t.Run(fmt.Sprintf("target%d/%s", target, phase), func(t *testing.T) {
				dir := privateDir(t)
				s, err := testOpenStore(dir)
				if err != nil {
					t.Fatal(err)
				}
				p, instance, req := contextReadinessProject(t, s, target)
				original, err := loadProjectMemberReadiness(s.db, req.OriginalReadinessOperationID)
				if err != nil {
					t.Fatal(err)
				}
				op, hit, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
				if err != nil || hit {
					t.Fatal(op, hit, err)
				}
				if phase != "planned" {
					if _, err = s.beginProjectMemberContextReadiness(op.ID); err != nil {
						t.Fatal(err)
					}
				}
				if phase == "observed" {
					if err = s.completeProjectMemberContextReadiness(op.ID, contextReadinessProof(t, s, op.ID)); err != nil {
						t.Fatal(err)
					}
				}
				before, err := loadProjectMemberContextReadiness(s.db, op.ID)
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
				if err = s.reconcileInterrupted(); err != nil {
					t.Fatal(err)
				}
				after, err := loadProjectMemberContextReadiness(s.db, op.ID)
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatal("reopen changed receipt", err)
				}
				replay, hit, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
				expected := "interrupted"
				if phase == "observed" {
					expected = "succeeded"
				}
				if err != nil || !hit || replay.ID != op.ID || replay.Status != expected {
					t.Fatal(replay, hit, err)
				}
				if _, err = s.beginProjectMemberContextReadiness(op.ID); err == nil {
					t.Fatal("old observation restarted")
				}
				req.RequestID = "new-context-readiness"
				next, hit, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
				if err != nil || hit || next.ID == op.ID {
					t.Fatal(next, hit, err)
				}
				unchanged, err := loadProjectMemberReadiness(s.db, req.OriginalReadinessOperationID)
				if err != nil || !reflect.DeepEqual(original, unchanged) {
					t.Fatal("original evidence rewritten", err)
				}
			})
		}
	}
}
func TestProjectMemberContextReadinessRefusals(t *testing.T) {
	for _, mode := range []string{"creator", "original", "revision", "digest", "active", "pending-stop", "completed-stop", "failed-insert", "wrong-proof", "changed-process", "stale-proof", "mirror", "orphan"} {
		t.Run(mode, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, instance, req := contextReadinessProject(t, s, 0)
			switch mode {
			case "creator":
				req.CreatorOperationID = "older-creator"
			case "original":
				req.OriginalReadinessOperationID = "foreign-readiness"
			case "revision":
				req.ExpectedRevision++
			case "digest":
				req.ReviewDigest = "changed"
			case "active":
				_, _, err = s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
				if err != nil {
					t.Fatal(err)
				}
				req.RequestID = "another-context"
			case "pending-stop", "completed-stop":
				resumeID := req.CreatorOperationID
				stopReq := observeGenerationFixture(t, s, p, resumeID, req.ReviewDigest, 2)
				stop, _, stopErr := s.acceptProjectMemberStop("actor", p.ID, instance, stopReq)
				if stopErr != nil {
					t.Fatal(stopErr)
				}
				if mode == "completed-stop" {
					if _, err = s.beginProjectMemberStop(stop.ID); err != nil {
						t.Fatal(err)
					}
					before, after := currentStopObservations(t, s, p.ID, instance)
					advanceStopFixture(t, s, p, stop.ID, "stop-dispatched", &before)
					advanceStopFixture(t, s, p, stop.ID, "stop-signalled", nil)
					advanceStopFixture(t, s, p, stop.ID, "stopped", &after)
					if err = s.finishProjectMemberStop(stop.ID); err != nil {
						t.Fatal(err)
					}
				}
			case "failed-insert":
				_, err = s.db.Exec("CREATE TRIGGER reject_context BEFORE INSERT ON project_member_context_readiness BEGIN SELECT RAISE(ABORT,'synthetic'); END")
				if err != nil {
					t.Fatal(err)
				}
			}
			var count int
			if err = s.db.QueryRow("SELECT count(*) FROM operations").Scan(&count); err != nil {
				t.Fatal(err)
			}
			op, _, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
			switch mode {
			case "creator", "original", "revision", "digest", "active", "pending-stop", "completed-stop", "failed-insert":
				if err == nil {
					t.Fatal("invalid observation admitted")
				}
				var after int
				if err = s.db.QueryRow("SELECT count(*) FROM operations").Scan(&after); err != nil || after != count {
					t.Fatal("failed acceptance wrote operation", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "mirror" {
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
				if err != nil {
					t.Fatal(err)
				}
				if validateProjectMemberContextReadiness(s.db) == nil {
					t.Fatal("changed mirror accepted")
				}
				return
			}
			if mode == "orphan" {
				_, err = s.db.Exec("DELETE FROM project_member_context_readiness WHERE operation_id=?", op.ID)
				if err != nil {
					t.Fatal(err)
				}
				if validateProjectMemberContextReadiness(s.db) == nil {
					t.Fatal("orphan accepted")
				}
				return
			}
			if _, err = s.beginProjectMemberContextReadiness(op.ID); err != nil {
				t.Fatal(err)
			}
			proof := contextReadinessProof(t, s, op.ID)
			switch mode {
			case "wrong-proof":
				proof.CreatorOperationID = "foreign"
			case "changed-process":
				proof.After.StartedAt = proof.After.StartedAt.Add(time.Second)
			case "stale-proof":
				proof.CapturedAt = proof.CapturedAt.Add(-time.Minute)
			}
			if err = s.completeProjectMemberContextReadiness(op.ID, proof); err == nil {
				t.Fatal("invalid completion accepted")
			}
			j, err := loadProjectMemberContextReadiness(s.db, op.ID)
			if err != nil || j.Phase != "observing" || j.Evidence != nil {
				t.Fatal("failed completion changed journal", err)
			}
		})
	}
}
func TestProjectMemberContextReadinessSurvivesLaterReplacement(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, instance, req := contextReadinessProject(t, s, 0)
	op, _, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberContextReadiness(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.completeProjectMemberContextReadiness(op.ID, contextReadinessProof(t, s, op.ID)); err != nil {
		t.Fatal(err)
	}
	before, _ := loadProjectMemberContextReadiness(s.db, op.ID)
	stopReq := observeGenerationFixture(t, s, p, req.CreatorOperationID, req.ReviewDigest, 2)
	nextReq := retireCurrentFixture(t, s, p, stopReq, 2)
	next, _, err := s.acceptProjectMemberResume("actor", p.ID, instance, nextReq)
	if err != nil {
		t.Fatal(err)
	}
	resumeThroughFixture(t, s, next.ID, "complete")
	if err = validateProjectMemberContextReadiness(s.db); err != nil {
		t.Fatal("old context lost validity", err)
	}
	after, _ := loadProjectMemberContextReadiness(s.db, op.ID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rewrote historical context")
	}
	req.RequestID = "stale-current-creator"
	if _, _, err = s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req); err == nil {
		t.Fatal("old creator accepted")
	}
}
