package managed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func loadOriginal61AppFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/grubbery-app-original61/"
	meta, err := os.ReadFile(base + "manifest.json")
	g02Check(t, err)
	var m struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		Packed string `json:"gzip_sha256"`
		SQL    string `json:"raw_sql_sha256"`
		Writer string `json:"writer_test_source_sha256"`
		Helper string `json:"fixture_helper_sha256"`
	}
	g02Check(t, json.Unmarshal(meta, &m))
	packed, err := os.ReadFile(base + "state.sql.gz")
	g02Check(t, err)
	writer, err := os.ReadFile(base + "writer-helper.go.txt")
	g02Check(t, err)
	helper, err := os.ReadFile(base + "fixture-helper.go.txt")
	g02Check(t, err)
	if m.Schema != 61 || m.Source != "6407cda23bdc3bbfa25a2ed661e4860fcf5f2763" {
		t.Fatal("original61 source changed")
	}
	if stopEvidenceDigest(packed) != m.Packed || stopEvidenceDigest(writer) != m.Writer || stopEvidenceDigest(helper) != m.Helper {
		t.Fatal("original61 helper/fixture bytes changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	raw, err := io.ReadAll(io.LimitReader(z, 4<<20+1))
	g02Check(t, err)
	g02Check(t, z.Close())
	if len(raw) > 4<<20 || stopEvidenceDigest(raw) != m.SQL {
		t.Fatal("original61 SQL changed")
	}
	f, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, f.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(raw))
	g02Check(t, err)
	g02Check(t, db.Close())
}
func TestGrubberyOriginal61MigrationPreservesLegacyPlanAndOriginalUncertainty(t *testing.T) {
	dir := privateDir(t)
	loadOriginal61AppFixture(t, dir)
	db := g02RawDB(t, dir)
	before := appAllRows(t, db)
	delete(before, "@schema")
	g02Check(t, db.Close())
	for i := 0; i < 3; i++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := appAllRows(t, s.db)
		delete(after, "@schema")
		if !reflect.DeepEqual(before, after) {
			t.Fatal("migration rewrote legacy plan/event/lease history")
		}
		j := grubberyJournal{s}
		r, err := j.Load(t.Context(), "1111111111111111111111111111111111111111111111111111111111111111")
		g02Check(t, err)
		if r.Version != 1 || r.Plan.Version != 1 || r.Revision != 2 {
			t.Fatal("legacy receipt was silently converted")
		}
		c := grubbery.Call{RunnerID: r.Plan.Binding.RunnerID, Mode: r.Plan.Binding.Mode, Actor: r.Plan.Binding.Actor, Target: r.Plan.Binding.Target, AppID: r.Plan.Binding.AppID, RequestID: r.Plan.RequestID, PlanDigest: r.Plan.Digest, StepID: r.Plan.Steps[0].ID}
		got, err := (grubbery.Execution{Store: j}).Advance(t.Context(), c)
		g02Check(t, err)
		if !reflect.DeepEqual(got, r) {
			t.Fatal("original uncertainty replay changed")
		}
		// Later pending phases are not executable through the corrected constructor.
		c.StepID = r.Plan.Steps[1].ID
		if _, err = (grubbery.Execution{Store: j}).Advance(t.Context(), c); !errors.Is(err, grubbery.ErrLegacyPlan) {
			t.Fatal("new worker admitted a legacy pending write", err)
		}
		if _, err = r.Plan.Request(c.StepID, nil); !errors.Is(err, grubbery.ErrLegacyPlan) {
			t.Fatal("legacy native request constructed")
		}
		g02Check(t, s.close())
	}
}
func TestGrubberyOriginal61CorruptionRollsBackSchema62(t *testing.T) {
	dir := privateDir(t)
	loadOriginal61AppFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec("DELETE FROM grubbery_app_events WHERE sequence=2")
	g02Check(t, err)
	before := appAllRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("truncated legacy history accepted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	if !reflect.DeepEqual(before, appAllRows(t, db)) {
		t.Fatal("failed migration repaired/relabelled original state")
	}
}
