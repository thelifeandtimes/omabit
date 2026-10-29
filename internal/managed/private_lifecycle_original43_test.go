//go:build linux && amd64

package managed

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

// The original43 writer retains uncertain effects and separate observation
// receipts. Future member persistence must preserve both histories verbatim.
func TestPrivateLifecycleOriginal43PreservesInitializationAndRecovery(t *testing.T) {
	raw, err := os.ReadFile("testdata/historical-upgrades/schema43.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "004391518e998771b9e739c094d7b6f80b0746dae7624ddfe697f7e9307bdbdc" {
		t.Fatal("original43 fixture changed")
	}
	var f g02Fixture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	g02Check(t, decoder.Decode(&f))
	archive, err := os.ReadFile("testdata/historical-upgrades/schema43-source.tar.gz")
	g02Check(t, err)
	if f.Schema != 43 || !f.Synthetic || f.SourceSnapshotSHA256 != "ff7e9860bc98dbb765e4e2f98535a77a4e8b0898d79327494c1d17b912876222" || fmt.Sprintf("%x", sha256.Sum256(archive)) != f.SourceSnapshotSHA256 {
		t.Fatal("original43 provenance changed")
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
			actor := "historical-private-actor-" + prefix
			registerReq := privateProjectRequest(t, "historical-register-"+prefix, prefix, uint16(30080+n*10))
			registerReq.ReplayOnly = true
			registered, hit, err := s.registerPrivateProject(actor, registerReq)
			g02Check(t, err)
			if !hit || registered.ProjectID == "" || registered.Status != "succeeded" {
				t.Fatal("registration replay changed")
			}
			p, err := loadPrivateProject(s.db, registered.ProjectID)
			g02Check(t, err)
			digest, err := p.Plan.Digest()
			g02Check(t, err)
			workspaceReq := privateWorkspaceRequest{RequestID: "historical-workspace-" + prefix, ExpectedRevision: 1, ReviewDigest: digest, ReplayOnly: true}
			prepared, hit, err := s.acceptPrivateWorkspace(actor+"-workspace", registered.ProjectID, workspaceReq, pierstore.DirectoryIdentity{})
			g02Check(t, err)
			w, err := loadPrivateWorkspace(s.db, registered.ProjectID)
			g02Check(t, err)
			if !hit || w.OperationID != prepared.ID || prepared.FinishedAt == nil {
				t.Fatal("workspace receipt lost")
			}
			if n == 0 && (prepared.Status != "succeeded" || w.Phase != "observed" || w.Root.Inode != 10 || w.Observation == nil) {
				t.Fatal("completed workspace changed")
			}
			if n == 1 && (prepared.Status != "interrupted" || w.Phase != "layout-dispatched" || w.Root.Inode != 20 || w.Observation != nil) {
				t.Fatal("uncertain workspace rewritten")
			}
			_, _, err = s.acceptPrivateWorkspace("another-actor", registered.ProjectID, workspaceReq, pierstore.DirectoryIdentity{})
			requireFault(t, err, "replay_not_found")
			_, err = s.beginPrivateWorkspace(registered.ProjectID, prepared.ID)
			requireFault(t, err, "stale_project_operation")
			readyWorkspace, workspaceEvidence, err := readyPrivateWorkspaceEvidence(s.db, registered.ProjectID)
			g02Check(t, err)
			if readyWorkspace.OperationID != prepared.ID || readyWorkspace.Phase != "observed" {
				t.Fatal("ready workspace lost original owner")
			}
			if n == 0 && workspaceEvidence.ID != prepared.ID {
				t.Fatal("direct workspace attribution changed")
			}
			if n == 1 {
				for _, suffix := range []string{"interrupted", "complete"} {
					req := privateWorkspaceObservationRequest{RequestID: "historical-workspace-observation-" + suffix, ExpectedRevision: 1, ReviewDigest: w.PlanDigest, SourceOperationID: w.OperationID, SourceDigest: privateWorkspaceSourceDigest(w), ReplayOnly: true}
					op, hit, err := s.acceptPrivateWorkspaceObservation(actor+"-observer", registered.ProjectID, req)
					g02Check(t, err)
					observation, err := loadPrivateWorkspaceObservation(s.db, op.ID)
					g02Check(t, err)
					if !hit || op.FinishedAt == nil || observation.SourceOperationID != prepared.ID {
						t.Fatal("workspace observation owner changed")
					}
					if suffix == "interrupted" && (op.Status != "interrupted" || observation.Phase != "observing" || observation.Result != nil) {
						t.Fatal("uncertain workspace observation changed")
					}
					if suffix == "complete" && (op.Status != "succeeded" || observation.Phase != "observed" || observation.Result == nil || workspaceEvidence.ID != op.ID) {
						t.Fatal("workspace recovery attribution changed")
					}
					_, _, err = s.acceptPrivateWorkspaceObservation("another-actor", registered.ProjectID, req)
					requireFault(t, err, "replay_not_found")
					_, err = s.beginPrivateWorkspaceObservation(op.ID)
					requireFault(t, err, "stale_project_operation")
				}
			}
			g, err := loadPrivateGateway(s.db, registered.ProjectID)
			g02Check(t, err)
			req := privateGatewayRequest{RequestID: "historical-gateway-" + prefix, ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: g.Selection.WorkspaceDigest, InputManifestSHA256: g.Selection.Inputs.ManifestSHA256, ReplayOnly: true}
			initialized, hit, err := s.acceptPrivateGateway(actor+"-gateway", registered.ProjectID, req, "", privatechain.RuntimeInputInspection{})
			g02Check(t, err)
			if !hit || initialized.ID != g.Selection.OperationID || initialized.FinishedAt == nil {
				t.Fatal("gateway receipt changed")
			}
			if n == 0 && (g.Phase != "started" || g.Observation != nil || initialized.Status != "interrupted") {
				t.Fatal("uncertain gateway initialization rewritten")
			}
			if n == 1 && (g.Phase != "ready" || g.Observation == nil || initialized.Status != "succeeded") {
				t.Fatal("completed gateway initialization changed")
			}
			_, _, err = s.acceptPrivateGateway("another-actor", registered.ProjectID, req, "", privatechain.RuntimeInputInspection{})
			requireFault(t, err, "replay_not_found")
			_, err = s.beginPrivateGateway(registered.ProjectID, initialized.ID)
			requireFault(t, err, "stale_project_operation")
			ready, evidence, err := readyPrivateGateway(s.db, registered.ProjectID)
			g02Check(t, err)
			if ready.Phase != "ready" || ready.Selection.OperationID != initialized.ID || ready.Observation == nil {
				t.Fatal("ready gateway lost original owner")
			}
			if n == 1 && evidence.ID != initialized.ID {
				t.Fatal("direct gateway attribution changed")
			}
			if n == 0 {
				for _, suffix := range []string{"interrupted", "complete"} {
					observe := privateGatewayObservationRequest{RequestID: "historical-gateway-observation-" + suffix, ExpectedRevision: 1, ReviewDigest: g.Selection.PlanDigest, SourceOperationID: initialized.ID, SourceDigest: privateGatewaySourceDigest(g), ReplayOnly: true}
					op, hit, err := s.acceptPrivateGatewayObservation(actor+"-observer", registered.ProjectID, observe)
					g02Check(t, err)
					observation, err := loadPrivateGatewayObservation(s.db, op.ID)
					g02Check(t, err)
					if !hit || op.FinishedAt == nil || observation.SourceOperationID != initialized.ID {
						t.Fatal("gateway observation owner changed")
					}
					if suffix == "interrupted" && (op.Status != "interrupted" || observation.Phase != "observing" || observation.Result != nil) {
						t.Fatal("uncertain gateway observation changed")
					}
					if suffix == "complete" && (op.Status != "succeeded" || observation.Phase != "observed" || observation.Result == nil || evidence.ID != op.ID) {
						t.Fatal("gateway recovery attribution changed")
					}
					_, _, err = s.acceptPrivateGatewayObservation("another-actor", registered.ProjectID, observe)
					requireFault(t, err, "replay_not_found")
					_, err = s.beginPrivateGatewayObservation(op.ID)
					requireFault(t, err, "stale_project_operation")
				}
			}
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
