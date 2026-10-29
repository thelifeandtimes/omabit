package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func projectRequest(t *testing.T, key, prefix string, first uint16) projectRegistration {
	t.Helper()
	p, err := projectnetwork.Build(projectnetwork.Request{Label: "Development " + prefix, ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{
		{InstanceID: prefix + "-zod", Ship: "~zod", HTTPPort: first}, {InstanceID: prefix + "-nec", Ship: "~nec", HTTPPort: first + 1},
	}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := p.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return projectRegistration{RequestID: key, Plan: p, ReviewDigest: digest}
}

func TestProjectRegistrationSurvivesRestartAndReplaysWithoutProbes(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	req := projectRequest(t, "request", "a", 18080)
	op, hit, err := s.registerProject("actor", req)
	if err != nil || hit || op.ProjectID == "" || op.InstanceID != "" || op.ConfigRevision != 1 {
		t.Fatal(op, hit, err)
	}
	var body []byte
	if err = s.db.QueryRow("SELECT body FROM projects WHERE id=?", op.ProjectID).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var p storedProject
	if err = json.Unmarshal(body, &p); err != nil {
		t.Fatal(err)
	}
	if p.Phase != "planned" || len(p.Members) != 2 || p.Members[0].RuntimeID == p.Members[1].RuntimeID {
		t.Fatal(p)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM instances").Scan(&n)
	if n != 0 {
		t.Fatal("registration created instances")
	}
	s.close()
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	s.portAvailable = func(string, uint16) bool { t.Error("replay probed host"); return false }
	req.ReplayOnly = true
	repeated, hit, err := s.registerProject("actor", req)
	if err != nil || !hit || !reflect.DeepEqual(op, repeated) {
		t.Fatal(repeated, hit, err)
	}
	if r, err := projectReservations(s.db); err != nil || len(r.Instances) != 2 || len(r.HTTPPorts) != 2 {
		t.Fatal(r, err)
	}
	req.RequestID = "new"
	_, _, err = s.registerProject("actor", req)
	requireFault(t, err, "replay_not_found")
	req.RequestID = "request"
	req.Plan.Label = "Changed"
	req.ReviewDigest, _ = req.Plan.Digest()
	_, _, err = s.registerProject("actor", req)
	requireFault(t, err, "request_conflict")
}

func TestProjectClaimsFenceStandaloneRegistrationConfigurationAndSuggestions(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.registerProject("a", projectRequest(t, "project", "a", 18080)); err != nil {
		t.Fatal(err)
	}
	_, _, err := s.register("a", registration("a-zod", "by-id", 19000))
	requireFault(t, err, "instance_reserved")
	_, _, err = s.register("a", registration("other", "by-port", 18081))
	requireFault(t, err, "port_conflict")
	if _, _, err = s.register("a", registration("other", "other", 19000)); err != nil {
		t.Fatal(err)
	}
	_, err = s.configure("a", "other", protocol.ConfigureRequest{ExpectedRevision: 1, Config: registration("x", "x", 18080).Config})
	requireFault(t, err, "port_conflict")
	if port, err := s.suggestHTTPPort(context.Background()); err != nil || port != 18082 {
		t.Fatal(port, err)
	}
	if port, err := s.choosePort(s.db, "a-zod", "tcp", 0); err != nil || port != 18082 {
		t.Fatal(port, err)
	}
	_, err = s.choosePort(s.db, "a-zod", "tcp", 18080)
	requireFault(t, err, "port_conflict")
	// Actor/request IDs share the existing receipt namespace across action types.
	_, _, err = s.register("a", registration("new", "project", 19001))
	requireFault(t, err, "request_conflict")
}

func TestProjectAdmissionIsAtomicAndAccountsForLegacyRuntimePorts(t *testing.T) {
	for _, kind := range []string{"instance", "desired-port", "runtime-port", "host-port", "stale-review", "failed-receipt"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			req := projectRequest(t, "request", "a", 18080)
			if kind == "instance" || kind == "desired-port" || kind == "runtime-port" {
				id, port := "legacy", uint16(19000)
				if kind == "instance" {
					id = "a-zod"
				}
				if kind == "desired-port" {
					port = 18081
				}
				if _, _, err := s.register("actor", registration(id, "legacy", port)); err != nil {
					t.Fatal(err)
				}
				if kind == "runtime-port" {
					v, err := loadInstance(s.db, id)
					if err != nil {
						t.Fatal(err)
					}
					v.Runtime.Prepared = true
					v.Runtime.Port = 18081
					tx, err := s.db.Begin()
					if err != nil {
						t.Fatal(err)
					}
					if err = saveInstance(tx, v); err != nil {
						t.Fatal(err)
					}
					if err = tx.Commit(); err != nil {
						t.Fatal(err)
					}
				}
			}
			if kind == "host-port" {
				s.portAvailable = func(_ string, p uint16) bool { return p != 18081 }
			}
			if kind == "stale-review" {
				req.Plan.Label = "different"
			}
			if kind == "failed-receipt" {
				_, err := s.db.Exec(`CREATE TRIGGER reject_project_receipt BEFORE INSERT ON operations WHEN new.request_id='request' BEGIN SELECT RAISE(ABORT,'synthetic receipt failure'); END;`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := s.registerProject("actor", req); err == nil {
				t.Fatal("admission unexpectedly succeeded")
			}
			for _, table := range []string{"projects", "project_members"} {
				var n int
				if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
					t.Fatal(table, n, err)
				}
			}
			var n int
			s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id='request'").Scan(&n)
			if n != 0 {
				t.Fatal("failed admission left receipt")
			}
		})
	}
}

func TestProjectConcurrentOperatorsCannotDuplicateClaims(t *testing.T) {
	dir := privateDir(t)
	a, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	b, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	req := projectRequest(t, "same-request", "a", 18080)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, s := range []*store{a, b} {
		wg.Add(1)
		go func(i int, s *store) {
			defer wg.Done()
			_, _, err := s.registerProject(fmt.Sprint(i), req)
			results <- err
		}(i, s)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			requireFault(t, err, "project_conflict")
		}
	}
	if successes != 1 {
		t.Fatal(successes)
	}
	// Duplicate fake identities in a different project are intentionally allowed.
	if _, _, err := b.registerProject("other", projectRequest(t, "same-request", "b", 19000)); err != nil {
		t.Fatal(err)
	}
	if r, err := projectReservations(a.db); err != nil || len(r.Instances) != 4 {
		t.Fatal(r, err)
	}
}

func TestProjectCorruptionCannotReleaseReservations(t *testing.T) {
	for _, query := range []string{
		"DELETE FROM project_members WHERE instance_id='a-zod'",
		"UPDATE project_members SET http_port=19000 WHERE instance_id='a-zod'",
		"UPDATE projects SET body='{}'",
	} {
		t.Run(query, func(t *testing.T) {
			s := testStore(t)
			if _, _, err := s.registerProject("a", projectRequest(t, "new", "a", 18080)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if _, err := projectReservations(s.db); err == nil {
				t.Fatal("altered ownership accepted")
			}
			if _, _, err := s.register("a", registration("unrelated", "standalone", 19000)); err == nil {
				t.Fatal("corruption bypassed registration fence")
			}
			if _, err := s.suggestHTTPPort(context.Background()); err == nil {
				t.Fatal("corruption bypassed port suggestion")
			}
			dir := s.directory
			s.close()
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt project ownership reopened")
			}
		})
	}
}

func TestProjectLabelsUniqueWithoutBreakingReplay(t *testing.T) {
	s := testStore(t)
	req := projectRequest(t, "first", "a", 28101)
	op, _, err := s.registerProject("actor", req)
	if err != nil {
		t.Fatal(err)
	}
	if replay, hit, err := s.registerProject("actor", req); err != nil || !hit || replay.ID != op.ID {
		t.Fatal("lost replay", err)
	}
	next := projectRequest(t, "second", "b", 28103)
	next.Plan.Label = req.Plan.Label
	next.ReviewDigest, _ = next.Plan.Digest()
	if _, _, err = s.registerProject("actor", next); err == nil {
		t.Fatal("duplicate label accepted")
	}
}
