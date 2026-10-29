package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func privateProjectRequest(t *testing.T, key, prefix string, port uint16) privateProjectRegistration {
	t.Helper()
	raw, err := os.ReadFile("../privatechain/testdata/qualified-foundation.json")
	g02Check(t, err)
	var f privatechain.Foundation
	g02Check(t, json.Unmarshal(raw, &f))
	p, err := privatechain.Build(privatechain.Request{Label: "Private development", Foundation: f, ConfirmSharedAdministration: true, Members: []privatechain.MemberRequest{{InstanceID: prefix + "-zod", Ship: "~zod", HTTPPort: port}, {InstanceID: prefix + "-nec", Ship: "~nec", HTTPPort: port + 1}}}, privatechain.Reservations{})
	g02Check(t, err)
	digest, err := p.Digest()
	g02Check(t, err)
	return privateProjectRegistration{RequestID: key, Plan: p, ReviewDigest: digest}
}
func readPrivateProject(t *testing.T, s *store, id string) (storedPrivateProject, []byte) {
	t.Helper()
	var raw []byte
	g02Check(t, s.db.QueryRow("SELECT body FROM private_projects WHERE id=?", id).Scan(&raw))
	var p storedPrivateProject
	g02Check(t, json.Unmarshal(raw, &p))
	return p, raw
}

func TestPrivateProjectRegistrationRestartAndActorReplay(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	req := privateProjectRequest(t, "private-request", "one", 18381)
	op, hit, err := s.registerPrivateProject("actor", req)
	g02Check(t, err)
	if hit || op.Kind != privateProjectRegistrationKind {
		t.Fatal(op, hit)
	}
	p, before := readPrivateProject(t, s, op.ProjectID)
	if p.Binding.RunnerID != s.runnerID || p.Binding.Generation != 1 || p.Members[0].RuntimeID == p.Members[1].RuntimeID {
		t.Fatal("wrong ownership")
	}
	for _, table := range []string{"instances", "projects", "project_jobs"} {
		var n int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
		if n != 0 {
			t.Fatal("registration created runtime/fake state", table)
		}
	}
	s.close()
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	s.portAvailable = func(string, uint16) bool { t.Error("replay probed a host port"); return false }
	req.ReplayOnly = true
	repeated, hit, err := s.registerPrivateProject("actor", req)
	g02Check(t, err)
	if !hit || !reflect.DeepEqual(op, repeated) {
		t.Fatal("receipt changed")
	}
	_, after := readPrivateProject(t, s, op.ProjectID)
	if !bytes.Equal(before, after) {
		t.Fatal("record changed on open/replay")
	}
	_, _, err = s.registerPrivateProject("other-actor", req)
	requireFault(t, err, "replay_not_found")
	req.Plan.Label = "Changed"
	req.ReviewDigest, _ = req.Plan.Digest()
	_, _, err = s.registerPrivateProject("actor", req)
	requireFault(t, err, "request_conflict")
}

func TestPrivateProjectReservationsFenceAllAdmissionPaths(t *testing.T) {
	s := testStore(t)
	op, _, err := s.registerPrivateProject("actor", privateProjectRequest(t, "private", "dev", 18080))
	g02Check(t, err)
	_, _, err = s.register("actor", registration("dev-zod", "by-name", 19000))
	requireFault(t, err, "instance_reserved")
	_, _, err = s.register("actor", registration("other", "by-port", 18081))
	requireFault(t, err, "port_conflict")
	_, _, err = s.register("actor", registration("other", "standalone", 19000))
	g02Check(t, err)
	_, err = s.configure("actor", "other", protocol.ConfigureRequest{ExpectedRevision: 1, Config: registration("x", "x", 18080).Config})
	requireFault(t, err, "port_conflict")
	for _, r := range []projectRegistration{projectRequest(t, "fake-name", "dev", 19010), projectRequest(t, "fake-port", "other", 18080)} {
		_, _, err = s.registerProject("actor", r)
		requireFault(t, err, "project_conflict")
	}
	if port, err := s.suggestHTTPPort(context.Background()); err != nil || port != 18082 {
		t.Fatal(port, err)
	}
	_, err = s.choosePort(s.db, "dev-zod", "tcp", 18080)
	requireFault(t, err, "port_conflict")
	_, _, err = s.registerProject("actor", projectRequest(t, "private", "unique", 19010))
	requireFault(t, err, "request_conflict")
	if _, err = loadProject(s.db, op.ProjectID); err == nil {
		t.Fatal("private project entered fake lifecycle")
	}
}

