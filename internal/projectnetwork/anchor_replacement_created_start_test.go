package projectnetwork

import (
	"context"
	"reflect"
	"testing"
)

func TestCreatedRecoveryReplacementStartsRetainedIDWithoutCreate(t *testing.T) {
	for _, fault := range []string{"", "start-lost-reply", "commit-start-dispatched", "stale-start", "inspect-created", "engine"} {
		t.Run(fault, func(t *testing.T) {
			f := newAnchorReplacementFixture(t, false)
			f.mode = "create-returned-id-error"
			created, err := ExecuteNextAnchorReplacement(context.Background(), f.h, f.saved, f, f.commit, f.clock)
			if err == nil || created.Members[0].Phase != "created" || f.creates != 1 || f.starts != 0 {
				t.Fatal("fixture lacks retained ID")
			}
			before := created.Clone()
			id := created.Members[0].Container.ID
			f.mode = fault
			readonlyStart := struct{ AnchorReplacementStartDriver }{f}
			after, err := StartCreatedAnchorReplacement(context.Background(), f.h, created, readonlyStart, f.commit, f.clock)
			if !reflect.DeepEqual(created, before) || f.creates != 1 || after.Members[0].Container.ID != id {
				t.Fatal("continuation changed ID or repeated Create")
			}
			if fault == "" {
				if err != nil || f.starts != 1 || after.Members[0].Phase != "running" {
					t.Fatal("retained replacement did not start", err)
				}
			} else if err == nil {
				t.Fatal("failed continuation succeeded")
			}
			if fault == "start-lost-reply" {
				if f.starts != 1 || after.Members[0].Phase != "start-dispatched" {
					t.Fatal("uncertain Start lost")
				}
				if _, err = StartCreatedAnchorReplacement(context.Background(), f.h, after, readonlyStart, f.commit, f.clock); err == nil || f.starts != 1 {
					t.Fatal("uncertain Start replayed")
				}
				f.mode = ""
				after, err = ObserveAnchorReplacementRunning(context.Background(), f.h, after, f, f.commit, f.clock)
				if err != nil || after.Members[0].Phase != "running" || after.Members[0].StartAcknowledged || f.starts != 1 {
					t.Fatal("read-only resolution changed lost acknowledgement", err)
				}
			}
			if fault != "" && fault != "start-lost-reply" && f.starts != 0 {
				t.Fatal("failed preflight dispatched Start")
			}
		})
	}
}
func TestCreatedRecoveryReplacementRefusesUnknownOrAlreadyStarted(t *testing.T) {
	for _, fault := range []string{"create-lost-id", "start-lost-reply", ""} {
		f := newAnchorReplacementFixture(t, false)
		f.mode = fault
		pending, _ := ExecuteNextAnchorReplacement(context.Background(), f.h, f.saved, f, f.commit, f.clock)
		creates, starts := f.creates, f.starts
		if _, err := StartCreatedAnchorReplacement(context.Background(), f.h, pending, f, f.commit, f.clock); err == nil {
			t.Fatal("non-created checkpoint admitted", fault)
		}
		if f.creates != creates || f.starts != starts {
			t.Fatal("refusal dispatched effects")
		}
	}
}
