package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateWorkspaceParent() pierstore.DirectoryIdentity {
	return pierstore.DirectoryIdentity{Device: 12, Inode: 9, Birth: "1000000000:1"}
}
func privateWorkspaceFixture(t *testing.T, s *store) (storedPrivateProject, privateWorkspaceRequest) {
	t.Helper()
	op, _, err := s.registerPrivateProject("owner", privateProjectRequest(t, "register-workspace", "workspace", 19080))
	g02Check(t, err)
	p, _ := readPrivateProject(t, s, op.ProjectID)
	digest, err := p.Plan.Digest()
	g02Check(t, err)
	return p, privateWorkspaceRequest{RequestID: "prepare-workspace", ExpectedRevision: 1, ReviewDigest: digest}
}

type privateWorkspaceStoreDriver struct {
	t     *testing.T
	s     *store
	fail  string
	calls []string
}

func (d *privateWorkspaceStoreDriver) check(name, phase string, w privatechain.Workspace) error {
	d.t.Helper()
	d.calls = append(d.calls, name)
	saved, err := loadPrivateWorkspace(d.s.db, w.Binding.ProjectID)
	g02Check(d.t, err)
	if !reflect.DeepEqual(saved, w) || saved.Phase != phase {
		d.t.Fatal("effect preceded its durable phase", name, saved, w)
	}
	if d.fail == name {
		return errors.New("synthetic lost reply")
	}
	return nil
}
func (d *privateWorkspaceStoreDriver) CreateRoot(_ context.Context, w privatechain.Workspace) (pierstore.DirectoryIdentity, error) {
	err := d.check("root", "root-dispatched", w)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	id := privateWorkspaceParent()
	id.Inode = 10
	return id, nil
}
func (d *privateWorkspaceStoreDriver) InspectRoot(_ context.Context, w privatechain.Workspace) error {
	return d.check("inspect", "root-created", w)
}
func (d *privateWorkspaceStoreDriver) CreateLayout(_ context.Context, w privatechain.Workspace) error {
	return d.check("layout", "layout-dispatched", w)
}
func (d *privateWorkspaceStoreDriver) ObserveLayout(_ context.Context, w privatechain.Workspace) (privatechain.WorkspaceObservation, error) {
	if err := d.check("observe", "layout-created", w); err != nil {
		return privatechain.WorkspaceObservation{}, err
	}
	piers := w.Root
	piers.Inode = 11
	a := w.Root
	a.Inode = 12
	b := w.Root
	b.Inode = 13
	return privatechain.WorkspaceObservation{ObservedAt: time.Now().UTC(), Parent: w.Parent, Root: w.Root, PiersRoot: piers, Mounts: []pierstore.DirectoryIdentity{a, b}, Complete: true, Private: true, Empty: true, Synced: true}, nil
}

func TestPrivateWorkspaceDurableExecutionAndActorReplay(t *testing.T) {
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	_, original := readPrivateProject(t, s, p.Binding.ProjectID)
	op, hit, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	if hit || op.Status != "queued" {
		t.Fatal(op, hit)
	}
	d := &privateWorkspaceStoreDriver{t: t, s: s}
	g02Check(t, s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, d))
	if !reflect.DeepEqual(d.calls, []string{"root", "inspect", "layout", "observe"}) {
		t.Fatal(d.calls)
	}
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	actual, err := loadOperation(s.db, op.ID)
	g02Check(t, err)
	if w.Phase != "observed" || actual.Status != "succeeded" {
		t.Fatal(w, actual)
	}
	req.ReplayOnly = true
	// Replaying never obtains new filesystem authority or changes the saved parent.
	replayed, hit, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, pierstore.DirectoryIdentity{})
	g02Check(t, err)
	if !hit || !reflect.DeepEqual(replayed, actual) {
		t.Fatal(replayed, hit)
	}
	_, _, err = s.acceptPrivateWorkspace("other", p.Binding.ProjectID, req, privateWorkspaceParent())
	requireFault(t, err, "replay_not_found")
	req.ReviewDigest = strings.Repeat("0", 64)
	_, _, err = s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	requireFault(t, err, "request_conflict")
	_, after := readPrivateProject(t, s, p.Binding.ProjectID)
	if !bytes.Equal(original, after) {
		t.Fatal("rewrote original project reservation")
	}
	if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil {
		t.Fatal("replayed completed creation")
	}
	if len(d.calls) != 4 {
		t.Fatal("driver ran on replay")
	}
}

