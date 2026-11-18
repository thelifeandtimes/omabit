package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// App journals are separate from the ordinary runtime queue. Opening or reading
// this store never dispatches, resumes, cancels, adopts, or edits native state.
var grubberyTables = []struct{ name, ddl string }{
	{"grubbery_app_clock", `CREATE TABLE grubbery_app_clock (singleton INTEGER PRIMARY KEY CHECK(singleton=1), ordinal INTEGER NOT NULL CHECK(ordinal>=0))`},
	{"grubbery_app_operations", `CREATE TABLE grubbery_app_operations (id TEXT PRIMARY KEY, ordinal INTEGER NOT NULL UNIQUE CHECK(ordinal>0), namespace TEXT NOT NULL, actor TEXT NOT NULL, plan_digest TEXT NOT NULL, header_digest TEXT NOT NULL, framework_id TEXT NOT NULL REFERENCES desk_operations(id), body BLOB NOT NULL, stamp TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision>=0 AND revision<=1024))`},
	{"grubbery_app_events", `CREATE TABLE grubbery_app_events (id TEXT NOT NULL REFERENCES grubbery_app_operations(id), sequence INTEGER NOT NULL CHECK(sequence>0 AND sequence<=1024), digest TEXT NOT NULL, body BLOB NOT NULL, PRIMARY KEY(id,sequence))`},
	{"grubbery_app_leases", `CREATE TABLE grubbery_app_leases (namespace TEXT PRIMARY KEY, id TEXT NOT NULL UNIQUE REFERENCES grubbery_app_operations(id))`},
}

func initializeGrubberyApps(tx *sql.Tx, version int) error {
	if version < 61 {
		for _, table := range grubberyTables {
			var count int
			if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name=?", table.name).Scan(&count); err != nil {
				return err
			}
			if count != 0 {
				return grubbery.ErrHistory
			}
			if _, err := tx.Exec(table.ddl); err != nil {
				return err
			}
		}
		if _, err := tx.Exec("INSERT INTO grubbery_app_clock(singleton,ordinal) VALUES(1,0)"); err != nil {
			return err
		}
	}
	for _, table := range grubberyTables {
		var ddl string
		if err := tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table.name).Scan(&ddl); err != nil {
			return err
		}
		if ddl != table.ddl {
			return grubbery.ErrHistory
		}
		var triggers int
		if err := tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND tbl_name=?", table.name).Scan(&triggers); err != nil {
			return err
		}
		if triggers != 0 {
			return grubbery.ErrHistory
		}
	}
	return validateGrubberyJournal(tx)
}

type grubberyHeader struct {
	id, namespace, actor, plan, header, framework, stamp string
	ordinal, revision                                    uint64
	body                                                 []byte
}

func (h grubberyHeader) digest() string {
	body, _ := json.Marshal(struct {
		Ordinal                                             uint64
		ID, Namespace, Actor, Plan, Header, Framework, Body string
	}{h.ordinal, h.id, h.namespace, h.actor, h.plan, h.header, h.framework, stopEvidenceDigest(h.body)})
	return stopEvidenceDigest(body)
}
func appCanonical(body []byte, out any, max int) error {
	if len(body) == 0 || len(body) > max {
		return grubbery.ErrHistory
	}
	if json.Unmarshal(body, out) != nil {
		return grubbery.ErrHistory
	}
	canonical, err := json.Marshal(out)
	if err != nil || !bytes.Equal(canonical, body) {
		return grubbery.ErrHistory
	}
	return nil
}
func grubberyRunner(q querier) (string, error) {
	var id string
	err := q.QueryRow("SELECT value FROM meta WHERE key='runner_id'").Scan(&id)
	return id, err
}

