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
)

func originalShipRouteRows(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	out := originalFrontendRows(t, db)
	rows, err := db.Query("SELECT body FROM desk_frontends ORDER BY body")
	g02Check(t, err)
	for rows.Next() {
		var b []byte
		g02Check(t, rows.Scan(&b))
		out["desk_frontends"] = append(out["desk_frontends"], string(b))
	}
	g02Check(t, rows.Err())
	g02Check(t, rows.Close())
	return out
}
func originalDeskSchema57(t *testing.T) string {
	t.Helper()
	dir := privateDir(t)
	sqlBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema57.sql"))
	g02Check(t, err)
	metaBytes, err := os.ReadFile(filepath.Join("testdata", "desk-schema57.json"))
	g02Check(t, err)
	var meta struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metaBytes, &meta))
	h := sha256.Sum256(sqlBytes)
	if meta.Schema != 57 || meta.Source != "bdb81b8561d81782e0bcf08c3bf68d1d3da8d778" || meta.SHA != hex.EncodeToString(h[:]) {
		t.Fatal("original schema57 fixture identity mismatch")
	}
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sqlBytes))
	g02Check(t, err)
	g02Check(t, db.Close())
	return dir
}
func TestShipRoutesSchema58PreservesOriginalSchema57(t *testing.T) {
	dir := originalDeskSchema57(t)
	db := g02RawDB(t, dir)
	before := originalShipRouteRows(t, db)
	metadata := originalDeskRows(t, db)
	g02Check(t, db.Close())
	for i := 0; i < 2; i++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, originalShipRouteRows(t, s.db)) || !reflect.DeepEqual(metadata, originalDeskRows(t, s.db)) {
			t.Fatal("route migration changed original state")
		}
		var n int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM ship_routes").Scan(&n))
		if n != 0 {
			t.Fatal("migration invented URL choices")
		}
		g02Check(t, s.close())
	}
}
func TestShipRoutesSchema58MigrationRollsBack(t *testing.T) {
	dir := originalDeskSchema57(t)
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
	var version, n int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('ship_routes','ship_routes_history')").Scan(&n))
	if version != 57 || n != 0 {
		t.Fatal("failed migration changed original state", version, n)
	}
}
