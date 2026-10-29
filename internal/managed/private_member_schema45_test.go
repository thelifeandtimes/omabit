package managed

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type birth45WorkspaceDriver struct{ *privateWorkspaceStoreDriver }

func (d *birth45WorkspaceDriver) CreateRoot(_ context.Context, w privatechain.Workspace) (pierstore.DirectoryIdentity, error) {
	id := w.Parent
	id.Inode = 50000
	return id, d.check("root", "root-dispatched", w)
}
func (d *birth45WorkspaceDriver) ObserveLayout(_ context.Context, w privatechain.Workspace) (privatechain.WorkspaceObservation, error) {
	piers, a, b := w.Root, w.Root, w.Root
	piers.Inode++
	a.Inode += 2
	b.Inode += 3
	return privatechain.WorkspaceObservation{ObservedAt: time.Now().UTC(), Parent: w.Parent, Root: w.Root, PiersRoot: piers, Mounts: []pierstore.DirectoryIdentity{a, b}, Complete: true, Private: true, Empty: true, Synced: true}, d.check("observe", "layout-created", w)
}

// A third, wholly synthetic project adds a genuine new-format capture while the
// original44 completed/uncertain histories remain untouched in the same store.
func populatePrivateBirth45(t *testing.T, s *store) privatechain.MemberLaunch {
	t.Helper()
	available := s.portAvailable
	s.portAvailable = func(string, uint16) bool { return true }
	defer func() { s.portAvailable = available }()
	p, req := privateWorkspaceFixture(t, s)
	op, hit, err := s.acceptPrivateWorkspace("owner", p.Binding.ProjectID, req, privateWorkspaceParent())
	g02Check(t, err)
	if hit {
		t.Fatal("new workspace replayed")
	}
	g02Check(t, s.runPrivateWorkspace(context.Background(), p.Binding.ProjectID, op.ID, &birth45WorkspaceDriver{&privateWorkspaceStoreDriver{t: t, s: s}}))
	w, err := readyPrivateWorkspace(s.db, p.Binding.ProjectID)
	g02Check(t, err)
	lifecycleGateway43(t, s, w, "ready")
	instance := p.Members[0].InstanceID
	mreq := privateMemberRequest(t, s, p.Binding.ProjectID)
	mreq.RequestID = "capture-member"
	op, hit, err = s.acceptPrivateMemberLaunch("birth-owner", p.Binding.ProjectID, instance, mreq, privateMemberStoreContract)
	g02Check(t, err)
	if hit {
		t.Fatal("new capture replayed")
	}
	g02Check(t, s.runPrivateMemberLaunch(context.Background(), p.Binding.ProjectID, instance, op.ID, &privateMemberStoreDriver{t: t, s: s}))
	j, err := loadPrivateMemberLaunch(s.db, instance)
	g02Check(t, err)
	if j.Version != 2 || j.Running == nil || j.Running.Birth == nil {
		t.Fatal("capture was not persisted")
	}
	return j
}

func assertOriginal44WithBirth45(t *testing.T, s *store, f g02Fixture) {
	t.Helper()
	added := map[string]int{"operations": 4, "private_projects": 1, "private_project_members": 2, "private_project_workspaces": 1, "private_project_gateways": 1, "private_member_launches": 1}
	for _, table := range f.Tables {
		rows, err := s.db.Query("SELECT * FROM " + table.Name)
		g02Check(t, err)
		columns, err := rows.Columns()
		g02Check(t, err)
		if !reflect.DeepEqual(columns, table.Columns) {
			t.Fatal("original columns changed", table.Name)
		}
		var actual [][]any
		for rows.Next() {
			values := make([]any, len(columns))
			ptr := make([]any, len(columns))
			for n := range values {
				ptr[n] = &values[n]
			}
			g02Check(t, rows.Scan(ptr...))
			actual = append(actual, values)
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
		if len(actual) != len(table.Rows)+added[table.Name] {
			t.Fatal("unexpected row count", table.Name, len(actual), len(table.Rows)+added[table.Name])
		}
		for _, row := range table.Rows {
			expected := make([]any, len(row))
			for n := range row {
				expected[n] = g02Cell(t, row[n])
			}
			count := 0
			for _, got := range actual {
				if reflect.DeepEqual(expected, got) {
					count++
				}
			}
			if count != 1 {
				t.Fatal("original row changed or duplicated", table.Name)
			}
		}
	}
}

func TestPrivateMemberSchema45PreservesOriginalAndCapturedHistory(t *testing.T) {
	f := originalPrivate44(t)
	dir := g02Materialize(t, f)
	s := g02Open(t, dir)
	g02AssertRows(t, s.db, f, nil)
	var version int
	g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
	if version != currentSchema {
		t.Fatal("real schema45 migration missing")
	}
	before := populatePrivateBirth45(t, s)
	assertOriginal44WithBirth45(t, s, f)
	g02Check(t, s.close())
	for pass := 0; pass < 3; pass++ {
		s = g02Open(t, dir)
		assertOriginal44WithBirth45(t, s, f)
		after, err := loadPrivateMemberLaunch(s.db, before.InstanceID)
		g02Check(t, err)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("capture changed on reopen")
		}
		req := privateMemberRequest(t, s, before.Gateway.Selection.Binding.ProjectID)
		req.RequestID = "capture-member"
		req.ReplayOnly = true
		op, hit, err := s.acceptPrivateMemberLaunch("birth-owner", before.Gateway.Selection.Binding.ProjectID, before.InstanceID, req, nil)
		g02Check(t, err)
		if !hit || op.ID != before.OperationID || op.Status != "succeeded" {
			t.Fatal("capture replay changed")
		}
		_, _, err = s.acceptPrivateMemberLaunch("another-owner", op.ProjectID, before.InstanceID, req, nil)
		requireFault(t, err, "replay_not_found")
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, f)
}

func TestPrivateMemberSchema45CorruptionRefusesMigrationUnchanged(t *testing.T) {
	f := originalPrivate44(t)
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
			t.Fatal("corrupt original launch migrated")
		}
		after, err := os.ReadFile(path)
		g02Check(t, err)
		if !bytes.Equal(before, after) {
			t.Fatal("refused migration wrote original database")
		}
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var version int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	if version != 44 {
		t.Fatal("partial migration committed")
	}
}
