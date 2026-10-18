package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// The Server, SQLite journal, HTTP admission, reconciliation and receipt
// publication are real. This fixture supplies only the driver's final proposal;
// runtimequalify's RC11 tests exercise the actual driver and filesystem checks
// with its injected account/runtime dependencies.
type rc11RecoveryCompletionBackend struct{ fakeBackend }

func (b *rc11RecoveryCompletionBackend) Apply(_ context.Context, action string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if action != "recover-import" || !state.ImportRecoveryEligible() {
		return state, runtimequalify.ManagedObservation{}, errors.New("unexpected or ineligible recovery dispatch")
	}
	b.mu.Lock()
	b.apply++
	b.mu.Unlock()
	completed := completedRecovery(storedInstance{Runtime: state})
	completed.Prepared = true
	if state.Initialized {
		completed.Adoption.FinishedAt = state.Adoption.FinishedAt
	}
	if state.Comet != nil {
		comet := *state.Comet
		if !state.Initialized {
			comet.StartedAt, comet.FinishedAt = completed.Adoption.FinishedAt, completed.Adoption.FinishedAt
		}
		completed.Comet = &comet
	}
	completed.ExternalImport.Events = append(append([]protocol.ImportEvent(nil), state.ExternalImport.Events...), protocol.ImportEvent{Stage: "complete", At: time.Now().UTC()})
	if err := persist(completed); err != nil {
		return state, runtimequalify.ManagedObservation{State: "unknown", RecoveryRequired: true}, err
	}
	return completed, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func (b *rc11RecoveryCompletionBackend) Observe(_ context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if state.UncertainProvisioning() {
		return state, runtimequalify.ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}, nil
	}
	return state, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func rc11RecoveryHTTP(r *Server, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func rc11RecoverySnapshot(t *testing.T, r *Server) protocol.Snapshot {
	t.Helper()
	w := rc11RecoveryHTTP(r, "GET", "/v1/snapshot", "")
	var snapshot protocol.Snapshot
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestRC11ImportCompletionServerRecoveryAcrossReopen(t *testing.T) {
	for _, phase := range []string{"unprepared", "prepared", "initialized"} {
		initialized := phase == "initialized"
		for _, live := range []bool{false, true} {
			for _, inPlace := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/live=%v/in-place=%v", phase, live, inPlace), func(t *testing.T) {
					must := func(err error) {
						t.Helper()
						if err != nil {
							t.Fatal(err)
						}
					}
					dir, home := privateDir(t), privateDir(t)
					s, err := testOpenStore(dir)
					must(err)
					s.mode = DevelopmentMode
					relative, ship, kind := "fakeships/one/zod", "~zod", "fake"
					if live {
						relative, ship, kind = "livenet/planets/one/sampel-palnet", "~sampel-palnet", "planet"
					}
					mount, err := pierstore.PrepareMount(home, &pierstore.Location{Path: relative})
					must(err)
					target := filepath.Join(mount, filepath.Base(relative))
					sourcePath := filepath.Join(privateDir(t), "source")
					if inPlace {
						sourcePath = target
					}
					must(os.MkdirAll(filepath.Join(sourcePath, ".urb/log"), 0700))
					must(os.Mkdir(filepath.Join(sourcePath, ".urb/chk"), 0700))
					must(os.WriteFile(filepath.Join(sourcePath, ".urb/chk/fixture"), []byte("retained fixture data"), 0600))
					source, err := pierstore.InspectExternal(context.Background(), home, sourcePath)
					must(err)
					req := protocol.EnrollPierRequest{ID: "one", RequestID: "import-before-upgrade", Ship: ship, Path: relative, HTTPPort: 18080, Fingerprint: source.Fingerprint, ConfirmStopped: true}
					old, err := s.enrollPierSource("original-importer", req, pierstore.Candidate{Kind: kind, Identity: source.Identity}, strings.Repeat("a", 64), &source)
					must(err)
					old, v, err := s.beginOperation(old.ID)
					must(err)
					if !inPlace {
						must(os.Rename(sourcePath, target))
						v.Runtime.ExternalImport.HeldPath = filepath.Join(filepath.Dir(sourcePath), ".omabit-import-"+v.Runtime.ID, "pier")
					}
					v.Runtime.Storage.Identity, err = pierstore.Identify(target)
					must(err)
					v.Runtime.Storage.MountIdentity, err = pierstore.IdentifyMount(mount)
					must(err)
					v.Runtime.Prepared, v.Runtime.Initialized = phase != "unprepared", initialized
					if v.Runtime.Prepared {
						v.Runtime.Runtime.Directory = filepath.Join(home, ".local/share/urbit-runner/managed", v.Runtime.ID)
					}
					v.Runtime.ExternalImport.Stage = "enrolling"
					v.Runtime.Adoption.Fingerprint = pierstore.Fingerprint(v.Runtime.Storage.Identity)
					finished := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
					if initialized {
						v.Runtime.Adoption.Uncertain, v.Runtime.Adoption.FinishedAt = false, &finished
						if live {
							v.Runtime.Comet.StartedAt, v.Runtime.Comet.FinishedAt = &finished, &finished
						}
					}
					must(s.runtimeUpdate("one", old.ID, v.Runtime))
					before, _, _, err := pierstore.TreeReview(context.Background(), target)
					must(err)
					// These bodies reproduce the prior release's callback states;
					// reopening also exercises schema19 -> current migration.
					_, err = s.db.Exec("PRAGMA user_version=19")
					must(err)
					must(s.close())
					b := &rc11RecoveryCompletionBackend{}
					r, err := newServerMode(dir, b, time.Hour, live)
					must(err)
					defer func() {
						if r != nil {
							r.Close()
						}
					}()
					first := rc11RecoverySnapshot(t, r)
					if len(first.Instances) != 1 || !first.Instances[0].RuntimeStatus.ImportRecoveryAvailable || first.Instances[0].DesiredState != "stopped" {
						t.Fatal("reopened server did not expose ordinary recovery", first.Instances)
					}
					prior, err := loadOperation(r.s.db, old.ID)
					must(err)
					if prior.Status != "interrupted" || prior.Actor != "original-importer" {
						t.Fatal("import receipt was not reconciled", prior)
					}
					start := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", `{"kind":"start","request_id":"still-fenced","expected_revision":1}`)
					if start.Code != 409 {
						t.Fatal("unfinished import admitted Start", start.Code, start.Body.String())
					}
					body := `{"kind":"recover-import","request_id":"complete-after-upgrade","expected_revision":1}`
					accepted := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", body)
					if accepted.Code != 202 {
						t.Fatal("ordinary recovery was not accepted", accepted.Code, accepted.Body.String())
					}
					var receipt protocol.Operation
					must(json.Unmarshal(accepted.Body.Bytes(), &receipt))
					deadline := time.Now().Add(3 * time.Second)
					for {
						current, err := loadOperation(r.s.db, receipt.ID)
						must(err)
						if current.Status == "succeeded" {
							receipt = current
							break
						}
						if current.Status == "failed" || time.Now().After(deadline) {
							t.Fatal("recovery did not complete through Server worker", current)
						}
						time.Sleep(time.Millisecond)
					}
					must(r.Close())
					r = nil
					r, err = newServerMode(dir, b, time.Hour, live)
					must(err)
					final := rc11RecoverySnapshot(t, r)
					got := final.Instances[0]
					if got.RuntimeStatus.ImportRecoveryAvailable || got.RecoveryRequired || got.ObservedState != "stopped" || got.DesiredState != "stopped" || got.ConfigRevision != 1 || !got.IdentityPending {
						t.Fatal("completion did not survive second reopen", got)
					}
					replay := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", body)
					if replay.Code != 200 {
						t.Fatal("completed receipt replay failed", replay.Code, replay.Body.String())
					}
					var repeated protocol.Operation
					must(json.Unmarshal(replay.Body.Bytes(), &repeated))
					if repeated.ID != receipt.ID || repeated.Status != "succeeded" || repeated.Actor != fmt.Sprintf("local-uid:%d", os.Geteuid()) {
						t.Fatal("recovery replay lost its actor-scoped receipt", repeated)
					}
					changed := strings.Replace(body, ":1", ":2", 1)
					if w := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", changed); w.Code != 409 {
						t.Fatal("recovery replay conflict not enforced", w.Code)
					}
					stored, err := loadInstance(r.s.db, "one")
					must(err)
					if stored.Runtime.ID != v.Runtime.ID || !reflect.DeepEqual(stored.Runtime.Storage, v.Runtime.Storage) || !reflect.DeepEqual(stored.Public.Config, v.Public.Config) || stored.Runtime.Port != v.Runtime.Port || stored.Runtime.AmesPort != v.Runtime.AmesPort {
						t.Fatal("recovery changed identity, revision, storage or configured ports")
					}
					if initialized && (!stored.Runtime.Adoption.FinishedAt.Equal(finished) || live && (!stored.Runtime.Comet.StartedAt.Equal(finished) || !stored.Runtime.Comet.FinishedAt.Equal(finished))) {
						t.Fatal("legacy completion timestamps changed")
					}
					after, _, _, err := pierstore.TreeReview(context.Background(), target)
					must(err)
					b.mu.Lock()
					calls := b.apply
					b.mu.Unlock()
					if before != after || calls != 1 || r.failed.Load() {
						t.Fatal("recovery modified pier data, redispatched or latched the server", calls)
					}
				})
			}
		}
	}
}