func TestPrivateWorkspaceStartupRetainsEveryInterruptedPhase(t *testing.T) {
	for _, phase := range []string{"planned", "root-dispatched", "root-created", "layout-dispatched", "layout-created"} {
		t.Run(phase, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			p, req := privateWorkspaceFixture(t, s)
			op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
			g02Check(t, err)
			w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if phase != "planned" {
				w, err = s.beginPrivateWorkspace(p.Binding.ProjectID, op.ID)
				g02Check(t, err)
				for _, nextPhase := range []string{"root-dispatched", "root-created", "layout-dispatched", "layout-created"} {
					root := w.Root
					if nextPhase == "root-created" {
						root = privateWorkspaceParent()
						root.Inode = 10
					}
					next, e := w.Advance(p.Plan, w.Phase, nextPhase, root, nil)
					g02Check(t, e)
					g02Check(t, s.advancePrivateWorkspace(p.Binding.ProjectID, op.ID, w.Phase, next))
					w = next
					if nextPhase == phase {
						break
					}
				}
			}
			before, _ := json.Marshal(w)
			g02Check(t, s.close())
			s, err = testOpenStore(dir)
			g02Check(t, err)
			defer s.close()
			g02Check(t, s.reconcileInterrupted())
			g02Check(t, s.reconcileInterrupted())
			got, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			after, _ := json.Marshal(got)
			if !bytes.Equal(before, after) {
				t.Fatal("startup changed retained filesystem intent")
			}
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if actual.Status != "interrupted" {
				t.Fatal(actual)
			}
			d := &privateWorkspaceStoreDriver{t: t, s: s}
			if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil || len(d.calls) != 0 {
				t.Fatal("startup/retry repeated mutation", err)
			}
			req.ReplayOnly = true
			replayed, hit, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, pierstore.DirectoryIdentity{})
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(replayed, actual) {
				t.Fatal("lost original receipt")
			}
			req.ReplayOnly = false
			req.RequestID = "replacement-request"
			_, _, err = s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
			requireFault(t, err, "project_recovery_required")
		})
	}
}

func TestPrivateWorkspaceDriverFailureRetainsOriginalIntent(t *testing.T) {
	for _, tc := range []struct{ call, phase string }{{"root", "root-dispatched"}, {"inspect", "root-created"}, {"layout", "layout-dispatched"}, {"observe", "layout-created"}} {
		t.Run(tc.call, func(t *testing.T) {
			s := testStore(t)
			p, req := privateWorkspaceFixture(t, s)
			op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
			g02Check(t, err)
			d := &privateWorkspaceStoreDriver{t: t, s: s, fail: tc.call}
			if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil {
				t.Fatal("lost reply reported success")
			}
			w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if w.Phase != tc.phase || actual.Status != "interrupted" || d.calls[len(d.calls)-1] != tc.call {
				t.Fatal(w, actual, d.calls)
			}
		})
	}
}

