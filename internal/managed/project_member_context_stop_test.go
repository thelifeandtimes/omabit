package managed

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func readyContextStopProject(t *testing.T, s *store, target int) (storedProject, string, projectMemberStopRequest) {
	t.Helper()
	p, instance, req := contextReadinessProject(t, s, target)
	op, _, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
	g02Check(t, err)
	_, err = s.beginProjectMemberContextReadiness(op.ID)
	g02Check(t, err)
	g02Check(t, s.completeProjectMemberContextReadiness(op.ID, contextReadinessProof(t, s, op.ID)))
	return p, instance, projectMemberStopRequest{RequestID: "context-stop", ReadinessOperationID: op.ID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
}
func contextStopObservations(t *testing.T, s *store, ready string) (projectnetwork.MemberStopObservation, projectnetwork.MemberStopObservation) {
	t.Helper()
	j, err := loadProjectMemberContextReadiness(s.db, ready)
	g02Check(t, err)
	var before projectnetwork.MemberStopObservation
	raw, _ := json.Marshal(j.Evidence.After)
	g02Check(t, json.Unmarshal(raw, &before))
	before.CapturedAt = time.Now().UTC()
	before.Runtime.ObservedAt = before.CapturedAt
	raw, _ = json.Marshal(before)
	var after projectnetwork.MemberStopObservation
	g02Check(t, json.Unmarshal(raw, &after))
	after.CapturedAt = before.CapturedAt.Add(time.Millisecond)
	after.Runtime.ObservedAt = after.CapturedAt
	after.FinishedAt = after.CapturedAt
	for i := range after.Runtime.Members {
		if after.Runtime.Members[i].InstanceID == j.InstanceID {
			after.Runtime.Members[i].Running = false
			after.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
		}
	}
	return before, after
}
func assertContextHistory(t *testing.T, s *store, history map[string]string) {
	t.Helper()
	now := currentStopHistory(t, s)
	for id, body := range history {
		if now[id] != body {
			t.Fatal("historical receipt changed", id)
		}
	}
}
func TestContextStopRestartReplayPreservesHistory(t *testing.T) {
	for target := 0; target < 2; target++ {
		for _, phase := range []string{"planned", "stop-dispatched", "stop-signalled", "stopped"} {
			t.Run(fmt.Sprintf("target%d/%s", target, phase), func(t *testing.T) {
				dir := privateDir(t)
				s, err := testOpenStore(dir)
				g02Check(t, err)
				p, instance, req := readyContextStopProject(t, s, target)
				history := currentStopHistory(t, s)
				op, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
				g02Check(t, err)
				if hit {
					t.Fatal("unexpected replay")
				}
				if phase != "planned" {
					_, err = s.beginProjectMemberStop(op.ID)
					g02Check(t, err)
					before, after := contextStopObservations(t, s, req.ReadinessOperationID)
					advanceStopFixture(t, s, p, op.ID, "stop-dispatched", &before)
					if phase != "stop-dispatched" {
						advanceStopFixture(t, s, p, op.ID, "stop-signalled", nil)
					}
					if phase == "stopped" {
						advanceStopFixture(t, s, p, op.ID, "stopped", &after)
						g02Check(t, s.finishProjectMemberStop(op.ID))
					}
				}
				saved, err := loadProjectMemberStop(s.db, op.ID)
				g02Check(t, err)
				assertContextHistory(t, s, history)
				g02Check(t, s.close())
				s, err = testOpenStore(dir)
				g02Check(t, err)
				defer s.close()
				g02Check(t, s.reconcileInterrupted())
				after, err := loadProjectMemberStop(s.db, op.ID)
				g02Check(t, err)
				if !reflect.DeepEqual(saved, after) {
					t.Fatal("restart rewrote Stop")
				}
				replay, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
				g02Check(t, err)
				want := "interrupted"
				if phase == "stopped" {
					want = "succeeded"
				}
				if !hit || replay.ID != op.ID || replay.Status != want {
					t.Fatal("replay changed", replay, hit)
				}
				g02Check(t, validateProjectMemberContextReadiness(s.db))
				assertContextHistory(t, s, history)
			})
		}
	}
}
func TestContextStopRejectsProcessMismatchBeforeCommit(t *testing.T) {
	for target := 0; target < 2; target++ {
		t.Run(fmt.Sprint(target), func(t *testing.T) {
			s := testStore(t)
			p, instance, req := readyContextStopProject(t, s, target)
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			g02Check(t, err)
			stop, err := s.beginProjectMemberStop(op.ID)
			g02Check(t, err)
			before, _ := contextStopObservations(t, s, req.ReadinessOperationID)
			before.StartedAt = before.StartedAt.Add(time.Second)
			next, err := stop.Advance(p.Plan, "planned", "stop-dispatched", &before)
			g02Check(t, err)
			history := currentStopHistory(t, s)
			if err = s.advanceProjectMemberStop(op.ID, "planned", next); err == nil {
				t.Fatal("new process persisted under old readiness")
			}
			assertContextHistory(t, s, history)
		})
	}
}
func TestContextReadinessRawSourceRejectsAnotherMembersEvidence(t *testing.T) {
	s := testStore(t)
	p, instance, req := contextReadinessProject(t, s, 0)
	var other string
	g02Check(t, s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE project_id=? AND instance_id!=?", p.ID, instance).Scan(&other))
	original, err := loadProjectMemberReadiness(s.db, req.OriginalReadinessOperationID)
	g02Check(t, err)
	foreign, err := loadProjectMemberReadiness(s.db, other)
	g02Check(t, err)
	original.Evidence = foreign.Evidence
	original.LaunchOperationID = foreign.LaunchOperationID
	body, err := json.Marshal(original)
	g02Check(t, err)
	_, err = s.db.Exec("UPDATE project_member_readiness SET body=? WHERE operation_id=?", body, original.OperationID)
	g02Check(t, err)
	_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", body, original.OperationID)
	g02Check(t, err)
	if _, err = projectContextReadinessSource(s.db, p.ID, instance, original.OperationID, nil); err == nil {
		t.Fatal("raw loader accepted another member's initialized pier")
	}
}
