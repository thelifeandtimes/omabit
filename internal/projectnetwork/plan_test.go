package projectnetwork

import (
	"reflect"
	"testing"
)

func pair() Request {
	return Request{Label: "Local app development", ConfirmSharedAdministration: true, Members: []MemberRequest{{"receiver", "~nec", 18282}, {"sender", "~zod", 18281}}}
}

func TestPlanIsDeterministicAndDoesNotMutateInput(t *testing.T) {
	req := pair()
	original := pair()
	a, err := Build(req, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(req, original) {
		t.Fatal("planning reordered request input")
	}
	req.Members[0], req.Members[1] = req.Members[1], req.Members[0]
	b, err := Build(req, Reservations{})
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("input order changed the complete plan", err)
	}
	da, err := a.Digest()
	if err != nil {
		t.Fatal(err)
	}
	db, err := b.Digest()
	if err != nil || da != db {
		t.Fatal("review digest changed", err)
	}
	if a.BindAddress != "127.0.0.1" || a.Members[0].HTTPPort != 18282 || a.Members[1].HTTPPort != 18281 || a.Members[0].InternalHTTPPort == a.Members[1].InternalHTTPPort {
		t.Fatal("public and internal endpoint bindings confused", a)
	}
	req.Members[0].HTTPPort = 18283
	changed, err := Build(req, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	dc, _ := changed.Digest()
	if dc == da {
		t.Fatal("changed host publication reused review digest")
	}
}

func TestInvalidAndConflictingMembersDoNotProducePartialPlans(t *testing.T) {
	cases := map[string]func(*Request){
		"unacknowledged trust": func(r *Request) { r.ConfirmSharedAdministration = false },
		"empty label":          func(r *Request) { r.Label = "" },
		"control label":        func(r *Request) { r.Label = "local\nproduction" },
		"duplicate instance":   func(r *Request) { r.Members[1].InstanceID = r.Members[0].InstanceID },
		"path instance":        func(r *Request) { r.Members[0].InstanceID = "../other" },
		"duplicate ship":       func(r *Request) { r.Members[1].Ship = r.Members[0].Ship },
		"noncanonical ship":    func(r *Request) { r.Members[0].Ship = "nec" },
		"fake planet":          func(r *Request) { r.Members[0].Ship = "~sampel-palnet" },
		"invalid galaxy":       func(r *Request) { r.Members[0].Ship = "~aaa" },
		"duplicate host port":  func(r *Request) { r.Members[1].HTTPPort = r.Members[0].HTTPPort },
		"privileged host port": func(r *Request) { r.Members[0].HTTPPort = 80 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := pair()
			mutate(&req)
			got, err := Build(req, Reservations{})
			if err == nil || !reflect.DeepEqual(got, Plan{}) {
				t.Fatal("invalid input returned usable plan", got, err)
			}
		})
	}
	for name, reserved := range map[string]Reservations{"existing instance": {Instances: map[string]bool{"receiver": true}}, "configured or retained port": {HTTPPorts: map[uint16]bool{18282: true}}} {
		t.Run(name, func(t *testing.T) {
			got, err := Build(pair(), reserved)
			if err == nil || !reflect.DeepEqual(got, Plan{}) {
				t.Fatal("reservation conflict admitted", got, err)
			}
		})
	}
}

func TestSeparateProjectsMayRepeatFakeIdentityButNotHostReservations(t *testing.T) {
	a, err := Build(pair(), Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	req := pair()
	req.Members[0].InstanceID = "other-receiver"
	req.Members[1].InstanceID = "other-sender"
	reserved := Reservations{HTTPPorts: map[uint16]bool{18281: true, 18282: true}, Instances: map[string]bool{"receiver": true, "sender": true}}
	if _, err := Build(req, reserved); err == nil {
		t.Fatal("second project reused host ports")
	}
	req.Members[0].HTTPPort = 18284
	req.Members[1].HTTPPort = 18283
	b, err := Build(req, reserved)
	if err != nil {
		t.Fatal("duplicate identity in distinct project rejected", err)
	}
	if b.Members[0].Ship != a.Members[0].Ship || b.Members[0].InternalHTTPPort != a.Members[0].InternalHTTPPort {
		t.Fatal("distinct namespaces unnecessarily changed their internal identities")
	}
}

func TestDigestRefusesUnreviewedOrAlteredContract(t *testing.T) {
	for _, mutate := range []func(*Plan){func(p *Plan) { p.BindAddress = "0.0.0.0" }, func(p *Plan) { p.Mode = "live" }, func(p *Plan) { p.Members[0].InternalHTTPPort = 12321 }, func(p *Plan) { p.Members[0], p.Members[1] = p.Members[1], p.Members[0] }, func(p *Plan) { p.SharedAdministration = false }} {
		p, err := Build(pair(), Reservations{})
		if err != nil {
			t.Fatal(err)
		}
		mutate(&p)
		if _, err := p.Digest(); err == nil {
			t.Fatal("altered plan received a review digest", p)
		}
	}
}

func TestEmptyProjectHasAStablePlanWithoutPortClaims(t *testing.T) {
	p, err := Build(Request{Label: "empty", ConfirmSharedAdministration: true}, Reservations{})
	if err != nil || len(p.Members) != 0 {
		t.Fatal(p, err)
	}
	if d, err := p.Digest(); err != nil || len(d) != 64 {
		t.Fatal(d, err)
	}
}
