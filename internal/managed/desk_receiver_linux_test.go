//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type receiverBackendFixture struct {
	*releaseBackendFixture
	subscription deskwork.Subscription
}
type receiverRuntimeFixture struct {
	releaseRuntimeFixture
	b *receiverBackendFixture
}

func (b *receiverBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	if b.opens.Add(1) == 3 && b.onWorker != nil {
		b.onWorker()
	}
	return receiverRuntimeFixture{releaseRuntimeFixture{b.releaseBackendFixture}, b}, nil
}
func (d receiverRuntimeFixture) Subscription(context.Context, string) (deskwork.Subscription, error) {
	return d.b.subscription, nil
}
func (d receiverRuntimeFixture) ReceiverSourceReady(context.Context, string) (bool, error) {
	return d.b.observation.Exists && d.b.observation.Revision > 0, nil
}
func (d receiverRuntimeFixture) InstallRemote(_ context.Context, _ string, p deskwork.Publisher) error {
	d.b.calls.Add(1)
	d.b.observation = deskwork.Observation{Exists: true, Revision: 1, ClayHash: p.ExpectedHash, Publication: "private"}
	d.b.app = deskwork.AppState{Zest: "live", SourceRevision: 1, SourceShip: p.Ship, SourceDesk: p.Desk, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	d.b.subscription = deskwork.Subscription{Present: true, Ship: p.Ship, Desk: p.Desk, Zest: "live", Hash: p.ExpectedHash}
	if d.b.lost {
		return errors.New("install reply lost")
	}
	return nil
}

func receiverServerFixture(t *testing.T) (*Server, deskwork.Intent, *receiverBackendFixture) {
	t.Helper()
	base := &deskBackendFixture{}
	r, initial := deskServerFixture(t, base)
	b := &receiverBackendFixture{releaseBackendFixture: &releaseBackendFixture{deskBackendFixture: base, identity: strings.Repeat("c", 64)}}
	r.b = b
	return r, initial, b
}

func TestDeskReceiverAPIEnforcesFreshGatesAndSingleInstall(t *testing.T) {
	for _, scenario := range []string{"ok", "default-block", "create-only", "commit-only", "policy-revoked", "workflow-changed", "ship-instructions-changed", "desk-occupied", "pending-subscription", "pier-changed", "lost-reply"} {
		t.Run(scenario, func(t *testing.T) {
			r, initial, b := receiverServerFixture(t)
			b.lost = scenario == "lost-reply"
			policy := apppolicy.Default()
			var err error
			if scenario != "default-block" {
				gates := apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Allow, Publish: apppolicy.Block}
				if scenario == "create-only" {
					gates.Commit = apppolicy.Block
				}
				if scenario == "commit-only" {
					gates.CreateDesk = apppolicy.Block
				}
				policy, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: initial.Target.Ship(), Gates: gates})
				g02Check(t, err)
			}
			publisher := deskwork.Publisher{Ship: "~nec", Desk: "demo", ExpectedHash: "0v123"}
			q := protocol.DeskReceiverRequest{Selection: deskSelection(initial), Publisher: publisher}
			out := deskHTTPRequest(t, r, "POST", protocol.DeskReceiverReviewPath, q)
			if out.Code != 200 {
				t.Fatal(out.Code, out.Body.String())
			}
			var review protocol.DeskReceiverReview
			g02Check(t, json.Unmarshal(out.Body.Bytes(), &review))
			g02Check(t, review.Validate())
			initial.Action, initial.SourceDigest, initial.PolicyRevision = "install-remote", publisher.Digest(), policy.Revision
			i := deskwork.ReceiverIntent{Intent: initial, Publisher: publisher, Continuity: b.identity, ReviewDigest: review.Review.Digest}
			b.onWorker = func() {
				switch scenario {
				case "policy-revoked":
					_, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: initial.Target.Ship(), ExpectedRevision: policy.Revision, Gates: apppolicy.Blocked()})
				case "workflow-changed":
					setting := deskwork.DefaultSetting(initial.Target)
					setting.Desk.Instructions = "changed development instructions"
					_, err = r.s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: setting.Target, Desk: setting.Desk})
				case "ship-instructions-changed":
					setting := deskwork.DefaultSetting(initial.Target.Ship())
					setting.Ship.Instructions = "changed ship instructions"
					_, err = r.s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: setting.Target, Ship: setting.Ship})
				case "desk-occupied":
					b.observation = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v999", Publication: "private"}
				case "pending-subscription":
					b.subscription = deskwork.Subscription{Present: true, Ship: "~bud", Desk: "demo", Zest: "live", Hash: "0v0"}
				case "pier-changed":
					b.identity = strings.Repeat("d", 64)
				}
				if err != nil {
					t.Error(err)
				}
			}
			out = deskHTTPRequest(t, r, "POST", protocol.DeskReceiversPath, i)
			if out.Code != 202 {
				t.Fatal(out.Code, out.Body.String())
			}
			receipt := deskWait(t, r, initial.ID)
			want, effects := "blocked", int32(0)
			if scenario == "ok" {
				want, effects = "observed", 1
			}
			if scenario == "lost-reply" {
				want, effects = "uncertain", 1
			}
			if receipt.Steps[len(receipt.Steps)-1].State != want || b.calls.Load() != effects {
				t.Fatal(receipt, b.calls.Load())
			}
			if scenario == "lost-reply" {
				_, err = r.s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: initial.Target.Ship(), ExpectedRevision: policy.Revision, Gates: apppolicy.Blocked()})
				g02Check(t, err)
				out = deskHTTPRequest(t, r, "POST", protocol.DeskOperationsPath+"/"+initial.ID+"/reconcile", struct{}{})
				if out.Code != 202 {
					t.Fatal(out.Code, out.Body.String())
				}
				r.wg.Wait()
				receipt = deskWait(t, r, initial.ID)
				if receipt.Steps[len(receipt.Steps)-1].State != "observed" {
					t.Fatal(receipt)
				}
			}
			out = deskHTTPRequest(t, r, "POST", protocol.DeskReceiversPath, i)
			if out.Code != 200 || b.calls.Load() != effects {
				t.Fatal("replayed install", out.Code, b.calls.Load())
			}
			i.ReviewDigest = strings.Repeat("f", 64)
			out = deskHTTPRequest(t, r, "POST", protocol.DeskReceiversPath, i)
			if out.Code != 409 {
				t.Fatal("rebound intent", out.Code)
			}
		})
	}
}

