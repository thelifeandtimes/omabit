package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type capture46WorkspaceDriver struct{ *birth45WorkspaceDriver }

func (d *capture46WorkspaceDriver) CreateRoot(_ context.Context, w privatechain.Workspace) (pierstore.DirectoryIdentity, error) {
	id := w.Parent
	id.Inode = 60000
	return id, d.check("root", "root-dispatched", w)
}

func capture46Project(t *testing.T, s *store) (storedPrivateProject, string, privateMemberLaunchRequest) {
	t.Helper()
	s.portAvailable = func(string, uint16) bool { return true }
	op, hit, err := s.registerPrivateProject("capture-owner", privateProjectRequest(t, "register-capture46", "capture46", 29080))
	g02Check(t, err)
	if hit {
		t.Fatal("fixture project replayed")
	}
	p, _ := readPrivateProject(t, s, op.ProjectID)
	digest, err := p.Plan.Digest()
	g02Check(t, err)
	req := privateWorkspaceRequest{RequestID: "prepare-capture46", ExpectedRevision: p.Revision, ReviewDigest: digest}
	op, hit, err = s.acceptPrivateWorkspace("capture-owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	if hit {
		t.Fatal("fixture workspace replayed")
	}
	driver := &capture46WorkspaceDriver{&birth45WorkspaceDriver{&privateWorkspaceStoreDriver{t: t, s: s}}}
	g02Check(t, s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, driver))
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	lifecycleGateway43(t, s, w, "ready")
	memberReq := privateMemberRequest(t, s, p.Binding.ProjectID)
	memberReq.RequestID = "capture46-member"
	return p, p.Members[0].InstanceID, memberReq
}

type capture46StoreDriver struct{ *privateMemberStoreDriver }

