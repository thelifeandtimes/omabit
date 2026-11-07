//go:build linux && amd64

package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"strings"
	"testing"
)

func reviewAdoptionFixture(t *testing.T, r *Server, stage deskwork.StageIntent) protocol.DeskAdoptionReview {
	t.Helper()
	q := protocol.DeskDeploymentRequest{Selection: deskSelection(deskIntentFixture()), StageID: stage.ID, SettingsRevision: stage.SettingsRevision}
	got := deskHTTPRequest(t, r, "POST", protocol.DeskAdoptionReviewPath, q)
	var out protocol.DeskAdoptionReview
	if got.Code != 200 || json.Unmarshal(got.Body.Bytes(), &out) != nil || out.Validate() != nil {
		t.Fatal(got.Code, got.Body.String())
	}
	return out
}
func adoptionIntentFromReview(v protocol.DeskAdoptionReview) deskwork.AdoptionIntent {
	return deskwork.AdoptionIntent{Intent: deskwork.Intent{Version: 1, ID: strings.Repeat("a", 64), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "adopt-source", Baseline: v.Observation, SourceDigest: v.SourceDigest}, StageID: v.Request.StageID, SettingsRevision: v.Request.SettingsRevision, SourceName: v.SourceName, Continuity: v.Continuity, PlanDigest: v.PlanDigest}
}
func TestDeskAdoptionAPIRequiresGateAndNeverWritesClay(t *testing.T) {
	for _, scenario := range []string{"closed", "allowed", "published-blocked", "published-allowed"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, stage := commitServerFixture(t)
			f.files["hello.txt"] = canonicalCommitFixture("hello.txt", []byte("hello\n"))
			if scenario != "closed" {
				gates := apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}
				if scenario == "published-allowed" {
					gates.Publish = apppolicy.Allow
				}
				_, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: gates})
				g02Check(t, err)
			}
			if strings.HasPrefix(scenario, "published-") {
				f.observation.Publication = "published"
			}
			v := reviewAdoptionFixture(t, r, stage)
			if v.PathCount != 1 || v.NewCount != 1 || v.ConflictCount != 0 {
				t.Fatal(v)
			}
			i := adoptionIntentFromReview(v)
			got := deskHTTPRequest(t, r, "POST", protocol.DeskAdoptionPath, i)
			if got.Code != 200 {
				t.Fatal(got.Code, got.Body.String())
			}
			receipt, err := loadDeskOperation(r.s.db, i.Intent.ID)
			g02Check(t, err)
			allowed := scenario == "allowed" || scenario == "published-allowed"
			if (receipt.Steps[1].State == "observed") != allowed || f.calls != 0 || f.observation != i.Intent.Baseline {
				t.Fatal("gate or ship mutation", receipt, f.calls)
			}
			owned, err := r.s.sourceOwnership(stage.Target, "example", f.continuity)
			g02Check(t, err)
			if (len(owned.Raw.Files) == 1) != allowed {
				t.Fatal(owned)
			}
			if allowed {
				again := reviewAdoptionFixture(t, r, stage)
				if again.NewCount+again.RefreshCount != 0 {
					t.Fatal("identical adoption not no-op")
				}
			}
			// Exact replay remains available after both source and policy change.
			f.observation.Revision++
			f.observation.ClayHash = "0v999"
			p, err := loadAppPolicy(r.s.db)
			g02Check(t, err)
			_, err = r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, ExpectedRevision: p.Revision, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
			g02Check(t, err)
			replay := deskHTTPRequest(t, r, "POST", protocol.DeskAdoptionPath, i)
			if replay.Code != 200 || f.calls != 0 {
				t.Fatal("replay changed ownership or ship", replay.Code)
			}
		})
	}
}
func TestDeskAdoptionAPIRefusesStaleDifferentMissingSource(t *testing.T) {
	for _, scenario := range []string{"different", "missing", "pier", "clay", "policy", "plan", "unresolved"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, stage := commitServerFixture(t)
			f.files["hello.txt"] = canonicalCommitFixture("hello.txt", []byte("hello\n"))
			v := reviewAdoptionFixture(t, r, stage)
			i := adoptionIntentFromReview(v)
			switch scenario {
			case "different":
				f.files["hello.txt"] = canonicalCommitFixture("hello.txt", []byte("external"))
			case "missing":
				delete(f.files, "hello.txt")
			case "pier":
				f.continuity = strings.Repeat("1", 64)
			case "clay":
				f.observation.Revision++
			case "policy":
				_, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
				g02Check(t, err)
			case "plan":
				i.PlanDigest = strings.Repeat("1", 64)
			case "unresolved":
				pending := deploymentRecordFixture(t)
				pending.Intent.Intent.ID = strings.Repeat("9", 64)
				_, _, err := r.s.createDeskDeployment(pending)
				g02Check(t, err)
			}
			got := deskHTTPRequest(t, r, "POST", protocol.DeskAdoptionPath, i)
			if got.Code != 409 || f.calls != 0 {
				t.Fatal(scenario, got.Code, got.Body.String(), f.calls)
			}
			var n int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM desk_adoptions").Scan(&n))
			if n != 0 {
				t.Fatal("changed review persisted adoption")
			}
		})
	}
}
