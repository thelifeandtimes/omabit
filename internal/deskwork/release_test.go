package deskwork

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReleaseRequestsBindReviewedRevisionAndRejectInjection(t *testing.T) {
	baseline := Observation{Exists: true, Revision: 1234, ClayHash: "0v123", Publication: "private"}
	constructors := []func(string, string, Observation) (LensRequest, error){PublishSourceAt, PublishTreatyAt, ActivateLocal}
	for _, makeRequest := range constructors {
		for _, name := range []string{"../base", "base\n!!", "%base", "base'", "", "hello_world"} {
			if _, err := makeRequest("~zod", name, baseline); err == nil {
				t.Fatal("unsafe desk", name)
			}
		}
		if _, err := makeRequest("~zod", "example", Observation{}); err == nil {
			t.Fatal("absent release accepted")
		}
		q, err := makeRequest("~zod", "example", baseline)
		if err != nil {
			t.Fatal(err)
		}
		body, err := q.Body()
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if json.Unmarshal(body, &value) != nil {
			t.Fatal("invalid typed envelope")
		}
		if !strings.Contains(q.expression, "1.234") || !strings.Contains(q.expression, "0v123") || q.mark == "" || q.input != nil {
			t.Fatal("unbound mutation", q)
		}
	}
	for _, bad := range []string{"~zod/", "~zod\n!!", "arbitrary"} {
		if _, err := AppInventory(bad, "base"); err == nil {
			t.Fatal("unsafe app ship")
		}
	}
	q, err := AppInventory("~zod", "example")
	if err != nil {
		t.Fatal(err)
	}
	if q.app != "" || q.mark != "" {
		t.Fatal("inventory is an effect")
	}
	for _, guard := range []string{"%gu /[ego]/treaty", "%gu /[ego]/docket", "sovereign", "frontend_docket_hash"} {
		if !strings.Contains(q.expression, guard) {
			t.Fatal("missing stage evidence", guard)
		}
	}
}
func TestAppStateRejectsContradictoryEvidence(t *testing.T) {
	valid := AppState{Zest: "dead", SourceRevision: 1, ExpectedAgents: []string{}, Agents: []AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*AppState){
		func(s *AppState) { s.TreatyListed = true }, func(s *AppState) { s.SourceShip = "~zod" }, func(s *AppState) { s.DocketAvailable = true },
		func(s *AppState) { s.FrontendState = "glob" }, func(s *AppState) { s.FrontendMode = "glob-http" },
		func(s *AppState) { s.Agents = []AppAgent{{"example", true}, {"example", false}} }, func(s *AppState) { s.ExpectedAgents = []string{"../bad"} },
	} {
		s := valid
		mutate(&s)
		if s.Validate() == nil {
			t.Fatal("contradictory evidence accepted", s)
		}
	}
	current := valid
	current.Zest = "live"
	current.DocketAvailable = true
	current.DocketHash = "0v123"
	current.FrontendDocketHash = "0v456"
	current.FrontendMode = "glob-ames"
	current.FrontendState = "glob"
	if err := current.Validate(); err != nil {
		t.Fatal("different current/loaded dockets are reportable, not ready", err)
	}
}
