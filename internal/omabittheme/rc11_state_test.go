package omabittheme

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
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shipaccess"
)

func TestRC11UpdateRejectsInvalidCallbackWithoutReplacingReadableState(t *testing.T) {
	ctx := context.Background()
	for _, corruption := range []string{"count", "size", "version", "preference", "migration"} {
		t.Run(corruption, func(t *testing.T) {
			store, err := OpenStore(filepath.Join(t.TempDir(), "theme"))
			if err != nil {
				t.Fatal(err)
			}
			ship := collection.Ship{ID: strings.Repeat("a", 32), Origin: "https://example.invalid", Ship: "~zod"}
			if err := store.Auto(ctx, ship, false); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(store.dir, "themes.json"))
			if err != nil {
				t.Fatal(err)
			}
			err = store.Update(ctx, func(state *State) error {
				switch corruption {
				case "count":
					for i := 0; i < maxPreferences; i++ {
						id := fmt.Sprintf("%032x", i)
						state.Ships[id] = Preference{ID: id, Revision: 1}
					}
				case "size":
					p := state.Ships[ship.ID]
					p.LastTheme = strings.Repeat("<", maxStateBytes/6) // JSON escaping counts too
					state.Ships[ship.ID] = p
				case "version":
					state.Version++
				case "preference":
					state.Ships[ship.ID] = Preference{ID: ship.ID}
				case "migration":
					state.Migration = &Migration{}
				}
				return nil
			})
			if err == nil {
				t.Fatal("invalid callback committed")
			}
			after, err := os.ReadFile(filepath.Join(store.dir, "themes.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("valid state was replaced", err)
			}
			if _, err := store.Snapshot(ctx); err != nil {
				t.Fatal("old state unreadable", err)
			}
		})
	}
}

