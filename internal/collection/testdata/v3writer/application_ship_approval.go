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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

const applicationShipApprovalsFile = "application-ship-approvals.json"

// ApplicationShipApproval is a standing user choice for an application/profile
// and a particular ship incarnation. It intentionally contains no origin or
// credential. Automatic connection attempts must independently select/verify a
// route and retain their existing durable, origin-bound attempt receipts.
type ApplicationShipApproval struct {
	Binding   shiproute.Binding `json:"binding"`
	Target    ApplicationTarget `json:"target"`
	Allowed   bool              `json:"allowed"`
	Revision  uint64            `json:"revision"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (a ApplicationShipApproval) valid() bool {
	return a.Binding.Validate() == nil && a.Target.Valid() && a.Revision > 0 && a.Revision <= apppolicy.MaxRevision && !a.UpdatedAt.IsZero()
}

func sameApplicationShip(a, b ApplicationShipApproval) bool {
	// A helper update replaces this logical grant; the old digest does not retain
	// an independent standing approval alongside it.
	return a.Binding == b.Binding && a.Target.PluginID == b.Target.PluginID && a.Target.Profile == b.Target.Profile
}

type ApplicationShipApprovals struct {
	Version  int                       `json:"version"`
	Revision uint64                    `json:"revision"`
	Entries  []ApplicationShipApproval `json:"entries"`
	History  []ApplicationShipApproval `json:"history"`
}

func (s ApplicationShipApprovals) validate() error {
	if s.Version != 1 || s.Entries == nil || s.History == nil || len(s.Entries) > 256 || len(s.History) > 2048 || uint64(len(s.History)) != s.Revision {
		return ErrApplicationJournal
	}
	current := []ApplicationShipApproval{}
	var at time.Time
	for i, change := range s.History {
		if !change.valid() || change.Revision != uint64(i+1) || change.UpdatedAt.Before(at) {
			return ErrApplicationJournal
		}
		at = change.UpdatedAt
		found := false
		for j, prior := range current {
			if sameApplicationShip(change, prior) {
				current[j], found = change, true
				break
			}
		}
		if !found {
			current = append(current, change)
		}
	}
	if len(current) != len(s.Entries) {
		return ErrApplicationJournal
	}
	for i := range current {
		if current[i] != s.Entries[i] {
			return ErrApplicationJournal
		}
	}
	return nil
}

// Allows must be called on the freshly locked state immediately before starting
// an automatic attempt. A copied approval is not a reusable authorization token.
func (s ApplicationShipApprovals) Allows(binding shiproute.Binding, target ApplicationTarget) bool {
	if s.validate() != nil || binding.Validate() != nil || !target.Valid() {
		return false
	}
	for _, grant := range s.Entries {
		if grant.Allowed && grant.Binding == binding && grant.Target == target {
			return true
		}
	}
	return false
}

func (r *Registry) readApplicationShipApprovals() (ApplicationShipApprovals, error) {
	state := ApplicationShipApprovals{Version: 1, Entries: []ApplicationShipApproval{}, History: []ApplicationShipApproval{}}
	f, err := privateFile(filepath.Join(r.dir, applicationShipApprovalsFile), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, ErrApplicationJournal
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRegistryBytes+1))
	if err != nil || len(data) > maxRegistryBytes || !uniqueApplicationJSON(data) {
		return state, ErrApplicationJournal
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || state.validate() != nil {
		return ApplicationShipApprovals{}, ErrApplicationJournal
	}
	return state, nil
}

func (r *Registry) ApplicationShipApprovals(ctx context.Context) (ApplicationShipApprovals, error) {
	var result ApplicationShipApprovals
	err := r.locked(ctx, func() error {
		var err error
		result, err = r.readApplicationShipApprovals()
		return err
	})
	return result, err
}

// SetApplicationShipApproval records an explicit user choice, not a connection.
// The caller must display the selected ship, helper and broad owner-session
// authority first. It cannot silently convert an old per-attempt approval into
// a standing grant. Revoke affects future attempts; existing sessions retain
// their ordinary explicit cleanup records.
func (r *Registry) SetApplicationShipApproval(ctx context.Context, expectedRevision uint64, binding shiproute.Binding, target ApplicationTarget, allowed bool) (ApplicationShipApprovals, error) {
	var result ApplicationShipApprovals
	if binding.Validate() != nil || !target.Valid() || expectedRevision > apppolicy.MaxRevision {
		return result, ErrApplicationConflict
	}
	err := r.locked(ctx, func() error {
		state, err := r.readApplicationShipApprovals()
		if err != nil {
			return err
		}
		if state.Revision != expectedRevision {
			return ErrApplicationConflict
		}
		change := ApplicationShipApproval{Binding: binding, Target: target, Allowed: allowed, Revision: state.Revision + 1, UpdatedAt: time.Now().UTC()}
		found := false
		for i, prior := range state.Entries {
			if sameApplicationShip(prior, change) {
				if prior.Target == target && prior.Allowed == allowed {
					result = state
					return nil
				}
				state.Entries[i], found = change, true
				break
			}
		}
		if !found {
			state.Entries = append(state.Entries, change)
		}
		state.History = append(state.History, change)
		state.Revision++
		if state.validate() != nil {
			return ErrApplicationJournal
		}
		body, err := json.MarshalIndent(state, "", "  ")
		if err != nil || len(body)+1 > maxRegistryBytes {
			return ErrApplicationLimit
		}
		if err := r.writeFile(applicationShipApprovalsFile, state); err != nil {
			return err
		}
		result = state
		return nil
	})
	return result, err
}
