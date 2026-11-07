package collection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// PreviousID is immutable intent, recorded before receiver preparation or login.
// SuccessorID is set atomically with a child's connected receipt, never inferred
// from a timeout. Original origin, credential and cleanup evidence stay intact.
type ApplicationReplacement struct {
	PreviousID        string `json:"previous_id,omitempty"`
	SuccessorID       string `json:"successor_id,omitempty"`
	AutomaticDisabled bool   `json:"automatic_disabled,omitempty"`
}

func (x ApplicationReplacement) valid(a ApplicationAttempt) bool {
	if x == (ApplicationReplacement{}) {
		return true
	}
	if a.Fleet.Version != 1 {
		return false
	}
	for _, id := range []string{x.PreviousID, x.SuccessorID} {
		if id != "" && (!validRef(id) || id == a.ID) {
			return false
		}
	}
	return x.PreviousID == "" || x.PreviousID != x.SuccessorID
}

func sameApplicationBinding(a, b ApplicationAttempt) bool {
	return a.Fleet.Version == 1 && b.Fleet.Version == 1 && a.ConnectionID == b.ConnectionID &&
		a.Fleet.Request.Binding == b.Fleet.Request.Binding && a.Target == b.Target && a.Credentials == b.Credentials
}

// A shared lease serializes all generations of one integration, including old
// cleanup and health probes. Registry permission writes remain short and can
// revoke future dispatch while a bounded external operation is in flight.
func (r *Registry) applicationScopeLock(a ApplicationAttempt) (*os.File, error) {
	digest := sha256.Sum256([]byte(a.ConnectionID + "\x00" + a.Target.PluginID + "\x00" + a.Target.Profile))
	f, err := privateFile(filepath.Join(r.dir, "application-scope-"+hex.EncodeToString(digest[:])+".lock"), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, ErrApplicationJournal
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrApplicationBusy
		}
		return nil, ErrApplicationJournal
	}
	return f, nil
}

// validateApplications resolves only the bounded lineage reachable from active
// records. Fully closed ancestors may be in immutable history; missing history
// is not permission to forget an older generation or create another branch.
func (r *Registry) validateApplications(state applicationsFile) error {
	if state.Version < 1 || state.Version > 4 || state.Attempts == nil || len(state.Attempts) > 64 {
		return ErrApplicationJournal
	}
	nodes := map[string]ApplicationAttempt{}
	refs := map[string]string{}
	add := func(a ApplicationAttempt) error {
		if !a.valid() || a.Fleet.Version != 0 && state.Version < 3 || (a.Replacement != (ApplicationReplacement{}) || a.Phase == "retiring") && state.Version < 4 {
			return ErrApplicationJournal
		}
		if _, ok := nodes[a.ID]; ok {
			return ErrApplicationJournal
		}
		if _, ok := refs[a.SessionRef]; ok {
			return ErrApplicationJournal
		}
		nodes[a.ID], refs[a.SessionRef] = a, a.ID
		return nil
	}
	queue := append([]ApplicationAttempt(nil), state.Attempts...)
	for _, a := range state.Attempts {
		if err := add(a); err != nil {
			return err
		}
	}
	for i := 0; i < len(queue); i++ {
		if len(queue) > 2048 {
			return ErrApplicationJournal
		}
		for _, id := range []string{queue[i].Replacement.PreviousID, queue[i].Replacement.SuccessorID} {
			if id == "" {
				continue
			}
			if _, ok := nodes[id]; ok {
				continue
			}
			receipt, ok, err := r.readApplicationArchive(id)
			if err != nil || !ok {
				return ErrApplicationJournal
			}
			if err = add(receipt.Attempt); err != nil {
				return err
			}
			queue = append(queue, receipt.Attempt)
		}
	}
	for _, a := range nodes {
		if id := a.Replacement.PreviousID; id != "" {
			p := nodes[id]
			if !sameApplicationBinding(a, p) || a.ApprovedAt.Before(p.ApprovedAt) || p.Phase == "connected" ||
				(!p.ReceiverReconciliation && p.ReceiverOutcome == "") {
				return ErrApplicationJournal
			}
			if (a.Phase == "connected" || a.Phase == "retiring") && p.Replacement.SuccessorID != a.ID {
				return ErrApplicationJournal
			}
			if p.Replacement.SuccessorID != "" && p.Replacement.SuccessorID != a.ID && a.blocksApproval() {
				return ErrApplicationJournal
			}
		}
		if id := a.Replacement.SuccessorID; id != "" {
			n := nodes[id]
			if n.Replacement.PreviousID != a.ID || !sameApplicationBinding(a, n) {
				return ErrApplicationJournal
			}
			switch n.Phase {
			case "connected", "retiring", "unresolved", "revoking", "revoked", "closed":
				if n.Outcome == "not-issued" || !n.ReceiverReconciliation && n.ReceiverOutcome == "" {
					return ErrApplicationJournal
				}
			default:
				return ErrApplicationJournal
			}
		}
		seen := map[string]bool{}
		for n := a; n.ID != ""; n = nodes[n.Replacement.PreviousID] {
			if seen[n.ID] {
				return ErrApplicationJournal
			}
			seen[n.ID] = true
		}
	}
	ancestor := func(a, b ApplicationAttempt) bool {
		for id := b.Replacement.PreviousID; id != ""; id = nodes[id].Replacement.PreviousID {
			if id == a.ID {
				return true
			}
		}
		return false
	}
	for i, a := range state.Attempts {
		if !a.blocksApproval() {
			continue
		}
		for _, b := range state.Attempts[:i] {
			if !b.blocksApproval() || a.Target.PluginID != b.Target.PluginID || a.Target.Profile != b.Target.Profile {
				continue
			}
			conflict := a.Origin == b.Origin || a.Fleet.Version != 0 && b.Fleet.Version != 0 && a.ConnectionID == b.ConnectionID || (a.Fleet.Version == 0) != (b.Fleet.Version == 0) && a.Ship == b.Ship
			if conflict && !(sameApplicationBinding(a, b) && (ancestor(a, b) || ancestor(b, a))) {
				return ErrApplicationJournal
			}
		}
	}
	return nil
}

