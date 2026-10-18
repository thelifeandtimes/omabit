package protocol

import "testing"

func TestSelectableAndHistoricalLoomBounds(t *testing.T) {
	for _, tc := range []struct {
		loom                   uint8
		selectable, historical bool
	}{{0, false, false}, {19, false, false}, {20, false, true}, {29, false, true}, {30, true, true}, {31, true, true}, {34, true, true}, {35, false, false}, {255, false, false}} {
		if ValidLoom(tc.loom) != tc.selectable || ValidRecordedLoom(tc.loom) != tc.historical {
			t.Fatalf("loom %d: selectable=%v historical=%v", tc.loom, ValidLoom(tc.loom), ValidRecordedLoom(tc.loom))
		}
	}
	if EffectiveLoom(0) != 31 || EffectiveLoom(29) != 29 {
		t.Fatal("historical defaults or explicit launch values changed")
	}
}
