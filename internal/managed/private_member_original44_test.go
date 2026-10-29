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

// Captured by the original schema44 writer before persistent birth/readiness.
// Later migrations must retain both the completed and uncertain first launches.
func originalPrivate44(t *testing.T) g02Fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/historical-upgrades/schema44.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "ba8b79265136a789efc7bf730f71f215a3fd23f9eadd401c63001e3579ccebda" {
		t.Fatal("original44 fixture changed")
	}
	var f g02Fixture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	g02Check(t, d.Decode(&f))
	archive, err := os.ReadFile("testdata/historical-upgrades/schema44-source.tar.gz")
	g02Check(t, err)
	if f.Schema != 44 || !f.Synthetic || f.SourceSnapshotSHA256 != "80e34af6c6569e9c2bd37a0589cb674261524f6dc1eec176f480afca2aece1a6" || fmt.Sprintf("%x", sha256.Sum256(archive)) != f.SourceSnapshotSHA256 {
		t.Fatal("original44 provenance changed")
	}
	return f
}

func TestPrivateMemberOriginal44PreservesCompletedAndUncertainLaunches(t *testing.T) {
	f := originalPrivate44(t)
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
		if len(members) != 2 {
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
			op, hit, err := s.acceptPrivateMemberLaunch("member-owner", project, j.InstanceID, req, nil)
			g02Check(t, err)
			if !hit || op.ID != j.OperationID || op.FinishedAt == nil {
				t.Fatal("replay lost original receipt")
			}
			switch j.Phase {
			case "running":
				complete++
				if op.Status != "succeeded" || j.Running == nil {
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
		if complete != 1 || uncertain != 1 {
			t.Fatal("both histories are required")
		}
		g02Check(t, privateGatewayState(s.db))
		g02AssertRows(t, s.db, f, nil)
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}