// ApproveFleetReplacement creates intent under the current standing permission.
// The caller must resolve a replacement-capable exact approved receiver first.
// Preparation can be retried only through a new, durably distinct attempt after
// all effects of a failed child are reconciled. Accepted children never branch.
func (r *Registry) ApproveFleetReplacement(ctx context.Context, previous string, revision uint64, id string, target ApplicationTarget, fleet FleetApplication) (ApplicationAttempt, error) {
	var result ApplicationAttempt
	if !validRef(id) || id == previous || revision == 0 || !target.Valid() || !fleet.valid() {
		return result, ErrApplicationConflict
	}
	err := r.WithApplication(ctx, previous, func(tx *ApplicationTransaction) error {
		return r.locked(ctx, func() error {
			state, err := r.readApplications()
			if err != nil {
				return err
			}
			for _, a := range state.Attempts {
				if a.ID == id {
					if a.Replacement.PreviousID != previous || a.Target != target || a.Fleet != fleet {
						return ErrApplicationConflict
					}
					result = a
					return nil // receipt only; no replay
				}
			}
			if _, ok, err := r.readApplicationArchive(id); err != nil {
				return err
			} else if ok {
				return ErrApplicationArchived
			}
			old := tx.Current()
			if old.Revision != revision || old.Fleet.Version != 1 || old.Target != target || old.Fleet.Request.Binding != fleet.Request.Binding ||
				(old.Phase != "connected" && old.Phase != "retiring") || old.Replacement.SuccessorID != "" || old.Revision == ^uint64(0) {
				return ErrApplicationConflict
			}
			if err := r.currentFleetGrant(fleet, target); err != nil {
				return err
			}
			if len(state.Attempts) >= 64 {
				return ErrApplicationLimit
			}
			now := time.Now().UTC()
			result = ApplicationAttempt{ID: id, ConnectionID: old.ConnectionID, ConnectionRevision: fleet.GrantRevision, Origin: fleet.Request.Route.Origin, Ship: old.Ship, Credentials: old.Credentials, SessionRef: NewSessionRef(), Target: target, Authority: old.Authority, Phase: "approved", Revision: 1, ApprovedAt: now, UpdatedAt: now, Fleet: fleet, Replacement: ApplicationReplacement{PreviousID: previous}}
			found := false
			for i, a := range state.Attempts {
				if a.ID == previous {
					if a != old {
						return ErrApplicationConflict
					}
					if old.Phase == "connected" || old.Replacement.AutomaticDisabled {
						a.Replacement.AutomaticDisabled = false
						a.Phase, a.Revision, a.UpdatedAt = "retiring", a.Revision+1, now
						state.Attempts[i] = a
					}
					found = true
				}
				if a.Replacement.PreviousID == previous && a.blocksApproval() {
					return ErrApplicationConflict
				}
			}
			if !found {
				return ErrApplicationConflict
			}
			state.Version = 4
			state.Attempts = append(state.Attempts, result)
			return r.saveApplications(state, true)
		})
	})
	if err != nil {
		return ApplicationAttempt{}, err
	}
	return result, nil
}

