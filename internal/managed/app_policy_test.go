package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestAppPolicyMigrationPersistenceAndOrdinaryEdits(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.register("fixture", registration("alpha", "register", 18080))
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("DROP TABLE app_development_policy; DROP TABLE app_development_policy_history; PRAGMA user_version=47"); err != nil {
		t.Fatal(err)
	}
	s.close()
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	after, err := s.snapshot()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("migration changed saved fleet: %v", err)
	}
	v, err := s.appPolicyStatus()
	if err != nil || v.Policy.Global != apppolicy.Blocked() || v.Policy.Revision != 0 {
		t.Fatal(v, err)
	}
	target := apppolicy.Target{InstanceID: "alpha", Desk: "app"}
	p, err := s.putAppPolicy("fixture", apppolicy.Update{Scope: "desk", Target: target, Gates: apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.configure("old-client", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18080, AmesPort: 34343, Loom: 31}})
	if err != nil {
		t.Fatal(err)
	}
	s.close()
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.appPolicyStatus()
	if err != nil || !reflect.DeepEqual(v.Policy, p) {
		t.Fatal("policy lost on settings edit/reopen", v, err)
	}
	_, err = s.putAppPolicy("other-desktop", apppolicy.Update{Scope: "runner", Gates: apppolicy.Blocked()})
	requireFault(t, err, "app_policy_revision_conflict")
	var count int
	if err = s.db.QueryRow("SELECT count(*) FROM app_development_policy_history WHERE actor='fixture'").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if _, err = s.db.Exec("UPDATE app_development_policy SET body='{}'"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.appPolicyStatus(); err == nil {
		t.Fatal("corrupt policy reset")
	}
	s.close()
	if reopened, err := testOpenStore(dir); err == nil {
		reopened.close()
		t.Fatal("corrupt policy admitted at startup")
	}
}

func TestAppPolicyServerAndEffectBoundary(t *testing.T) {
	r, c, _ := testEndpoint(t, privateDir(t), &fakeBackend{})
	_, _, err := r.s.register("fixture", registration("alpha", "register", 18080))
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.Hello(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	identity := client.Identity{RunnerID: h.RunnerID, Mode: h.Mode}
	v, err := c.AppPolicy(context.Background(), identity)
	if err != nil || v.DeskOperations {
		t.Fatal(v, err)
	}
	target := apppolicy.Target{InstanceID: "alpha", Desk: "app"}
	effects := 0
	do := func(revision uint64, publication string) error {
		return r.withAppPolicyEffect(target, revision, "commit", func() (string, error) { return publication, nil }, func() error { effects++; return nil })
	}
	if do(0, "private") == nil || effects != 0 {
		t.Fatal("default gate")
	}
	p, err := c.SetAppPolicy(context.Background(), identity, apppolicy.Update{Scope: "desk", Target: target, Gates: apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	if err != nil {
		t.Fatal(err)
	}
	if do(p.Revision, "private") != nil || effects != 1 {
		t.Fatal("allowed commit not admitted")
	}
	if do(p.Revision, "published") == nil || do(p.Revision, "unknown") == nil || effects != 1 {
		t.Fatal("publication bypass")
	}
	queuedRevision := p.Revision
	p, err = c.SetAppPolicy(context.Background(), identity, apppolicy.Update{ExpectedRevision: p.Revision, Scope: "desk", Target: target, Gates: apppolicy.Blocked()})
	if err != nil {
		t.Fatal(err)
	}
	if do(queuedRevision, "private") == nil || do(p.Revision, "private") == nil || effects != 1 {
		t.Fatal("queued work survived revocation")
	}
	for _, body := range []string{`{"scope":"runner","gates":{"commit":"allow","publish":"allow"},"agent_override":true}`, `{"scope":"runner","gates":{"commit":"allow","publish":"allow"}}`} {
		q := httptest.NewRequest(http.MethodPut, protocol.AppPolicyPath, strings.NewReader(body))
		q.Header.Set("Content-Type", "application/json")
		q.Header.Set(protocol.Header, "1")
		q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		if w.Code < 400 {
			t.Fatal("bypass accepted", w.Body.String())
		}
	}
	q := httptest.NewRequest(http.MethodPut, protocol.AppPolicyPath, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.serveAppPolicy(w, q, "observer", "observer")
	if w.Code != 403 {
		t.Fatal("observer edit", w.Code)
	}
}

func TestAppPolicyProjectGenerationAndMissingTarget(t *testing.T) {
	s := testStore(t)
	p, instance, _ := contextReadinessProject(t, s, 0)
	v, err := s.appPolicyStatus()
	if err != nil {
		t.Fatal(err)
	}
	var target apppolicy.Target
	for _, item := range v.Targets {
		if item.Target.ProjectID == p.ID && item.Target.InstanceID == instance && item.Ready {
			target = item.Target
		}
	}
	if target.Generation == 0 || target.Creator == "" {
		raw, _ := json.Marshal(v)
		t.Fatal("no generation", string(raw))
	}
	target.Desk = "app"
	update := apppolicy.Update{Scope: "desk", Target: target, Gates: apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Allow}}
	saved, err := s.putAppPolicy("fixture", update)
	if err != nil {
		t.Fatal(err)
	}
	update.ExpectedRevision = saved.Revision
	update.Target.Generation++
	_, err = s.putAppPolicy("fixture", update)
	requireFault(t, err, "app_policy_target_changed")
	update.Target = apppolicy.Target{InstanceID: "missing", Desk: "app"}
	_, err = s.putAppPolicy("fixture", update)
	requireFault(t, err, "app_policy_target_changed")
}

func TestAppPolicyRetirementIsAtomicAndDoesNotGrantReusedName(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("fixture", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	target := apppolicy.Target{InstanceID: "alpha", Desk: "app"}
	p, err := s.putAppPolicy("fixture", apppolicy.Update{Scope: "desk", Target: target, Gates: apppolicy.Gates{Commit: apppolicy.Allow, Publish: apppolicy.Allow}})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = retireAppPolicyInstance(tx, "delete-fixture", "alpha"); err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	got, err := loadAppPolicy(s.db)
	if err != nil || got.Revision != p.Revision || got.Effective(target).Commit.Value != apppolicy.Allow {
		t.Fatal("rollback lost permission", got, err)
	}
	tx, err = s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = retireAppPolicyInstance(tx, "delete-fixture", "alpha"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	got, err = loadAppPolicy(s.db)
	if err != nil || got.Revision != p.Revision+1 || got.Effective(target).Commit.Value != apppolicy.Block {
		t.Fatal("reused name retained grant", got, err)
	}
}
