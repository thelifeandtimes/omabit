//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type applicationBackendFixture struct {
	*deskBackendFixture
	variant string
}
type applicationRuntimeFixture struct {
	deskRuntimeFixture
	variant string
}

func (b *applicationBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return applicationRuntimeFixture{deskRuntimeFixture{b.deskBackendFixture}, b.variant}, nil
}
func (d applicationRuntimeFixture) Application(context.Context, string) (deskwork.AppState, error) {
	s := deskwork.AppState{Zest: "live", SourceRevision: 1, ExpectedAgents: []string{"example"}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	switch d.variant {
	case "unavailable":
		return s, errors.New("unavailable")
	case "invalid":
		s.TreatyListed = true
	case "different-revision":
		s.SourceRevision++
	case "changed-during-read":
		d.b.scenario = "absent"
	}
	return s, nil
}
func TestDeskApplicationObservationsAreSeparateBoundedReadOnlyFacts(t *testing.T) {
	for _, variant := range []string{"ok", "unavailable", "invalid", "different-revision", "changed-during-read", "absent", "stale-runtime", "observer"} {
		t.Run(variant, func(t *testing.T) {
			b := &deskBackendFixture{scenario: "desk-exists"}
			r, i := deskServerFixture(t, b)
			r.b = &applicationBackendFixture{b, variant}
			if variant == "absent" {
				b.scenario = "absent"
			}
			selection := deskSelection(i)
			if variant == "stale-runtime" {
				selection.ConfigRevision++
			}
			var out *httptest.ResponseRecorder
			if variant == "observer" {
				body, _ := json.Marshal(selection)
				out = httptest.NewRecorder()
				r.serveDeskApplication(out, httptest.NewRequest("POST", protocol.DeskApplicationPath, bytes.NewReader(body)), "observer")
			} else {
				out = deskHTTPRequest(t, r, "POST", protocol.DeskApplicationPath, selection)
			}
			if variant == "ok" {
				if out.Code != 200 {
					t.Fatal(out.Code, out.Body.String())
				}
				var got protocol.DeskApplication
				g02Check(t, json.Unmarshal(out.Body.Bytes(), &got))
				g02Check(t, got.Validate())
				if len(got.Application.Agents) != 0 || len(got.Application.ExpectedAgents) != 1 || got.Application.FrontendState != "unavailable" {
					t.Fatal("invented readiness", got)
				}
			} else if out.Code == 200 {
				t.Fatal("invalid observation accepted", out.Body.String())
			}
			p, err := loadAppPolicy(r.s.db)
			g02Check(t, err)
			if p.Global != apppolicy.Blocked() || p.Revision != 0 || b.calls.Load() != 0 {
				t.Fatal("observation changed authority")
			}
			ids, err := deskOperationIDs(r.s.db)
			g02Check(t, err)
			if len(ids) != 0 {
				t.Fatal("observation queued an effect")
			}
		})
	}
}
