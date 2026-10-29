package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func closeUnissued(t *testing.T, r *Registry, id string) ApplicationAttempt {
	t.Helper()
	var a ApplicationAttempt
	if err := r.WithApplication(context.Background(), id, func(tx *ApplicationTransaction) error {
		if err := tx.Advance(context.Background(), "closed", "not-issued"); err != nil {
			return err
		}
		a = tx.Current()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestApplicationHistoryOutlivesActiveLimitAndFencesReplay(t *testing.T) {
	ctx := context.Background()
	r, s, target := applicationFixture(t)
	pendingTarget := target
	pendingTarget.PluginID = "pending.dev"
	pending, err := r.ApproveApplication(ctx, s, NewSessionRef(), pendingTarget)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(r.ApplicationCacheDir(), 0700); err != nil {
		t.Fatal(err)
	}
	retained := filepath.Join(r.ApplicationCacheDir(), "needed-helper")
	if err := os.WriteFile(retained, []byte("retained pending recovery helper"), 0600); err != nil {
		t.Fatal(err)
	}
	var first ApplicationAttempt
	for i := 0; i < 96; i++ {
		a, err := r.ApproveApplication(ctx, s, fmt.Sprintf("%032x", i+1), target)
		if err != nil {
			t.Fatal(i, err)
		}
		a = closeUnissued(t, r, a.ID)
		if i == 0 {
			first = a
		}
		if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision+1); !errors.Is(err, ErrApplicationConflict) {
			t.Fatal("stale archive", err)
		}
		archived, err := r.ArchiveApplication(ctx, a.ID, a.Revision)
		if err != nil || archived != a {
			t.Fatal(i, archived, err)
		}
		retry, err := r.ArchiveApplication(ctx, a.ID, a.Revision)
		if err != nil || retry != a {
			t.Fatal("lost-result retry", err)
		}
		active, err := r.Applications(ctx)
		if err != nil || len(active) != 1 || active[0] != pending {
			t.Fatal("active record lost", active, err)
		}
	}
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	got, archived, err := reopened.Application(ctx, first.ID)
	if err != nil || !archived || got != first {
		t.Fatal(got, archived, err)
	}
	if _, err := reopened.ApproveApplication(ctx, s, first.ID, target); !errors.Is(err, ErrApplicationArchived) {
		t.Fatal("replayed archived approval", err)
	}
	if got, err := os.ReadFile(retained); err != nil || string(got) != "retained pending recovery helper" {
		t.Fatal("archive removed needed receiver", err)
	}
	raw, err := os.ReadFile(filepath.Join(r.dir, "applications.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state applicationsFile
	if json.Unmarshal(raw, &state) != nil || state.Version != 2 {
		t.Fatal("old issuer would accept archived IDs", string(raw))
	}
}

func TestApplicationHistoryRefusesUnfinishedPhases(t *testing.T) {
	ctx := context.Background()
	r, s, target := applicationFixture(t)
	a, err := r.ApproveApplication(ctx, s, NewSessionRef(), target)
	if err != nil {
		t.Fatal(err)
	}
	steps := []struct{ phase, outcome string }{{"issuing", ""}, {"saved", ""}, {"delivering", ""}, {"connected", ""}, {"revoking", ""}, {"revoked", "revoked"}, {"closed", "revoked"}}
	for _, step := range steps {
		if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision); !errors.Is(err, ErrApplicationConflict) {
			t.Fatal("archived", a.Phase, err)
		}
		if err := r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error {
			if err := tx.Advance(ctx, step.phase, step.outcome); err != nil {
				return err
			}
			a = tx.Current()
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("discarded receiver reconciliation", err)
	}
	if err := r.WithApplication(ctx, a.ID, func(tx *ApplicationTransaction) error {
		if err := tx.Reconciled(ctx, "removed"); err != nil {
			return err
		}
		a = tx.Current()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision); err != nil {
		t.Fatal(err)
	}
}

func TestApplicationHistoryCrashAfterReceiptPublication(t *testing.T) {
	if dir := os.Getenv("OMABIT_HISTORY_CRASH_FIXTURE"); dir != "" {
		r, err := Open(dir)
		if err != nil {
			os.Exit(91)
		}
		entries, err := r.Applications(context.Background())
		if err != nil || len(entries) != 1 {
			os.Exit(92)
		}
		if err := r.locked(context.Background(), func() error { return r.saveApplicationArchive(entries[0]) }); err != nil {
			os.Exit(93)
		}
		os.Exit(94) // Exact durable-receipt / still-active publication boundary.
	}
	ctx := context.Background()
	r, s, target := applicationFixture(t)
	a, err := r.ApproveApplication(ctx, s, NewSessionRef(), target)
	if err != nil {
		t.Fatal(err)
	}
	a = closeUnissued(t, r, a.ID)
	child := exec.Command(os.Args[0], "-test.run=^TestApplicationHistoryCrashAfterReceiptPublication$")
	child.Env = append(os.Environ(), "OMABIT_HISTORY_CRASH_FIXTURE="+r.dir)
	err = child.Run()
	var exited *exec.ExitError
	if !errors.As(err, &exited) || exited.ExitCode() != 94 {
		t.Fatal("wrong crash boundary", err)
	}
	r, err = Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	active, err := r.Applications(ctx)
	if err != nil || len(active) != 1 {
		t.Fatal(active, err)
	}
	if _, err := r.ApproveApplication(ctx, s, a.ID, target); !errors.Is(err, ErrApplicationArchived) {
		t.Fatal("partial archive replayed", err)
	}
	receipt := filepath.Join(r.dir, "application-history", a.ID+".json")
	before, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(receipt)
	if err != nil || string(before) != string(after) {
		t.Fatal("archive receipt replaced on resume", err)
	}
	if rows, err := r.Applications(ctx); err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}

func TestApplicationHistoryRejectsCorruptionAndLinks(t *testing.T) {
	for _, kind := range []string{"malformed", "public", "symlink", "hardlink", "directory-link"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			r, s, target := applicationFixture(t)
			a, err := r.ApproveApplication(ctx, s, NewSessionRef(), target)
			if err != nil {
				t.Fatal(err)
			}
			a = closeUnissued(t, r, a.ID)
			if err := r.locked(ctx, func() error { return r.saveApplicationArchive(a) }); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(r.dir, "application-history", a.ID+".json")
			switch kind {
			case "malformed":
				err = os.WriteFile(path, []byte(`{"version":1,"version":1}`), 0600)
			case "public":
				err = os.Chmod(path, 0644)
			case "symlink":
				if err = os.Rename(path, path+".original"); err == nil {
					err = os.Symlink(path+".original", path)
				}
			case "hardlink":
				err = os.Link(path, path+".link")
			case "directory-link":
				dir := filepath.Dir(path)
				if err = os.Rename(dir, dir+".original"); err == nil {
					err = os.Symlink(dir+".original", dir)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ArchiveApplication(ctx, a.ID, a.Revision); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("unsafe archive accepted", err)
			}
			if _, err := r.ApproveApplication(ctx, s, a.ID, target); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("unsafe receipt ignored", err)
			}
			rows, err := r.Applications(ctx)
			if err != nil || len(rows) != 1 || rows[0] != a {
				t.Fatal("active evidence lost", rows, err)
			}
		})
	}
}