func TestPrivateWorkspaceAdmissionAndAdvanceRollback(t *testing.T) {
	for _, kind := range []string{"insert-journal", "mirror", "advance"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			p, req := privateWorkspaceFixture(t, s)
			if kind != "advance" {
				stmt := `CREATE TRIGGER fail_workspace BEFORE INSERT ON private_project_workspaces BEGIN SELECT RAISE(ABORT,'synthetic journal failure'); END;`
				if kind == "mirror" {
					stmt = `CREATE TRIGGER fail_workspace BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`
				}
				_, err := s.db.Exec(stmt)
				g02Check(t, err)
				if _, _, err = s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent()); err == nil {
					t.Fatal("incomplete acceptance succeeded")
				}
				var n int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM private_project_workspaces").Scan(&n))
				if n != 0 {
					t.Fatal("partial journal")
				}
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM operations WHERE request_id=?", req.RequestID).Scan(&n))
				if n != 0 {
					t.Fatal("partial receipt")
				}
				return
			}
			op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
			g02Check(t, err)
			w, err := s.beginPrivateWorkspace(p.Binding.ProjectID, op.ID)
			g02Check(t, err)
			_, err = s.db.Exec(`CREATE TRIGGER fail_workspace BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`)
			g02Check(t, err)
			next, err := w.Advance(p.Plan, w.Phase, "root-dispatched", w.Root, nil)
			g02Check(t, err)
			if err = s.advancePrivateWorkspace(p.Binding.ProjectID, op.ID, w.Phase, next); err == nil {
				t.Fatal("partial advance succeeded")
			}
			got, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if !reflect.DeepEqual(got, w) {
				t.Fatal("checkpoint changed after failed mirror")
			}
		})
	}
}

func TestPrivateWorkspaceCorruptionRefusesOpenAndReplay(t *testing.T) {
	for name, mutation := range map[string]string{
		"journal":       `UPDATE private_project_workspaces SET body=json_set(body,'$.binding.runner_id','wrong')`,
		"unknown-field": `UPDATE private_project_workspaces SET body=json_set(body,'$.unknown',true)`,
		"mirror":        `UPDATE operations SET runtime='{}' WHERE json_extract(body,'$.kind')='prepare-private-project-workspace'`,
		"fingerprint":   `UPDATE operations SET fingerprint='changed' WHERE json_extract(body,'$.kind')='prepare-private-project-workspace'`,
		"actor":         `UPDATE operations SET actor='changed' WHERE json_extract(body,'$.kind')='prepare-private-project-workspace'`,
		"orphan":        `DELETE FROM private_project_workspaces`,
		"wrong-phase":   `UPDATE private_project_workspaces SET body=json_set(body,'$.phase','observed')`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			p, req := privateWorkspaceFixture(t, s)
			_, _, err = s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
			g02Check(t, err)
			_, err = s.db.Exec(mutation)
			g02Check(t, err)
			if _, _, err = s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent()); err == nil {
				t.Fatal("corrupt replay passed")
			}
			g02Check(t, s.close())
			broken, err := testOpenStore(dir)
			if err == nil {
				broken.close()
				t.Fatal("corrupt journal reopened")
			}
		})
	}
}

func TestPrivateWorkspaceCannotRebindParentOrImmutableCallback(t *testing.T) {
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	w, err := s.beginPrivateWorkspace(p.Binding.ProjectID, op.ID)
	g02Check(t, err)
	next, err := w.Advance(p.Plan, w.Phase, "root-dispatched", w.Root, nil)
	g02Check(t, err)
	for _, change := range []func(*privatechain.Workspace){func(w *privatechain.Workspace) { w.Parent.Inode++ }, func(w *privatechain.Workspace) { w.Binding.RunnerID = "runner-other" }, func(w *privatechain.Workspace) { w.OperationID = "operation-other" }} {
		bad := next.Clone()
		change(&bad)
		if err = s.advancePrivateWorkspace(p.Binding.ProjectID, op.ID, w.Phase, bad); err == nil {
			t.Fatal("changed ownership")
		}
	}
	second, _, err := s.registerPrivateProject("owner", privateProjectRequest(t, "second", "second", 20080))
	g02Check(t, err)
	p2, _ := readPrivateProject(t, s, second.ProjectID)
	digest, _ := p2.Plan.Digest()
	parent := privateWorkspaceParent()
	parent.Inode++
	_, _, err = s.acceptPrivateWorkspace("owner", p2.Binding.ProjectID, privateWorkspaceRequest{RequestID: "prepare-second", ExpectedRevision: 1, ReviewDigest: digest}, parent)
	if err == nil {
		t.Fatal("same runner accepted another workspace parent")
	}
}
