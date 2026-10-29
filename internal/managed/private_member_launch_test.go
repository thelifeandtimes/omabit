package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateMemberStoreContract(_ privatechain.Plan, _ privatechain.Workspace, _ privatechain.Gateway, instance, runtime, operation string) (string, error) {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(instance+runtime+operation))), nil
}

func privateMemberRequest(t *testing.T, s *store, project string) privateMemberLaunchRequest {
	t.Helper()
	g, op, err := readyPrivateGateway(s.db, project)
	g02Check(t, err)
	return privateMemberLaunchRequest{RequestID: "initial-member", ExpectedRevision: g.Selection.Revision, ReviewDigest: g.Selection.PlanDigest, GatewayEvidenceOperationID: op.ID, GatewayDigest: privateGatewaySourceDigest(g)}
}
func privateMemberStoreFixture(t *testing.T) (*store, storedPrivateProject, string, privateMemberLaunchRequest) {
	t.Helper()
	s, p, req, inputs := privateGatewayStoreFixture(t)
	op, _, err := s.acceptPrivateGateway("gateway-owner", p.Binding.ProjectID, req, "private-engine", inputs)
	g02Check(t, err)
	g02Check(t, s.runPrivateGateway(context.Background(), p.Binding.ProjectID, op.ID, &privateGatewayStoreDriver{t: t, s: s}))
	return s, p, p.Members[0].InstanceID, privateMemberRequest(t, s, p.Binding.ProjectID)
}

type privateMemberStoreDriver struct {
	t      *testing.T
	s      *store
	events []string
	fail   string
}

