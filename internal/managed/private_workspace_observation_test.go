//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateWorkspaceObservationFixture(t *testing.T) (*store, storedPrivateProject, privateWorkspaceObservationRequest) {
	t.Helper()
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	d := &privateWorkspaceStoreDriver{t: t, s: s, fail: "layout"}
	if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, d); err == nil {
		t.Fatal("missing simulated layout failure")
	}
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if w.Phase != "layout-dispatched" {
		t.Fatal(w.Phase)
	}
	return s, p, privateWorkspaceObservationRequest{RequestID: "observe-private-workspace", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, SourceOperationID: w.OperationID, SourceDigest: privateWorkspaceSourceDigest(w)}
}

type workspaceObservationOnly struct {
	t         *testing.T
	s         *store
	operation string
	calls     int
	fail      bool
	modify    func(*privatechain.WorkspaceObservation)
}

func (o *workspaceObservationOnly) ObserveLayout(_ context.Context, w privatechain.Workspace) (privatechain.WorkspaceObservation, error) {
	o.t.Helper()
	o.calls++
	j, err := loadPrivateWorkspaceObservation(o.s.db, o.operation)
	g02Check(o.t, err)
	if j.Phase != "observing" || j.SourceOperationID != w.OperationID || j.SourceDigest != privateWorkspaceSourceDigest(w) {
		o.t.Fatal("observation preceded attributed receipt")
	}
	if o.fail {
		return privatechain.WorkspaceObservation{}, errors.New("private observation detail")
	}
	piers, a, b := w.Root, w.Root, w.Root
	piers.Inode = w.Root.Inode + 1
	a.Inode = w.Root.Inode + 2
	b.Inode = w.Root.Inode + 3
	result := privatechain.WorkspaceObservation{ObservedAt: time.Now().UTC(), Parent: w.Parent, Root: w.Root, PiersRoot: piers, Mounts: []pierstore.DirectoryIdentity{a, b}, Complete: true, Private: true, Empty: true, Synced: true}
	if o.modify != nil {
		o.modify(&result)
	}
	return result, nil
}