// Called inside publication's registry lock. A parent disconnect blocks pending
// dispatch. Accepted observation, in contrast, remains recordable after revoke.
func replaceApplicationParent(state *applicationsFile, old, next ApplicationAttempt) error {
	if old.Replacement.PreviousID == "" {
		return nil
	}
	for i, p := range state.Attempts {
		if p.ID != old.Replacement.PreviousID {
			continue
		}
		if !sameApplicationBinding(p, old) {
			return ErrApplicationConflict
		}
		switch next.Phase {
		case "preparing", "prepared", "issuing", "delivering":
			if p.Phase != "retiring" || p.Replacement.SuccessorID != "" || p.Replacement.AutomaticDisabled || old.Replacement.AutomaticDisabled {
				return ErrApplicationConflict
			}
		case "connected":
			if p.Replacement.SuccessorID != "" && p.Replacement.SuccessorID != old.ID || p.Revision == ^uint64(0) {
				return ErrApplicationConflict
			}
			p.Replacement.SuccessorID, p.Revision, p.UpdatedAt = old.ID, p.Revision+1, time.Now().UTC()
			state.Attempts[i] = p
		}
		return nil
	}
	// A fully archived parent only permits cleanup of its existing child. It
	// cannot be revived to prepare or dispatch a new credential.
	switch next.Phase {
	case "preparing", "prepared", "issuing", "delivering", "connected":
		return ErrApplicationConflict
	}
	return nil
}

// DisableReplacementAutomation records an explicit Disconnect without widening
// it into logout of an already accepted successor. Pending handoff intent is
// cancelled in both directions, so a later maintenance pass cannot undo it.
func (tx *ApplicationTransaction) DisableReplacementAutomation(ctx context.Context) error {
	if !tx.active {
		return ErrApplicationConflict
	}
	old := tx.current
	if old.Fleet.Version != 1 || old.Replacement == (ApplicationReplacement{}) && old.Phase != "retiring" {
		return nil
	}
	var updated ApplicationAttempt
	err := tx.r.locked(ctx, func() error {
		state, err := tx.r.readApplications()
		if err != nil {
			return err
		}
		found := false
		for _, a := range state.Attempts {
			if a.ID == old.ID {
				if a != old {
					return ErrApplicationConflict
				}
				found = true
			}
		}
		if !found {
			return ErrApplicationConflict
		}
		for i, a := range state.Attempts {
			cancel := a.ID == old.ID || a.ID == old.Replacement.PreviousID && a.Replacement.SuccessorID == "" || a.Replacement.PreviousID == old.ID && old.Replacement.SuccessorID == ""
			if cancel && a.Phase != "closed" && !a.Replacement.AutomaticDisabled {
				if a.Revision == ^uint64(0) {
					return ErrApplicationConflict
				}
				a.Replacement.AutomaticDisabled = true
				a.Revision++
				a.UpdatedAt = time.Now().UTC()
				state.Attempts[i] = a
			}
			if a.ID == old.ID {
				updated = a
			}
		}
		return tx.r.saveApplications(state, false)
	})
	if err == nil {
		tx.current = updated
	}
	return err
}