func loadGrubberyRecord(q querier, id string) (grubbery.Record, error) {
	var h grubberyHeader
	err := q.QueryRow("SELECT id,ordinal,namespace,actor,plan_digest,header_digest,framework_id,body,stamp,revision FROM grubbery_app_operations WHERE id=?", id).Scan(&h.id, &h.ordinal, &h.namespace, &h.actor, &h.plan, &h.header, &h.framework, &h.body, &h.stamp, &h.revision)
	if errors.Is(err, sql.ErrNoRows) {
		return grubbery.Record{}, grubbery.ErrMissing
	}
	if err != nil {
		return grubbery.Record{}, err
	}
	var r grubbery.Record
	if appCanonical(h.body, &r, 2<<20) != nil || r.Validate() != nil || r.Revision != 0 || h.ordinal == 0 || h.id != id || h.stamp != h.digest() {
		return r, grubbery.ErrHistory
	}
	n, _ := r.Plan.Binding.Namespace()
	runner, err := grubberyRunner(q)
	if err != nil {
		return r, err
	}
	if h.id != r.Plan.RequestID || h.namespace != n.DeskRoot || h.actor != r.Plan.Binding.Actor || h.plan != r.Plan.Digest || h.header != r.HeaderDigest || h.framework != r.Plan.Binding.FrameworkReceiptID || r.Plan.Binding.RunnerID != runner {
		return r, grubbery.ErrHistory
	}
	if err = checkGrubberyFramework(q, r.Plan.Binding); err != nil {
		return r, err
	}
	rows, err := q.Query("SELECT sequence,digest,body FROM grubbery_app_events WHERE id=? ORDER BY sequence", id)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var seq uint64
		var saved string
		var body []byte
		var event grubbery.Event
		if err = rows.Scan(&seq, &saved, &body); err != nil {
			break
		}
		if appCanonical(body, &event, 16<<10) != nil || seq != uint64(len(r.Events)+1) || seq > 1024 || saved != event.Digest || seq != event.Sequence {
			err = grubbery.ErrHistory
			break
		}
		r.Events = append(r.Events, event)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return r, err
	}
	r.Revision = h.revision
	if r.Validate() != nil {
		return r, grubbery.ErrHistory
	}
	return r, nil
}

func checkGrubberyFramework(q querier, b grubbery.Binding) error {
	receipt, err := loadDeskOperation(q, b.FrameworkReceiptID)
	if err != nil {
		return grubbery.ErrHistory
	}
	deployment, err := loadDeskDeployment(q, b.FrameworkReceiptID)
	if err != nil {
		return grubbery.ErrHistory
	}
	if receipt.Intent.Action != "commit" || receipt.Intent.Target != b.Target || receipt.Intent.SourceDigest != grubbery.FrameworkSourceDigest || deployment.Intent.Continuity != b.Continuity || deployment.Plan.Source.Digest != grubbery.FrameworkSourceDigest || deployment.Intent.SourceName != "grubbery" {
		return grubbery.ErrHistory
	}
	last := receipt.Steps[len(receipt.Steps)-1]
	if last.State != "observed" || last.VerifiedSourceDigest != grubbery.FrameworkSourceDigest || last.Observation == nil || *last.Observation != b.FrameworkBaseline {
		return grubbery.ErrHistory
	}
	return nil
}

func appIDs(q querier, namespace string) ([]string, error) {
	query := "SELECT id FROM grubbery_app_operations"
	var args []any
	if namespace != "" {
		query += " WHERE namespace=?"
		args = append(args, namespace)
	}
	query += " ORDER BY ordinal"
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	return ids, errors.Join(err, rows.Err(), rows.Close())
}

