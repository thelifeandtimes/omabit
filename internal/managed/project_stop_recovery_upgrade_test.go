package managed

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func loadOriginal59StopFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/project-stop-recovery/"
	var manifest struct {
		Schema  int    `json:"schema"`
		Source  string `json:"original_source"`
		SQL     string `json:"sql_sha256"`
		Fixture string `json:"fixture_sha256"`
		Helper  string `json:"writer_helper_sha256"`
	}
	meta, err := os.ReadFile(base + "manifest.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(meta, &manifest))
	packed, err := os.ReadFile(base + "schema59.sql.gz")
	g02Check(t, err)
	helper, err := os.ReadFile(base + "schema59-writer-test.go.txt")
	g02Check(t, err)
	if manifest.Schema != 59 || manifest.Source != "a9623329a538ebf83d63365053d2d1011ade091e" || stopEvidenceDigest(packed) != manifest.Fixture || stopEvidenceDigest(helper) != manifest.Helper {
		t.Fatal("original59 provenance changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	body, err := io.ReadAll(io.LimitReader(z, 2<<20))
	g02Check(t, err)
	g02Check(t, z.Close())
	if stopEvidenceDigest(body) != manifest.SQL {
		t.Fatal("original59 SQL changed")
	}
	file, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, file.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(body))
	g02Check(t, err)
	var schema int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&schema))
	if schema != 59 {
		t.Fatal("fixture is not original schema59")
	}
	g02Check(t, db.Close())
}

func stopUpgradeRows(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	g02Check(t, err)
	var tables []string
	for rows.Next() {
		var table string
		g02Check(t, rows.Scan(&table))
		tables = append(tables, table)
	}
	g02Check(t, rows.Err())
	g02Check(t, rows.Close())
	out := map[string][]string{}
	for _, table := range tables {
		if table == "project_stop_reconciliations" {
			continue
		}
		r, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(table, `"`, `""`) + `"`)
		g02Check(t, err)
		columns, err := r.Columns()
		g02Check(t, err)
		out[table] = []string{}
		for r.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			g02Check(t, r.Scan(pointers...))
			body, err := json.Marshal(values)
			g02Check(t, err)
			out[table] = append(out[table], string(body))
		}
		g02Check(t, r.Err())
		g02Check(t, r.Close())
		sort.Strings(out[table])
	}
	return out
}

func TestProjectStopOriginal59MigrationPreservesEveryHistoricalRow(t *testing.T) {
	dir := privateDir(t)
	loadOriginal59StopFixture(t, dir)
	db := g02RawDB(t, dir)
	before := stopUpgradeRows(t, db)
	g02Check(t, db.Close())
	for n := 0; n < 3; n++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, stopUpgradeRows(t, s.db)) {
			t.Fatal("migration changed original rows")
		}
		op, err := loadOperation(s.db, "op-original59-stop")
		g02Check(t, err)
		selected, err := s.projectStopSelection(s.db, "actor", op.ProjectID, op.InstanceID, op.ID)
		g02Check(t, err)
		if selected.review.EvidenceBasis != "legacy-admission-window" || selected.member.StopIntent != nil {
			t.Fatal("migration invented pre-signal evidence")
		}
		var schema int
		g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&schema))
		if schema != StateSchemaVersion {
			t.Fatal("wrong successor schema")
		}
		g02Check(t, s.close())
	}
}

func TestProjectStopMigrationRollsBackWhenResolutionEvidenceIsMissing(t *testing.T) {
	dir := privateDir(t)
	loadOriginal59StopFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec("UPDATE operations SET body=json_set(body,'$.kind','reconcile-project-ship-stop') WHERE id='op-original59-stop'")
	g02Check(t, err)
	before := stopUpgradeRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("missing reconciliation evidence accepted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var schema, table int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&schema))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='project_stop_reconciliations'").Scan(&table))
	if schema != 59 || table != 0 || !reflect.DeepEqual(before, stopUpgradeRows(t, db)) {
		t.Fatal("failed migration did not roll back unchanged")
	}
}
