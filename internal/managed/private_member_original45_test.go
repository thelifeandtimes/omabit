package managed

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// Captured by the original schema45 writer with durable birth, before managed readiness.
// Later migrations must retain both the completed and uncertain first launches.
func originalPrivate45(t *testing.T) g02Fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/historical-upgrades/schema45.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "8908b59a97cad17e10b9cecc3745c207364325c6fab6a3ac5e686a1543f6ed44" {
		t.Fatal("original45 fixture changed")
	}
	var f g02Fixture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	g02Check(t, d.Decode(&f))
	archive, err := os.ReadFile("testdata/historical-upgrades/schema45-source.tar.gz")
	g02Check(t, err)
	if f.Schema != 45 || !f.Synthetic || f.SourceSnapshotSHA256 != "b366caf7df4c488816cb69b24fbeefb082d6b2618283ab76fe2d721a06496da9" || fmt.Sprintf("%x", sha256.Sum256(archive)) != f.SourceSnapshotSHA256 {
		t.Fatal("original45 provenance changed")
	}
	return f
}

func TestPrivateMemberOriginal45PreservesCompletedAndUncertainLaunches(t *testing.T) {
	f := originalPrivate45(t)
	dir := g02Materialize(t, f)
	for pass := 0; pass < 3; pass++ {
		s := g02Open(t, dir)
		if s.runnerID != f.RunnerID {
			t.Fatal("runner identity changed")
		}
		g02AssertRows(t, s.db, f, nil)
		g02Replay(t, s, f)
		members, err := privateMemberLaunches(s.db)
		g02Check(t, err)
		if len(members) != 3 {
			t.Fatal("original member history lost")
		}
		complete, uncertain := 0, 0
		for _, j := range members {
			project := j.Gateway.Selection.Binding.ProjectID
			p, err := loadPrivateProject(s.db, project)
			g02Check(t, err)
			w, err := readyPrivateWorkspace(s.db, project)
			g02Check(t, err)
			g02Check(t, j.Validate(p.Plan, w))
			g, evidence, err := readyPrivateGateway(s.db, project)
			g02Check(t, err)
			if !reflect.DeepEqual(g, j.Gateway) || evidence.Status != "succeeded" {
				t.Fatal("gateway ancestry changed")
			}
			req := privateMemberRequest(t, s, project)
			req.RequestID = "member-" + project
			req.ReplayOnly = true
			actor := "member-owner"
			if j.InstanceID == "workspace-nec" {
				req.RequestID = "capture-member"
				actor = "birth-owner"
			}
			op, hit, err := s.acceptPrivateMemberLaunch(actor, project, j.InstanceID, req, nil)
			g02Check(t, err)
			if !hit || op.ID != j.OperationID || op.FinishedAt == nil {
				t.Fatal("replay lost original receipt")
			}
			switch j.Phase {
			case "running":
				complete++
				if op.Status != "succeeded" || j.Version != 2 || j.Running == nil || j.Running.Birth == nil {
					t.Fatal("completed launch changed")
				}
			case "start-dispatched":
				uncertain++
				if op.Status != "interrupted" || j.Running != nil || j.ContainerID == "" {
					t.Fatal("uncertain launch promoted or lost object")
				}
			default:
				t.Fatal("unexpected original phase", j.Phase)
			}
			_, _, err = s.acceptPrivateMemberLaunch("other-member-owner", project, j.InstanceID, req, nil)
			requireFault(t, err, "replay_not_found")
			_, err = s.beginPrivateMemberLaunch(project, j.InstanceID, j.OperationID)
			requireFault(t, err, "stale_project_operation")
		}
		if complete != 2 || uncertain != 1 {
			t.Fatal("both histories are required")
		}
		g02Check(t, privateGatewayState(s.db))
		g02AssertRows(t, s.db, f, nil)
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}
