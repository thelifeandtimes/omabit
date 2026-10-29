//go:build linux && amd64

package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"os"
	"strings"
	"testing"
)

type publicMemberFixture struct {
	r       *Server
	p       storedProject
	q       protocol.ProjectActionRequest
	release chan struct{}
	setID   func(string)
}

func publicMemberActionFixture(t *testing.T, kind string) publicMemberFixture {
	var f publicMemberFixture
	f.q.Kind = kind
	f.q.ConfirmAction = true
	f.q.RequestID = "public-member-" + kind
	switch kind {
	case "stop":
		r, b, p, q := stopQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.ReadinessOperationID
		f.setID = func(id string) { b.opID = id }
	case "observe-exit":
		r, b, p, q := stopExitQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.StopOperationID
	case "retire":
		r, b, p, q := retirementQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.ExitOperationID
	case "resume":
		r, b, p, q := resumeQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.RetirementOperationID
	case "observe-resumed":
		r, b, p, q := resumedReadinessQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.ResumeOperationID
		f.setID = func(id string) { b.opID = id }
	case "observe-current":
		r, b, p, q := contextReadinessQueueFixture(t)
		f.r, f.p, f.release = r, p, b.release
		f.q.InstanceID = b.instance
		f.q.ReviewDigest = q.ReviewDigest
		f.q.ExpectedRevision = q.ExpectedRevision
		f.q.AfterOperationID = q.CreatorOperationID
		f.setID = func(id string) { b.opID = id }
		f.q.OriginalReadinessOperationID = q.OriginalReadinessOperationID
	default:
		t.Fatal(kind)
	}
	f.r.uid = uint32(os.Geteuid())
	return f
}
func TestProjectMemberPublicUnknownPrerequisite(t *testing.T) {
	for _, kind := range []string{"stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current"} {
		t.Run(kind, func(t *testing.T) {
			f := publicMemberActionFixture(t, kind)
			f.q.AfterOperationID = "op-unknown-selection"
			w := initialActionHTTP(t, f.r, f.p.ID, f.q, false)
			code := "project_prerequisite_missing"
			if kind == "observe-current" {
				code = "project_recovery_required"
			}
			if w.Code != 409 || !strings.Contains(w.Body.String(), code) || f.r.failed.Load() || f.r.updateWorkers != 0 {
				t.Fatalf("status=%d failed=%v workers=%d body=%s", w.Code, f.r.failed.Load(), f.r.updateWorkers, w.Body.String())
			}
		})
	}
}
func TestProjectMemberPublicQueuesAndReplay(t *testing.T) {
	for _, kind := range []string{"stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current"} {
		t.Run(kind, func(t *testing.T) {
			f := publicMemberActionFixture(t, kind)
			r := f.r
			r.workGate <- struct{}{}
			w := initialActionHTTP(t, r, f.p.ID, f.q, false)
			if w.Code != 202 {
				t.Fatal(w.Code, w.Body.String())
			}
			var a protocol.ProjectActionResult
			if json.Unmarshal(w.Body.Bytes(), &a) != nil {
				t.Fatal(w.Body.String())
			}
			_, opKind, _ := protocol.ProjectAction(kind)
			if a.Operation.Kind != opKind || a.Operation.Status != "queued" || a.Operation.InstanceID != f.q.InstanceID || a.Operation.ProjectID != f.p.ID {
				t.Fatal(a)
			}
			if f.setID != nil {
				f.setID(a.Operation.ID)
			}
			hit := initialActionHTTP(t, r, f.p.ID, f.q, true)
			if hit.Code != 200 || hit.Body.String() != w.Body.String() {
				t.Fatal(hit.Code, hit.Body.String())
			}
			missing := f.q
			missing.RequestID = "missing-lookup"
			hit = initialActionHTTP(t, r, f.p.ID, missing, true)
			if hit.Code != 409 || !strings.Contains(hit.Body.String(), "replay_not_found") || r.updateWorkers != 1 {
				t.Fatal(hit.Code, hit.Body.String(), r.updateWorkers)
			}
			changed := f.q
			changed.AfterOperationID = "op-changed-selection"
			hit = initialActionHTTP(t, r, f.p.ID, changed, true)
			if hit.Code != 409 || r.updateWorkers != 1 {
				t.Fatal("changed replay accepted", hit.Code, hit.Body.String())
			}
			<-r.workGate
			close(f.release)
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, a.Operation.ID)
			if err != nil || actual.Status != "succeeded" {
				t.Fatal(actual, err)
			}
			hit = initialActionHTTP(t, r, f.p.ID, f.q, true)
			if hit.Code != 200 || r.updateWorkers != 0 {
				t.Fatal(hit.Code, hit.Body.String())
			}
			var done protocol.ProjectActionResult
			json.Unmarshal(hit.Body.Bytes(), &done)
			if done.Operation.ID != actual.ID || done.Operation.Status != "succeeded" {
				t.Fatal(done)
			}
		})
	}
}