func TestPrivateWorkspaceObservationPreservesSourceAndEnablesGateway(t *testing.T) {
	s, p, req := privateWorkspaceObservationFixture(t)
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	source, err := loadOperation(s.db, w.OperationID)
	g02Check(t, err)
	if _, err = readyPrivateWorkspace(s.db, p.Binding.ProjectID); err == nil {
		t.Fatal("unobserved workspace was ready")
	}
	op, hit, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	if hit {
		t.Fatal("new request replayed")
	}
	o := &workspaceObservationOnly{t: t, s: s, operation: op.ID}
	g02Check(t, s.runPrivateWorkspaceObservation(context.Background(), op.ID, o))
	j, err := loadPrivateWorkspaceObservation(s.db, op.ID)
	g02Check(t, err)
	after, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	originalOp, err := loadOperation(s.db, w.OperationID)
	g02Check(t, err)
	if !reflect.DeepEqual(w, after) || !reflect.DeepEqual(source, originalOp) || j.Phase != "observed" || o.calls != 1 {
		t.Fatal("recovery rewrote original intent")
	}
	req.ReplayOnly = true
	replayed, hit, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	if !hit || replayed.ID != op.ID || replayed.Status != "succeeded" {
		t.Fatal("observation replay changed")
	}
	_, _, err = s.acceptPrivateWorkspaceObservation("another-actor", p.Binding.ProjectID, req)
	requireFault(t, err, "replay_not_found")
	ready, evidence, err := readyPrivateWorkspaceEvidence(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	if ready.OperationID != w.OperationID || evidence.ID != op.ID || ready.Phase != "observed" {
		t.Fatal("wrong readiness attribution")
	}
	inputs := privatechain.RuntimeInputInspection{Format: privatechain.RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.Binding.FoundationDigest, Files: 12352, Bytes: 834035190}
	g, err := privatechain.NewGateway(p.Plan, ready, "prototype-gateway", "private-engine", inputs)
	g02Check(t, err)
	initReq := privateGatewayRequest{RequestID: "initialize-recovered-workspace", ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: g.Selection.WorkspaceDigest, InputManifestSHA256: inputs.ManifestSHA256}
	_, _, err = s.acceptPrivateGateway("owner", p.Binding.ProjectID, initReq, "private-engine", inputs)
	g02Check(t, err)
	if err = s.runPrivateWorkspaceObservation(context.Background(), op.ID, o); err == nil || o.calls != 1 {
		t.Fatal("completed observation reexecuted")
	}
	req.RequestID = "duplicate-observation"
	req.ReplayOnly = false
	_, _, err = s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
	requireFault(t, err, "project_operation_pending")
}

func TestPrivateWorkspaceObservationInterruptedAttemptCanBeInspectedAgain(t *testing.T) {
	for _, phase := range []string{"planned", "observing", "failed-observer"} {
		t.Run(phase, func(t *testing.T) {
			s, p, req := privateWorkspaceObservationFixture(t)
			op, _, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			if phase == "observing" {
				_, err = s.beginPrivateWorkspaceObservation(op.ID)
				g02Check(t, err)
			}
			if phase == "failed-observer" {
				o := &workspaceObservationOnly{t: t, s: s, operation: op.ID, fail: true}
				if err = s.runPrivateWorkspaceObservation(context.Background(), op.ID, o); err == nil || strings.Contains(err.Error(), "private observation detail") {
					t.Fatal(err)
				}
			}
			before, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			dir := s.directory
			g02Check(t, s.close())
			s = g02Open(t, dir)
			g02Check(t, s.reconcileInterrupted())
			g02Check(t, s.reconcileInterrupted())
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if actual.Status != "interrupted" {
				t.Fatal(actual.Status)
			}
			_, err = s.beginPrivateWorkspaceObservation(op.ID)
			requireFault(t, err, "stale_project_operation")
			req.RequestID = "observe-again"
			second, _, err := s.acceptPrivateWorkspaceObservation("another-observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			o := &workspaceObservationOnly{t: t, s: s, operation: second.ID}
			g02Check(t, s.runPrivateWorkspaceObservation(context.Background(), second.ID, o))
			after, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
			g02Check(t, err)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("read-only recovery changed original journal")
			}
		})
	}
}

func TestPrivateWorkspaceObservationRejectsIncompleteAndChangedObjects(t *testing.T) {
	for name, modify := range map[string]func(*privatechain.WorkspaceObservation){
		"partial":  func(o *privatechain.WorkspaceObservation) { o.Complete = false },
		"root":     func(o *privatechain.WorkspaceObservation) { o.Root.Inode++ },
		"mount":    func(o *privatechain.WorkspaceObservation) { o.Mounts[0] = o.Mounts[1] },
		"contents": func(o *privatechain.WorkspaceObservation) { o.Empty = false },
		"sync":     func(o *privatechain.WorkspaceObservation) { o.Synced = false },
		"expired":  func(o *privatechain.WorkspaceObservation) { o.ObservedAt = o.ObservedAt.Add(-time.Minute) },
	} {
		t.Run(name, func(t *testing.T) {
			s, p, req := privateWorkspaceObservationFixture(t)
			op, _, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
			g02Check(t, err)
			o := &workspaceObservationOnly{t: t, s: s, operation: op.ID, modify: modify}
			if err = s.runPrivateWorkspaceObservation(context.Background(), op.ID, o); err == nil {
				t.Fatal("bad result committed")
			}
			j, err := loadPrivateWorkspaceObservation(s.db, op.ID)
			g02Check(t, err)
			actual, err := loadOperation(s.db, op.ID)
			g02Check(t, err)
			if j.Result != nil || j.Phase != "observing" || actual.Status != "interrupted" {
				t.Fatal("failed observation fabricated readiness")
			}
		})
	}
}

func TestPrivateWorkspaceObservationAtomicReceiptAndMirror(t *testing.T) {
	for _, phase := range []string{"accept", "begin", "complete"} {
		t.Run(phase, func(t *testing.T) {
			s, p, req := privateWorkspaceObservationFixture(t)
			var id string
			if phase != "accept" {
				op, _, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
				g02Check(t, err)
				id = op.ID
			}
			if phase == "complete" {
				_, err := s.beginPrivateWorkspaceObservation(id)
				g02Check(t, err)
			}
			_, err := s.db.Exec(`CREATE TRIGGER fail_observation BEFORE UPDATE OF runtime ON operations WHEN json_extract(new.body,'$.kind')='observe-private-project-workspace' BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`)
			g02Check(t, err)
			switch phase {
			case "accept":
				_, _, err = s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
			case "begin":
				_, err = s.beginPrivateWorkspaceObservation(id)
			case "complete":
				w, e := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
				g02Check(t, e)
				o := &workspaceObservationOnly{t: t, s: s, operation: id}
				observed, e := o.ObserveLayout(context.Background(), w)
				g02Check(t, e)
				w.Phase = "observed"
				w.Observation = &observed
				err = s.completePrivateWorkspaceObservation(id, w)
			}
			if err == nil {
				t.Fatal("partial transaction committed")
			}
			g02Check(t, privateWorkspaceObservationState(s.db))
			if phase == "accept" {
				var n int
				g02Check(t, s.db.QueryRow("SELECT count(*) FROM private_workspace_observations").Scan(&n))
				if n != 0 {
					t.Fatal("partial receipt")
				}
			}
			if phase != "accept" {
				j, e := loadPrivateWorkspaceObservation(s.db, id)
				g02Check(t, e)
				want := "planned"
				if phase == "complete" {
					want = "observing"
				}
				if j.Phase != want || j.Result != nil {
					t.Fatal("partial journal")
				}
			}
		})
	}
}

func TestPrivateWorkspaceObservationRefusesSourceRebinding(t *testing.T) {
	s, p, req := privateWorkspaceObservationFixture(t)
	op, _, err := s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req)
	g02Check(t, err)
	_, err = s.db.Exec(`UPDATE private_project_workspaces SET body=json_set(body,'$.parent.birth','1000000001:1')`)
	g02Check(t, err)
	_, err = s.db.Exec(`UPDATE operations SET runtime=json_set(runtime,'$.parent.birth','1000000001:1') WHERE id=?`, req.SourceOperationID)
	g02Check(t, err)
	o := &workspaceObservationOnly{t: t, s: s, operation: op.ID}
	if err = s.runPrivateWorkspaceObservation(context.Background(), op.ID, o); err == nil || o.calls != 0 {
		t.Fatal("changed original source reached observer")
	}
	if _, _, err = s.acceptPrivateWorkspaceObservation("observer", p.Binding.ProjectID, req); err == nil {
		t.Fatal("changed original source replayed")
	}
}

