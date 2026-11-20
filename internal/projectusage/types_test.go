package projectusage

import (
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func modelObservation() Observation {
	id := pierstore.DirectoryIdentity{Device: 1, Inode: 2, Birth: "1:1"}
	r := Root{ID: "root", Kind: "member-mount", InstanceID: "member", RuntimeID: strings.Repeat("a", 32), Path: "/private/fixture", Expected: id}
	totals := Totals{Directories: 1, RegularFiles: 1, UniqueFileInodes: 1, LogicalBytes: 100, AllocatedBytes: 4096}
	return Observation{Version: 1, Binding: Binding{RunnerID: "runner", Mode: "development-managed", ProjectID: "project", Revision: 1, Generation: 1, PlanDigest: strings.Repeat("a", 64), LedgerSHA256: strings.Repeat("b", 64)}, Scope: Scope, Status: "measured", DeclaredRoots: []Root{r}, Roots: []RootUsage{{r, id, totals}}, Totals: &totals, ExcludedComponents: Exclusions(), SharedAllowanceStatus: "not-configured", StartedAt: time.Unix(100, 0).UTC(), ReadAt: time.Unix(101, 0).UTC(), Basis: "inert model fixture"}
}

func TestAllocationGrowthUsesExactBindingsAndUnsignedDeltas(t *testing.T) {
	before := modelObservation()
	after := before.Clone()
	after.Totals.AllocatedBytes = 512
	after.Roots[0].Totals.AllocatedBytes = 512
	after.Totals.LogicalBytes = ^uint64(0)
	after.Roots[0].Totals.LogicalBytes = ^uint64(0)
	got := Compare(before, after, time.Minute)
	if got.Validate() != nil || got.Status != "measured" || got.Allocated.Direction != "decreased" || got.Allocated.Bytes != 3584 || got.Logical.Direction != "increased" || got.Logical.Bytes != ^uint64(0)-100 || got.ElapsedNanos != int64(time.Minute) {
		t.Fatal("exact growth delta lost", got)
	}
	for _, variant := range []string{"binding", "root", "unknown", "short", "expired", "first"} {
		a, b := before.Clone(), after.Clone()
		elapsed := time.Minute
		want := ""
		switch variant {
		case "binding":
			b.Binding.LedgerSHA256 = strings.Repeat("c", 64)
			want = "binding-changed"
		case "root":
			b.Roots[0].Observed.Inode++
			b.Roots[0].Root.Expected.Inode++
			b.DeclaredRoots[0].Expected.Inode++
			want = "binding-changed"
		case "unknown":
			b.Status, b.Reason, b.Roots, b.Totals = "unknown", "unavailable", nil, nil
			want = "current-unavailable"
		case "short":
			elapsed = time.Millisecond
			want = "window-too-short"
		case "expired":
			elapsed = 6 * time.Minute
			want = "sample-expired"
		case "first":
			a = Observation{}
			want = "need-two-samples"
		}
		out := Compare(a, b, elapsed)
		if out.Validate() != nil || out.Status != "unknown" || out.Reason != want || out.Allocated != nil || out.Logical != nil || out.ElapsedNanos != 0 {
			t.Fatal("invalid growth exposed stale delta", variant, out)
		}
	}
	if before.Totals.AllocatedBytes != 4096 || before.Roots[0].Root.Expected.Inode != 2 {
		t.Fatal("clone retained mutable fields")
	}
}

func TestAllocationModelRejectsUnsafeClaimsAndOverflow(t *testing.T) {
	base := modelObservation()
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"complete", "zero-shared", "missing-exclusion", "partial-unknown", "unknown-growth", "wrong-sum", "wrong-root", "zero-directory", "bad-runtime"} {
		b := base.Clone()
		switch change {
		case "complete":
			b.FullProjectBudgetKnown = true
		case "zero-shared":
			b.SharedAllowanceStatus = "zero"
		case "missing-exclusion":
			b.ExcludedComponents = b.ExcludedComponents[:1]
		case "partial-unknown":
			b.Status, b.Reason = "unknown", "unavailable"
		case "unknown-growth":
			growth := Compare(base, base, time.Second)
			b.Status, b.Reason, b.Roots, b.Totals = "unknown", "unavailable", nil, nil
			b.Growth = &growth
		case "wrong-sum":
			b.Totals.AllocatedBytes++
		case "wrong-root":
			b.Roots[0].Root.ID = "other"
		case "zero-directory":
			b.Totals.Directories = 0
			b.Roots[0].Totals.Directories = 0
		case "bad-runtime":
			b.DeclaredRoots[0].RuntimeID = "arbitrary"
		}
		if b.Validate() == nil {
			t.Fatal("unsafe measurement claim accepted", change)
		}
	}
	if _, err := (Totals{AllocatedBytes: ^uint64(0)}).Add(Totals{AllocatedBytes: 1}); err == nil {
		t.Fatal("overflow wrapped")
	}
	if _, err := (Totals{Directories: MaxEntries}).Add(Totals{Directories: 1}); err == nil {
		t.Fatal("entry overflow accepted")
	}
	request := Request{RunnerID: "runner", ProjectID: "project", ExpectedRevision: 1, ReviewDigest: strings.Repeat("a", 64), IncludePrivateData: true}
	if request.Validate() != nil {
		t.Fatal("exact selection refused")
	}
	request.IncludePrivateData = false
	if request.Validate() == nil {
		t.Fatal("private consent omitted")
	}
}