func TestProjectMemberPublicSelectionAndCorruption(t *testing.T) {
	for _, scenario := range []string{"wrong-kind", "wrong-member", "original-missing", "corrupt-journal"} {
		t.Run(scenario, func(t *testing.T) {
			kind := "stop"
			if scenario == "original-missing" {
				kind = "observe-current"
			}
			f := publicMemberActionFixture(t, kind)
			switch scenario {
			case "wrong-kind":
				if err := f.r.s.db.QueryRow("SELECT operation_id FROM project_member_launches WHERE instance_id=?", f.q.InstanceID).Scan(&f.q.AfterOperationID); err != nil {
					t.Fatal(err)
				}
			case "wrong-member":
				f.q.InstanceID = "not-reserved"
			case "original-missing":
				f.q.OriginalReadinessOperationID = "op-unknown-original"
			case "corrupt-journal":
				if _, err := f.r.s.db.Exec("DELETE FROM project_member_readiness WHERE operation_id=?", f.q.AfterOperationID); err != nil {
					t.Fatal(err)
				}
			}
			w := initialActionHTTP(t, f.r, f.p.ID, f.q, false)
			want := 409
			failed := false
			if scenario == "corrupt-journal" {
				want = 500
				failed = true
			}
			if w.Code != want || f.r.failed.Load() != failed || f.r.updateWorkers != 0 {
				t.Fatal(scenario, w.Code, w.Body.String(), f.r.failed.Load(), f.r.updateWorkers)
			}
		})
	}
}
func TestProjectInitialPublicPrerequisiteMissing(t *testing.T) {
	for _, kind := range []string{"launch", "observe"} {
		t.Run(kind, func(t *testing.T) {
			var r *Server
			var p storedProject
			var digest, instance, opID string
			if kind == "launch" {
				server, _, project, q := memberLaunchQueueFixture(t)
				r, p, digest = server, project, q.ReviewDigest
				instance = p.Members[0].InstanceID
				if err := r.s.db.QueryRow("SELECT operation_id FROM project_anchor_starts WHERE project_id=?", p.ID).Scan(&opID); err != nil {
					t.Fatal(err)
				}
				if _, err := r.s.db.Exec("DELETE FROM project_anchor_starts WHERE project_id=?", p.ID); err != nil {
					t.Fatal(err)
				}
			} else {
				server, b, project, q := readinessQueueFixture(t)
				r, p, digest, instance = server, project, q.ReviewDigest, b.instance
				// Last initial launch has no later launch referencing its peer evidence.
				target := p.Members[len(p.Members)-1].InstanceID
				if err := r.s.db.QueryRow("SELECT operation_id FROM project_member_launches WHERE instance_id=?", target).Scan(&opID); err != nil {
					t.Fatal(err)
				}
				if _, err := r.s.db.Exec("DELETE FROM project_member_launches WHERE instance_id=?", target); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := r.s.db.Exec("DELETE FROM operations WHERE id=?", opID); err != nil {
				t.Fatal(err)
			}
			r.uid = uint32(os.Geteuid())
			q := protocol.ProjectActionRequest{Kind: kind, InstanceID: instance, RequestID: "missing-initial", ExpectedRevision: 1, ReviewDigest: digest, ConfirmAction: true}
			w := initialActionHTTP(t, r, p.ID, q, false)
			if w.Code != 409 || r.failed.Load() || r.updateWorkers != 0 {
				t.Fatal(w.Code, w.Body.String(), r.failed.Load())
			}
		})
	}
}