func (d *privateMemberStoreDriver) check(name string, j privatechain.MemberLaunch) error {
	d.t.Helper()
	d.events = append(d.events, name)
	saved, err := loadPrivateMemberLaunch(d.s.db, j.InstanceID)
	g02Check(d.t, err)
	if !reflect.DeepEqual(j, saved) {
		d.t.Fatal("effect preceded committed journal", name)
	}
	if d.fail == name {
		return errors.New("PRIVATE DRIVER DIAGNOSTIC")
	}
	return nil
}
func (d *privateMemberStoreDriver) VerifyPreparation(_ context.Context, j privatechain.MemberLaunch) error {
	return d.check("verify-"+j.Phase, j)
}
func (d *privateMemberStoreDriver) PrepareArtifact(_ context.Context, j privatechain.MemberLaunch) (pierstore.DirectoryIdentity, error) {
	id := j.Gateway.Observation.Root
	id.Inode += 1000
	return id, d.check("artifact", j)
}
func (d *privateMemberStoreDriver) VerifyKey(_ context.Context, j privatechain.MemberLaunch) (privatechain.MemberKeyAcceptance, error) {
	now := time.Now().UTC()
	g := j.Gateway.Observation.Clone()
	g.ObservedAt = now
	return privatechain.MemberKeyAcceptance{CheckedAt: now, ContractDigest: j.ContractDigest, KeySHA256: strings.Repeat("a", 64), Gateway: g, Verified: true}, d.check("key", j)
}
func (d *privateMemberStoreDriver) CreateMember(_ context.Context, j privatechain.MemberLaunch) (string, error) {
	return fmt.Sprintf("%x", sha256.Sum256([]byte("member:"+j.InstanceID))), d.check("create", j)
}
func (d *privateMemberStoreDriver) InspectCreated(_ context.Context, j privatechain.MemberLaunch) error {
	return d.check("inspect", j)
}
func (d *privateMemberStoreDriver) StartMember(_ context.Context, j privatechain.MemberLaunch) error {
	return d.check("start", j)
}
func (d *privateMemberStoreDriver) ObserveRunning(_ context.Context, j privatechain.MemberLaunch) (privatechain.MemberRunningObservation, error) {
	now := time.Now().UTC()
	g := j.Gateway.Observation.Clone()
	g.ObservedAt = now
	process := privatechain.MemberBirthProcess{ObservedAt: now, StartedAt: j.Key.CheckedAt, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, Namespace: g.Namespace, CompleteContract: true, Running: true}
	birth := privatechain.MemberBirthEvidence{InstanceID: j.InstanceID, OperationID: j.OperationID, RuntimeID: j.RuntimeID, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, EmptyObservedAt: j.Key.CheckedAt, CapturedAt: now, Before: process, After: process, Pier: pierstore.Identity{Device: g.Root.Device, Inode: g.Root.Inode + 2000, UrbDevice: g.Root.Device, UrbInode: g.Root.Inode + 2001, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}
	return privatechain.MemberRunningObservation{ObservedAt: now, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, StartedAt: j.Key.CheckedAt, Namespace: g.Namespace, Gateway: g, Peers: j.Peers, CompleteContract: true, Running: true, Birth: &birth}, d.check("observe", j)
}

var privateMemberStorePhases = []string{"planned", "artifact-dispatched", "artifact-created", "key-verified", "create-dispatched", "created", "start-dispatched", "started", "running"}

func advancePrivateMemberStoreFixture(t *testing.T, s *store, p storedPrivateProject, j privatechain.MemberLaunch, phase string) privatechain.MemberLaunch {
	t.Helper()
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	d := &privateMemberStoreDriver{t: t, s: s}
	result := privatechain.MemberLaunchResult{}
	switch phase {
	case "artifact-created":
		result.Artifact, err = d.PrepareArtifact(context.Background(), j)
	case "key-verified":
		k, e := d.VerifyKey(context.Background(), j)
		result.Key = &k
		err = e
	case "created":
		result.ContainerID, err = d.CreateMember(context.Background(), j)
	case "running":
		o, e := d.ObserveRunning(context.Background(), j)
		result.Running = &o
		err = e
	}
	g02Check(t, err)
	next, err := j.Advance(p.Plan, w, j.Phase, phase, result)
	g02Check(t, err)
	g02Check(t, s.advancePrivateMemberLaunch(p.Binding.ProjectID, j.InstanceID, j.OperationID, j.Phase, next))
	return next
}

func TestPrivateMemberStoreExecutionReplayAndReadinessFence(t *testing.T) {
	s, p, instance, req := privateMemberStoreFixture(t)
	project := p.Binding.ProjectID
	op, hit, err := s.acceptPrivateMemberLaunch("owner", project, instance, req, privateMemberStoreContract)
	g02Check(t, err)
	if hit {
		t.Fatal("new admission replayed")
	}
	d := &privateMemberStoreDriver{t: t, s: s}
	g02Check(t, s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d))
	saved, err := loadPrivateMemberLaunch(s.db, instance)
	g02Check(t, err)
	actual, err := loadOperation(s.db, op.ID)
	g02Check(t, err)
	if saved.Phase != "running" || actual.Status != "succeeded" || len(d.events) != 10 || saved.RuntimeID != p.Members[0].RuntimeID {
		t.Fatal(saved.Phase, actual.Status, d.events)
	}
	req.ReplayOnly = true
	again, hit, err := s.acceptPrivateMemberLaunch("owner", project, instance, req, nil)
	g02Check(t, err)
	if !hit || !reflect.DeepEqual(again, actual) {
		t.Fatal("replay changed")
	}
	_, _, err = s.acceptPrivateMemberLaunch("another", project, instance, req, nil)
	requireFault(t, err, "replay_not_found")
	changed := req
	changed.GatewayDigest = strings.Repeat("f", 64)
	_, _, err = s.acceptPrivateMemberLaunch("owner", project, instance, changed, nil)
	requireFault(t, err, "request_conflict")
	req.RequestID = "second-request"
	req.ReplayOnly = false
	for _, target := range []string{instance, p.Members[1].InstanceID} {
		_, _, err = s.acceptPrivateMemberLaunch("owner", project, target, req, privateMemberStoreContract)
		requireFault(t, err, "project_recovery_required")
	}
	if s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d) == nil || len(d.events) != 10 {
		t.Fatal("completed launch reran")
	}
}

