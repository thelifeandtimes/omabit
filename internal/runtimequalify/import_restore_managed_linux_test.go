//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestManagedImportRestorePreservesDurableStateAndStopFence(t *testing.T) {
	for _, phase := range []string{"none", "retaining-destination", "restoring-source", "complete"} {
		t.Run(phase, func(t *testing.T) {
			f, state, _ := partialImportRestoreFixture(t)
			ctx := context.Background()
			review, err := previewImportRestore(ctx, state, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state.ImportRestore = &ManagedImportRestore{Digest: review.Digest}
			durable := state
			failed := false
			persist := func(s ManagedState) error {
				if !failed && s.ImportRestore.Journal != nil && s.ImportRestore.Journal.Stage == phase {
					failed = true
					return errors.New("journal unavailable")
				}
				durable = s
				if s.ImportRestore.Journal != nil {
					copy := *s.ImportRestore
					copy.Journal = roundTripRestoreJournal(t, *copy.Journal)
					durable.ImportRestore = &copy
				}
				return nil
			}
			result, obs, err := applyManaged(ctx, "restore-import", state, persist, f.d)
			if phase != "none" {
				if err == nil || !failed || !result.UncertainProvisioning() {
					t.Fatal("did not preserve failed publication", err)
				}
				if _, _, err := applyManaged(ctx, "start", durable, persist, f.d); err == nil {
					t.Fatal("restoration allowed Start")
				}
				durable, _, err = applyManaged(ctx, "stop", durable, persist, f.d)
				if err != nil || durable.ImportRestore == nil {
					t.Fatal("Stop discarded restoration", err)
				}
				if _, err := previewImportRestore(ctx, durable, f.d); err != nil {
					t.Fatal("resume preview failed", err)
				}
				result, obs, err = applyManaged(ctx, "restore-import", durable, persist, f.d)
			}
			if err != nil || obs.State != "restored" || !obs.RecoveryRequired || result.ImportRestore.Journal.Stage != "complete" || !result.UncertainProvisioning() {
				t.Fatal("completion escaped retirement fence", obs, err)
			}
			if _, err := os.Stat(review.OriginalPath); err != nil {
				t.Fatal("source not restored", err)
			}
			if _, err := os.Stat(review.RetainedPath); err != nil {
				t.Fatal("partial destination lost", err)
			}
			if len(f.calls) != 0 || f.downloads != 0 {
				t.Fatal("restoration dispatched runtime")
			}
		})
	}
}
