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
	set("runner", Target{}, Gates{Allow, Inherit})
	set("ship", a.Ship(), Gates{Block, Inherit})
	set("desk", a, Gates{Allow, Inherit})
	if p.Effective(a).Commit.Source != "desk" || p.Effective(b).Commit != (Decision{Block, "ship"}) {
		t.Fatal("scope escaped", p)
	}
	if p.Check(a, p.Revision, "commit", "private") != nil || p.Check(a, p.Revision, "commit", "published") == nil || p.Check(a, p.Revision, "commit", "unknown") == nil {
		t.Fatal("effect gates")
	}
	set("desk", a, Gates{Allow, Allow})
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
	set("global", Target{}, Gates{Allow, Allow})
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
	p.Version = 2
	if p.Effective(Target{}).Commit.Value != Block || p.Check(Target{InstanceID: "a", Desk: "app"}, 0, "publish", "") == nil {
		t.Fatal("unknown format grants")
	}
	p = Default()
	if p.Apply(Update{0, "global", Target{}, Inherited()}) == nil {
		t.Fatal("global inherit accepted")
	}
	project := Target{InstanceID: "zod", ProjectID: "p", Generation: 1, Creator: "launch", Desk: "app"}
	if err := p.Apply(Update{0, "desk", project, Gates{Allow, Allow}}); err != nil {
		t.Fatal(err)
	}
	project.Generation = 2
	project.Creator = "resume"
	if p.Effective(project).Commit.Value != Block {
		t.Fatal("permission carried to replacement")
	}
}