func TestPrivateMemberStoreRestartAtEveryPhase(t *testing.T) {
	for _, phase := range privateMemberStorePhases {
		t.Run(phase, func(t *testing.T) {
			s, p, instance, req := privateMemberStoreFixture(t)
			project := p.Binding.ProjectID
			op, _, err := s.acceptPrivateMemberLaunch("owner", project, instance, req, privateMemberStoreContract)
			g02Check(t, err)
			j, err := loadPrivateMemberLaunch(s.db, instance)
			g02Check(t, err)
			if phase != "planned" {
				j, err = s.beginPrivateMemberLaunch(project, instance, op.ID)
				g02Check(t, err)
				for _, next := range privateMemberStorePhases[1:] {
					j = advancePrivateMemberStoreFixture(t, s, p, j, next)
					if next == phase {
						break
					}
				}
			}
			raw, _ := json.Marshal(j)
			dir := s.directory
			g02Check(t, s.close())
			for pass := 0; pass < 3; pass++ {
				s, err = testOpenStore(dir)
				g02Check(t, err)
				g02Check(t, s.reconcileInterrupted())
				after, e := loadPrivateMemberLaunch(s.db, instance)
				g02Check(t, e)
				body, _ := json.Marshal(after)
				if !bytes.Equal(raw, body) {
					t.Fatal("restart rewrote launch evidence")
				}
				req.ReplayOnly = true
				receipt, hit, e := s.acceptPrivateMemberLaunch("owner", project, instance, req, nil)
				g02Check(t, e)
				status := "interrupted"
				if phase == "running" {
					status = "succeeded"
				}
				if !hit || receipt.Status != status {
					t.Fatal("restart lost receipt", receipt.Status)
				}
				d := &privateMemberStoreDriver{t: t, s: s}
				if s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d) == nil || len(d.events) != 0 {
					t.Fatal("restart authorized boot")
				}
				g02Check(t, s.close())
			}
		})
	}
}

func TestPrivateMemberStoreKnownObjectsSurviveDriverErrors(t *testing.T) {
	for name, phase := range map[string]string{"artifact": "artifact-created", "key": "artifact-created", "create": "created", "inspect": "created", "start": "start-dispatched", "observe": "started"} {
		t.Run(name, func(t *testing.T) {
			s, p, instance, req := privateMemberStoreFixture(t)
			project := p.Binding.ProjectID
			op, _, err := s.acceptPrivateMemberLaunch("owner", project, instance, req, privateMemberStoreContract)
			g02Check(t, err)
			d := &privateMemberStoreDriver{t: t, s: s, fail: name}
			err = s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d)
			if err == nil || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("missing or exposed error", err)
			}
			j, err := loadPrivateMemberLaunch(s.db, instance)
			g02Check(t, err)
			receipt, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if j.Phase != phase || receipt.Status != "interrupted" {
				t.Fatal(j.Phase, receipt.Status)
			}
			count := len(d.events)
			if s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d) == nil || len(d.events) != count {
				t.Fatal("failed launch repeated")
			}
		})
	}
}

