package applicationauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

const fixtureCode = "lidlut-tabwed-pillex-ridrup"

type fixture struct {
	threadUnavailable   bool
	mu                  sync.Mutex
	sessions            map[string]bool
	logins, logoutCalls int
	offline             bool
	dir                 string
	r                   *collection.Registry
	ship                collection.Ship
	owner               *eyre.Client
	e                   *Engine
	store               credentials.Store
}

func setup(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{sessions: map[string]bool{}, dir: filepath.Join(t.TempDir(), "collector")}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path == "/~/host" {
			fmt.Fprint(w, "~zod")
			return
		}
		cookie, _ := r.Cookie("urbauth-~zod")
		if r.URL.Path == "/~/login" {
			if cookie != nil {
				t.Error("parent cookie sent to new login")
			}
			if r.ParseForm() != nil || r.Form.Get("password") != fixtureCode {
				w.WriteHeader(403)
				return
			}
			f.logins++
			token := fmt.Sprintf("fixture-cookie-%d", f.logins)
			f.sessions[token] = true
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: token, Path: "/", MaxAge: 3600})
			return
		}
		if r.URL.Path == "/~/logout" {
			f.logoutCalls++
			if f.offline {
				w.WriteHeader(503)
				return
			}
			body, _ := io.ReadAll(r.Body)
			if r.URL.RawQuery != "" || len(body) > 0 || r.Method != "POST" {
				t.Error("logout was not exact-session only")
			}
		}
		if cookie == nil || !f.sessions[cookie.Value] {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/~/name":
			fmt.Fprint(w, "~zod")
		case "/spider/base/json/code/ship":
			if f.threadUnavailable {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "%q", "~"+fixtureCode)
		case "/~/logout":
			delete(f.sessions, cookie.Value)
			w.WriteHeader(303)
		default:
			t.Error("unexpected fixture request", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	var err error
	f.r, err = collection.Open(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	f.ship, err = f.r.EnrollShip(context.Background(), "fixture", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	f.owner, err = eyre.New(server.URL, "~zod")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.owner.Close)
	if err := f.owner.Login(context.Background(), []byte(fixtureCode)); err != nil {
		t.Fatal(err)
	}
	f.store, err = credentials.NewFile(f.r.CredentialsDir())
	if err != nil {
		t.Fatal(err)
	}
	ref := collection.NewSessionRef()
	data, _ := f.owner.ExportSession()
	defer clear(data)
	f.ship, err = f.r.SaveShipSession(context.Background(), f.ship, ref, func(ctx context.Context) error {
		return f.store.Save(ctx, credentials.Scope(f.ship.ID, f.ship.Origin, ref), data)
	})
	if err != nil {
		t.Fatal(err)
	}
	f.e = New(f.r)
	return f
}

type receiver struct {
	target  collection.ApplicationTarget
	verify  func(context.Context) error
	deliver func(context.Context, []byte) ([]byte, error)
	calls   int
	private []byte
}

func newReceiver() *receiver {
	return &receiver{target: collection.ApplicationTarget{PluginID: "fixture.auth-lab", Profile: "auth-lab", SHA256: strings.Repeat("a", 64)}}
}
func (r *receiver) Target() collection.ApplicationTarget { return r.target }
func (r *receiver) Verify(ctx context.Context) error {
	if r.verify != nil {
		return r.verify(ctx)
	}
	return nil
}
func (r *receiver) Deliver(ctx context.Context, data []byte) ([]byte, error) {
	r.calls++
	r.private = append([]byte(nil), data...)
	if r.deliver != nil {
		return r.deliver(ctx, data)
	}
	return ack(data), nil
}
func ack(data []byte) []byte {
	var offer map[string]json.RawMessage
	_ = json.Unmarshal(data, &offer)
	delete(offer, "session")
	offer["status"] = json.RawMessage(`"accepted"`)
	out, _ := json.Marshal(offer)
	return out
}
func (f *fixture) approve(t *testing.T, r *receiver) collection.ApplicationAttempt {
	t.Helper()
	a, err := f.r.ApproveApplication(context.Background(), f.ship, collection.NewSessionRef(), r.target)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func (f *fixture) reopen(t *testing.T) {
	t.Helper()
	r, err := collection.Open(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	f.r, f.e = r, New(r)
}
func (f *fixture) counts() (int, int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins, f.logoutCalls, len(f.sessions)
}

func TestDurableHandoffAndExactDisconnect(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	one, two := newReceiver(), newReceiver()
	two.target.PluginID = "other.auth-lab"
	a, b := f.approve(t, one), f.approve(t, two)
	for _, pair := range []struct {
		a collection.ApplicationAttempt
		r *receiver
	}{{a, one}, {b, two}} {
		got, err := f.e.Connect(ctx, pair.a.ID, pair.r)
		if err != nil || got.Phase != "connected" {
			t.Fatal(got, err)
		}
	}
	// Duplicate approval/call has no new cookie, receiver invocation or logout.
	again, err := f.r.ApproveApplication(ctx, f.ship, a.ID, one.target)
	if err != nil || again.Phase != "connected" {
		t.Fatal(again, err)
	}
	if _, err := f.e.Connect(ctx, a.ID, one); err != nil || one.calls != 1 {
		t.Fatal(err)
	}
	f.reopen(t)
	if got, err := f.e.Recover(ctx, a.ID); err != nil || got.Phase != "connected" {
		t.Fatal(got, err)
	}
	if n, out, active := f.counts(); n != 3 || out != 0 || active != 3 {
		t.Fatal(n, out, active)
	}
	got, err := f.e.Disconnect(ctx, a.ID)
	if err != nil || got.Phase != "closed" || got.Outcome != "revoked" || !got.ReceiverReconciliation {
		t.Fatal(got, err)
	}
	if _, err := f.e.Disconnect(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if n, out, active := f.counts(); n != 3 || out != 1 || active != 2 {
		t.Fatal(n, out, active)
	}
	if err := f.owner.Check(ctx); err != nil {
		t.Fatal("parent lost", err)
	}
	data, err := f.store.Load(ctx, credentialKey(b))
	if err != nil {
		t.Fatal(err)
	}
	other, _ := eyre.New(b.Origin, b.Ship)
	defer other.Close()
	if err := other.RestoreSession(data); err != nil {
		t.Fatal(err)
	}
	if err := other.Check(ctx); err != nil {
		t.Fatal("other plugin lost", err)
	}
	if _, err := f.store.Load(ctx, credentialKey(a)); !errors.Is(err, credentials.ErrMissing) {
		t.Fatal("retained revoked credential", err)
	}
	if _, err := f.r.ApproveApplication(ctx, f.ship, collection.NewSessionRef(), one.target); !errors.Is(err, collection.ErrApplicationConflict) {
		t.Fatal("ignored receiver reconciliation", err)
	}
	journal, _ := os.ReadFile(filepath.Join(f.dir, "applications.json"))
	if bytes.Contains(journal, []byte("fixture-cookie")) || bytes.Contains(journal, []byte(fixtureCode)) {
		t.Fatal("credential in journal")
	}
}

func TestLostOrInvalidAcknowledgementsRevokeAndFenceRetry(t *testing.T) {
	cases := map[string]func([]byte) ([]byte, error){
		"lost": func([]byte) ([]byte, error) { return nil, errors.New("private cookie should not leak") },
		"wrong-binding": func(b []byte) ([]byte, error) {
			return bytes.Replace(ack(b), []byte(`"auth-lab"`), []byte(`"default"`), 1), nil
		},
		"duplicate-key": func(b []byte) ([]byte, error) { return append([]byte(`{"status":"rejected",`), ack(b)[1:]...), nil },
		"extra-field":   func(b []byte) ([]byte, error) { return append([]byte(`{"cookie":"private",`), ack(b)[1:]...), nil },
		"oversized":     func([]byte) ([]byte, error) { return []byte(strings.Repeat("x", 4097)), nil },
		"exit-error":    func(b []byte) ([]byte, error) { return ack(b), errors.New("failed exit") },
		"trailing-json": func(b []byte) ([]byte, error) { return append(ack(b), []byte(`{}`)...), nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f, r := setup(t), newReceiver()
			r.deliver = func(_ context.Context, b []byte) ([]byte, error) { return mutate(b) }
			a := f.approve(t, r)
			got, err := f.e.Connect(context.Background(), a.ID, r)
			if !errors.Is(err, ErrDelivery) || got.Phase != "closed" || !got.ReceiverReconciliation {
				t.Fatal(got, err)
			}
			if strings.Contains(err.Error(), "private") {
				t.Fatal("private error escaped")
			}
			if _, err := f.e.Connect(context.Background(), a.ID, r); !errors.Is(err, ErrRecovery) || r.calls != 1 {
				t.Fatal(err)
			}
			if _, out, active := f.counts(); out != 1 || active != 1 {
				t.Fatal(out, active)
			}
		})
	}
}

func TestOfflineCleanupSurvivesParentForgetAndRestarts(t *testing.T) {
	ctx := context.Background()
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	r.deliver = func(context.Context, []byte) ([]byte, error) {
		f.mu.Lock()
		f.offline = true
		f.mu.Unlock()
		return nil, io.ErrUnexpectedEOF
	}
	got, err := f.e.Connect(ctx, a.ID, r)
	if !errors.Is(err, ErrCleanup) || got.Phase != "revoking" {
		t.Fatal(got, err)
	}
	f.reopen(t)
	if _, err := f.e.Recover(ctx, a.ID); !errors.Is(err, ErrCleanup) {
		t.Fatal(err)
	}
	// Losing the parent enrollment must not erase the application's exact origin,
	// identity, adapter or opaque reference needed for later cleanup.
	f.ship, err = f.r.SetShipSession(ctx, f.ship, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Delete(ctx, credentials.Scope(f.ship.ID, f.ship.Origin, f.ship.Retired[0])); err != nil {
		t.Fatal(err)
	}
	if err := f.r.CompleteCredentialCleanup(ctx, f.ship.ID, f.ship.Retired); err != nil {
		t.Fatal(err)
	}
	if err := f.r.ForgetShipExpected(ctx, f.ship.Name, f.ship.Origin, f.ship.ID); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.offline = false
	f.mu.Unlock()
	f.reopen(t)
	got, err = f.e.Recover(ctx, a.ID)
	if err != nil || got.Phase != "closed" || got.Outcome != "revoked" {
		t.Fatal(got, err)
	}
	if err := f.owner.Check(ctx); err != nil {
		t.Fatal("parent revoked", err)
	}
}

func TestArtifactMismatchStaleShipAndConcurrentRecovery(t *testing.T) {
	ctx := context.Background()
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	r.target.SHA256 = strings.Repeat("b", 64)
	if _, err := f.e.Connect(ctx, a.ID, r); !errors.Is(err, collection.ErrApplicationConflict) {
		t.Fatal(err)
	}
	if n, _, _ := f.counts(); n != 1 {
		t.Fatal("issued to changed artifact")
	}
	r.target = a.Target
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	r.deliver = func(_ context.Context, data []byte) ([]byte, error) { close(entered); <-release; return ack(data), nil }
	go func() { _, err := f.e.Connect(ctx, a.ID, r); done <- err }()
	<-entered
	if _, err := New(f.r).Recover(ctx, a.ID); !errors.Is(err, collection.ErrApplicationBusy) {
		t.Error(err)
	}
	if _, err := New(f.r).Connect(ctx, a.ID, newReceiver()); !errors.Is(err, collection.ErrApplicationBusy) {
		t.Error(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if n, _, _ := f.counts(); n != 2 {
		t.Fatal("double issuance")
	}
	// A separate approved attempt cannot silently use a newer parent login.
	r2 := newReceiver()
	r2.target.PluginID = "stale.auth-lab"
	b := f.approve(t, r2)
	if _, err := f.r.SetShipSession(ctx, f.ship, ""); err != nil {
		t.Fatal(err)
	}
	got, err := f.e.Connect(ctx, b.ID, r2)
	if !errors.Is(err, ErrNotIssued) || got.Phase != "closed" || got.Outcome != "not-issued" || r2.calls != 0 {
		t.Fatal(got, err)
	}
	if n, _, _ := f.counts(); n != 2 {
		t.Fatal("stale approval issued session")
	}
}

type storeHook struct {
	credentials.Store
	afterSave, afterDelete func()
	failSave               bool
}

func (s storeHook) Save(ctx context.Context, key string, data []byte) error {
	if s.failSave {
		return credentials.ErrStore
	}
	if err := s.Store.Save(ctx, key, data); err != nil {
		return err
	}
	if s.afterSave != nil {
		s.afterSave()
	}
	return nil
}
func (s storeHook) Delete(ctx context.Context, key string) error {
	if err := s.Store.Delete(ctx, key); err != nil {
		return err
	}
	if s.afterDelete != nil {
		s.afterDelete()
	}
	return nil
}

// This helper exits without defers to exercise real process-release locks and
// durable bytes, not merely a successful return or a simulated error value.
func TestCrashHelper(t *testing.T) {
	mode := os.Getenv("OMABIT_APP_CRASH_FIXTURE")
	if mode == "" {
		return
	}
	r, err := collection.Open(os.Getenv("OMABIT_APP_FIXTURE_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	e := New(r)
	id := os.Getenv("OMABIT_APP_FIXTURE_ID")
	stop := func() { os.Exit(73) }
	originalIssue, originalStore, originalRevoke := e.issue, e.store, e.revoke
	e.issue = func(ctx context.Context, a collection.ApplicationAttempt) ([]byte, error) {
		if mode == "before-issue" {
			stop()
		}
		data, err := originalIssue(ctx, a)
		if mode == "after-issue" && err == nil {
			stop()
		}
		return data, err
	}
	e.store = func(a collection.ApplicationAttempt) (credentials.Store, error) {
		s, err := originalStore(a)
		if err != nil {
			return nil, err
		}
		h := storeHook{Store: s}
		if mode == "after-save" {
			h.afterSave = stop
		}
		if mode == "after-delete" {
			h.afterDelete = stop
		}
		return h, nil
	}
	e.revoke = func(ctx context.Context, a collection.ApplicationAttempt, b []byte) (string, error) {
		out, err := originalRevoke(ctx, a, b)
		if mode == "after-revoke" && err == nil {
			stop()
		}
		return out, err
	}
	receiver := newReceiver()
	if mode == "during-delivery" {
		receiver.deliver = func(context.Context, []byte) ([]byte, error) { stop(); return nil, nil }
	}
	if mode == "after-revoke" || mode == "after-delete" {
		receiver.deliver = func(context.Context, []byte) ([]byte, error) { return nil, io.ErrUnexpectedEOF }
	}
	if _, err := e.Connect(context.Background(), id, receiver); err != nil {
		t.Fatal(err)
	}
	if mode == "connected" {
		stop()
	}
	t.Fatal("crash checkpoint not reached")
}

func TestProcessCrashRecoveryBoundaries(t *testing.T) {
	for _, mode := range []string{"before-issue", "after-issue", "after-save", "during-delivery", "after-revoke", "after-delete", "connected"} {
		t.Run(mode, func(t *testing.T) {
			f, r := setup(t), newReceiver()
			a := f.approve(t, r)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCrashHelper$")
			cmd.Env = append(os.Environ(), "OMABIT_APP_CRASH_FIXTURE="+mode, "OMABIT_APP_FIXTURE_DIR="+f.dir, "OMABIT_APP_FIXTURE_ID="+a.ID)
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 73 {
				t.Fatalf("fixture process: %v %s", err, out)
			}
			f.reopen(t)
			got, err := f.e.Recover(ctx, a.ID)
			switch mode {
			case "before-issue", "after-issue":
				if !errors.Is(err, ErrCleanup) || got.Phase != "unresolved" {
					t.Fatal(got, err)
				}
				if _, err := f.e.Connect(ctx, a.ID, r); !errors.Is(err, ErrRecovery) {
					t.Fatal("retried uncertain issuance", err)
				}
			case "connected":
				if err != nil || got.Phase != "connected" {
					t.Fatal(got, err)
				}
				if _, n, active := f.counts(); n != 0 || active != 2 {
					t.Fatal(n, active)
				}
			default:
				if err != nil || got.Phase != "closed" {
					t.Fatal(got, err)
				}
				if _, _, active := f.counts(); active != 1 {
					t.Fatal("orphan token", active)
				}
				if mode == "after-revoke" && got.Outcome != "rejected" {
					t.Fatal("replayed logout reported fresh revocation")
				}
				if mode == "after-delete" && got.Outcome != "revoked" {
					t.Fatal("lost durable remote evidence")
				}
			}
			if err := f.owner.Check(ctx); err != nil {
				t.Fatal("parent was revoked", err)
			}
		})
	}
}

func TestFailedStorageNeverDeliversAndCancellationStillCleansUp(t *testing.T) {
	for _, cancelDelivery := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelDelivery), func(t *testing.T) {
			f, r := setup(t), newReceiver()
			a := f.approve(t, r)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelDelivery {
				r.deliver = func(context.Context, []byte) ([]byte, error) { cancel(); return nil, context.Canceled }
			} else {
				f.e.store = func(collection.ApplicationAttempt) (credentials.Store, error) {
					return storeHook{Store: f.store, failSave: true}, nil
				}
			}
			got, err := f.e.Connect(ctx, a.ID, r)
			if err == nil || got.Phase != "closed" || got.Outcome != "revoked" {
				t.Fatal(got, err)
			}
			if !cancelDelivery && r.calls != 0 {
				t.Fatal("delivered before durable credential")
			}
			if _, _, active := f.counts(); active != 1 {
				t.Fatal("orphan session")
			}
		})
	}
}

func TestMissingCodeThreadDoesNotInventAnUnresolvedSession(t *testing.T) {
	f, r := setup(t), newReceiver()
	a := f.approve(t, r)
	f.mu.Lock()
	f.threadUnavailable = true
	f.mu.Unlock()
	got, err := f.e.Connect(context.Background(), a.ID, r)
	if !errors.Is(err, ErrNotIssued) || got.Phase != "closed" || got.Outcome != "not-issued" || r.calls != 0 {
		t.Fatal(got, err)
	}
	if n, out, active := f.counts(); n != 1 || out != 0 || active != 1 {
		t.Fatal(n, out, active)
	}
}
