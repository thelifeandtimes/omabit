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

func originalDeskSchema52(t *testing.T) string {
	t.Helper()
	dir := privateDir(t)
	sqlBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema52.sql"))
	g02Check(t, err)
	metaBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema52.json"))
	g02Check(t, err)
	var meta struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metaBytes, &meta))
	hash := sha256.Sum256(sqlBytes)
	if meta.Schema != 52 || meta.Source != "f7beb08dfb6249dc4979c6d8db17b2e2a2bc34c1" || meta.SHA != hex.EncodeToString(hash[:]) {
		t.Fatal("original schema52 fixture identity mismatch")
	}
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sqlBytes))
	g02Check(t, err)
	g02Check(t, db.Close())
	return dir
}
func TestDeskSchema53PreservesOriginalSchema52SourceAndJournal(t *testing.T) {
	dir := originalDeskSchema52(t)
	db := g02RawDB(t, dir)
	before := originalDeskRows(t, db)
	receipt, err := loadDeskOperation(db, deskIntentFixture().ID)
	g02Check(t, err)
	target := apppolicy.Target{InstanceID: "alpha", Desk: "example"}
	beforeSetting, err := loadDeskSetting(db, deskSettingsKey(target))
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
		if !reflect.DeepEqual(beforeSetting, setting) {
			t.Fatal("migration changed workflow", setting)
		}
		b := sourceBaselineFixture()
		b.Binding.SettingsRevision = setting.Revision
		base, err := s.sourceBaseline(b.Binding)
		g02Check(t, err)
		if base == nil || base.Validate() != nil {
			t.Fatal("migration lost canonical baseline")
		}
		var count int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM desk_source_ownership").Scan(&count))
		if count != 0 {
			t.Fatal("migration invented source ownership")
		}
		g02Check(t, s.close())
	}
}
func TestDeskSchema53MigrationFailureIsAtomic(t *testing.T) {
	dir := originalDeskSchema52(t)
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
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('desk_deployments','desk_source_ownership')").Scan(&count))
	if version != 52 || count != 0 {
		t.Fatal("failed migration changed schema", version, count)
	}
}
