package collection

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	previous "git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection/testdata/v3writer"
)

func connectedFleet(t *testing.T) (*Registry, ApplicationAttempt) {
	t.Helper()
	r, target, fleet := fleetAttemptFixture(t)
	ctx := context.Background()
	if _, err := r.SetApplicationShipApproval(ctx, 0, fleet.Request.Binding, target, true); err != nil {
		t.Fatal(err)
	}
	a, err := r.ApproveFleetApplication(ctx, NewSessionRef(), target, fleet, "file")
	if err != nil {
		t.Fatal(err)
	}
	advanceReplacement(t, r, a.ID, "issuing", "saved", "delivering", "connected")
	a, _, err = r.Application(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	return r, a
}

func advanceReplacement(t *testing.T, r *Registry, id string, phases ...string) {
	t.Helper()
	for _, phase := range phases {
		if err := r.WithApplication(context.Background(), id, func(tx *ApplicationTransaction) error { return tx.Advance(context.Background(), phase, "") }); err != nil {
			t.Fatalf("%s: %v", phase, err)
		}
	}
}

func childFleet(t *testing.T, r *Registry, parent ApplicationAttempt, origin string) ApplicationAttempt {
	t.Helper()
	f := parent.Fleet
	f.Request.Route.Origin = origin
	a, err := r.ApproveFleetReplacement(context.Background(), parent.ID, parent.Revision, NewSessionRef(), parent.Target, f)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func closeReplacement(t *testing.T, r *Registry, id string) ApplicationAttempt {
	t.Helper()
	ctx := context.Background()
	err := r.WithApplication(ctx, id, func(tx *ApplicationTransaction) error {
		if err := tx.Advance(ctx, "revoking", ""); err != nil {
			return err
		}
		if err := tx.Advance(ctx, "revoked", "revoked"); err != nil {
			return err
		}
		if err := tx.Advance(ctx, "closed", "revoked"); err != nil {
			return err
		}
		return tx.Reconciled(ctx, "removed")
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := r.Application(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReplacementJournalAtomicLineageAndArchivedOriginalCleanup(t *testing.T) {
	ctx := context.Background()
	r, original := connectedFleet(t)
	child := childFleet(t, r, original, "https://other.example")
	parent, _, err := r.Application(ctx, original.ID)
	if err != nil || parent.Phase != "retiring" || parent.Revision != original.Revision+1 || parent.Origin != original.Origin || parent.SessionRef != original.SessionRef || parent.Target != original.Target || parent.Fleet != original.Fleet {
		t.Fatal(parent, err)
	}
	if child.SessionRef == original.SessionRef || child.ConnectionID != original.ConnectionID || child.Replacement.PreviousID != original.ID {
		t.Fatal(child)
	}
	if _, err := r.ApproveFleetReplacement(ctx, parent.ID, parent.Revision, NewSessionRef(), parent.Target, child.Fleet); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("second child admitted", err)
	}
	if err := r.WithApplication(ctx, child.ID, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, "issuing", "") }); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("skipped preparation", err)
	}
	advanceReplacement(t, r, child.ID, "preparing", "prepared", "issuing", "saved", "delivering", "connected")
	parent, _, _ = r.Application(ctx, original.ID)
	if parent.Replacement.SuccessorID != child.ID || parent.Phase != "retiring" || !parent.ReceiverReconciliation {
		t.Fatal("lost original cleanup", parent)
	}
	parent = closeReplacement(t, r, parent.ID)
	if _, err := r.ArchiveApplication(ctx, parent.ID, parent.Revision); err != nil {
		t.Fatal(err)
	}
	child, _, _ = r.Application(ctx, child.ID)
	grandchild := childFleet(t, r, child, original.Origin)
	advanceReplacement(t, r, grandchild.ID, "preparing", "prepared", "issuing", "saved", "delivering", "connected")
	reopened, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	all, err := reopened.Applications(ctx)
	if err != nil || len(all) != 2 {
		t.Fatal(all, err)
	}
	got, archived, err := reopened.Application(ctx, parent.ID)
	if err != nil || !archived || got != parent {
		t.Fatal("archived ancestor changed", got, err)
	}
	// Broken immutable ancestry is not silently discarded to admit a fresh login.
	history := filepath.Join(r.dir, "application-history", parent.ID+".json")
	if err := os.Rename(history, history+".held"); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Applications(ctx); !errors.Is(err, ErrApplicationJournal) {
		t.Fatal("missing ancestor accepted", err)
	}
}

func TestReplacementJournalScopeLockAndRevocation(t *testing.T) {
	ctx := context.Background()
	r, parent := connectedFleet(t)
	child := childFleet(t, r, parent, parent.Origin)
	err := r.WithApplication(ctx, child.ID, func(tx *ApplicationTransaction) error {
		if err := r.WithApplication(ctx, parent.ID, func(*ApplicationTransaction) error { t.Fatal("parent entered child lease"); return nil }); !errors.Is(err, ErrApplicationBusy) {
			t.Fatal(err)
		}
		if _, err := r.SetApplicationShipApproval(ctx, 1, parent.Fleet.Request.Binding, parent.Target, false); err != nil {
			return err
		}
		if err := tx.Advance(ctx, "preparing", ""); !errors.Is(err, ErrApplicationPermission) {
			t.Fatal("revocation ignored", err)
		}
		return tx.Advance(ctx, "closed", "not-issued")
	})
	if err != nil {
		t.Fatal(err)
	}
	// Retired original remains independently cleanable after permission removal.
	closeReplacement(t, r, parent.ID)
}

func TestReplacementJournalFailedPreparationCanUseDistinctAttempt(t *testing.T) {
	ctx := context.Background()
	r, parent := connectedFleet(t)
	child := childFleet(t, r, parent, parent.Origin)
	advanceReplacement(t, r, child.ID, "preparing")
	closed := closeUnissued(t, r, child.ID)
	if _, err := r.ArchiveApplication(ctx, child.ID, closed.Revision); err != nil {
		t.Fatal(err)
	}
	parent, _, _ = r.Application(ctx, parent.ID)
	next := childFleet(t, r, parent, "https://retry.example")
	if next.ID == child.ID || next.SessionRef == child.SessionRef {
		t.Fatal("reused failed attempt")
	}
	// An archived ID remains fenced even though preparation itself is repeatable.
	if _, err := r.ApproveFleetReplacement(ctx, parent.ID, parent.Revision, child.ID, parent.Target, child.Fleet); !errors.Is(err, ErrApplicationArchived) {
		t.Fatal(err)
	}
}

func TestReplacementJournalParentDisconnectStopsDispatchAndArchive(t *testing.T) {
	ctx := context.Background()
	r, parent := connectedFleet(t)
	child := childFleet(t, r, parent, parent.Origin)
	parent = closeReplacement(t, r, parent.ID)
	if _, err := r.ArchiveApplication(ctx, parent.ID, parent.Revision); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("pending child lost its mutable parent", err)
	}
	if err := r.WithApplication(ctx, child.ID, func(tx *ApplicationTransaction) error { return tx.Advance(ctx, "preparing", "") }); !errors.Is(err, ErrApplicationConflict) {
		t.Fatal("disconnected parent revived", err)
	}
	closeUnissued(t, r, child.ID)
	if _, err := r.ArchiveApplication(ctx, parent.ID, parent.Revision); err != nil {
		t.Fatal(err)
	}
}

func TestReplacementJournalRejectsCorruptAncestryWithoutMutation(t *testing.T) {
	for _, kind := range []string{"version", "missing", "self", "cycle", "binding", "helper", "credentials", "unretired-parent", "unaccepted-child", "forged-successor", "branch", "duplicate-session"} {
		t.Run(kind, func(t *testing.T) {
			r, parent := connectedFleet(t)
			child := childFleet(t, r, parent, parent.Origin)
			path := filepath.Join(r.dir, "applications.json")
			raw, _ := os.ReadFile(path)
			var state applicationsFile
			if json.Unmarshal(raw, &state) != nil {
				t.Fatal("decode")
			}
			p, c := &state.Attempts[0], &state.Attempts[1]
			switch kind {
			case "version":
				state.Version = 3
			case "missing":
				c.Replacement.PreviousID = NewSessionRef()
			case "self":
				c.Replacement.PreviousID = c.ID
			case "cycle":
				p.Replacement.PreviousID = c.ID
			case "binding":
				c.Fleet.Request.Binding.Continuity = string(bytes.Repeat([]byte("2"), 64))
				c.ConnectionID = FleetConnectionID(c.Fleet.Request.Binding)
			case "helper":
				c.Target.SHA256 = string(bytes.Repeat([]byte("b"), 64))
			case "credentials":
				c.Credentials = "secret-service"
			case "unretired-parent":
				p.Phase = "connected"
			case "unaccepted-child":
				c.Phase = "connected"
				c.ReceiverReconciliation = true
			case "forged-successor":
				p.Replacement.SuccessorID = c.ID
			case "branch":
				other := child
				other.ID = NewSessionRef()
				other.SessionRef = NewSessionRef()
				state.Attempts = append(state.Attempts, other)
			case "duplicate-session":
				c.SessionRef = p.SessionRef
			}
			raw, _ = json.Marshal(state)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Applications(context.Background()); !errors.Is(err, ErrApplicationJournal) {
				t.Fatal("corrupt lineage accepted", err)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(raw, after) {
				t.Fatal("corrupt state changed")
			}
		})
	}
}

func TestReplacementJournalActualPreviousWriterPreservationAndRefusal(t *testing.T) {
	ctx := context.Background()
	// Verify retained production files against their frozen source identity.
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	raw, err := os.ReadFile("testdata/v3writer/source.json")
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		t.Fatal(err)
	}
	for name, want := range manifest.Files {
		data, err := os.ReadFile(filepath.Join("testdata/v3writer", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("previous writer modified", name)
		}
	}
	dir := filepath.Join(t.TempDir(), "registry")
	old, err := previous.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, target, fleet := fleetAttemptFixture(t)
	ot := previous.ApplicationTarget(target)
	of := previous.FleetApplication(fleet)
	if _, err = old.SetApplicationShipApproval(ctx, 0, fleet.Request.Binding, ot, true); err != nil {
		t.Fatal(err)
	}
	a, err := old.ApproveFleetApplication(ctx, NewSessionRef(), ot, of, "file")
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"issuing", "saved", "delivering", "connected"} {
		if err = old.WithApplication(ctx, a.ID, func(tx *previous.ApplicationTransaction) error { return tx.Advance(ctx, phase, "") }); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "applications.json")
	original, _ := os.ReadFile(path)
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	parent, _, err := r.Application(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, _ := os.ReadFile(path)
	if !bytes.Equal(original, unchanged) {
		t.Fatal("read migrated original bytes")
	}
	child := childFleet(t, r, parent, "https://successor.example")
	upgraded, _ := os.ReadFile(path)
	if _, err := old.Applications(ctx); !errors.Is(err, previous.ErrApplicationJournal) {
		t.Fatal("actual previous reader accepted v4", err)
	}
	if err := old.WithApplication(ctx, a.ID, func(tx *previous.ApplicationTransaction) error { return tx.Advance(ctx, "revoking", "") }); !errors.Is(err, previous.ErrApplicationJournal) {
		t.Fatal("actual previous writer changed v4", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(upgraded, after) {
		t.Fatal("previous writer modified new format")
	}
	advanceReplacement(t, r, child.ID, "preparing", "prepared", "issuing", "saved", "delivering", "connected")
	for i := 0; i < 3; i++ {
		reopened, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		got, _, err := reopened.Application(ctx, parent.ID)
		if err != nil || got.SessionRef != parent.SessionRef || got.Origin != parent.Origin || got.Fleet != parent.Fleet || got.Replacement.SuccessorID != child.ID {
			t.Fatal("original evidence lost", got, err)
		}
	}
}