// Creation order comes from a transactional, gap-free clock, never timestamps.
// The independent clock detects removal of a tail/header, including all headers.
func validateGrubberyJournal(q querier) error {
	var clock, count uint64
	if err := q.QueryRow("SELECT count(*),coalesce(max(ordinal),0) FROM grubbery_app_clock WHERE singleton=1").Scan(&count, &clock); err != nil {
		return err
	}
	if count != 1 {
		return grubbery.ErrHistory
	}
	ids, err := appIDs(q, "")
	if err != nil {
		return err
	}
	if clock != uint64(len(ids)) {
		return grubbery.ErrHistory
	}
	byNamespace := map[string][]grubbery.Record{}
	for index, id := range ids {
		var ordinal uint64
		if err = q.QueryRow("SELECT ordinal FROM grubbery_app_operations WHERE id=?", id).Scan(&ordinal); err != nil {
			return err
		}
		if ordinal != uint64(index+1) {
			return grubbery.ErrHistory
		}
		r, err := loadGrubberyRecord(q, id)
		if err != nil {
			return err
		}
		n, _ := r.Plan.Binding.Namespace()
		byNamespace[n.DeskRoot] = append(byNamespace[n.DeskRoot], r)
	}
	expected := map[string]string{}
	for ns, records := range byNamespace {
		o, err := grubbery.DeriveOwnership(records)
		if err != nil {
			return err
		}
		if o.PendingRequestID != "" {
			expected[ns] = o.PendingRequestID
		}
	}
	rows, err := q.Query("SELECT namespace,id FROM grubbery_app_leases")
	if err != nil {
		return err
	}
	for rows.Next() {
		var ns, id string
		if err = rows.Scan(&ns, &id); err != nil {
			break
		}
		if expected[ns] != id {
			err = grubbery.ErrHistory
			break
		}
		delete(expected, ns)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return err
	}
	if len(expected) != 0 {
		return grubbery.ErrHistory
	}
	var orphan int
	if err = q.QueryRow("SELECT count(*) FROM grubbery_app_events e LEFT JOIN grubbery_app_operations o ON e.id=o.id WHERE o.id IS NULL").Scan(&orphan); err != nil {
		return err
	}
	if orphan != 0 {
		return grubbery.ErrHistory
	}
	return nil
}

// grubberyJournal is an internal persistence adapter, not a public dispatch
// surface. The authenticated server must construct Call identity separately.
type grubberyJournal struct{ s *store }

var _ grubbery.Store = grubberyJournal{}

