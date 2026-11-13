package deskwork

import (
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func receiverReviewFixture(t *testing.T) (ReceiverReview, ReceiverIntent) {
	t.Helper()
	target := apppolicy.Target{InstanceID: "receiver", Desk: "example"}
	review := ReceiverReview{Version: 1, Publisher: Publisher{Ship: "~zod", Desk: "demo", ExpectedHash: "0v123"}, ShipSetting: DefaultSetting(target.Ship()), DeskSetting: DefaultSetting(target), Continuity: strings.Repeat("c", 64)}
	if err := review.Seal(); err != nil {
		t.Fatal(err)
	}
	intent := ReceiverIntent{Intent: Intent{Version: 1, ID: strings.Repeat("a", 64), Target: target, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1, Action: "install-remote", SourceDigest: review.Publisher.Digest()}, Publisher: review.Publisher, Continuity: review.Continuity, ReviewDigest: review.Digest}
	return review, intent
}

func TestReceiverReviewBindsPublisherAndRejectsCrossProjectInstallation(t *testing.T) {
	review, intent := receiverReviewFixture(t)
	if err := intent.CheckReview(review); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*ReceiverReview){
		func(r *ReceiverReview) { r.Publisher.ExpectedHash = "0v456" },
		func(r *ReceiverReview) { r.Publisher.Ship = "~bud" },
		func(r *ReceiverReview) { r.Continuity = strings.Repeat("d", 64) },
		func(r *ReceiverReview) { r.DeskSetting.Desk.Instructions = "changed instructions" },
	} {
		r := review
		r.DeskSetting = DefaultSetting(review.DeskSetting.Target)
		change(&r)
		if err := r.Seal(); err != nil {
			t.Fatal(err)
		}
		if intent.CheckReview(r) == nil {
			t.Fatal("changed review accepted")
		}
	}
	project := apppolicy.Target{InstanceID: "receiver", ProjectID: "network", Generation: 2, Creator: "member-create"}
	review.ShipSetting = DefaultSetting(project)
	project.Desk = "example"
	review.DeskSetting = DefaultSetting(project)
	if review.Seal() == nil {
		t.Fatal("project admitted external publisher")
	}
	review.PublisherTarget = apppolicy.Target{InstanceID: "publisher", ProjectID: "different", Generation: 1, Creator: "created"}
	if review.Seal() == nil {
		t.Fatal("cross-project publisher accepted")
	}
	review.PublisherTarget.ProjectID = "network"
	if err := review.Seal(); err != nil {
		t.Fatal(err)
	}
	review.PublisherTarget.InstanceID = "receiver"
	if review.Seal() == nil {
		t.Fatal("self publisher accepted")
	}
}

func TestReceiverReceiptVerifiesFrozenPublisherWithoutClaimingHealth(t *testing.T) {
	_, i := receiverReviewFixture(t)
	now := time.Now().UTC()
	app := AppState{Zest: "live", SourceRevision: 1, SourceShip: "~zod", SourceDesk: "demo", ExpectedAgents: []string{"example"}, Agents: []AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	observed := Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
	r := Receipt{Version: 1, Intent: i.Intent, Actor: "fixture", Steps: []Step{
		{Kind: "install-remote", State: "planned", At: now}, {Kind: "install-remote", State: "dispatched", At: now},
		{Kind: "install-remote", State: "acknowledged", At: now}, {Kind: "install-remote", State: "observed", At: now, Observation: &observed, Application: &app},
	}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Observation, *AppState){
		func(o *Observation, a *AppState) { o.ClayHash = "0v456" },
		func(o *Observation, a *AppState) { a.SourceDesk = "other" },
		func(o *Observation, a *AppState) { a.SourceShip = "~bud" },
		func(o *Observation, a *AppState) { o.Publication = "published" },
	} {
		o, a := observed, app
		change(&o, &a)
		r.Steps[3].Observation, r.Steps[3].Application = &o, &a
		if r.Validate() == nil {
			t.Fatal("unreviewed source accepted")
		}
	}
	r.Steps = r.Steps[:3]
	recovered, err := r.Recover(now)
	if err != nil || recovered.Steps[len(recovered.Steps)-1].State != "uncertain" {
		t.Fatal("restart authorized retry", recovered, err)
	}
}

func TestReceiverInstallRequiresBothCreationAndCommitAtCurrentRevision(t *testing.T) {
	_, i := receiverReviewFixture(t)
	for _, gates := range []apppolicy.Gates{
		apppolicy.Blocked(), {CreateDesk: apppolicy.Allow, Commit: apppolicy.Block, Publish: apppolicy.Allow},
		{CreateDesk: apppolicy.Block, Commit: apppolicy.Allow, Publish: apppolicy.Allow},
		{CreateDesk: apppolicy.Allow, Commit: apppolicy.Allow, Publish: apppolicy.Block},
	} {
		p := apppolicy.Default()
		p.Global = gates
		err := i.Intent.Check(p, Observation{})
		want := gates.CreateDesk == apppolicy.Allow && gates.Commit == apppolicy.Allow
		if (err == nil) != want {
			t.Fatal(gates, err)
		}
		p.Revision++
		if i.Intent.Check(p, Observation{}) == nil {
			t.Fatal("stale policy accepted")
		}
	}
}
