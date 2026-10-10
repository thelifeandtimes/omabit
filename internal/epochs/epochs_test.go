package epochs

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func inventory(now time.Time) *Inventory {
	return &Inventory{Source: Source, PierID: strings.Repeat("a", 32), ObservedAt: now, Digest: strings.Repeat("d", 64), Entries: []Entry{
		{Name: "0i0", LogicalBytes: 100, AllocatedBytes: 4096, FileCount: 3},
		{Name: "0i10", LogicalBytes: 200, AllocatedBytes: 8192, FileCount: 4, SharedFiles: true},
		{Name: "0i9007199254740993", LogicalBytes: 300, AllocatedBytes: 12288, FileCount: 4, Protected: true},
		{Name: "0i18446744073709551615", LogicalBytes: 400, AllocatedBytes: 16384, FileCount: 4, Protected: true},
	}}
}

func TestPrefixPreviewPreservesProtectedSuffixAndExactNames(t *testing.T) {
	now := time.Now().UTC()
	inv := inventory(now)
	p, err := Plan(inv, "0i10", inv.Digest, now)
	if err != nil || !reflect.DeepEqual(p.Selected, inv.Entries[:2]) || !reflect.DeepEqual(p.Retained, inv.Entries[2:]) || p.SelectedLogicalBytes != 300 || p.SelectedAllocatedBytes != 12288 || !p.SharedFiles || p.DeletionSupported || !p.RequiresStoppedRevalidation {
		t.Fatalf("incorrect prefix preview: %+v %v", p, err)
	}
	p.Selected[0].Name = "changed"
	if inv.Entries[0].Name != "0i0" {
		t.Fatal("preview aliased inventory")
	}
}

func TestUnsafeOrStalePreviewsAreRejected(t *testing.T) {
	for _, fault := range []string{"nil", "stale", "future", "digest", "source", "empty", "duplicate", "unsorted", "negative", "overflow", "protection", "protected cutoff", "missing cutoff", "noncanonical", "one epoch"} {
		t.Run(fault, func(t *testing.T) {
			now := time.Now().UTC()
			inv := inventory(now)
			cutoff, digest := "0i10", inv.Digest
			switch fault {
			case "nil":
				inv = nil
			case "stale":
				inv.ObservedAt = now.Add(-MaxAge - time.Second)
			case "future":
				inv.ObservedAt = now.Add(time.Minute)
			case "digest":
				digest = strings.Repeat("e", 64)
			case "source":
				inv.Source = "unknown"
			case "empty":
				inv.Entries = nil
			case "duplicate":
				inv.Entries[1].Name = "0i0"
			case "unsorted":
				inv.Entries[0].Name = "0i11"
			case "negative":
				inv.Entries[0].AllocatedBytes = -1
			case "overflow":
				inv.Entries[0].LogicalBytes = math.MaxInt64
			case "protection":
				inv.Entries[3].Protected = false
			case "protected cutoff":
				cutoff = inv.Entries[2].Name
			case "missing cutoff":
				cutoff = "0i9"
			case "noncanonical":
				cutoff = "0i010"
			case "one epoch":
				inv.Entries = inv.Entries[:1]
				inv.Entries[0].Protected = true
				cutoff = "0i0"
			}
			if _, err := Plan(inv, cutoff, digest, now); err == nil {
				t.Fatal("unsafe preview accepted")
			}
		})
	}
}

func TestCanonicalEpochNames(t *testing.T) {
	for _, name := range []string{"0i0", "0i1", "0i9007199254740993", "0i18446744073709551615"} {
		if _, err := Number(name); err != nil {
			t.Fatal(name, err)
		}
	}
	for _, name := range []string{"", "0i", "0i00", "0i+1", "0i-1", "0i1/..", "0i1\n", "0i18446744073709551616", "0I1", "0i 1"} {
		if _, err := Number(name); err == nil {
			t.Fatal("noncanonical name accepted", name)
		}
	}
}

func TestGenesisScopeIsNarrowerThanGeneralPrefixSelection(t *testing.T) {
	now := time.Now().UTC()
	inv := inventory(now)
	for i := range inv.Entries {
		inv.Entries[i].Digest = strings.Repeat("a", 64)
		inv.Entries[i].SharedFiles = false
		inv.Entries[i].FileCount = 5
		inv.Entries[i].SnapshotBytes = 100
	}
	inv.Entries[0].FileCount, inv.Entries[0].SnapshotBytes = 4, 0
	if _, err := GenesisPlan(inv, "0i0", inv.Digest, now); err != nil {
		t.Fatal(err)
	}
	if _, err := GenesisPlan(inv, "0i10", inv.Digest, now); err == nil {
		t.Fatal("unqualified non-genesis prefix accepted")
	}
	inv.Entries[1].SharedFiles = true
	if _, err := GenesisPlan(inv, "0i0", inv.Digest, now); err == nil {
		t.Fatal("shared files accepted for deletion")
	}
}

func TestQualifiedPrefixPlanProtectsNewestAndRequiresLayoutEvidence(t *testing.T) {
	now := time.Now().UTC()
	inv := inventory(now)
	for i := range inv.Entries {
		inv.Entries[i].Digest = strings.Repeat("a", 64)
		inv.Entries[i].SharedFiles = false
		inv.Entries[i].FileCount = 5
		inv.Entries[i].SnapshotBytes = 100
	}
	inv.Entries[0].FileCount, inv.Entries[0].SnapshotBytes = 4, 0
	p, err := PrunePlan(inv, "0i10", inv.Digest, now)
	if err != nil || len(p.Selected) != 2 || len(p.Retained) != 2 {
		t.Fatalf("valid selected prefix refused: %+v %v", p, err)
	}
	if _, err := PrunePlan(inv, inv.Entries[2].Name, inv.Digest, now); err == nil {
		t.Fatal("protected suffix selected")
	}
	inv.Entries = inv.Entries[1:]
	inv.Entries[0].Protected = false
	p, err = PrunePlan(inv, "0i10", inv.Digest, now)
	if err != nil || len(p.Selected) != 1 || len(p.Retained) != 2 {
		t.Fatal("prefix after genesis removal refused")
	}
	inv.Entries[0].SnapshotBytes = 0
	if _, err := PrunePlan(inv, "0i10", inv.Digest, now); err == nil {
		t.Fatal("missing selected snapshot accepted")
	}
}

func TestPrefixSelectionRemainsBoundedAtInventoryLimit(t *testing.T) {
	now := time.Now().UTC()
	inv := &Inventory{Source: Source, PierID: strings.Repeat("a", 32), Digest: strings.Repeat("d", 64), ObservedAt: now}
	for i := 0; i < MaxEpochs; i++ {
		e := Entry{Name: fmt.Sprintf("0i%d", i), Digest: strings.Repeat("a", 64), FileCount: 5, SnapshotBytes: 10, LogicalBytes: 10, AllocatedBytes: 4096, Protected: i >= MaxEpochs-2}
		if i == 0 {
			e.FileCount, e.SnapshotBytes = 4, 0
		}
		inv.Entries = append(inv.Entries, e)
	}
	p, err := PrunePlan(inv, fmt.Sprintf("0i%d", MaxEpochs-3), inv.Digest, now)
	if err != nil || len(p.Selected) != MaxEpochs-2 || len(p.Retained) != 2 {
		t.Fatalf("bounded prefix: %+v %v", p, err)
	}
	inv.Entries = append(inv.Entries, Entry{Name: "0i999"})
	if _, err := PrunePlan(inv, "0i0", inv.Digest, now); err == nil {
		t.Fatal("over-limit inventory admitted")
	}
}