func (d *capture46StoreDriver) CaptureBirth(_ context.Context, j privatechain.MemberLaunch) (privatechain.MemberBirthEvidence, error) {
	now := time.Now().UTC()
	process := privatechain.MemberBirthProcess{ObservedAt: now, StartedAt: j.Key.CheckedAt, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, Namespace: j.Gateway.Observation.Namespace, CompleteContract: true, Running: true}
	b := privatechain.MemberBirthEvidence{InstanceID: j.InstanceID, OperationID: j.OperationID, RuntimeID: j.RuntimeID, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, EmptyObservedAt: j.Key.CheckedAt, CapturedAt: now, Before: process, After: process, Pier: pierstore.Identity{Device: j.Gateway.Observation.Root.Device, Inode: 62000, UrbDevice: j.Gateway.Observation.Root.Device, UrbInode: 62001, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}
	return b, d.check("capture", j)
}
func (d *capture46StoreDriver) ObserveRunning(_ context.Context, j privatechain.MemberLaunch) (privatechain.MemberRunningObservation, error) {
	if j.Phase != "captured" || j.Capture == nil {
		d.t.Fatal("observation preceded durable capture")
	}
	now := time.Now().UTC()
	g := j.Gateway.Observation.Clone()
	g.ObservedAt = now
	b := *j.Capture
	return privatechain.MemberRunningObservation{ObservedAt: now, ContainerID: j.ContainerID, ContractDigest: j.ContractDigest, StartedAt: b.After.StartedAt, Namespace: g.Namespace, Gateway: g, Peers: j.Peers, CompleteContract: true, Running: true, Birth: &b}, d.check("observe", j)
}

func TestPrivateCapture46ReopenPreservesOriginalAndNewCapture(t *testing.T) {
	for _, kind := range []string{"running", "captured", "started"} {
		t.Run(kind, func(t *testing.T) {
			original := originalPrivate45(t)
			dir := g02Materialize(t, original)
			s := g02Open(t, dir)
			g02AssertRows(t, s.db, original, nil)
			var version int
			g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
			if version != StateSchemaVersion {
				t.Fatal("successor format not fenced")
			}
			p, instance, req := capture46Project(t, s)
			project := p.Binding.ProjectID
			op, hit, err := s.acceptCapturedPrivateMemberLaunch("capture-owner", project, instance, req, privateMemberStoreContract)
			g02Check(t, err)
			if hit {
				t.Fatal("new launch replayed")
			}
			d := &capture46StoreDriver{&privateMemberStoreDriver{t: t, s: s}}
			if kind == "captured" {
				d.fail = "observe"
			}
			if kind == "started" {
				d.fail = "capture"
			}
			err = s.runPrivateMemberLaunch(context.Background(), project, instance, op.ID, d)
			if (err == nil) != (kind == "running") {
				t.Fatal("unexpected execution outcome", err)
			}
			before, err := loadPrivateMemberLaunch(s.db, instance)
			g02Check(t, err)
			receipt, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if before.Version != 3 || before.Phase != kind || (before.Capture == nil) != (kind == "started") {
				t.Fatal("wrong capture history")
			}
			assertOriginal44WithBirth45(t, s, original)
			g02Check(t, s.close())
			for pass := 0; pass < 3; pass++ {
				s = g02Open(t, dir)
				assertOriginal44WithBirth45(t, s, original)
				after, err := loadPrivateMemberLaunch(s.db, instance)
				g02Check(t, err)
				if !reflect.DeepEqual(before, after) {
					t.Fatal("capture changed on reopen")
				}
				req.ReplayOnly = true
				again, hit, err := s.acceptCapturedPrivateMemberLaunch("capture-owner", project, instance, req, nil)
				g02Check(t, err)
				if !hit || !reflect.DeepEqual(again, receipt) {
					t.Fatal("actor replay changed")
				}
				_, _, err = s.acceptCapturedPrivateMemberLaunch("foreign-owner", project, instance, req, nil)
				requireFault(t, err, "replay_not_found")
				_, err = s.beginPrivateMemberLaunch(project, instance, op.ID)
				requireFault(t, err, "stale_project_operation")
				g02Check(t, privateGatewayState(s.db))
				g02Check(t, s.close())
			}
			g02Sidecars(t, dir, original)
		})
	}
}

func capture46Started(t *testing.T, s *store) (storedPrivateProject, privatechain.MemberLaunch, privatechain.MemberLaunch) {
	t.Helper()
	p, instance, req := capture46Project(t, s)
	op, _, err := s.acceptCapturedPrivateMemberLaunch("capture-owner", p.Binding.ProjectID, instance, req, privateMemberStoreContract)
	g02Check(t, err)
	j, err := s.beginPrivateMemberLaunch(p.Binding.ProjectID, instance, op.ID)
	g02Check(t, err)
	for _, phase := range privateMemberStorePhases[1 : len(privateMemberStorePhases)-1] {
		j = advancePrivateMemberStoreFixture(t, s, p, j, phase)
	}
	d := &capture46StoreDriver{&privateMemberStoreDriver{t: t, s: s}}
	b, err := d.CaptureBirth(context.Background(), j)
	g02Check(t, err)
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	next, err := j.Advance(p.Plan, w, "started", "captured", privatechain.MemberLaunchResult{Capture: &b})
	g02Check(t, err)
	return p, j, next
}

func TestPrivateCapture46CommitMirrorAtomicityAndFreshness(t *testing.T) {
	for _, kind := range []string{"mirror-write-failure", "future", "substituted-selection"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			p, j, next := capture46Started(t, s)
			if kind == "mirror-write-failure" {
				_, err := s.db.Exec(`CREATE TRIGGER capture_test_refusal BEFORE UPDATE OF runtime ON operations WHEN json_extract(NEW.runtime,'$.phase')='captured' BEGIN SELECT RAISE(ABORT,'synthetic mirror refusal'); END`)
				g02Check(t, err)
			} else if kind == "future" {
				next.Capture.CapturedAt = time.Now().UTC().Add(6 * time.Second)
			} else {
				next.RuntimeID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			}
			if err := s.advancePrivateMemberLaunch(p.Binding.ProjectID, j.InstanceID, j.OperationID, "started", next); err == nil {
				t.Fatal("invalid/partial commit accepted")
			}
			after, err := loadPrivateMemberLaunch(s.db, j.InstanceID)
			g02Check(t, err)
			if !reflect.DeepEqual(j, after) {
				t.Fatal("journal changed after refused commit")
			}
			var mirror []byte
			g02Check(t, s.db.QueryRow("SELECT runtime FROM operations WHERE id=?", j.OperationID).Scan(&mirror))
			want, _ := json.Marshal(j)
			if !bytes.Equal(mirror, want) {
				t.Fatal("operation mirror changed after refused commit")
			}
		})
	}
}