func TestRC11ExistingCompactStateRemainsReadableAtByteLimit(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "theme"))
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	state := State{Version: 1, Ships: map[string]Preference{id: {ID: id, Revision: 1}}}
	compact, _ := json.Marshal(state)
	p := state.Ships[id]
	// Add precisely enough unescaped data to keep the historical compact body
	// readable while its indented replacement would exceed the writer limit.
	p.LastTheme = strings.Repeat("x", maxStateBytes-len(compact)-len(`,"last_theme":""`))
	state.Ships[id] = p
	compact, _ = json.Marshal(state)
	if len(compact) != maxStateBytes {
		t.Fatal("fixture not at boundary", len(compact))
	}
	path := filepath.Join(store.dir, "themes.json")
	if err := os.WriteFile(path, compact, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(context.Background()); err != nil {
		t.Fatal("read-only open refused supported compact state", err)
	}
	if err := store.Update(context.Background(), func(state *State) error {
		p := state.Ships[id]
		p.Revision++
		state.Ships[id] = p
		return nil
	}); err == nil {
		t.Fatal("oversized indented replacement accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(compact, after) {
		t.Fatal("old compact generation lost", err)
	}
}

func rc11MigrationFixture(t *testing.T, fingerprint string) (*collection.Registry, *Store, *migrationFixture, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("migration revoked a shared remote session")
		}
		fmt.Fprint(w, "~zod")
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, "legacy")
	if err := os.Mkdir(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	old := LegacyState{Version: 2, Ships: []LegacyShip{{ID: strings.Repeat("b", 64), Ship: "~zod", URL: server.URL, Automatic: true, Fingerprint: fingerprint}}}
	raw, _ := json.Marshal(old)
	if err := os.WriteFile(filepath.Join(legacy, "state.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := collection.Open(filepath.Join(dir, "collector"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Join(dir, "theme"))
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := credentials.NewFile(filepath.Join(dir, "synthetic-credentials"))
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := json.Marshal(map[string]string{"url": server.URL, "ship": "~zod", "cookieName": "urbauth-~zod", "cookieValue": "synthetic-fixture"})
	return r, store, &migrationFixture{dir: legacy, data: secret, store: secrets}, digest(raw)
}

func TestRC11MigrationPrunesForgottenPreferencesAndRemainsReadable(t *testing.T) {
	r, store, backend, expected := rc11MigrationFixture(t, "")
	ctx := context.Background()
	old, _, err := readLegacy(backend.dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxPreferences; i++ {
		ship, err := r.EnrollShip(ctx, "fixture", old.Ships[0].URL, "~zod", "file")
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Auto(ctx, ship, false); err != nil {
			t.Fatal(err)
		}
		if err := r.ForgetShip(ctx, ship.Name, ship.Origin); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyMigration(ctx, r, store, backend.dir, expected, backend); err != nil {
		t.Fatal("full forgotten-preference migration failed", err)
	}
	state, err := store.Snapshot(ctx)
	if err != nil || len(state.Ships) != 1 || !state.Migration.Complete || !backend.cleared {
		t.Fatal("migration did not preserve readable progress", err)
	}
	ships, err := r.Ships(ctx)
	if err != nil || len(ships) != 1 {
		t.Fatal(err)
	}
	secret, err := backend.store.Load(ctx, shipaccess.Key(ships[0]))
	clear(secret)
	if err != nil {
		t.Fatal("migrated credential lost", err)
	}
}

func TestRC11MigrationPreflightsSerializedGrowthBeforeSideEffects(t *testing.T) {
	// Each half fits; original plus imported preference and recovery journal do
	// not. Preflight must account for JSON escaping and eventual progress entries.
	r, store, backend, expected := rc11MigrationFixture(t, strings.Repeat("<", 23000))
	ctx := context.Background()
	if err := ApplyMigration(ctx, r, store, backend.dir, expected, backend); err == nil {
		t.Fatal("oversized projected migration accepted")
	}
	state, err := store.Snapshot(ctx)
	if err != nil || state.Migration != nil || backend.disabled || backend.cleared {
		t.Fatal("preflight altered migration/legacy state", err)
	}
	ships, err := r.Ships(ctx)
	if err != nil || len(ships) != 0 {
		t.Fatal("preflight enrolled a ship", err)
	}
	old, _, err := readLegacy(backend.dir)
	if err != nil || !old.Ships[0].Automatic {
		t.Fatal("preflight paused legacy publisher", err)
	}
}

type interruptedCredentialSave struct {
	credentials.Store
	fail bool
}

func (s *interruptedCredentialSave) Save(ctx context.Context, key string, data []byte) error {
	if err := s.Store.Save(ctx, key, data); err != nil {
		return err
	}
	if s.fail {
		return errors.New("fixture interruption after credential persistence")
	}
	return nil
}

func TestRC11MigrationResumesAfterCredentialPersistenceBeforePublication(t *testing.T) {
	r, store, backend, expected := rc11MigrationFixture(t, "")
	secrets := &interruptedCredentialSave{Store: backend.store, fail: true}
	backend.store = secrets
	ctx := context.Background()
	if err := ApplyMigration(ctx, r, store, backend.dir, expected, backend); err == nil {
		t.Fatal("interruption not propagated")
	}
	ships, err := r.Ships(ctx)
	if err != nil || len(ships) != 1 || ships[0].SessionRef != "" || len(ships[0].Retired) != 1 || backend.cleared {
		t.Fatal("lost pending credential or prematurely cleared legacy", err)
	}
	pending := ships[0]
	pending.SessionRef = pending.Retired[0]
	if _, err := secrets.Load(ctx, shipaccess.Key(pending)); err != nil {
		t.Fatal("pending credential unreadable", err)
	}
	secrets.fail = false
	if err := ApplyMigration(ctx, r, store, backend.dir, expected, backend); err != nil {
		t.Fatal("resume failed", err)
	}
	ships, err = r.Ships(ctx)
	if err != nil || ships[0].SessionRef == "" || len(ships[0].Retired) != 0 {
		t.Fatal("session not promoted or local duplicate retained", err)
	}
	if _, err := secrets.Load(ctx, shipaccess.Key(pending)); !errors.Is(err, credentials.ErrMissing) {
		t.Fatal("abandoned copy retained", err)
	}
	state, err := store.Snapshot(ctx)
	if err != nil || !state.Migration.Complete {
		t.Fatal("recovery journal incomplete", err)
	}
}
