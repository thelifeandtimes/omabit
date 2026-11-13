package deskwork

import (
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func releaseReviewFixture(t *testing.T, action string) ReleaseReview {
	t.Helper()
	i := intentFixture()
	desk := DefaultSetting(i.Target)
	desk.Desk.Publication = "source"
	if action == "publish-treaty" {
		desk.Desk.Publication = "treaty"
	}
	app := AppState{Zest: "dead", SourceRevision: 1, ExpectedAgents: []string{"example"}, Agents: []AppAgent{}, TreatyAvailable: true, DocketAvailable: true, DocketHash: "0v456", FrontendMode: "site", FrontendState: "absent"}
	r := ReleaseReview{Version: 1, Action: action, Observation: i.Baseline, Application: app, ShipSetting: DefaultSetting(i.Target.Ship()), DeskSetting: desk, Continuity: strings.Repeat("c", 64)}
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestReleaseReviewBindsWorkflowAndExactCheckEvidence(t *testing.T) {
	r := releaseReviewFixture(t, "publish-treaty")
	r.DeskSetting.Desk.Checks = []string{"Application upgrade test", "Receiver smoke test"}
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	checks := []ReleaseCheck{{Name: r.DeskSetting.Desk.Checks[0], ClayHash: r.Observation.ClayHash, Passed: true, Evidence: "fixture report 1"}, {Name: r.DeskSetting.Desk.Checks[1], ClayHash: r.Observation.ClayHash, Passed: true, Evidence: "fixture report 2"}}
	i := intentFixture()
	i.Action = r.Action
	i.SourceDigest = ""
	intent := ReleaseIntent{Intent: i, Continuity: r.Continuity, ReviewDigest: r.Digest, ChecksDigest: ReleaseChecksDigest(checks)}
	record := ReleaseRecord{Version: 1, Intent: intent, Review: r, Checks: checks, Actor: "fixture", CreatedAt: time.Now().UTC()}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*ReleaseRecord){
		func(v *ReleaseRecord) { v.Review.Observation.Revision++ },
		func(v *ReleaseRecord) { v.Review.Continuity = strings.Repeat("d", 64) },
		func(v *ReleaseRecord) { v.Review.ShipSetting = DefaultSetting(apppolicy.Target{InstanceID: "other"}) },
		func(v *ReleaseRecord) { v.Intent.ShipSettingsRevision++ },
		func(v *ReleaseRecord) { v.Checks = nil },
		func(v *ReleaseRecord) { v.Checks[0].Passed = false },
		func(v *ReleaseRecord) { v.Checks[0].ClayHash = "0v999" },
		func(v *ReleaseRecord) { v.Checks[0].Name = "replacement test" },
		func(v *ReleaseRecord) { v.Checks[0].Evidence = "" },
	} {
		v := record
		v.Checks = append([]ReleaseCheck{}, checks...)
		change(&v)
		if v.Validate() == nil {
			t.Fatal("changed release evidence accepted", v)
		}
	}
	for _, bad := range []string{"none", "source"} {
		r := releaseReviewFixture(t, "publish-treaty")
		r.DeskSetting.Desk.Publication = bad
		if err := r.Seal(); err != nil {
			t.Fatal(err)
		}
		if r.CheckConfiguredRelease() == nil {
			t.Fatal("changed publication settings accepted")
		}
	}
	activation := releaseReviewFixture(t, "activate-local")
	activation.DeskSetting.Desk.Checks = []string{"Test running app before publishing"}
	if err := activation.Seal(); err != nil {
		t.Fatal(err)
	}
	if err := CheckReleaseEvidence(activation, []ReleaseCheck{}); err != nil {
		t.Fatal("activation blocked on checks that require it", err)
	}
	activation.Application.SourceShip = "~nec"
	activation.Application.SourceDesk = "source"
	if err := activation.Seal(); err != nil {
		t.Fatal(err)
	}
	if activation.CheckConfiguredRelease() == nil {
		t.Fatal("remote subscription replacement allowed")
	}
}

func TestReleaseReceiptsRequireExactStageAndNeverReplay(t *testing.T) {
	for _, action := range []string{"activate-local", "publish-source", "publish-treaty"} {
		t.Run(action, func(t *testing.T) {
			r := releaseReviewFixture(t, action)
			i := intentFixture()
			i.Action = action
			i.SourceDigest = ""
			p := apppolicy.Default()
			if i.Check(p, i.Baseline) == nil {
				t.Fatal("release bypassed closed gates")
			}
			p.Global.Commit = apppolicy.Allow
			if action != "activate-local" && i.Check(p, i.Baseline) == nil {
				t.Fatal("commit enabled publication")
			}
			p.Global.Publish = apppolicy.Allow
			if err := i.Check(p, i.Baseline); err != nil {
				t.Fatal(err)
			}
			if action == "activate-local" {
				i.Baseline.Publication = "published"
				p.Global.Publish = apppolicy.Block
				if i.Check(p, i.Baseline) == nil {
					t.Fatal("public activation bypassed publish gate")
				}
				i.Baseline = r.Observation
			}
			after := r.Observation
			app := r.Application
			app.Zest = "live"
			app.Agents = []AppAgent{{Name: "example", Live: true}}
			if action != "activate-local" {
				after.Publication = "published"
			}
			if action == "publish-treaty" {
				app.TreatyListed = true
				app.TreatyHash = after.ClayHash
			}
			at := time.Now().UTC()
			receipt := Receipt{Version: 1, Intent: i, Actor: "fixture", Steps: []Step{{Kind: action, State: "planned", At: at}, {Kind: action, State: "dispatched", At: at}}}
			uncertain, err := receipt.Recover(at)
			if err != nil {
				t.Fatal(err)
			}
			observed := uncertain
			observed.Steps = append(append([]Step{}, uncertain.Steps...), Step{Kind: action, State: "observed", At: at, Observation: &after, Application: &app})
			if err = observed.Validate(); err != nil {
				t.Fatal(err)
			}
			if next, err := uncertain.Recover(at); err != nil || len(next.Steps) != len(uncertain.Steps) {
				t.Fatal("recovery repeated an effect", err)
			}
			after.ClayHash = "0v999"
			if observed.Validate() == nil {
				t.Fatal("another release source accepted")
			}
			after = r.Observation
			if action != "activate-local" {
				after.Publication = "published"
			}
			if action == "publish-treaty" {
				app.TreatyHash = "0v456"
				if observed.Validate() == nil {
					t.Fatal("another Treaty revision accepted")
				}
			}
			if action == "activate-local" {
				app.Agents = []AppAgent{}
				if observed.Validate() == nil {
					t.Fatal("missing declared agent accepted")
				}
			}
		})
	}
}
