package deskwork

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"strings"
	"testing"
	"time"
)

func intentFixture() Intent {
	return Intent{Version: 1, ID: strings.Repeat("1", 64), Target: apppolicy.Target{InstanceID: "publisher", Desk: "example"}, RuntimeID: strings.Repeat("2", 64), ConfigRevision: 1, Action: "commit", Baseline: Observation{true, 1, "0v123", "private"}, SourceDigest: strings.Repeat("3", 64)}
}
func TestPolicyIsFreshAndPublicationRequiresBothGates(t *testing.T) {
	i := intentFixture()
	p := apppolicy.Default()
	if i.Check(p, i.Baseline) == nil {
		t.Fatal("default gate opened")
	}
	p.Global.Commit = apppolicy.Allow
	if e := i.Check(p, i.Baseline); e != nil {
		t.Fatal(e)
	}
	changed := i.Baseline
	changed.Revision++
	if i.Check(p, changed) == nil {
		t.Fatal("changed desk accepted")
	}
	p.Revision++
	if i.Check(p, i.Baseline) == nil {
		t.Fatal("stale policy accepted")
	}
	i.PolicyRevision = p.Revision
	i.Baseline.Publication = "published"
	if i.Check(p, i.Baseline) == nil {
		t.Fatal("published commit bypassed publication gate")
	}
	p.Global.Publish = apppolicy.Allow
	if e := i.Check(p, i.Baseline); e != nil {
		t.Fatal(e)
	}
	p.Global.Commit = apppolicy.Block
	if i.Check(p, i.Baseline) == nil {
		t.Fatal("publish allowed commit")
	}
	i.Action = "create-desk"
	i.Baseline = Observation{}
	i.SourceDigest = ""
	if i.Check(p, i.Baseline) == nil {
		t.Fatal("creation gate bypassed")
	}
	p.Global.CreateDesk = apppolicy.Allow
	if e := i.Check(p, i.Baseline); e != nil {
		t.Fatal(e)
	}
	i.Target.ProjectID = "network"
	if i.Validate() == nil {
		t.Fatal("unresolved project accepted")
	}
}
func TestUnknownEffectsAreNeverReplayedOrAssumedComplete(t *testing.T) {
	at := time.Now().UTC()
	i := intentFixture()
	r := Receipt{1, i, "desktop", []Step{{Kind: i.Action, State: "planned", At: at}}}
	interrupted, e := r.Recover(at)
	if e != nil || interrupted.Steps[1].State != "interrupted" || len(r.Steps) != 1 {
		t.Fatal(interrupted, e)
	}
	r.Steps = append(r.Steps, Step{Kind: i.Action, State: "dispatched", At: at})
	uncertain, e := r.Recover(at)
	if e != nil || uncertain.Steps[2].State != "uncertain" {
		t.Fatal(uncertain, e)
	}
	again, e := uncertain.Recover(at)
	if e != nil || len(again.Steps) != 3 {
		t.Fatal(again, e)
	}
	replay := again
	replay.Steps = append(append([]Step{}, again.Steps...), Step{Kind: i.Action, State: "dispatched", At: at})
	if replay.Validate() == nil {
		t.Fatal("uncertain effect replay accepted")
	}
	observed := Observation{true, 2, "0v234", "private"}
	done := again
	done.Steps = append(append([]Step{}, again.Steps...), Step{Kind: i.Action, State: "observed", At: at, Observation: &observed})
	if done.Validate() == nil {
		t.Fatal("unrelated revision mistaken for effect")
	}
	done.Steps[len(done.Steps)-1].VerifiedSourceDigest = i.SourceDigest
	if e = done.Validate(); e != nil {
		t.Fatal(e)
	}
	observed.Revision = 1
	if done.Validate() == nil {
		t.Fatal("unchanged revision claimed committed")
	}
}
