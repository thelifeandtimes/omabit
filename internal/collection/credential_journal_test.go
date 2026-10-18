package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
)

func journalShip(t *testing.T) (*Registry, Ship, credentials.Store) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "~zod") }))
	t.Cleanup(server.Close)
	r := registry(t)
	s, err := r.EnrollShip(context.Background(), "fixture", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	store, err := credentials.NewFile(r.CredentialsDir())
	if err != nil {
		t.Fatal(err)
	}
	return r, s, store
}

func TestCredentialJournalSurvivesInterruptedSaveAndPublication(t *testing.T) {
	for _, afterSave := range []bool{false, true} {
		t.Run(fmt.Sprint(afterSave), func(t *testing.T) {
			r, s, store := journalShip(t)
			ctx := context.Background()
			ref := NewSessionRef()
			key := credentials.Scope(s.ID, s.Origin, ref)
			interrupted := errors.New("fixture interruption")
			func() {
				defer func() {
					if recovered := recover(); recovered != interrupted {
						t.Fatal("missing interruption", recovered)
					}
				}()
				_, err := r.SaveShipSession(ctx, s, ref, func(ctx context.Context) error {
					// Inspect durable bytes at the actual pre-Save boundary. The old
					// version-1 schema can still decode and clean this reference.
					data, err := os.ReadFile(filepath.Join(r.dir, "ships.json"))
					var state shipsFile
					if err != nil || json.Unmarshal(data, &state) != nil || state.Version != 1 || !slices.Contains(state.Ships[0].Retired, ref) || state.Ships[0].SessionRef != "" {
						t.Fatal("Save preceded durable intent", err)
					}
					if afterSave {
						if err := store.Save(ctx, key, []byte("isolated synthetic token")); err != nil {
							t.Fatal(err)
						}
					}
					panic(interrupted)
				})
				t.Fatal("unexpected completion", err)
			}()
			reopened, err := Open(r.dir)
			if err != nil {
				t.Fatal(err)
			}
			got, err := reopened.Ship(ctx, s.Name)
			if err != nil || got.SessionRef != "" || !slices.Contains(got.Retired, ref) || got.Revision != s.Revision {
				t.Fatal("lost recoverable intent", got, err)
			}
			if err := reopened.ForgetShip(ctx, s.Name, s.Origin); err == nil {
				t.Fatal("forgot pending token")
			}
			if err := store.Delete(ctx, key); err != nil {
				t.Fatal(err)
			}
			if err := reopened.CompleteCredentialCleanup(ctx, s.ID, []string{ref}); err != nil {
				t.Fatal(err)
			}
			if err := reopened.ForgetShip(ctx, s.Name, s.Origin); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCredentialSaveExcludesCleanupAndRejectsCASLoserBeforeSave(t *testing.T) {
	r, s, store := journalShip(t)
	ctx := context.Background()
	ref := NewSessionRef()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := r.SaveShipSession(ctx, s, ref, func(ctx context.Context) error {
			close(entered)
			<-release
			return store.Save(ctx, credentials.Scope(s.ID, s.Origin, ref), []byte("synthetic winner"))
		})
		done <- err
	}()
	<-entered
	wait, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	_, err := r.Ship(wait, s.Name)
	cancel()
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("cleanup could observe an in-flight save", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	called := false
	if _, err := r.SaveShipSession(ctx, s, NewSessionRef(), func(context.Context) error { called = true; return nil }); !errors.Is(err, ErrShipConflict) || called {
		t.Fatal("CAS loser persisted an untracked credential", err, called)
	}
	got, err := r.Ship(ctx, s.Name)
	if err != nil || got.SessionRef != ref || len(got.Retired) != 0 {
		t.Fatal("winner was not atomically published", err)
	}
}
