package managed

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func initializeAppPolicy(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS app_development_policy (id INTEGER PRIMARY KEY CHECK(id=1), body BLOB NOT NULL);
	CREATE TABLE IF NOT EXISTS app_development_policy_history (revision INTEGER PRIMARY KEY, actor TEXT NOT NULL, updated_at TEXT NOT NULL, body BLOB NOT NULL);`)
	if err != nil {
		return err
	}
	// Preserve the exact historical receipt and append a migration revision.
	var oldBody []byte
	err = tx.QueryRow("SELECT body FROM app_development_policy WHERE id=1").Scan(&oldBody)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		var old apppolicy.Policy
		if json.Unmarshal(oldBody, &old) != nil {
			return errors.New("invalid legacy application policy")
		}
		if old.Version == 1 {
			var receipt []byte
			var actor, at string
			if err = tx.QueryRow("SELECT actor,updated_at,body FROM app_development_policy_history WHERE revision=?", old.Revision).Scan(&actor, &at, &receipt); err != nil {
				return err
			}
			if _, err = time.Parse(time.RFC3339Nano, at); err != nil || actor == "" || !bytes.Equal(oldBody, receipt) || old.Revision == 0 || old.Revision >= apppolicy.MaxRevision {
				return errors.New("legacy application policy audit receipt mismatch")
			}
			upgraded, err := apppolicy.UpgradeLegacy(old)
			if err != nil {
				return err
			}
			upgraded.Revision++
			if err = saveAppPolicy(tx, "schema49-migration", upgraded); err != nil {
				return err
			}
		}
	}
	_, err = loadAppPolicy(tx)
	return err
}
func loadAppPolicy(q querier) (apppolicy.Policy, error) {
	p := apppolicy.Default()
	var body []byte
	err := q.QueryRow("SELECT body FROM app_development_policy WHERE id=1").Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		var count int
		if err = q.QueryRow("SELECT count(*) FROM app_development_policy_history").Scan(&count); err != nil {
			return p, err
		}
		if count != 0 {
			return p, errors.New("application policy is missing its current revision")
		}
		return p, nil
	}
	if err != nil {
		return p, err
	}
	p = apppolicy.Policy{}
	if json.Unmarshal(body, &p) != nil || p.Validate() != nil || p.Revision == 0 {
		return apppolicy.Policy{}, errors.New("invalid stored application policy")
	}
	var mirrored []byte
	var actor, at string
	if err = q.QueryRow("SELECT actor,updated_at,body FROM app_development_policy_history WHERE revision=?", p.Revision).Scan(&actor, &at, &mirrored); err != nil {
		return p, err
	}
	if _, err = time.Parse(time.RFC3339Nano, at); err != nil || actor == "" || !bytes.Equal(body, mirrored) {
		return p, errors.New("application policy audit receipt mismatch")
	}
	return p, nil
}
func (s *store) appPolicyStatus() (protocol.AppPolicyStatus, error) {
	out := protocol.AppPolicyStatus{Version: 1, RunnerID: s.runnerID, Mode: s.mode, Targets: []protocol.AppPolicyTarget{}}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out.Policy, err = loadAppPolicy(tx)
	if err != nil {
		return out, err
	}
	out.Targets, err = appPolicyTargets(tx)
	if err != nil {
		return out, err
	}
	out.ReadAt = time.Now().UTC()
	return out, tx.Commit()
}
func appPolicyTarget(q querier, t apppolicy.Target) error {
	all, err := appPolicyTargets(q)
	if err != nil {
		return err
	}
	for _, v := range all {
		if v.Target == t.Ship() && v.Ready {
			return nil
		}
	}
	return conflict("app_policy_target_changed", "ship or project generation changed or is unresolved; refresh policy")
}
func (s *store) putAppPolicy(actor string, u apppolicy.Update) (apppolicy.Policy, error) {
	var p apppolicy.Policy
	if err := u.Validate(); err != nil {
		return p, bad(err.Error())
	}
	tx, err := s.db.Begin()
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	p, err = loadAppPolicy(tx)
	if err != nil {
		return p, err
	}
	if p.Revision != u.ExpectedRevision {
		return p, conflict("app_policy_revision_conflict", "application policy changed; refresh before saving")
	}
	if u.Target.InstanceID != "" {
		if err = appPolicyTarget(tx, u.Target); err != nil {
			return p, err
		}
	}
	if err = p.Apply(u); err != nil {
		return p, bad(err.Error())
	}
	if err = saveAppPolicy(tx, actor, p); err != nil {
		return p, err
	}
	return p, tx.Commit()
}
func saveAppPolicy(tx *sql.Tx, actor string, p apppolicy.Policy) error {
	if p.Validate() != nil || p.Revision == 0 || actor == "" {
		return errors.New("invalid application policy receipt")
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO app_development_policy(id,body) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", body); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO app_development_policy_history(revision,actor,updated_at,body) VALUES(?,?,?,?)", p.Revision, actor, time.Now().UTC().Format(time.RFC3339Nano), body); err != nil {
		return err
	}
	return nil
}

// Deletion and import restoration allow the instance name to be reused. Drop
// its grants in the same transaction as the deletion, preserving audit history.
func retireAppPolicyInstance(tx *sql.Tx, actor, instance string) error {
	p, err := loadAppPolicy(tx)
	if err != nil {
		return err
	}
	entries := make([]apppolicy.Entry, 0, len(p.Entries))
	for _, e := range p.Entries {
		if e.Target.InstanceID != instance || e.Target.ProjectID != "" {
			entries = append(entries, e)
		}
	}
	if len(entries) == len(p.Entries) {
		return nil
	}
	if p.Revision >= apppolicy.MaxRevision {
		return errors.New("application policy revision exhausted")
	}
	p.Entries = entries
	p.Revision++
	return saveAppPolicy(tx, actor, p)
}

// Called under the runner mutation lock. Policy authoring uses a dedicated
// Setup endpoint, never the instance configuration or agent operation schema.
func (r *Server) serveAppPolicy(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method == http.MethodGet {
		v, err := r.s.appPolicyStatus()
		if err != nil {
			r.respondError(w, err)
			return
		}
		respond(w, 200, v)
		return
	}
	if req.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "Setup policy changes require operator access"}})
		return
	}
	var u apppolicy.Update
	if err := decode(w, req, &u); err != nil {
		r.respondError(w, err)
		return
	}
	p, err := r.s.putAppPolicy(actor, u)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, p)
}

// withAppPolicyEffect is the runner effect boundary for future typed desk
// adapters. Never retain its decision across queueing, restart, or retries.
// The lock serializes revocation with effects; inspect must obtain fresh trusted
// desk evidence (absence for creation, publication state for commit). No runtime
// operation is implied by this helper; typed adapters also journal dispatch.
func (r *Server) withAppPolicyEffect(target apppolicy.Target, revision uint64, effect string, inspect func() (string, error), apply func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.updateFence.Held {
		return conflict("runner_unavailable", "runner cannot perform application effects")
	}
	if err := appPolicyTarget(r.s.db, target); err != nil {
		return err
	}
	p, err := loadAppPolicy(r.s.db)
	if err != nil {
		return err
	}
	publication, err := inspect()
	if err != nil {
		return err
	}
	if err = p.Check(target, revision, effect, publication); err != nil {
		return &fault{403, protocol.APIError{Code: "app_policy_blocked", Message: err.Error()}}
	}
	return apply()
}
