package omabittheme

import (
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

type migrationFixture struct {
	dir               string
	data              []byte
	cleared, disabled bool
	failClear         bool
	store             credentials.Store
}

func (f *migrationFixture) DisableLegacy(context.Context) error { f.disabled = true; return nil }
func (f *migrationFixture) LookupLegacy(context.Context, string) ([]byte, error) {
	state, _, err := readLegacy(f.dir)
	if err != nil || !f.disabled || state.Ships[0].Automatic {
		return nil, errors.New("old publisher not paused")
	}
	return append([]byte{}, f.data...), nil
}
func (f *migrationFixture) ClearLegacy(context.Context, string) error {
	if f.failClear {
		return errors.New("fixture keyring locked")
	}
	f.cleared = true
	return nil
}
func (f *migrationFixture) Store(*collection.Registry, collection.Ship) (credentials.Store, error) {
	return f.store, nil
}
func TestMigrationResumesWithoutLoggingOutSharedSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("migration performed a mutation/logout")
		}
		if r.URL.Path == "/~/name" {
			cookie, err := r.Cookie("urbauth-~zod")
			if err != nil || cookie.Value != "0vfixture" {
				t.Error("wrong migrated credential")
			}
		}
		fmt.Fprint(w, "~zod")
	}))
	defer server.Close()
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	legacy := filepath.Join(dir, "legacy")
	os.Mkdir(legacy, 0700)
	old := LegacyState{Version: 2, Ships: []LegacyShip{{ID: strings.Repeat("a", 64), Ship: "~zod", URL: server.URL, Automatic: true, Fingerprint: palette().Fingerprint(), LastTheme: "miasma"}}}
	raw, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(legacy, "state.json"), raw, 0600)
	r, err := collection.Open(filepath.Join(dir, "collector"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(filepath.Join(dir, "omabit"))
	if err != nil {
		t.Fatal(err)
	}
	credentialStore, err := credentials.NewFile(filepath.Join(dir, "credentials"))
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := json.Marshal(map[string]string{"url": server.URL, "ship": "~zod", "cookieName": "urbauth-~zod", "cookieValue": "0vfixture"})
	backend := &migrationFixture{dir: legacy, data: secret, failClear: true, store: credentialStore}
	ctx := context.Background()
	plan, err := PlanMigration(ctx, r, legacy)
	if err != nil || len(plan.Issues) != 0 {
		t.Fatal(err)
	}
	if err := ApplyMigration(ctx, r, s, legacy, plan.Digest, backend); err == nil {
		t.Fatal("incomplete cleanup reported complete")
	}
	state, err := s.Snapshot(ctx)
	if err != nil || state.Migration.Complete {
		t.Fatal("missing recovery journal")
	}
	ships, _ := r.Ships(ctx)
	if len(ships) != 1 || ships[0].SessionRef == "" {
		t.Fatal("credential import missing")
	}
	data, err := credentialStore.Load(ctx, shipaccess.Key(ships[0]))
	if err != nil {
		t.Fatal(err)
	}
	clear(data)
	backend.failClear = false
	if err := ApplyMigration(ctx, r, s, legacy, plan.Digest, backend); err != nil {
		t.Fatal(err)
	}
	state, _ = s.Snapshot(ctx)
	pref := state.Ships[ships[0].ID]
	if !state.Migration.Complete || !backend.cleared || !pref.Automatic || pref.Pending || pref.Fingerprint != old.Ships[0].Fingerprint {
		t.Fatal("migration did not preserve preferences/credentials")
	}
	if err := ApplyMigration(ctx, r, s, legacy, plan.Digest, backend); err != nil {
		t.Fatal("completed migration not idempotent")
	}
}
