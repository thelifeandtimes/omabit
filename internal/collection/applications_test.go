package collection

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
)

func applicationFixture(t *testing.T) (*Registry, Ship, ApplicationTarget) {
	t.Helper()
	r, s, store := journalShip(t)
	ref := NewSessionRef()
	s, err := r.SaveShipSession(context.Background(), s, ref, func(ctx context.Context) error {
		return store.Save(ctx, credentials.Scope(s.ID, s.Origin, ref), []byte("synthetic parent"))
	})
	if err != nil {
		t.Fatal(err)
	}
	return r, s, ApplicationTarget{PluginID: "fixture.dev", Profile: "dev", SHA256: strings.Repeat("a", 64)}
}

func TestApplicationApprovalIsIdempotentAndBoundToSelection(t *testing.T) {
	ctx := context.Background()
	r, s, target := applicationFixture(t)
	id := NewSessionRef()
	a, err := r.ApproveApplication(ctx, s, id, target)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionRef == s.SessionRef || a.Phase != "approved" || a.Authority != "eyre-owner-session" {
		t.Fatal(a)
	}
	got, err := r.ApproveApplication(ctx, s, id, target)
	if err != nil || got != a {
		t.Fatal(got, err)
	}
	if _, err := r.ApproveApplication(ctx, s, NewSessionRef(), target); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("duplicate active selection", err)
	}
	changed := target
	changed.SHA256 = strings.Repeat("b", 64)
	if _, err := r.ApproveApplication(ctx, s, id, changed); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("ID rebound", err)
	}
	other := target
	other.PluginID = "other.dev"
	stale := s
	stale.Revision++
	if _, err := r.ApproveApplication(ctx, stale, NewSessionRef(), other); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("stale selection approved", err)
	}
	if err := r.WithApplication(ctx, id, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, "connected", "") }); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("skipped delivery", err)
	}
	var retained *ApplicationTransaction
	if err := r.WithApplication(ctx, id, func(tx *ApplicationTransaction) error { retained = tx; return tx.Advance(ctx, "closed", "not-issued") }); err != nil {
		t.Fatal(err)
	}
	if err := retained.Advance(ctx, "issuing", ""); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("used released lease", err)
	}
	if _, err := r.ApproveApplication(ctx, s, NewSessionRef(), target); err != nil {
		t.Fatal("unused approval could not be replaced", err)
	}
}

func TestApplicationJournalRefusesUnsafeAndUnknownRecordsWithoutReset(t *testing.T) {
	for _, kind := range []string{"version", "unknown-field", "bad-phase", "missing-reference", "duplicate-id", "duplicate-field", "public", "symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			r, s, target := applicationFixture(t)
			if _, err := r.ApproveApplication(context.Background(), s, NewSessionRef(), target); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(r.dir, "applications.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var state applicationsFile
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "version":
				state.Version = 3
			case "bad-phase":
				state.Attempts[0].Phase = "unknown"
			case "missing-reference":
				state.Attempts[0].SessionRef = ""
			case "duplicate-id":
				state.Attempts = append(state.Attempts, state.Attempts[0])
			}
			data, err = json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "unknown-field" {
				data = append([]byte(`{"new_field":true,`), data[1:]...)
			}
			if kind == "duplicate-field" {
				data = append([]byte(`{"version":2,`), data[1:]...)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "public":
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".original", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+".link"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := r.Applications(context.Background()); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("unsafe registry accepted", err)
			}
			if _, err := r.ApproveApplication(context.Background(), s, NewSessionRef(), target); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("unsafe registry replaced", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(data) {
				t.Fatal("unsafe registry was modified", err)
			}
		})
	}
}