func (j grubberyJournal) Load(ctx context.Context, id string) (grubbery.Record, error) {
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return grubbery.Record{}, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return grubbery.Record{}, err
	}
	return loadGrubberyRecord(tx, id)
}
func (j grubberyJournal) Create(ctx context.Context, r grubbery.Record) (grubbery.Record, error) {
	if r.Validate() != nil || r.Revision != 0 || r.Plan.Binding.Mode != j.s.mode || r.Plan.Binding.RunnerID != j.s.runnerID {
		return grubbery.Record{}, grubbery.ErrConflict
	}
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return grubbery.Record{}, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return grubbery.Record{}, err
	}
	if old, err := loadGrubberyRecord(tx, r.Plan.RequestID); err == nil {
		if old.Plan.Digest != r.Plan.Digest || old.Plan.Binding.Actor != r.Plan.Binding.Actor {
			return old, grubbery.ErrConflict
		}
		return old, nil
	} else if !errors.Is(err, grubbery.ErrMissing) {
		return grubbery.Record{}, err
	}
	if err = checkGrubberyFramework(tx, r.Plan.Binding); err != nil {
		return grubbery.Record{}, err
	}
	n, _ := r.Plan.Binding.Namespace()
	var held string
	err = tx.QueryRow("SELECT id FROM grubbery_app_leases WHERE namespace=?", n.DeskRoot).Scan(&held)
	if err == nil {
		return grubbery.Record{}, grubbery.ErrNamespaceBusy
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return grubbery.Record{}, err
	}
	ids, err := appIDs(tx, n.DeskRoot)
	if err != nil {
		return grubbery.Record{}, err
	}
	records := make([]grubbery.Record, 0, len(ids)+1)
	for _, id := range ids {
		old, err := loadGrubberyRecord(tx, id)
		if err != nil {
			return grubbery.Record{}, err
		}
		records = append(records, old)
	}
	if _, err = grubbery.DeriveOwnership(append(records, r)); err != nil {
		return grubbery.Record{}, grubbery.ErrHistory
	}
	var ordinal uint64
	if err = tx.QueryRow("SELECT ordinal FROM grubbery_app_clock WHERE singleton=1").Scan(&ordinal); err != nil {
		return grubbery.Record{}, err
	}
	if ordinal >= 1<<53-1 {
		return grubbery.Record{}, grubbery.ErrHistory
	}
	ordinal++
	body, _ := json.Marshal(r)
	if len(body) > 2<<20 {
		return grubbery.Record{}, grubbery.ErrHistory
	}
	h := grubberyHeader{id: r.Plan.RequestID, ordinal: ordinal, namespace: n.DeskRoot, actor: r.Plan.Binding.Actor, plan: r.Plan.Digest, header: r.HeaderDigest, framework: r.Plan.Binding.FrameworkReceiptID, body: body}
	if _, err = tx.Exec("INSERT INTO grubbery_app_operations(id,ordinal,namespace,actor,plan_digest,header_digest,framework_id,body,stamp,revision) VALUES(?,?,?,?,?,?,?,?,?,0)", h.id, h.ordinal, h.namespace, h.actor, h.plan, h.header, h.framework, h.body, h.digest()); err != nil {
		return grubbery.Record{}, err
	}
	if _, err = tx.Exec("INSERT INTO grubbery_app_leases(namespace,id) VALUES(?,?)", n.DeskRoot, h.id); err != nil {
		return grubbery.Record{}, err
	}
	if _, err = tx.Exec("UPDATE grubbery_app_clock SET ordinal=? WHERE singleton=1", ordinal); err != nil {
		return grubbery.Record{}, err
	}
	return r, tx.Commit()
}
func (j grubberyJournal) Append(ctx context.Context, id string, revision uint64, event grubbery.Event) (grubbery.Record, error) {
	tx, err := j.s.db.BeginTx(ctx, nil)
	if err != nil {
		return grubbery.Record{}, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return grubbery.Record{}, err
	}
	r, err := loadGrubberyRecord(tx, id)
	if err != nil {
		return r, err
	}
	if r.Revision != revision {
		return r, grubbery.ErrCAS
	}
	if r.Plan.Binding.RunnerID != j.s.runnerID || r.Plan.Binding.Mode != j.s.mode {
		return r, grubbery.ErrConflict
	}
	terminal, err := r.Terminal()
	if err != nil {
		return r, err
	}
	if terminal {
		return r, grubbery.ErrPhase
	}
	next := r
	next.Events = append(append([]grubbery.Event{}, r.Events...), event)
	next.Revision++
	if next.Validate() != nil {
		return r, grubbery.ErrHistory
	}
	n, _ := r.Plan.Binding.Namespace()
	var held string
	if err = tx.QueryRow("SELECT id FROM grubbery_app_leases WHERE namespace=?", n.DeskRoot).Scan(&held); err != nil || held != id {
		return r, grubbery.ErrHistory
	}
	body, _ := json.Marshal(event)
	if len(body) > 16<<10 {
		return r, grubbery.ErrHistory
	}
	if _, err = tx.Exec("INSERT INTO grubbery_app_events(id,sequence,digest,body) VALUES(?,?,?,?)", id, next.Revision, event.Digest, body); err != nil {
		return r, err
	}
	result, err := tx.Exec("UPDATE grubbery_app_operations SET revision=? WHERE id=? AND revision=?", next.Revision, id, revision)
	if err != nil {
		return r, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return r, grubbery.ErrCAS
	}
	terminal, err = next.Terminal()
	if err != nil {
		return r, err
	}
	if terminal {
		result, err = tx.Exec("DELETE FROM grubbery_app_leases WHERE namespace=? AND id=?", n.DeskRoot, id)
		if err != nil {
			return r, err
		}
		affected, err = result.RowsAffected()
		if err != nil || affected != 1 {
			return r, grubbery.ErrHistory
		}
	}
	return next, tx.Commit()
}

func grubberyTableNames() []string {
	names := make([]string, len(grubberyTables))
	for i, t := range grubberyTables {
		names[i] = t.name
	}
	return names
}
