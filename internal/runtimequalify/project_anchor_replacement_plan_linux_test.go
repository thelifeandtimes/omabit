//go:build linux && amd64

package runtimequalify

import (
	"context"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectRecoveryReplacementPlanningIsReadOnly(t *testing.T) {
	f := completedRecoveryRetirement(t)
	h := anchorDrainHistory(f.selection.Source)
	selection := ProjectAnchorReplacementPlanSelection{Source: f.selection.Source, Retirement: f.saved.Clone(), OperationID: "planned-recovery"}
	for _, m := range f.saved.Members {
		selection.Members = append(selection.Members, projectnetwork.RecoveryJournalMemberID{InstanceID: m.InstanceID, OperationID: "planned-" + m.InstanceID})
	}
	removed := len(f.removes)
	for _, mode := range []string{"valid", "reordered", "reused-id", "partial-retirement", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			copy := selection
			copy.Members = append([]projectnetwork.RecoveryJournalMemberID(nil), selection.Members...)
			copy.Retirement = selection.Retirement.Clone()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "reordered":
				copy.Members[0], copy.Members[1] = copy.Members[1], copy.Members[0]
			case "reused-id":
				copy.Members[0].OperationID = copy.Members[1].OperationID
			case "partial-retirement":
				copy.Retirement.Members[0].Phase = "remove-dispatched"
			case "cancelled":
				cancel()
			}
			got, err := planProjectAnchorReplacement(ctx, copy, f.x.f.f.d)
			if mode != "valid" {
				if err == nil {
					t.Fatal("invalid replacement plan accepted")
				}
				return
			}
			if err != nil || got.Validate(h) != nil || got.OperationID != selection.OperationID {
				t.Fatal("planning failed", err)
			}
			if !reflect.DeepEqual(got.Retirement, f.saved) || len(f.removes) != removed {
				t.Fatal("planning changed retained retirement")
			}
			for i, m := range got.Members {
				if m.Creation.OperationID != selection.Members[i].OperationID || m.Container.ID != "" || m.Container.ContractDigest == "" || m.Phase != "planned" || m.Creation.Storage != f.saved.Members[i].After.Piers[i].Storage {
					t.Fatal("planning lost reserved creator or pier")
				}
			}
		})
	}
	if len(f.removes) != removed {
		t.Fatal("planning performed a removal")
	}
}
