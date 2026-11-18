package managed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The fixture is the closed store from the actual frozen62 binary, including
// original59/60 ancestry. It is not a current store with a lowered marker.
func loadOriginal62AppFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/grubbery-app-original62/"
	raw, err := os.ReadFile(base + "manifest.json")
	g02Check(t, err)
	var m struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		Packed string `json:"gzip_sha256"`
		SQL    string `json:"raw_sql_sha256"`
		Helper string `json:"declared_helper_sha256"`
	}
	g02Check(t, json.Unmarshal(raw, &m))
	packed, err := os.ReadFile(base + "state.sql.gz")
	g02Check(t, err)
	helper, err := os.ReadFile(base + "writer-helper.go.txt")
	g02Check(t, err)
	if m.Schema != 62 || m.Source != "3e7808791c6af9357eb0bc808620b1ec1daca349" || stopEvidenceDigest(packed) != m.Packed || stopEvidenceDigest(helper) != m.Helper {
		t.Fatal("actual62 writer or packed fixture changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	sql, err := io.ReadAll(io.LimitReader(z, (4<<20)+1))
	g02Check(t, err)
	g02Check(t, z.Close())
	if len(sql) > 4<<20 || stopEvidenceDigest(sql) != m.SQL {
		t.Fatal("actual62 SQL changed")
	}
	f, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, f.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(sql))
	g02Check(t, err)
	g02Check(t, db.Close())
}

func TestAppRuntimeOriginal62MigrationPreservesEveryOriginalRowAndUncertainty(t *testing.T) {
	dir := privateDir(t)
	loadOriginal62AppFixture(t, dir)
	db := g02RawDB(t, dir)
	before := appAllRows(t, db)
	delete(before, "@schema")
	var id string
	g02Check(t, db.QueryRow("SELECT id FROM grubbery_app_operations").Scan(&id))
	original, err := loadGrubberyRecord(db, id)
	g02Check(t, err)
	if original.Plan.Version != 2 || original.Revision != 2 {
		t.Fatal("wrong original62 uncertainty fixture")
	}
	g02Check(t, db.Close())
	for pass := 0; pass < 3; pass++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := rowsWithoutRuntimeRecovery(appAllRows(t, s.db))
		delete(after, "@schema")
		if !reflect.DeepEqual(before, after) {
			t.Fatal("migration changed an original62 row, DDL, access or history")
		}
		got, err := (grubberyJournal{s}).Load(t.Context(), id)
		g02Check(t, err)
		if !reflect.DeepEqual(original, got) {
			t.Fatal("migration rewrote an original plan, event, lease or acknowledgement")
		}
		var audits, heads int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_runtime_recoveries").Scan(&audits))
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_runtime_heads WHERE ordinal=0 AND digest=''").Scan(&heads))
		if audits != 0 || heads != 1 {
			t.Fatal("migration invented runtime recovery authority")
		}
		g02Check(t, s.close())
	}
}

func TestAppRuntimeOriginal62CorruptionRollsBackUnchanged(t *testing.T) {
	dir := privateDir(t)
	loadOriginal62AppFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec("DELETE FROM grubbery_app_events WHERE sequence=2")
	g02Check(t, err)
	before := appAllRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("corrupt original prefix admitted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	if !reflect.DeepEqual(before, appAllRows(t, db)) {
		t.Fatal("failed63 migration repaired or relabeled an actual62 store")
	}
}
