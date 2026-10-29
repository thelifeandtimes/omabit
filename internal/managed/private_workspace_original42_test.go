package managed

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// The frozen original42 writer creates both a completed workspace and one
// interrupted after layout dispatch. Keep their exact operation/object bindings
// when private gateway initialization and observation recovery gain persistence.
func TestPrivateWorkspaceOriginal42PreservesCompletedAndInterruptedIntent(t *testing.T) {
	raw, err := os.ReadFile("testdata/historical-upgrades/schema42.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "470bcb0c37df88ca40e45dbb6d06bffb57915455e291b4e13e0be0359ee711d5" {
		t.Fatal("original42 fixture changed")
	}
	var f g02Fixture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	g02Check(t, decoder.Decode(&f))
	archive, err := os.ReadFile("testdata/historical-upgrades/schema42-source.tar.gz")
	g02Check(t, err)
	if f.Schema != 42 || !f.Synthetic || f.SourceSnapshotSHA256 != "5f743bba5b7df16e4bbaa6ccb3f96582e357c12c313882c154c959a64947ccf3" || fmt.Sprintf("%x", sha256.Sum256(archive)) != f.SourceSnapshotSHA256 {
		t.Fatal("original42 provenance changed")
	}
	dir := g02Materialize(t, f)
	for pass := 0; pass < 3; pass++ {
		s := g02Open(t, dir)
		if s.runnerID != f.RunnerID {
			t.Fatal("runner identity changed")
		}
		g02AssertRows(t, s.db, f, nil)
		g02Replay(t, s, f)
		for n, prefix := range []string{"historical-private-a", "historical-private-b"} {
			req := privateProjectRequest(t, "historical-register-"+prefix, prefix, uint16(30080+n*10))
			req.ReplayOnly = true
			actor := "historical-private-actor-" + prefix
			op, hit, err := s.registerPrivateProject(actor, req)
			g02Check(t, err)
			if !hit || op.ProjectID == "" || op.Status != "succeeded" {
				t.Fatal("historical registration replay changed")
			}
			p, err := loadPrivateProject(s.db, op.ProjectID)
			g02Check(t, err)
			digest, err := p.Plan.Digest()
			g02Check(t, err)
			workspaceReq := privateWorkspaceRequest{RequestID: "historical-workspace-" + prefix, ExpectedRevision: 1, ReviewDigest: digest, ReplayOnly: true}
			prepared, hit, err := s.acceptPrivateWorkspace(actor+"-workspace", op.ProjectID, workspaceReq, pierstore.DirectoryIdentity{})
			g02Check(t, err)
			w, err := loadPrivateWorkspace(s.db, op.ProjectID)
			g02Check(t, err)
			if !hit || w.OperationID != prepared.ID || w.Validate(p.Plan) != nil || prepared.FinishedAt == nil {
				t.Fatal("workspace replay/ownership changed")
			}
			if n == 0 && (prepared.Status != "succeeded" || w.Phase != "observed" || w.Root.Inode != 10 || w.Observation == nil) {
				t.Fatal("completed original workspace changed")
			}
			if n == 1 && (prepared.Status != "interrupted" || w.Phase != "layout-dispatched" || w.Root.Inode != 20 || w.Observation != nil) {
				t.Fatal("interrupted original workspace changed")
			}
			_, _, err = s.acceptPrivateWorkspace("another-actor", op.ProjectID, workspaceReq, pierstore.DirectoryIdentity{})
			requireFault(t, err, "replay_not_found")
			_, err = s.beginPrivateWorkspace(op.ProjectID, prepared.ID)
			requireFault(t, err, "stale_project_operation")
		}
		reserved, err := projectReservations(s.db)
		g02Check(t, err)
		for _, port := range []uint16{28080, 28081, 30080, 30081, 30090, 30091} {
			if !reserved.HTTPPorts[port] {
				t.Fatal("historical shared reservation lost", port)
			}
		}
		g02AssertRows(t, s.db, f, nil)
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}
