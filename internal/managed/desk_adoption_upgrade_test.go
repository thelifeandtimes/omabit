package managed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func originalDeskSchema54(t *testing.T) string {
	t.Helper()
	dir := privateDir(t)
	sqlBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema54.sql"))
	g02Check(t, err)
	metaBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema54.json"))
	g02Check(t, err)
	var meta struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metaBytes, &meta))
	hash := sha256.Sum256(sqlBytes)
	if meta.Schema != 54 || meta.Source != "829cfedc19b3fe94fb8721b98f03e622deb49b81" || meta.SHA != hex.EncodeToString(hash[:]) {
		t.Fatal("original schema54 fixture identity mismatch")
	}
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sqlBytes))
	g02Check(t, err)
	g02Check(t, db.Close())
	return dir
}
func originalAdoptionRows(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, table := range []string{"desk_operations", "desk_operation_history", "desk_settings", "desk_settings_history", "desk_source_baselines", "desk_deployments", "desk_source_ownership", "desk_releases"} {
		rows, err := db.Query("SELECT body FROM " + table + " ORDER BY body")
		g02Check(t, err)
		for rows.Next() {
			var b []byte
			g02Check(t, rows.Scan(&b))
			out[table] = append(out[table], string(b))
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
	}
	return out
}
func TestDeskSchema55PreservesOriginalSchema54OwnershipAndHistory(t *testing.T) {
	dir := originalDeskSchema54(t)
	db := g02RawDB(t, dir)
	before := originalAdoptionRows(t, db)
	metadata := originalDeskRows(t, db)
	g02Check(t, db.Close())
	for i := 0; i < 2; i++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, originalAdoptionRows(t, s.db)) || !reflect.DeepEqual(metadata, originalDeskRows(t, s.db)) {
			t.Fatal("adoption migration changed original state")
		}
		owned, err := loadOwnedSource(s.db, deskSettingsKey(apppolicy.Target{InstanceID: "alpha", Desk: "preview"}))
		g02Check(t, err)
		if len(owned.Ownership.Raw.Files) != 1 {
			t.Fatal("verified ownership lost")
		}
		var n int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM desk_adoptions").Scan(&n))
		if n != 0 {
			t.Fatal("migration invented adoption evidence")
		}
		g02Check(t, s.close())
	}
}
func TestDeskSchema55MigrationFailureRollsBack(t *testing.T) {
	dir := originalDeskSchema54(t)
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
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='desk_adoptions'").Scan(&count))
	if version != 54 || count != 0 {
		t.Fatal("failed migration changed state", version, count)
	}
}