func TestPrivateWorkspaceObservationCannotClaimOtherProjectDirectories(t *testing.T) {
	raw, err := os.ReadFile("testdata/historical-upgrades/schema42.json")
	g02Check(t, err)
	var fixture g02Fixture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	g02Check(t, decoder.Decode(&fixture))
	s := g02Open(t, g02Materialize(t, fixture))
	claims, err := privateWorkspaceClaims(s.db)
	g02Check(t, err)
	var pending, complete privatechain.Workspace
	for _, w := range claims {
		if w.Phase == "observed" {
			complete = w
		} else {
			pending = w
		}
	}
	if pending.Phase != "layout-dispatched" || complete.Observation == nil {
		t.Fatal("original42 coverage changed")
	}
	req := privateWorkspaceObservationRequest{RequestID: "cross-project-observation", ExpectedRevision: 1, ReviewDigest: pending.PlanDigest, SourceOperationID: pending.OperationID, SourceDigest: privateWorkspaceSourceDigest(pending)}
	op, _, err := s.acceptPrivateWorkspaceObservation("observer", pending.Binding.ProjectID, req)
	g02Check(t, err)
	o := &workspaceObservationOnly{t: t, s: s, operation: op.ID, modify: func(result *privatechain.WorkspaceObservation) {
		result.PiersRoot = complete.Observation.PiersRoot
		result.Mounts = append([]pierstore.DirectoryIdentity(nil), complete.Observation.Mounts...)
	}}
	if err = s.runPrivateWorkspaceObservation(context.Background(), op.ID, o); err == nil {
		t.Fatal("observation claimed another project's directories")
	}
	j, err := loadPrivateWorkspaceObservation(s.db, op.ID)
	g02Check(t, err)
	if j.Result != nil {
		t.Fatal("cross-project result committed")
	}
	req.RequestID = "correct-project-observation"
	second, _, err := s.acceptPrivateWorkspaceObservation("observer", pending.Binding.ProjectID, req)
	g02Check(t, err)
	g02Check(t, s.runPrivateWorkspaceObservation(context.Background(), second.ID, &workspaceObservationOnly{t: t, s: s, operation: second.ID}))
	ready, _, err := readyPrivateWorkspaceEvidence(s.db, pending.Binding.ProjectID)
	g02Check(t, err)
	if ready.Observation.PiersRoot.SameObject(complete.Observation.PiersRoot) {
		t.Fatal("separate workspace objects lost")
	}
}

