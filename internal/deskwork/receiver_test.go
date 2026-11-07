package deskwork

import (
	"strings"
	"testing"
)

func TestReceiverInstallSelectsPublisherAndRefusesReplacement(t *testing.T) {
	p := Publisher{Ship: "~zod", Desk: "example", ExpectedHash: "0v123"}
	q, err := InstallRemote("~nec", "received", p)
	if err != nil || q.app != "hood" || q.mark != "kiln-install" || !strings.Contains(q.expression, "[%received ~zod %example]") || !strings.Contains(q.expression, "!(~(has by pikes) %received)") || !strings.Contains(q.expression, "%cd /[ego]//[wen])) %received)") {
		t.Fatal(q, err)
	}
	for _, bad := range []Publisher{{"~zod\n!!", "example", "0v123"}, {"~zod", "../base", "0v123"}, {"~zod", "example", "0v';"}, {"~nec", "example", "0v123"}} {
		if _, err := InstallRemote("~nec", "received", bad); err == nil {
			t.Fatal("unsafe publisher accepted", bad)
		}
	}
	if _, err := InstallRemote("~nec", "../outside", p); err == nil {
		t.Fatal("unsafe target")
	}
	r, err := SubscriptionInventory("~nec", "received")
	if err != nil || r.app != "" || r.mark != "" {
		t.Fatal("subscription read has effect", r, err)
	}
	r, err = ReceiverSourceReady("~nec", "received")
	if err != nil || r.app != "" || r.mark != "" || !strings.Contains(r.expression, "?:(present (gth let:") {
		t.Fatal("pending receiver must have a read-only source test", r, err)
	}
	if _, err := ReceiverSourceReady("~nec", "../base"); err == nil {
		t.Fatal("unsafe pending source selection")
	}
}
func TestReceiverVerificationRequiresExactSourceAndSeparatesHealth(t *testing.T) {
	p := Publisher{Ship: "~zod", Desk: "example", ExpectedHash: "0v123"}
	o := Observation{Exists: true, Revision: 4, ClayHash: "0v123", Publication: "private"}
	a := AppState{Zest: "live", SourceRevision: 4, SourceShip: "~zod", SourceDesk: "example", ExpectedAgents: []string{"demo"}, Agents: []AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	if err := VerifyReceiver(p, o, a); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Observation, *AppState){func(o *Observation, a *AppState) { o.ClayHash = "0v456" }, func(o *Observation, a *AppState) { a.SourceShip = "~bud" }, func(o *Observation, a *AppState) { a.SourceDesk = "other" }, func(o *Observation, a *AppState) { a.SourceRevision++ }, func(o *Observation, a *AppState) { a.Zest = "held" }} {
		b, c := o, a
		mutate(&b, &c)
		if VerifyReceiver(p, b, c) == nil {
			t.Fatal("unverified receiver accepted", b, c)
		}
	}
	if (Subscription{Present: false, Ship: "~zod"}).Validate() == nil {
		t.Fatal("contradictory absence")
	}
	if (Subscription{Present: true, Ship: "~zod", Desk: "example", Zest: "held", Hash: "0v0"}).Validate() != nil {
		t.Fatal("pending subscription cannot be observed")
	}
}
