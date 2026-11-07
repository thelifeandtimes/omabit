//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type releaseBackendFixture struct {
	*deskBackendFixture
	observation deskwork.Observation
	app         deskwork.AppState
	identity    string
	opens       atomic.Int32
	onWorker    func()
	lost        bool
}
type releaseRuntimeFixture struct{ b *releaseBackendFixture }

func (b *releaseBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	if b.opens.Add(1) == 3 && b.onWorker != nil {
		b.onWorker()
	}
	return releaseRuntimeFixture{b}, nil
}
func (d releaseRuntimeFixture) Observe(context.Context, string) (deskwork.Observation, error) {
	return d.b.observation, nil
}
func (d releaseRuntimeFixture) Application(context.Context, string) (deskwork.AppState, error) {
	return d.b.app, nil
}
func (d releaseRuntimeFixture) ListDesks(context.Context) ([]string, error) {
	return []string{"example"}, nil
}
func (d releaseRuntimeFixture) Create(context.Context, string) error {
	return errors.New("unexpected creation")
}
func (d releaseRuntimeFixture) SourceIdentity(context.Context) (string, error) {
	return d.b.identity, nil
}
func (d releaseRuntimeFixture) Release(_ context.Context, _ string, action string, _ deskwork.Observation) error {
	b := d.b
	b.calls.Add(1)
	switch action {
	case "activate-local":
		b.app.Zest = "live"
		b.app.Agents = []deskwork.AppAgent{{Name: "example", Live: true}}
	case "publish-source":
		b.observation.Publication = "published"
	case "publish-treaty":
		b.observation.Publication = "published"
		b.app.TreatyListed = true
		b.app.TreatyHash = b.observation.ClayHash
	default:
		return errors.New("unknown action")
	}
	if b.lost {
		return errors.New("lost effect acknowledgement")
	}
	return nil
}
func TestDeskReleaseAPIGatesFreshReviewsAndReadOnlyReconciliation(t *testing.T) {
	for _, action := range []string{"activate-local", "publish-source", "publish-treaty"} {
		t.Run(action, func(t *testing.T) {
			for _, scenario := range []string{"ok", "default-block", "policy-revoked", "workflow-changed", "ship-instructions-changed", "source-changed", "pier-changed", "lost-reply"} {
				t.Run(scenario, func(t *testing.T) {
					base := &deskBackendFixture{}
					r, initial := deskServerFixture(t, base)
					b := &releaseBackendFixture{deskBackendFixture: base, observation: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, app: deskwork.AppState{Zest: "dead", SourceRevision: 1, ExpectedAgents: []string{"example"}, Agents: []deskwork.AppAgent{}, TreatyAvailable: true, DocketAvailable: true, DocketHash: "0v456", FrontendMode: "site", FrontendState: "absent"}, identity: strings.Repeat("c", 64), lost: scenario == "lost-reply"}
					r.b = b
					setting := deskwork.DefaultSetting(initial.Target)
					setting.Desk.Publication = "source"
					if action == "publish-treaty" {
						setting.Desk.Publication = "treaty"
					}
					setting, err := r.s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: initial.Target, Desk: setting.Desk})
					g02Check(t, err)
					policy := apppolicy.Default()
					if scenario != "default-block" {
						policy, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "desk", Target: initial.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Allow}})
						g02Check(t, err)
					}
					q := protocol.DeskReleaseRequest{Selection: deskSelection(initial), Action: action}
					out := deskHTTPRequest(t, r, "POST", protocol.DeskReleaseReviewPath, q)
					if out.Code != 200 {
						t.Fatal(out.Code, out.Body.String())
					}
					var review protocol.DeskReleaseReview
					g02Check(t, json.Unmarshal(out.Body.Bytes(), &review))
					g02Check(t, review.Validate())
					i := initial
					i.Action = action
					i.Baseline = review.Review.Observation
					i.PolicyRevision = policy.Revision
					checks := []deskwork.ReleaseCheck{}
					submission := protocol.DeskReleaseSubmission{Intent: deskwork.ReleaseIntent{Intent: i, SettingsRevision: setting.Revision, Continuity: b.identity, ReviewDigest: review.Review.Digest, ChecksDigest: deskwork.ReleaseChecksDigest(checks)}, Checks: checks}
					b.onWorker = func() {
						switch scenario {
						case "policy-revoked":
							_, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "desk", Target: initial.Target, ExpectedRevision: policy.Revision, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
						case "workflow-changed":
							setting.Desk.Instructions = "new instructions"
							_, err = r.s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: initial.Target, Desk: setting.Desk, ExpectedRevision: setting.Revision})
						case "ship-instructions-changed":
							ship := deskwork.DefaultSetting(initial.Target.Ship())
							ship.Ship.Instructions = "new ship instructions"
							_, err = r.s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: ship.Target, Ship: ship.Ship})
						case "source-changed":
							b.observation.ClayHash = "0v999"
						case "pier-changed":
							b.identity = strings.Repeat("d", 64)
						}
						if err != nil {
							t.Error(err)
						}
					}
					out = deskHTTPRequest(t, r, "POST", protocol.DeskReleasePath, submission)
					if out.Code != 202 {
						t.Fatal(out.Code, out.Body.String())
					}
					receipt := deskWait(t, r, i.ID)
					want := "blocked"
					effects := int32(0)
					if scenario == "ok" {
						want = "observed"
						effects = 1
					}
					if scenario == "lost-reply" {
						want = "uncertain"
						effects = 1
					}
					if receipt.Steps[len(receipt.Steps)-1].State != want || b.calls.Load() != effects {
						t.Fatal("unexpected effect", receipt, b.calls.Load())
					}
					if scenario == "lost-reply" {
						_, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "desk", Target: initial.Target, ExpectedRevision: policy.Revision, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
						g02Check(t, err)
						out = deskHTTPRequest(t, r, "POST", protocol.DeskOperationsPath+"/"+i.ID+"/reconcile", struct{}{})
						if out.Code != 202 {
							t.Fatal(out.Code, out.Body.String())
						}
						// Reconciliation retains the same read-only authority after revocation.
						r.wg.Wait()
						receipt = deskWait(t, r, i.ID)
						if receipt.Steps[len(receipt.Steps)-1].State != "observed" {
							t.Fatal("unobserved recovery", receipt)
						}
					}
					out = deskHTTPRequest(t, r, "POST", protocol.DeskReleasePath, submission)
					if out.Code != 200 || b.calls.Load() != effects {
						t.Fatal("replay dispatched", out.Code, b.calls.Load())
					}
					changed := submission
					changed.Intent.ReviewDigest = strings.Repeat("f", 64)
					out = deskHTTPRequest(t, r, "POST", protocol.DeskReleasePath, changed)
					if out.Code != 409 {
						t.Fatal("intent rebound", out.Code)
					}
				})
			}
		})
	}
}
