package apppolicy

import "testing"

func TestPermissionsInheritanceAndEffects(t *testing.T) {
	p := Default()
	a := Target{InstanceID: "publisher", Desk: "app"}
	b := Target{InstanceID: "publisher", Desk: "other"}
	if p.Effective(a).Commit != (Decision{Block, "global"}) || p.Check(a, 0, "commit", "private") == nil || p.Check(a, 0, "publish", "") == nil {
		t.Fatal("default grant")
	}
	set := func(scope string, target Target, g Gates) {
		t.Helper()
		if err := p.Apply(Update{p.Revision, scope, target, g}); err != nil {
			t.Fatal(err)
		}
	}
	set("runner", Target{}, Gates{Allow, Inherit, Inherit})
	set("ship", a.Ship(), Gates{Block, Inherit, Inherit})
	set("desk", a, Gates{Allow, Inherit, Inherit})
	if p.Effective(a).Commit.Source != "desk" || p.Effective(b).Commit != (Decision{Block, "ship"}) {
		t.Fatal("scope escaped", p)
	}
	if p.Check(a, p.Revision, "commit", "private") != nil || p.Check(a, p.Revision, "commit", "published") == nil || p.Check(a, p.Revision, "commit", "unknown") == nil {
		t.Fatal("effect gates")
	}
	set("desk", a, Gates{Allow, Allow, Inherit})
	if p.Check(a, p.Revision, "commit", "published") != nil || p.Check(a, p.Revision, "publish", "") != nil {
		t.Fatal("explicit permission ignored")
	}
	if p.Check(a, p.Revision-1, "commit", "private") == nil || p.Check(a, p.Revision, "arbitrary-poke", "private") == nil {
		t.Fatal("stale or unsupported effect")
	}
	set("desk", a, Inherited())
	if p.Effective(a).Commit.Source != "ship" {
		t.Fatal("inherit did not remove override")
	}
	set("global", Target{}, Gates{Allow, Allow, Block})
	if p.Effective(a).Commit.Value != Block {
		t.Fatal("global wiped specific block")
	}
}
func TestInvalidPoliciesFailClosed(t *testing.T) {
	for _, target := range []Target{{InstanceID: "a", Desk: "../base"}, {Desk: "app"}, {InstanceID: "a", ProjectID: "p", Generation: 2}, {InstanceID: "a", Creator: "c"}} {
		if target.Validate() == nil {
			t.Fatal(target)
		}
	}
	p := Default()
	p.Version = 99
	if p.Effective(Target{}).Commit.Value != Block || p.Check(Target{InstanceID: "a", Desk: "app"}, 0, "publish", "") == nil {
		t.Fatal("unknown format grants")
	}
	p = Default()
	if p.Apply(Update{0, "global", Target{}, Inherited()}) == nil {
		t.Fatal("global inherit accepted")
	}
	project := Target{InstanceID: "zod", ProjectID: "p", Generation: 1, Creator: "launch", Desk: "app"}
	if err := p.Apply(Update{0, "desk", project, Gates{Allow, Allow, Inherit}}); err != nil {
		t.Fatal(err)
	}
	project.Generation = 2
	project.Creator = "resume"
	if p.Effective(project).Commit.Value != Block {
		t.Fatal("permission carried to replacement")
	}
}

func TestCreateDeskIsShipScopedAndNewDesksInherit(t *testing.T) {
	p := Default()
	target := Target{InstanceID: "publisher", Desk: "new-app"}
	if p.Check(target, 0, "create-desk", "absent") == nil {
		t.Fatal("desk creation defaulted open")
	}
	ship := Gates{Commit: Inherit, Publish: Inherit, CreateDesk: Allow}
	if err := p.Apply(Update{p.Revision, "ship", target.Ship(), ship}); err != nil {
		t.Fatal(err)
	}
	if err := p.Check(target, p.Revision, "create-desk", "absent"); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"", "unknown", "private", "published"} {
		if p.Check(target, p.Revision, "create-desk", state) == nil {
			t.Fatal("creation without authoritative absence", state)
		}
	}
	if p.Check(target, p.Revision-1, "create-desk", "absent") == nil || p.Check(Target{InstanceID: "sibling", Desk: "new-app"}, p.Revision, "create-desk", "absent") == nil {
		t.Fatal("creation grant escaped revision or ship")
	}
	if p.Effective(target).Commit != (Decision{Block, "global"}) || p.Check(target, p.Revision, "commit", "private") == nil || p.Check(target, p.Revision, "publish", "") == nil {
		t.Fatal("creating a desk granted commit/publication")
	}
	if p.Apply(Update{p.Revision, "desk", target, Gates{Allow, Allow, Allow}}) == nil {
		t.Fatal("desk can grant sibling creation")
	}
	if err := p.Apply(Update{p.Revision, "ship", target.Ship(), Gates{Allow, Block, Allow}}); err != nil {
		t.Fatal(err)
	}
	if p.Effective(target).Commit != (Decision{Allow, "ship"}) || p.Effective(target).Publish != (Decision{Block, "ship"}) {
		t.Fatal("new desk does not inherit ship permissions")
	}
}

func TestLegacyPolicyUpgradePreservesChoicesAndClosesCreation(t *testing.T) {
	p := Policy{Version: 1, Revision: 7, Global: Gates{Commit: Allow, Publish: Block}, Runner: Gates{Commit: Inherit, Publish: Allow}, Entries: []Entry{
		{Target{InstanceID: "a"}, Gates{Commit: Block, Publish: Inherit}},
		{Target{InstanceID: "a", Desk: "base"}, Gates{Commit: Allow, Publish: Inherit}},
	}}
	upgraded, err := UpgradeLegacy(p)
	if err != nil || upgraded.Version != Version || upgraded.Revision != 7 {
		t.Fatal(upgraded, err)
	}
	e := upgraded.Effective(Target{InstanceID: "a", Desk: "base"})
	if e.Commit != (Decision{Allow, "desk"}) || e.Publish != (Decision{Allow, "runner"}) || e.CreateDesk != (Decision{Block, "global"}) {
		t.Fatal(e)
	}
	p.Entries[0].Gates.CreateDesk = Allow
	if _, err := UpgradeLegacy(p); err == nil {
		t.Fatal("legacy creation grant accepted")
	}
}
