package managed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func originalDeskSchema50(t *testing.T) string {
	t.Helper()
	dir := privateDir(t)
	sqlBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema50.sql"))
	g02Check(t, err)
	metaBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema50.json"))
	g02Check(t, err)
	var meta struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metaBytes, &meta))
	hash := sha256.Sum256(sqlBytes)
	if meta.Schema != 50 || meta.Source != "9f45b42a2b13b886549ebea72082ce8f73bf4be0" || meta.SHA != hex.EncodeToString(hash[:]) {
		t.Fatal("original schema50 fixture identity mismatch")
	}
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sqlBytes))
	g02Check(t, err)
	g02Check(t, db.Close())
	return dir
}
func TestDeskSchema51PreservesOriginalSchema50JournalAndPolicy(t *testing.T) {
	dir := originalDeskSchema50(t)
	db := g02RawDB(t, dir)
	before := originalDeskRows(t, db)
	receipt, err := loadDeskOperation(db, deskIntentFixture().ID)
	g02Check(t, err)
	g02Check(t, db.Close())
	for i := 0; i < 2; i++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, originalDeskRows(t, s.db)) {
			t.Fatal("migration changed original rows")
		}
		after, err := loadDeskOperation(s.db, receipt.Intent.ID)
		g02Check(t, err)
		if !reflect.DeepEqual(after, receipt) {
			t.Fatal("migration changed original journal")
		}
		setting, err := s.deskSetting(apppolicy.Target{InstanceID: "alpha", Desk: "example"})
		g02Check(t, err)
		if setting.Revision != 0 || setting.Desk.Publication != "none" {
			t.Fatal("migration invented workflow grants", setting)
		}
		g02Check(t, s.close())
	}
}
func TestDeskSchema51MigrationFailureIsAtomic(t *testing.T) {
	dir := originalDeskSchema50(t)
	db := g02RawDB(t, dir)
	_, err := db.Exec("UPDATE meta SET value='invalid' WHERE key='runner_id'")
	g02Check(t, err)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("invalid original state accepted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var version, count int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE 'desk_settings%'").Scan(&count))
	if version != 50 || count != 0 {
		t.Fatal("failed migration changed schema", version, count)
	}
}