type lostLayoutAcknowledgement struct {
	*privatechain.WorkspaceFilesystem
}

func (d lostLayoutAcknowledgement) CreateLayout(ctx context.Context, w privatechain.Workspace) error {
	if err := d.WorkspaceFilesystem.CreateLayout(ctx, w); err != nil {
		return err
	}
	return errors.New("synthetic lost layout acknowledgement")
}

type workspaceObserverView struct{ privatechain.WorkspaceObserver }

func TestPrivateWorkspaceObservationRealDirectoriesAfterLostReply(t *testing.T) {
	s := testStore(t)
	p, req := privateWorkspaceFixture(t, s)
	parentPath := filepath.Join(s.directory, "private-projects")
	g02Check(t, os.Mkdir(parentPath, 0700))
	parent, err := privatechain.InspectWorkspaceParent(context.Background(), parentPath)
	g02Check(t, err)
	op, _, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, parent)
	g02Check(t, err)
	w, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	driver, err := privatechain.OpenWorkspaceFilesystem(context.Background(), parentPath, parent, p.Plan, w)
	g02Check(t, err)
	if err = s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, lostLayoutAcknowledgement{driver}); err == nil {
		t.Fatal("missing lost-reply result")
	}
	g02Check(t, driver.Close())
	w, err = loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	sourceRaw, _ := json.Marshal(w)
	if w.Phase != "layout-dispatched" {
		t.Fatal(w.Phase)
	}
	observationReq := privateWorkspaceObservationRequest{RequestID: "observe-real-workspace", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, SourceOperationID: w.OperationID, SourceDigest: privateWorkspaceSourceDigest(w)}
	receipt, _, err := s.acceptPrivateWorkspaceObservation("recovery-actor", p.Binding.ProjectID, observationReq)
	g02Check(t, err)
	observer, err := privatechain.OpenWorkspaceFilesystem(context.Background(), parentPath, parent, p.Plan, w)
	g02Check(t, err)
	g02Check(t, s.runPrivateWorkspaceObservation(context.Background(), receipt.ID, workspaceObserverView{observer}))
	g02Check(t, observer.Close())
	ready, evidence, err := readyPrivateWorkspaceEvidence(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	after, err := loadPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	afterRaw, _ := json.Marshal(after)
	if !bytes.Equal(sourceRaw, afterRaw) || ready.Phase != "observed" || !ready.Root.SameObject(w.Root) || evidence.ID != receipt.ID {
		t.Fatal("real recovery lost ownership or original intent")
	}
	for _, m := range w.Mounts {
		rows, err := os.ReadDir(filepath.Join(s.directory, m.Path))
		g02Check(t, err)
		if len(rows) != 0 {
			t.Fatal("observation changed empty mount parent")
		}
	}
}
