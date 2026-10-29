package managed

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func originalPrivate43(t *testing.T) g02Fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/historical-upgrades/schema43.json")
	g02Check(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "004391518e998771b9e739c094d7b6f80b0746dae7624ddfe697f7e9307bdbdc" {
		t.Fatal("original43 fixture changed")
	}
	var f g02Fixture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	g02Check(t, d.Decode(&f))
	if f.Schema != 43 {
		t.Fatal("original schema changed")
	}
	return f
}

// Synthetic first-member launch for each original project, one completed and
// one uncertain. Their original gateway/workspace initialization and separately
// attributed recovery receipts must remain byte-for-byte intact.
func populatePrivateMember44(t *testing.T, s *store) {
	t.Helper()
	claims, err := privateWorkspaceClaims(s.db)
	g02Check(t, err)
	if len(claims) != 2 {
		t.Fatal("original43 requires both projects")
	}
	for n, w := range claims {
		project := w.Binding.ProjectID
		p, err := loadPrivateProject(s.db, project)
		g02Check(t, err)
		instance := p.Members[0].InstanceID
		req := privateMemberRequest(t, s, project)
		req.RequestID = "member-" + project
		op, hit, err := s.acceptPrivateMemberLaunch("member-owner", project, instance, req, privateMemberStoreContract)
		g02Check(t, err)
		if hit {
			t.Fatal("fresh member replayed")
		}
		j, err := s.beginPrivateMemberLaunch(project, instance, op.ID)
		g02Check(t, err)
		for _, phase := range privateMemberStorePhases[1:] {
			j = advancePrivateMemberStoreFixture(t, s, p, j, phase)
			if n == 1 && phase == "start-dispatched" {
				break
			}
		}
		if n == 1 {
			g02Check(t, s.interruptPrivateMemberLaunch(project, instance, op.ID))
		}
	}
}

func TestPrivateMemberSchema44PreservesOriginalAndNewHistory(t *testing.T) {
	f := originalPrivate43(t)
	dir := g02Materialize(t, f)
	s := g02Open(t, dir)
	g02AssertRows(t, s.db, f, nil)
	var version int
	g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
	if version != StateSchemaVersion {
		t.Fatal("real schema44 migration did not run")
	}
	populatePrivateMember44(t, s)
	assertOriginalPrivateRowsWithAdditional(t, s, f, 2)
	before, err := privateMemberLaunches(s.db)
	g02Check(t, err)
	g02Check(t, s.close())
	for pass := 0; pass < 3; pass++ {
		s = g02Open(t, dir)
		assertOriginalPrivateRowsWithAdditional(t, s, f, 2)
		after, e := privateMemberLaunches(s.db)
		g02Check(t, e)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("restart rewrote new member history")
		}
		for _, j := range after {
			project := j.Gateway.Selection.Binding.ProjectID
			req := privateMemberRequest(t, s, project)
			req.RequestID = "member-" + project
			req.ReplayOnly = true
			op, hit, e := s.acceptPrivateMemberLaunch("member-owner", project, j.InstanceID, req, nil)
			g02Check(t, e)
			want := "interrupted"
			if j.Phase == "running" {
				want = "succeeded"
			}
			if !hit || op.ID != j.OperationID || op.Status != want {
				t.Fatal("member replay lost")
			}
		}
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}

func TestPrivateMemberSchema44InvalidAncestryRollsBackMigration(t *testing.T) {
	f := originalPrivate43(t)
	dir := g02Materialize(t, f)
	db := g02RawDB(t, dir)
	_, err := db.Exec("UPDATE private_project_gateways SET body='{}' WHERE project_id=(SELECT project_id FROM private_project_gateways ORDER BY project_id LIMIT 1)")
	g02Check(t, err)
	g02Check(t, db.Close())
	path := filepath.Join(dir, "runner.sqlite")
	before, err := os.ReadFile(path)
	g02Check(t, err)
	for pass := 0; pass < 2; pass++ {
		s, e := openStore(dir)
		if e == nil {
			s.close()
			t.Fatal("corrupt ancestry migrated")
		}
		after, e := os.ReadFile(path)
		g02Check(t, e)
		if !bytes.Equal(before, after) {
			t.Fatal("failed migration modified original database")
		}
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var version, count int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='private_member_launches'").Scan(&count))
	if version != 43 || count != 0 {
		t.Fatal("partial migration committed", version, count)
	}
}
