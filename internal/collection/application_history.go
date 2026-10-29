package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

var ErrApplicationArchived = errors.New("application approval was completed and archived; it cannot be issued again")

type applicationArchive struct {
	Version    int                `json:"version"`
	Attempt    ApplicationAttempt `json:"attempt"`
	ArchivedAt time.Time          `json:"archived_at"`
}

func (r *Registry) applicationHistory(create bool) (*Registry, error) {
	dir := filepath.Join(r.dir, "application-history")
	if !create {
		if _, err := os.Lstat(dir); err != nil {
			return nil, err
		}
	}
	history, err := Open(dir)
	if err != nil {
		return nil, ErrApplicationJournal
	}
	if create {
		parent, err := os.Open(r.dir)
		if err != nil {
			return nil, ErrApplicationJournal
		}
		err = parent.Sync()
		closeErr := parent.Close()
		if errors.Join(err, closeErr) != nil {
			return nil, ErrApplicationJournal
		}
	}
	return history, nil
}

// readApplicationArchive is called under the registry lock. Receipts are bounded
// individually; they do not share the active journal's 64-attempt ceiling.
func (r *Registry) readApplicationArchive(id string) (applicationArchive, bool, error) {
	var receipt applicationArchive
	if !validRef(id) {
		return receipt, false, ErrApplicationConflict
	}
	history, err := r.applicationHistory(false)
	if errors.Is(err, os.ErrNotExist) {
		return receipt, false, nil
	}
	if err != nil {
		return receipt, false, ErrApplicationJournal
	}
	f, err := privateFile(filepath.Join(history.dir, id+".json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return receipt, false, nil
	}
	if err != nil {
		return receipt, false, ErrApplicationJournal
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil || len(raw) > 16384 || !uniqueApplicationJSON(raw) {
		return receipt, false, ErrApplicationJournal
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || decoder.Decode(new(any)) != io.EOF || receipt.Version != 1 || receipt.Attempt.ID != id || !receipt.Attempt.valid() || receipt.Attempt.blocksApproval() || receipt.ArchivedAt.IsZero() || receipt.ArchivedAt.Before(receipt.Attempt.UpdatedAt) {
		return receipt, false, ErrApplicationJournal
	}
	return receipt, true, nil
}

func (r *Registry) saveApplicationArchive(a ApplicationAttempt) error {
	if !a.valid() || a.blocksApproval() {
		return ErrApplicationConflict
	}
	existing, found, err := r.readApplicationArchive(a.ID)
	if err != nil {
		return err
	}
	if found {
		if existing.Attempt != a {
			return ErrApplicationJournal
		}
		history, err := r.applicationHistory(true)
		if err != nil {
			return err
		}
		directory, err := os.Open(history.dir)
		if err != nil {
			return ErrApplicationJournal
		}
		defer directory.Close()
		return directory.Sync()
	}
	history, err := r.applicationHistory(true)
	if err != nil {
		return err
	}
	return history.writeFile(a.ID+".json", applicationArchive{1, a, time.Now().UTC()})
}

// Application reads one active or archived attempt without contacting a ship,
// credential store or receiver. Archived receipts have no executable next action.
func (r *Registry) Application(ctx context.Context, id string) (ApplicationAttempt, bool, error) {
	var result ApplicationAttempt
	archived := false
	if !validRef(id) {
		return result, false, ErrApplicationConflict
	}
	err := r.locked(ctx, func() error {
		state, err := r.readApplications()
		if err != nil {
			return err
		}
		for _, a := range state.Attempts {
			if a.ID == id {
				result = a
				return nil
			}
		}
		receipt, found, err := r.readApplicationArchive(id)
		if err != nil {
			return err
		}
		if !found {
			return ErrApplicationConflict
		}
		result, archived = receipt.Attempt, true
		return nil
	})
	return result, archived, err
}

// ArchiveApplication removes only a fully closed/reconciled attempt from the
// active list. The receipt is durable BEFORE removal, so a crash leaves either
// the active record or a permanent replay fence (possibly both). The same exact
// revision can be retried after a lost result. No session/receiver IO occurs.
func (r *Registry) ArchiveApplication(ctx context.Context, id string, revision uint64) (ApplicationAttempt, error) {
	var result ApplicationAttempt
	if revision == 0 {
		return result, ErrApplicationConflict
	}
	err := r.WithApplication(ctx, id, func(tx *ApplicationTransaction) error {
		a := tx.Current()
		result = a
		if a.Revision != revision || a.blocksApproval() {
			return ErrApplicationConflict
		}
		return r.locked(ctx, func() error {
			state, err := r.readApplications()
			if err != nil {
				return err
			}
			for i, current := range state.Attempts {
				if current.ID != id {
					continue
				}
				if current != a {
					return ErrApplicationConflict
				}
				if err := r.saveApplicationArchive(a); err != nil {
					return err
				}
				// Older issuer implementations must refuse this journal instead of
				// accepting an archived approval ID they do not know to look up.
				state.Version = 2
				state.Attempts = append(state.Attempts[:i], state.Attempts[i+1:]...)
				return r.writeFile("applications.json", state)
			}
			return ErrApplicationConflict
		})
	})
	if errors.Is(err, ErrApplicationConflict) {
		a, archived, readErr := r.Application(ctx, id)
		if readErr == nil && archived && a.Revision == revision {
			return a, nil
		}
	}
	return result, err
}
