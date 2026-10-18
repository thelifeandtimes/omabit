package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

const capacityActiveToken = "synthetic-capacity-active"
const capacityWinnerToken = "synthetic-capacity-winner"

type logoutCapacityFixture struct {
	dir      string
	ship     collection.Ship
	refs     []string
	mu       sync.Mutex
	logouts  map[string]int
	onLogout func(string)
}

func (f *logoutCapacityFixture) logoutCount(token string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logouts[token]
}

func (f *logoutCapacityFixture) hook(fn func(string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onLogout = fn
}

func (f *logoutCapacityFixture) reopen(t *testing.T) (*collection.Registry, collection.Ship) {
	t.Helper()
	r, err := collection.Open(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.Ship(context.Background(), f.ship.Name)
	if err != nil {
		t.Fatal(err)
	}
	return r, s
}

func (f *logoutCapacityFixture) credentialPath(ref string) string {
	s := f.ship
	s.SessionRef = ref
	return filepath.Join(f.dir, "credentials", secretKey(s))
}

func capacitySession(t *testing.T, s collection.Ship, token string) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"origin": s.Origin, "ship": s.Ship, "cookie_name": "urbauth-~zod", "cookie_value": token, "expires": time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func newLogoutCapacityFixture(t *testing.T, persisted func(int) bool) *logoutCapacityFixture {
	t.Helper()
	f := &logoutCapacityFixture{dir: filepath.Join(t.TempDir(), "collector"), logouts: map[string]int{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/~/login":
			http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: capacityWinnerToken, Path: "/", MaxAge: 3600})
		case "/~/logout":
			cookie, err := req.Cookie("urbauth-~zod")
			if err != nil {
				t.Error("logout lacked synthetic session")
				w.WriteHeader(401)
				return
			}
			f.mu.Lock()
			f.logouts[cookie.Value]++
			hook := f.onLogout
			f.mu.Unlock()
			if hook != nil {
				hook(cookie.Value)
			}
			w.WriteHeader(http.StatusSeeOther)
		default:
			fmt.Fprint(w, "~zod")
		}
	}))
	t.Cleanup(server.Close)
	ctx := context.Background()
	r, err := collection.Open(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.EnrollShip(ctx, "fixture", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	store, err := storeFor(r, s)
	if err != nil {
		t.Fatal(err)
	}
	s.SessionRef = collection.NewSessionRef()
	active := s
	s.SessionRef = ""
	data := capacitySession(t, s, capacityActiveToken)
	f.ship, err = r.SaveShipSession(ctx, s, active.SessionRef, func(ctx context.Context) error { return store.Save(ctx, secretKey(active), data) })
	clear(data)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("synthetic interrupted Save")
	for i := 0; i < 32; i++ {
		pending := f.ship
		pending.SessionRef = collection.NewSessionRef()
		f.refs = append(f.refs, pending.SessionRef)
		data := capacitySession(t, f.ship, fmt.Sprintf("synthetic-retired-%02d", i))
		_, err := r.SaveShipSession(ctx, f.ship, pending.SessionRef, func(ctx context.Context) error {
			if persisted(i) {
				if err := store.Save(ctx, secretKey(pending), data); err != nil {
					return err
				}
			}
			return interrupted
		})
		clear(data)
		if !errors.Is(err, interrupted) {
			t.Fatal("could not fill pending journal", i, err)
		}
	}
	_, current := f.reopen(t)
	if current.SessionRef != f.ship.SessionRef || len(current.Retired) != 32 {
		t.Fatal("fixture did not reach full capacity")
	}
	return f
}

type capacityCommandResult struct {
	code     int
	out, err string
}

func capacityCommand(ctx context.Context, dir string, args ...string) capacityCommandResult {
	var out, errs bytes.Buffer
	code := run(ctx, append([]string{"--config-dir", dir}, args...), &out, &errs)
	return capacityCommandResult{code, out.String(), errs.String()}
}

func assertCapacityActive(t *testing.T, f *logoutCapacityFixture, want collection.Ship, token string) {
	t.Helper()
	r, s := f.reopen(t)
	if s.ID != want.ID || s.Revision != want.Revision || s.SessionRef != want.SessionRef {
		t.Fatal("active session was detached or replaced", s.Revision, want.Revision)
	}
	store, err := storeFor(r, s)
	if err != nil {
		t.Fatal(err)
	}
	data, err := store.Load(context.Background(), secretKey(s))
	defer clear(data)
	if err != nil || !bytes.Contains(data, []byte(token)) {
		t.Fatal("active credential was deleted or changed", err)
	}
	if f.logoutCount(token) != 0 {
		t.Fatal("active session was remotely revoked")
	}
}

func assertCapacityLoggedOutAndForget(t *testing.T, f *logoutCapacityFixture) {
	t.Helper()
	_, s := f.reopen(t)
	if s.SessionRef != "" || len(s.Retired) != 0 {
		t.Fatal("logout left active or retired references")
	}
	files, err := os.ReadDir(filepath.Join(f.dir, "credentials"))
	if err != nil || len(files) != 0 {
		t.Fatal("logout retained synthetic credentials", err)
	}
	// Repeating the actual command after reopening is idempotent, then forget
	// can complete using the same immutable enrollment selection.
	if result := capacityCommand(context.Background(), f.dir, "ship-logout", s.Name, "--expected-ship-id", s.ID); result.code != 0 {
		t.Fatal(result)
	}
	if result := capacityCommand(context.Background(), f.dir, "ship-forget", s.Name, "--url", s.Origin, "--expected-ship-id", s.ID); result.code != 0 {
		t.Fatal(result)
	}
}

func TestRC11LogoutFullCredentialJournal(t *testing.T) {
	for _, kind := range []string{"missing", "mixed", "persisted"} {
		t.Run(kind, func(t *testing.T) {
			f := newLogoutCapacityFixture(t, func(i int) bool { return kind == "persisted" || kind == "mixed" && i%2 == 0 })
			assertCapacityActive(t, f, f.ship, capacityActiveToken)
			result := capacityCommand(context.Background(), f.dir, "ship-logout", f.ship.Name, "--expected-ship-id", f.ship.ID)
			if result.code != 0 {
				t.Fatal("full journal stranded logout", result)
			}
			if f.logoutCount(capacityActiveToken) != 1 {
				t.Fatal("selected session was not logged out exactly once")
			}
			if strings.Contains(result.out+result.err, "synthetic-") {
				t.Fatal("credential leaked into command output")
			}
			assertCapacityLoggedOutAndForget(t, f)
		})
	}
}

func TestRC11LogoutFullJournalInterruptedCleanupResumes(t *testing.T) {
	for _, failure := range []string{"delete", "journal-after-delete"} {
		t.Run(failure, func(t *testing.T) {
			f := newLogoutCapacityFixture(t, func(i int) bool { return i%2 == 0 })
			const boundary = 10
			credential := f.credentialPath(f.refs[boundary])
			lock := filepath.Join(f.dir, "registry.lock")
			if failure == "delete" {
				if err := os.Chmod(credential, 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				f.hook(func(token string) {
					if token == "synthetic-retired-10" {
						// Local credential deletion can complete, but the following
						// journal update fails. Retry must tolerate the missing file.
						if err := os.Chmod(lock, 0644); err != nil {
							t.Error(err)
						}
					}
				})
			}
			result := capacityCommand(context.Background(), f.dir, "ship-logout", f.ship.Name)
			if result.code != 1 || !strings.Contains(result.err, "cleanup is pending") {
				t.Fatal("cleanup interruption not reported", result)
			}
			// Remove only the injected filesystem fault before reopening.
			if err := os.Chmod(lock, 0600); err != nil {
				t.Fatal(err)
			}
			f.hook(nil)
			assertCapacityActive(t, f, f.ship, capacityActiveToken)
			_, current := f.reopen(t)
			if len(current.Retired) != 32-boundary || !slices.Equal(current.Retired, f.refs[boundary:]) {
				t.Fatal("cleanup lost retry intent or completed progress")
			}
			if failure == "delete" {
				// A repeated command against the still-failing adapter preserves
				// both progress and the active credential across another reopen.
				if result := capacityCommand(context.Background(), f.dir, "ship-logout", f.ship.Name); result.code != 1 {
					t.Fatal(result)
				}
				assertCapacityActive(t, f, f.ship, capacityActiveToken)
				if err := os.Chmod(credential, 0600); err != nil {
					t.Fatal(err)
				}
			} else if _, err := os.Stat(credential); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("fixture did not interrupt between deletion and journal completion", err)
			}
			if result := capacityCommand(context.Background(), f.dir, "ship-forget", f.ship.Name, "--url", f.ship.Origin); result.code != 1 {
				t.Fatal("forget lost pending/active references", result)
			}
			if result := capacityCommand(context.Background(), f.dir, "ship-logout", f.ship.Name); result.code != 0 {
				t.Fatal("cleanup did not resume", result)
			}
			if f.logoutCount("synthetic-retired-10") != 1 {
				t.Fatal("missing retired file caused duplicate remote revocation")
			}
			assertCapacityLoggedOutAndForget(t, f)
		})
	}
}

func TestRC11LogoutFullJournalPreservesConcurrentLogin(t *testing.T) {
	for _, phase := range []string{"retired-cleanup", "after-detach"} {
		t.Run(phase, func(t *testing.T) {
			f := newLogoutCapacityFixture(t, func(i int) bool { return i%2 == 0 })
			entered, release := make(chan struct{}), make(chan struct{})
			var blocked atomic.Bool
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			target := "synthetic-retired-02" // prior slots have already been drained
			if phase == "after-detach" {
				target = capacityActiveToken
			}
			f.hook(func(token string) {
				if token == target && blocked.CompareAndSwap(false, true) {
					close(entered)
					<-release
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := make(chan capacityCommandResult, 1)
			go func() { done <- capacityCommand(ctx, f.dir, "ship-logout", f.ship.Name) }()
			select {
			case <-entered:
			case result := <-done:
				t.Fatal("logout never reached cleanup", result)
			case <-ctx.Done():
				t.Fatal("cleanup boundary not reached")
			}
			if phase == "retired-cleanup" {
				assertCapacityActive(t, f, f.ship, capacityActiveToken)
			}
			var out, errs bytes.Buffer
			fail := func(err error, code int) int { fmt.Fprintln(&errs, err); return code }
			// Exercise the real login command against the isolated HTTP fixture,
			// including durable Save/promotion and its own retired cleanup.
			code := runShips(ctx, "ship-login", []string{f.ship.Name, "--code-stdin", "--expected-ship-id", f.ship.ID}, f.dir, "", "", "json", strings.NewReader("aaaaaa-bbbbbb-cccccc-dddddd"), &out, fail)
			if code != 0 {
				t.Fatal("competing synthetic login failed", code, errs.String())
			}
			_, winner := f.reopen(t)
			if winner.SessionRef == "" || winner.SessionRef == f.ship.SessionRef {
				t.Fatal("competing session not promoted")
			}
			assertCapacityActive(t, f, winner, capacityWinnerToken)
			unblock()
			select {
			case result := <-done:
				if phase == "retired-cleanup" {
					if result.code != 1 || !strings.Contains(result.err, collection.ErrShipConflict.Error()) {
						t.Fatal("stale logout adopted competing login", result)
					}
				} else if result.code != 0 {
					t.Fatal("already-detached logout failed", result)
				}
			case <-ctx.Done():
				t.Fatal("logout did not complete after concurrent login")
			}
			assertCapacityActive(t, f, winner, capacityWinnerToken)
			_, current := f.reopen(t)
			if len(current.Retired) != 0 {
				t.Fatal("concurrent cleanup lost progress")
			}
			if result := capacityCommand(ctx, f.dir, "ship-status", f.ship.Name); result.code != 0 || !strings.Contains(result.out, "authenticated") {
				t.Fatal("winning session no longer readable", result)
			}
			if result := capacityCommand(ctx, f.dir, "ship-forget", f.ship.Name, "--url", f.ship.Origin); result.code != 1 {
				t.Fatal("forgot competing active session", result)
			}
		})
	}
}

func TestRC11LogoutFullJournalCancellationPreservesActiveSession(t *testing.T) {
	f := newLogoutCapacityFixture(t, func(i int) bool { return i%2 == 0 })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.hook(func(token string) {
		if token == "synthetic-retired-00" {
			cancel()
		}
	})
	result := capacityCommand(ctx, f.dir, "ship-logout", f.ship.Name)
	if result.code != 1 {
		t.Fatal("canceled logout detached the active session", result)
	}
	f.hook(nil)
	assertCapacityActive(t, f, f.ship, capacityActiveToken)
	_, current := f.reopen(t)
	if len(current.Retired) != 0 {
		t.Fatal("caller cancellation prevented independent cleanup progress")
	}
	if result := capacityCommand(context.Background(), f.dir, "ship-logout", f.ship.Name); result.code != 0 {
		t.Fatal("canceled logout could not be retried", result)
	}
	assertCapacityLoggedOutAndForget(t, f)
}
