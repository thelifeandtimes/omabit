package managed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func deskIntentFixture() deskwork.Intent {
	return deskwork.Intent{Version: 1, ID: strings.Repeat("a", 64), Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1, Action: "create-desk"}
}
func deskStep(r deskwork.Receipt, state string) deskwork.Step {
	return deskwork.Step{Kind: r.Intent.Action, State: state, At: time.Now().UTC()}
}
func TestDeskJournalDeduplicationRecoveryAndReconciliation(t *testing.T) {
	for _, phase := range []string{"planned", "dispatched", "acknowledged"} {
		t.Run(phase, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			intent := deskIntentFixture()
			r, fresh, err := s.createDeskOperation("desktop", intent)
			g02Check(t, err)
			if !fresh {
				t.Fatal("new intent replayed")
			}
			if phase != "planned" {
				r, err = s.appendDeskStep(r, deskStep(r, "dispatched"))
				g02Check(t, err)
			}
			if phase == "acknowledged" {
				r, err = s.appendDeskStep(r, deskStep(r, "acknowledged"))
				g02Check(t, err)
			}
			replay, fresh, err := s.createDeskOperation("desktop", intent)
			g02Check(t, err)
			if fresh || !reflect.DeepEqual(replay, r) {
				t.Fatal("duplicate redispatched or changed")
			}
			_, _, err = s.createDeskOperation("another-desktop", intent)
			requireFault(t, err, "desk_request_conflict")
			altered := intent
			altered.ConfigRevision++
			_, _, err = s.createDeskOperation("desktop", altered)
			requireFault(t, err, "desk_request_conflict")
			alternate := intent
			alternate.ID = strings.Repeat("c", 64)
			_, _, err = s.createDeskOperation("desktop", alternate)
			requireFault(t, err, "desk_outcome_unresolved")
			before := r
			g02Check(t, s.close())
			s, err = testOpenStore(dir)
			g02Check(t, err)
			defer s.close()
			g02Check(t, s.reconcileDeskOperations())
			after, err := loadDeskOperation(s.db, intent.ID)
			g02Check(t, err)
			want := "uncertain"
			if phase == "planned" {
				want = "interrupted"
			}
			if len(after.Steps) != len(before.Steps)+1 || after.Steps[len(after.Steps)-1].State != want {
				t.Fatal("wrong recovery", after)
			}
			if !reflect.DeepEqual(before.Steps, after.Steps[:len(before.Steps)]) {
				t.Fatal("recovery rewrote history")
			}
			g02Check(t, s.reconcileDeskOperations())
			again, err := loadDeskOperation(s.db, intent.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(again, after) {
				t.Fatal("recovery appended twice")
			}
			if _, err = s.appendDeskStep(before, deskStep(before, "dispatched")); err == nil {
				t.Fatal("stale receipt changed")
			}
			if _, err = s.appendDeskStep(after, deskStep(after, "dispatched")); err == nil {
				t.Fatal("recovery became redispatch")
			}
			if want == "uncertain" {
				observed := deskStep(after, "observed")
				observed.Observation = &deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
				after, err = s.appendDeskStep(after, observed)
				g02Check(t, err)
				g02Check(t, s.close())
				s, err = testOpenStore(dir)
				g02Check(t, err)
				defer s.close()
				got, err := loadDeskOperation(s.db, intent.ID)
				g02Check(t, err)
				if !reflect.DeepEqual(got, after) {
					t.Fatal("reconciliation lost on reopen")
				}
			}
		})
	}
}
func TestDeskJournalCorruptionRefusesOpenWithoutRepair(t *testing.T) {
	for _, query := range []string{
		"UPDATE desk_operations SET body='{}'",
		"UPDATE desk_operations SET actor='other'",
		"DELETE FROM desk_operation_history",
		"UPDATE desk_operation_history SET body='{}'",
		"UPDATE desk_operation_history SET sequence=2",
		"DROP TABLE desk_operation_history",
	} {
		t.Run(query, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			_, _, err = s.createDeskOperation("desktop", deskIntentFixture())
			g02Check(t, err)
			_, err = s.db.Exec(query)
			g02Check(t, err)
			g02Check(t, s.close())
			before, err := os.ReadFile(filepath.Join(dir, "runner.sqlite"))
			g02Check(t, err)
			reopened, err := testOpenStore(dir)
			if err == nil {
				reopened.close()
				t.Fatal("corruption accepted")
			}
			after, err := os.ReadFile(filepath.Join(dir, "runner.sqlite"))
			g02Check(t, err)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected open rewrote corrupted state")
			}
		})
	}
}
func originalDeskSchema49(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile("testdata/desk-schema49.sql")
	g02Check(t, err)
	metadata, err := os.ReadFile("testdata/desk-schema49.json")
	g02Check(t, err)
	var pin struct {
		Schema int    `json:"schema"`
		Source string `json:"source_commit"`
		SHA    string `json:"sql_sha256"`
	}
	g02Check(t, json.Unmarshal(metadata, &pin))
	sum := sha256.Sum256(body)
	if pin.Schema != 49 || pin.Source != "e2f6377da3706ceff54ddc33915ea29b15f397c9" || pin.SHA != hex.EncodeToString(sum[:]) {
		t.Fatal("original writer fixture changed")
	}
	dir := privateDir(t)
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), nil, 0600))
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(body))
	g02Check(t, err)
	g02Check(t, db.Close())
	return dir
}
func originalDeskRows(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, table := range []string{"meta", "instances", "operations", "app_development_policy", "app_development_policy_history"} {
		rows, err := db.Query("SELECT * FROM " + table + " ORDER BY 1")
		g02Check(t, err)
		columns, err := rows.Columns()
		g02Check(t, err)
		var all [][]any
		for rows.Next() {
			values := make([]any, len(columns))
			ptr := make([]any, len(columns))
			for i := range values {
				ptr[i] = &values[i]
			}
			g02Check(t, rows.Scan(ptr...))
			all = append(all, values)
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
		body, err := json.Marshal(all)
		g02Check(t, err)
		result[table] = string(body)
	}
	return result
}
func TestDeskSchema50PreservesOriginalWriterAndPolicy(t *testing.T) {
	dir := originalDeskSchema49(t)
	db := g02RawDB(t, dir)
	before := originalDeskRows(t, db)
	g02Check(t, db.Close())
	s, err := testOpenStore(dir)
	g02Check(t, err)
	if !reflect.DeepEqual(before, originalDeskRows(t, s.db)) {
		t.Fatal("migration rewrote original state")
	}
	p, err := loadAppPolicy(s.db)
	g02Check(t, err)
	if p.Global != apppolicy.Blocked() || p.Revision != 1 || len(p.Entries) != 1 || p.Entries[0].Gates.Commit != apppolicy.Allow {
		t.Fatal("original policy lost", p)
	}
	r, _, err := s.createDeskOperation("fixture", deskIntentFixture())
	g02Check(t, err)
	r, err = s.appendDeskStep(r, deskStep(r, "dispatched"))
	g02Check(t, err)
	g02Check(t, s.close())
	for i := 0; i < 2; i++ {
		s, err = testOpenStore(dir)
		g02Check(t, err)
		if !reflect.DeepEqual(before, originalDeskRows(t, s.db)) {
			t.Fatal("reopen changed original state")
		}
		g02Check(t, s.reconcileDeskOperations())
		g02Check(t, s.close())
	}
}
func TestDeskSchema50MigrationFailureIsAtomic(t *testing.T) {
	dir := originalDeskSchema49(t)
	db := g02RawDB(t, dir)
	_, err := db.Exec("UPDATE meta SET value='invalid' WHERE key='runner_id'")
	g02Check(t, err)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("invalid old state accepted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	var version, count int
	g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE 'desk_operation%'").Scan(&count))
	if version != 49 || count != 0 {
		t.Fatal("failed migration changed schema", version, count)
	}
}
