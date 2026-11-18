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

func appAllRows(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	rows, err := db.Query("SELECT name,sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	g02Check(t, err)
	tables := map[string]string{}
	for rows.Next() {
		var name, ddl string
		g02Check(t, rows.Scan(&name, &ddl))
		tables[name] = ddl
	}
	g02Check(t, rows.Err())
	g02Check(t, rows.Close())
	out := map[string][]string{}
	for table, ddl := range tables {
		r, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(table, `"`, `""`) + `"`)
		g02Check(t, err)
		cols, err := r.Columns()
		g02Check(t, err)
		values := []string{}
		for r.Next() {
			raw := make([]any, len(cols))
			ptr := make([]any, len(cols))
			for i := range raw {
				ptr[i] = &raw[i]
			}
			g02Check(t, r.Scan(ptr...))
			body, err := json.Marshal(raw)
			g02Check(t, err)
			values = append(values, string(body))
		}
		g02Check(t, r.Err())
		g02Check(t, r.Close())
		sort.Strings(values)
		out[table] = append([]string{ddl}, values...)
	}
	var version int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	v, _ := json.Marshal(version)
	out["@schema"] = []string{string(v)}
	return out
}
func loadOriginal60AppFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/grubbery-app-original60/"
	meta, err := os.ReadFile(base + "manifest.json")
	g02Check(t, err)
	var m struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		Helper string `json:"helper_sha256"`
		Packed string `json:"gzip_sha256"`
		SQL    string `json:"raw_sql_sha256"`
	}
	g02Check(t, json.Unmarshal(meta, &m))
	packed, err := os.ReadFile(base + "state.sql.gz")
	g02Check(t, err)
	helper, err := os.ReadFile(base + "writer-helper.go.txt")
	g02Check(t, err)
	if m.Schema != 60 || m.Source != "d24c960281dd3e4c0e2029f35e07b0aa546d665b" || m.Helper != stopEvidenceDigest(helper) || m.Packed != stopEvidenceDigest(packed) {
		t.Fatal("original60 provenance changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	body, err := io.ReadAll(io.LimitReader(z, 2<<20+1))
	g02Check(t, err)
	g02Check(t, z.Close())
	if len(body) > 2<<20 || stopEvidenceDigest(body) != m.SQL {
		t.Fatal("original60 SQL changed")
	}
	f, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, f.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(body))
	g02Check(t, err)
	g02Check(t, db.Close())
}
func appOriginalRows(all map[string][]string) map[string][]string {
	out := map[string][]string{}
	for name, rows := range all {
		if name != "@schema" && !strings.HasPrefix(name, "grubbery_app_") {
			out[name] = rows
		}
	}
	return out
}
func TestGrubberyOriginal60MigrationPreservesEveryRowAndExactResolution(t *testing.T) {
	dir := privateDir(t)
	loadOriginal60AppFixture(t, dir)
	db := g02RawDB(t, dir)
	before := appOriginalRows(appAllRows(t, db))
	g02Check(t, db.Close())
	for n := 0; n < 3; n++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, appOriginalRows(appAllRows(t, s.db))) {
			t.Fatal("migration rewrote original60 state")
		}
		original, err := loadOperation(s.db, "op-original59-stop")
		g02Check(t, err)
		resolution, err := loadProjectStopResolution(s.db, original.ID)
		g02Check(t, err)
		got, hit, err := s.replayProjectStopResolution("actor", original.ProjectID, resolution.Request)
		g02Check(t, err)
		if !hit || !reflect.DeepEqual(got, resolution.Result) {
			t.Fatal("original60 resolution replay changed")
		}
		var version, clock int
		g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
		g02Check(t, s.db.QueryRow("SELECT ordinal FROM grubbery_app_clock").Scan(&clock))
		if version != StateSchemaVersion || clock != 0 {
			t.Fatal("migration invented app history")
		}
		g02Check(t, s.close())
	}
}
func TestGrubberyOriginal60MigrationRollbackAndUnknownTableRefusal(t *testing.T) {
	for _, kind := range []string{"corrupt-previous", "preexisting-successor-table"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			loadOriginal60AppFixture(t, dir)
			db := g02RawDB(t, dir)
			if kind == "corrupt-previous" {
				_, err := db.Exec("UPDATE project_stop_reconciliations SET body='{}'")
				g02Check(t, err)
			} else {
				_, err := db.Exec("CREATE TABLE grubbery_app_operations(unowned TEXT)")
				g02Check(t, err)
			}
			before := appAllRows(t, db)
			g02Check(t, db.Close())
			if s, err := testOpenStore(dir); err == nil {
				s.close()
				t.Fatal("invalid migration accepted")
			}
			db = g02RawDB(t, dir)
			defer db.Close()
			if !reflect.DeepEqual(before, appAllRows(t, db)) {
				t.Fatal("failed migration changed rows/schema")
			}
		})
	}
}

// Explicit external rehearsal hook: called only in the private, networkless
// writer capsule with its synthetic historical store mounted at /tmp. It adds
// real app journal rows before the actual old binary is tested against schema61.
func TestGrubberyOriginal60MigrationRehearsal(t *testing.T) {
	if os.Getenv("OMABIT_TEST_GRUBBERY_UPGRADE") != "networkless-original60" {
		t.Skip("explicit isolated original-writer rehearsal only")
	}
	dir := "/tmp/schema-state"
	db := g02RawDB(t, dir)
	before := appOriginalRows(appAllRows(t, db))
	g02Check(t, db.Close())
	s, err := testOpenStore(dir)
	g02Check(t, err)
	if !reflect.DeepEqual(before, appOriginalRows(appAllRows(t, s.db))) {
		t.Fatal("original writer state changed on migration")
	}
	r, _ := grubberyFixture(t, s)
	j := grubberyJournal{s}
	_, err = j.Create(t.Context(), r)
	g02Check(t, err)
	r = appAppendFixture(t, j, r, r.Plan.Steps[0], "dispatched")
	r = appAppendFixture(t, j, r, r.Plan.Steps[0], "uncertain")
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	saved, err := (grubberyJournal{s}).Load(t.Context(), r.Plan.RequestID)
	g02Check(t, err)
	if !reflect.DeepEqual(r, saved) {
		t.Fatal("populated migration lost uncertainty")
	}
}