func TestPrivateProjectAdmissionAtomicConflicts(t *testing.T) {
	for _, kind := range []string{"fake-name", "fake-port", "private-name", "private-port", "standalone-name", "desired-port", "retained-port", "host-port", "stale-review", "receipt-failure", "member-failure", "fake-mode"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			req := privateProjectRequest(t, "request", "a", 18080)
			switch kind {
			case "fake-name":
				_, _, err := s.registerProject("a", projectRequest(t, "existing", "a", 19000))
				g02Check(t, err)
			case "fake-port":
				_, _, err := s.registerProject("a", projectRequest(t, "existing", "b", 18080))
				g02Check(t, err)
			case "private-name":
				_, _, err := s.registerPrivateProject("a", privateProjectRequest(t, "existing", "a", 19000))
				g02Check(t, err)
			case "private-port":
				_, _, err := s.registerPrivateProject("a", privateProjectRequest(t, "existing", "b", 18080))
				g02Check(t, err)
			case "standalone-name", "desired-port", "retained-port":
				id, port := "standalone", uint16(19000)
				if kind == "standalone-name" {
					id = "a-zod"
				}
				if kind == "desired-port" {
					port = 18081
				}
				_, _, err := s.register("a", registration(id, "existing", port))
				g02Check(t, err)
				if kind == "retained-port" {
					v, err := loadInstance(s.db, id)
					g02Check(t, err)
					v.Runtime.Prepared = true
					v.Runtime.Port = 18081
					tx, err := s.db.Begin()
					g02Check(t, err)
					g02Check(t, saveInstance(tx, v))
					g02Check(t, tx.Commit())
				}
			case "host-port":
				s.portAvailable = func(_ string, p uint16) bool { return p != 18081 }
			case "stale-review":
				req.Plan.Label = "changed"
			case "receipt-failure":
				_, err := s.db.Exec(`CREATE TRIGGER fail_private_receipt BEFORE INSERT ON operations WHEN new.request_id='request' BEGIN SELECT RAISE(ABORT,'synthetic receipt failure'); END;`)
				g02Check(t, err)
			case "member-failure":
				_, err := s.db.Exec(`CREATE TRIGGER fail_private_member BEFORE INSERT ON private_project_members WHEN new.instance_id='a-zod' BEGIN SELECT RAISE(ABORT,'synthetic member failure'); END;`)
				g02Check(t, err)
			case "fake-mode":
				req.Plan.Mode = "fake-galaxies-v1"
			}
			counts := map[string]int{}
			for _, table := range []string{"private_projects", "private_project_members", "operations"} {
				var n int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
				counts[table] = n
			}
			if _, _, err := s.registerPrivateProject("a", req); err == nil {
				t.Fatal("invalid admission passed")
			}
			for table, want := range counts {
				var n int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
				if n != want {
					t.Fatal("partial transaction", table, n, want)
				}
			}
		})
	}
}

func TestPrivateProjectCorruptionRefusesOpenAndReplay(t *testing.T) {
	changes := map[string]string{
		"missing-member":    "DELETE FROM private_project_members WHERE instance_id='a-zod'",
		"changed-port":      "UPDATE private_project_members SET http_port=18389 WHERE instance_id='a-zod'",
		"changed-runtime":   "UPDATE private_project_members SET runtime_id='altered' WHERE instance_id='a-zod'",
		"wrong-runner":      `UPDATE private_projects SET body=json_set(body,'$.binding.runner_id','runner-other')`,
		"wrong-generation":  `UPDATE private_projects SET body=json_set(body,'$.binding.generation',2)`,
		"wrong-foundation":  `UPDATE private_projects SET body=json_set(body,'$.binding.foundation_digest','wrong')`,
		"unknown-field":     `UPDATE private_projects SET body=json_set(body,'$.unknown',true)`,
		"wrong-fingerprint": "UPDATE operations SET fingerprint='altered' WHERE kindPlaceholder",
		"wrong-actor":       "UPDATE operations SET actor='other' WHERE kindPlaceholder",
		"missing-receipt":   "DELETE FROM operations WHERE kindPlaceholder",
		"orphan-receipt":    "DELETE FROM private_project_members; DELETE FROM private_projects",
	}
	for kind, mutation := range changes {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			req := privateProjectRequest(t, "register", "a", 18381)
			op, _, err := s.registerPrivateProject("actor", req)
			g02Check(t, err)
			mutation = strings.ReplaceAll(mutation, "kindPlaceholder", "id='"+op.ID+"'")
			_, err = s.db.Exec(mutation)
			g02Check(t, err)
			if _, _, err = s.registerPrivateProject("actor", req); err == nil {
				t.Fatal("corrupt replay passed")
			}
			s.close()
			bad, err := testOpenStore(dir)
			if err == nil {
				bad.close()
				t.Fatal("corrupt state reopened")
			}
		})
	}
}

func TestPrivateProjectIndependentBindingsAndConcurrentClaims(t *testing.T) {
	dir := privateDir(t)
	a, err := testOpenStore(dir)
	g02Check(t, err)
	defer a.close()
	b, err := testOpenStore(dir)
	g02Check(t, err)
	defer b.close()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, s := range []*store{a, b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := s.registerPrivateProject(fmt.Sprint(i), privateProjectRequest(t, fmt.Sprint("req-", i), "same", 18381))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		} else {
			requireFault(t, err, "project_conflict")
		}
	}
	if success != 1 {
		t.Fatal("overlapping claims admitted", success)
	}
	first, _, err := a.registerPrivateProject("actor", privateProjectRequest(t, "one", "first", 19381))
	g02Check(t, err)
	second, _, err := a.registerPrivateProject("actor", privateProjectRequest(t, "two", "second", 20381))
	g02Check(t, err)
	p, _ := readPrivateProject(t, a, first.ProjectID)
	q, _ := readPrivateProject(t, a, second.ProjectID)
	if p.Binding.FoundationDigest != q.Binding.FoundationDigest || p.Binding == q.Binding || privatechain.Match(p.Binding, q.Binding, p.Plan.Foundation) == nil {
		t.Fatal("cloneable foundation became project identity")
	}
}
