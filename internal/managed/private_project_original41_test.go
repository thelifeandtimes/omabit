package managed

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// This is an original41-writer fixture, not a relabeled current model. Keep it
// as a preservation baseline when private initialization gains durable state.
func TestPrivateProjectOriginal41PreservesReservationsAndLegacyHistory(t *testing.T) {
	raw, err := os.ReadFile("testdata/historical-upgrades/schema41.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "5da8b5eb7aa65ddf8bca03eb8233cd5bacfc541b97e86693e55b9337942c7ac3" {
		t.Fatal("original41 fixture changed")
	}
	var f g02Fixture
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	g02Check(t, decoder.Decode(&f))
	archive, err := os.ReadFile("testdata/historical-upgrades/schema41-source.tar.gz")
	g02Check(t, err)
	if f.Schema != 41 || !f.Synthetic || f.SourceSnapshotSHA256 != "eb4bd69f6ac1bb692a83768d84f58ea00afb4c1ba3096a637f215c200b50beb2" || fmt.Sprintf("%x", sha256.Sum256(archive)) != f.SourceSnapshotSHA256 {
		t.Fatal("original41 provenance changed")
	}
	dir := g02Materialize(t, f)
	for pass := 0; pass < 3; pass++ {
		s := g02Open(t, dir)
		if s.runnerID != f.RunnerID {
			t.Fatal("runner identity changed")
		}
		g02AssertRows(t, s.db, f, nil)
		g02Replay(t, s, f)
		s.portAvailable = func(string, uint16) bool { t.Error("saved reservation replay probed a host port"); return false }
		for n, prefix := range []string{"historical-private-a", "historical-private-b"} {
			req := privateProjectRequest(t, "historical-register-"+prefix, prefix, uint16(30080+n*10))
			req.ReplayOnly = true
			op, hit, err := s.registerPrivateProject("historical-private-actor-"+prefix, req)
			g02Check(t, err)
			if !hit || op.ProjectID == "" || op.Status != "succeeded" {
				t.Fatal("historical reservation replay changed")
			}
			_, _, err = s.registerPrivateProject("another-actor", req)
			requireFault(t, err, "replay_not_found")
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