func TestPrivateCapture46GlobalCaptureClaimRollsBackCollision(t *testing.T) {
	f := originalPrivate45(t)
	s := g02Open(t, g02Materialize(t, f))
	defer s.close()
	originals, err := privateMemberLaunches(s.db)
	g02Check(t, err)
	var occupied pierstore.Identity
	for _, j := range originals {
		if j.Running != nil && j.Running.Birth != nil {
			occupied = j.Running.Birth.Pier
			break
		}
	}
	p, j, next := capture46Started(t, s)
	next.Capture.Pier = occupied
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	g02Check(t, next.Validate(p.Plan, w))
	if err = s.advancePrivateMemberLaunch(p.Binding.ProjectID, j.InstanceID, j.OperationID, "started", next); err == nil {
		t.Fatal("capture claimed another project's pier")
	}
	after, err := loadPrivateMemberLaunch(s.db, j.InstanceID)
	g02Check(t, err)
	if !reflect.DeepEqual(after, j) {
		t.Fatal("collision changed journal")
	}
	g02Check(t, privateGatewayState(s.db))
}

func TestPrivateCapture46InvalidOriginalRollsBackMigration(t *testing.T) {
	f := originalPrivate45(t)
	dir := g02Materialize(t, f)
	db := g02RawDB(t, dir)
	_, err := db.Exec("UPDATE private_member_launches SET body='{}' WHERE instance_id=(SELECT instance_id FROM private_member_launches ORDER BY instance_id LIMIT 1)")
	g02Check(t, err)
	g02Check(t, db.Close())
	path := filepath.Join(dir, "runner.sqlite")
	before, err := os.ReadFile(path)
	g02Check(t, err)
	for pass := 0; pass < 2; pass++ {
		s, err := openStore(dir)
		if err == nil {
			s.close()
			t.Fatal("corrupt prior state migrated")
		}
	}
	after, err := os.ReadFile(path)
	g02Check(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("failed migration changed database")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var version int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	if version != 45 {
		t.Fatal("failed migration changed schema")
	}
}

// Populate an interrupted capture in a genuine successor for the original-reader
// harness. This uses a separate project and leaves original45 histories intact.
func populatePrivateCapture46(t *testing.T, s *store) privatechain.MemberLaunch {
	t.Helper()
	p, instance, req := capture46Project(t, s)
	op, hit, err := s.acceptCapturedPrivateMemberLaunch("capture-owner", p.Binding.ProjectID, instance, req, privateMemberStoreContract)
	g02Check(t, err)
	if hit {
		t.Fatal("successor capture replayed")
	}
	d := &capture46StoreDriver{&privateMemberStoreDriver{t: t, s: s, fail: "observe"}}
	if s.runPrivateMemberLaunch(context.Background(), p.Binding.ProjectID, instance, op.ID, d) == nil {
		t.Fatal("successor requires interrupted capture")
	}
	j, err := loadPrivateMemberLaunch(s.db, instance)
	g02Check(t, err)
	if j.Version != 3 || j.Phase != "captured" || j.Capture == nil {
		t.Fatal("successor capture missing")
	}
	return j
}