func TestRC11LegacyCompletionReceiptFailureRemainsRecoverableAfterReopen(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	v := seedRecovery(t, s)
	v.Runtime = completedRecovery(v)
	v.Runtime.ExternalImport.Stage = "enrolling"
	saveRecoveryFixture(t, s, v)
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	b := &rc11RecoveryCompletionBackend{}
	r, err := newServer(dir, b, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r != nil {
			r.Close()
		}
	}()
	_, err = r.s.db.Exec(`CREATE TRIGGER rc11_fail_completion BEFORE UPDATE OF body ON operations WHEN json_extract(NEW.body,'$.status')='succeeded' BEGIN SELECT RAISE(ABORT,'injected RC11 receipt failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"kind":"recover-import","request_id":"lost-receipt","expected_revision":1}`
	first := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", body)
	if first.Code != 202 {
		t.Fatal(first.Code, first.Body.String())
	}
	for deadline := time.Now().Add(3 * time.Second); !r.failed.Load(); {
		if time.Now().After(deadline) {
			t.Fatal("terminal receipt failure not exercised")
		}
		time.Sleep(time.Millisecond)
	}
	retained, err := loadInstance(r.s.db, "one")
	if err != nil || !retained.Runtime.ImportRecoveryEligible() || !reflect.DeepEqual(retained.Runtime, v.Runtime) {
		t.Fatal("completion escaped the failed receipt transaction", err)
	}
	if _, err := r.s.db.Exec("DROP TRIGGER rc11_fail_completion"); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r = nil
	r, err = newServer(dir, b, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot := rc11RecoverySnapshot(t, r); !snapshot.Instances[0].RuntimeStatus.ImportRecoveryAvailable {
		t.Fatal("failed receipt lost ordinary recovery after reopen")
	}
	replay := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", body)
	var old protocol.Operation
	if replay.Code != 200 || json.Unmarshal(replay.Body.Bytes(), &old) != nil || old.Status != "interrupted" {
		t.Fatal("interrupted receipt was not retained", replay.Code, replay.Body.String())
	}
	second := rc11RecoveryHTTP(r, "POST", "/v1/instances/one/operations", strings.Replace(body, "lost-receipt", "explicit-continuation", 1))
	if second.Code != 202 {
		t.Fatal("explicit continuation not accepted", second.Code, second.Body.String())
	}
	for deadline := time.Now().Add(3 * time.Second); ; {
		snapshot := rc11RecoverySnapshot(t, r)
		got := snapshot.Instances[0]
		if got.ActiveOperationID == "" && !got.RecoveryRequired && got.ObservedState == "stopped" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("explicit continuation did not complete", snapshot)
		}
		time.Sleep(time.Millisecond)
	}
}
