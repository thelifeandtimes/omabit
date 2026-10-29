//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
)

func PreviewImportRestore(ctx context.Context, state ManagedState) (ImportRestoreReview, error) {
	return previewImportRestore(ctx, state, defaults())
}

func previewImportRestore(ctx context.Context, state ManagedState, d dependencies) (ImportRestoreReview, error) {
	if state.ImportRestore != nil && state.ImportRestore.Journal != nil {
		j := *state.ImportRestore.Journal
		if err := validateRestoreJournal(state, j, state.ImportRestore.Digest, d); err != nil {
			return ImportRestoreReview{}, err
		}
		if _, err := inspectRestoreLocations(ctx, j); err != nil {
			return ImportRestoreReview{}, err
		}
		return j.Review, nil
	}
	return reviewImportRestore(ctx, state, d)
}

func applyImportRestore(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if state.ImportRestore == nil || persist == nil {
		return state, unknown, errors.New("source restoration requires an approved digest and durable intent")
	}
	approved := state.ImportRestore.Digest
	_, err := restoreImportMoves(ctx, state, approved, state.ImportRestore.Journal, func(j ImportRestoreJournal) error {
		next := state
		next.ImportRestore = &ManagedImportRestore{Digest: approved, Journal: &j}
		if err := persist(next); err != nil {
			return err
		}
		state = next
		return nil
	}, d, importRestoreIO{})
	if err != nil {
		return state, unknown, err
	}
	return state, ManagedObservation{State: "restored", Health: "unknown", RecoveryRequired: true}, nil
}