func TestDeskReceiverReviewRejectsExistingAndSelfPublisher(t *testing.T) {
	for _, scenario := range []string{"pending", "occupied", "self"} {
		t.Run(scenario, func(t *testing.T) {
			r, initial, b := receiverServerFixture(t)
			publisher := deskwork.Publisher{Ship: "~nec", Desk: "demo", ExpectedHash: "0v123"}
			switch scenario {
			case "pending":
				b.subscription = deskwork.Subscription{Present: true, Ship: "~bud", Desk: "demo", Zest: "live", Hash: "0v0"}
			case "occupied":
				b.observation = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
			case "self":
				publisher.Ship = "~zod"
			}
			out := deskHTTPRequest(t, r, "POST", protocol.DeskReceiverReviewPath, protocol.DeskReceiverRequest{Selection: deskSelection(initial), Publisher: publisher})
			if out.Code < 400 || b.calls.Load() != 0 {
				t.Fatal(out.Code, out.Body.String())
			}
		})
	}
}

func TestDeskReceiverStatusSeparatesPendingSourceFromApplicationStages(t *testing.T) {
	for _, scenario := range []string{"absent", "pending", "received", "changed-revision"} {
		t.Run(scenario, func(t *testing.T) {
			r, initial, b := receiverServerFixture(t)
			if scenario != "absent" {
				b.subscription = deskwork.Subscription{Present: true, Ship: "~nec", Desk: "demo", Zest: "live", Hash: "0v0"}
			}
			if scenario == "received" || scenario == "changed-revision" {
				b.observation = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
				b.app = deskwork.AppState{Zest: "live", SourceRevision: 1, SourceShip: "~nec", SourceDesk: "demo", ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
			}
			if scenario == "changed-revision" {
				b.app.SourceRevision = 2
			}
			out := deskHTTPRequest(t, r, "POST", protocol.DeskReceiverStatusPath, deskSelection(initial))
			if scenario == "changed-revision" {
				if out.Code != 502 {
					t.Fatal(out.Code, out.Body.String())
				}
			} else {
				if out.Code != 200 {
					t.Fatal(out.Code, out.Body.String())
				}
				var status protocol.DeskReceiverStatus
				g02Check(t, json.Unmarshal(out.Body.Bytes(), &status))
				g02Check(t, status.Validate())
				if status.SourceReady != (scenario == "received") || status.Subscription.Present != (scenario != "absent") {
					t.Fatal(status)
				}
			}
			if b.calls.Load() != 0 {
				t.Fatal("status dispatched install")
			}
		})
	}
}
