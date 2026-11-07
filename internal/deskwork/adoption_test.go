package deskwork

import (
	"reflect"
	"testing"
)

func TestAdoptionMatchesSelectedClayAndRetainsOtherOwnership(t *testing.T) {
	previous := Ownership{1, deployManifest(t, map[string]string{"kept.txt": "old raw", "desk.docket-0": "old docket"}, false), deployManifest(t, map[string]string{"kept.txt": "old canonical", "desk.docket-0": "old docket canonical"}, true)}
	current := deployManifest(t, map[string]string{"kept.txt": "external edit", "desk.docket-0": "new docket canonical", "app/demo.hoon": "app canonical", "generated.txt": "scaffold"}, true)
	source := deployManifest(t, map[string]string{"desk.docket-0": "new docket raw", "app/demo.hoon": "app raw"}, false)
	known := deployManifest(t, map[string]string{"desk.docket-0": "new docket canonical", "app/demo.hoon": "app canonical"}, true)
	p, err := PlanAdoption(previous, current, source, known)
	if err != nil || p.Validate() != nil || len(p.Conflicts) != 0 || p.NewCount != 1 || p.RefreshCount != 1 || len(p.Next.Raw.Files) != 3 {
		t.Fatal(p, err)
	}
	if p.Next.Clay.Index()["kept.txt"] != previous.Clay.Index()["kept.txt"] {
		t.Fatal("implicitly refreshed unselected external edit")
	}
	if _, ok := p.Next.Raw.Index()["generated.txt"]; ok {
		t.Fatal("adopted unspecified generated source")
	}
	// Adoption permits a later selected update, but not overwriting unrelated drift.
	update := deployManifest(t, map[string]string{"desk.docket-0": "next", "app/demo.hoon": "next", "kept.txt": "new raw"}, false)
	canonical := deployManifest(t, map[string]string{"desk.docket-0": "next canonical", "app/demo.hoon": "next canonical", "kept.txt": "new canonical"}, true)
	commit, err := PlanDeployment(p.Next, current, update, canonical)
	if err != nil || !reflect.DeepEqual(commit.Conflicts, []string{"kept.txt"}) || len(commit.Changes) != 2 {
		t.Fatal(commit, err)
	}
	p2, err := PlanAdoption(p.Next, current, source, known)
	if err != nil || p2.NewCount+p2.RefreshCount != 0 {
		t.Fatal("repeated adoption changed ownership", p2, err)
	}
	p.NewCount++
	if p.Validate() == nil {
		t.Fatal("altered adoption accepted")
	}
}
func TestAdoptionNeverTreatsDifferentMissingOrUnknownFilesAsOwned(t *testing.T) {
	current := deployManifest(t, map[string]string{"different.txt": "existing", "same.txt": "canonical same", "unknown.txt": "existing"}, true)
	source := deployManifest(t, map[string]string{"different.txt": "different", "missing.txt": "new", "same.txt": "raw same", "unknown.txt": "raw"}, false)
	known := deployManifest(t, map[string]string{"different.txt": "different canonical", "same.txt": "canonical same"}, true)
	p, err := PlanAdoption(EmptyOwnership(), current, source, known)
	if err != nil || p.Validate() != nil || !reflect.DeepEqual(p.Conflicts, []string{"different.txt", "missing.txt", "unknown.txt"}) || p.NewCount != 1 || len(p.Next.Raw.Files) != 1 {
		t.Fatal(p, err)
	}
	if _, err = PlanAdoption(EmptyOwnership(), current, EmptyOwnership().Raw, EmptyOwnership().Clay); err == nil {
		t.Fatal("empty adoption accepted")
	}
}