func TestPrivateMemberStoreAtomicAdmissionAndPhaseCommit(t *testing.T) {
	for _, target := range []string{"insert-journal", "insert-operation", "mirror", "advance-journal", "advance-mirror", "completion"} {
		t.Run(target, func(t *testing.T) {
			s, p, instance, req := privateMemberStoreFixture(t)
			project := p.Binding.ProjectID
			event, table := "INSERT", "private_member_launches"
			if target == "insert-operation" {
				table = "operations"
			}
			if target == "mirror" {
				event, table = "UPDATE OF runtime", "operations"
			}
			if strings.HasPrefix(target, "advance-") || target == "completion" {
				op, _, err := s.acceptPrivateMemberLaunch("owner", project, instance, req, privateMemberStoreContract)
				g02Check(t, err)
				j, err := s.beginPrivateMemberLaunch(project, instance, op.ID)
				g02Check(t, err)
				if target == "completion" {
					for _, phase := range privateMemberStorePhases[1 : len(privateMemberStorePhases)-1] {
						j = advancePrivateMemberStoreFixture(t, s, p, j, phase)
					}
				}
				event, table = "UPDATE OF body", "private_member_launches"
				if target == "advance-mirror" {
					event, table = "UPDATE OF runtime", "operations"
				}
				if target == "completion" {
					event, table = "UPDATE OF body", "operations"
				}
				_, err = s.db.Exec("CREATE TRIGGER fail_member BEFORE " + event + " ON " + table + " BEGIN SELECT RAISE(ABORT,'synthetic failure'); END;")
				g02Check(t, err)
				w, err := readyPrivateWorkspace(s.db, project)
				g02Check(t, err)
				phase := "artifact-dispatched"
				result := privatechain.MemberLaunchResult{}
				if target == "completion" {
					phase = "running"
					d := &privateMemberStoreDriver{t: t, s: s}
					o, e := d.ObserveRunning(context.Background(), j)
					g02Check(t, e)
					result.Running = &o
				}
				next, err := j.Advance(p.Plan, w, j.Phase, phase, result)
				g02Check(t, err)
				if s.advancePrivateMemberLaunch(project, instance, op.ID, j.Phase, next) == nil {
					t.Fatal("injected failure accepted")
				}
				got, err := loadPrivateMemberLaunch(s.db, instance)
				g02Check(t, err)
				if !reflect.DeepEqual(got, j) {
					t.Fatal("partial phase committed")
				}
				g02Check(t, validatePrivateMemberLaunches(s.db))
				return
			}
			_, err := s.db.Exec("CREATE TRIGGER fail_member BEFORE " + event + " ON " + table + " BEGIN SELECT RAISE(ABORT,'synthetic failure'); END;")
			g02Check(t, err)
			if _, _, err = s.acceptPrivateMemberLaunch("owner", project, instance, req, privateMemberStoreContract); err == nil {
				t.Fatal("injected admission failure accepted")
			}
			var a, b int
			g02Check(t, s.db.QueryRow("SELECT count(*) FROM private_member_launches").Scan(&a))
			g02Check(t, s.db.QueryRow("SELECT count(*) FROM operations WHERE actor='owner' AND request_id=?", req.RequestID).Scan(&b))
			if a != 0 || b != 0 {
				t.Fatal("partial admission committed", a, b)
			}
		})
	}
}

func TestPrivateMemberStoreCorruptionRefusesUnchangedOpen(t *testing.T) {
	cases := map[string]string{
		"journal":       `UPDATE private_member_launches SET body=json_set(body,'$.runtime_id','foreign')`,
		"encoding":      `UPDATE private_member_launches SET body=body||' '`,
		"unknown-field": `UPDATE private_member_launches SET body=json_set(body,'$.unknown',true)`,
		"mirror":        `UPDATE operations SET runtime='{}' WHERE json_extract(body,'$.kind')='launch-private-project-member'`,
		"fingerprint":   `UPDATE operations SET fingerprint='changed' WHERE json_extract(body,'$.kind')='launch-private-project-member'`,
		"actor":         `UPDATE operations SET actor='changed' WHERE json_extract(body,'$.kind')='launch-private-project-member'`,
		"orphan":        `DELETE FROM private_member_launches`,
		"phase":         `UPDATE private_member_launches SET body=json_set(body,'$.phase','running')`,
	}
	for name, stmt := range cases {
		t.Run(name, func(t *testing.T) {
			s, p, instance, req := privateMemberStoreFixture(t)
			_, _, err := s.acceptPrivateMemberLaunch("owner", p.Binding.ProjectID, instance, req, privateMemberStoreContract)
			g02Check(t, err)
			_, err = s.db.Exec(stmt)
			g02Check(t, err)
			dir := s.directory
			g02Check(t, s.close())
			path := filepath.Join(dir, "runner.sqlite")
			before, err := os.ReadFile(path)
			g02Check(t, err)
			opened, err := openStore(dir)
			if err == nil {
				opened.close()
				t.Fatal("corrupt launch opened")
			}
			after, err := os.ReadFile(path)
			g02Check(t, err)
			if !bytes.Equal(before, after) {
				t.Fatal("refused open changed evidence")
			}
		})
	}
}
