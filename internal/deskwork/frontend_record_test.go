package deskwork

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"strings"
	"testing"
	"time"
)

func frontendRecordFixture(t *testing.T) FrontendRecord {
	t.Helper()
	i := intentFixture()
	i.Action = "upload-frontend"
	i.SourceDigest = strings.Repeat("f", 64)
	ref := FrontendReference{Mode: "glob-ames", Base: "demo", Ship: "~zod", GlobHash: "0v1", DocketHash: "0v2"}
	app := AppState{Zest: "live", SourceRevision: i.Baseline.Revision, ExpectedAgents: []string{}, Agents: []AppAgent{}, DocketAvailable: true, DocketHash: ref.DocketHash, FrontendMode: ref.Mode, FrontendState: "glob", FrontendDocketHash: ref.DocketHash}
	r := FrontendReview{Version: 1, Ship: "~zod", Observation: i.Baseline, Application: app, Frontend: FrontendState{SourceRevision: i.Baseline.Revision, DocketAvailable: true, Stage: "glob", Declared: ref, Current: ref}, ShipSetting: DefaultSetting(i.Target.Ship()), DeskSetting: DefaultSetting(i.Target), Continuity: strings.Repeat("c", 64), Stage: StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: i.Target, SourceDigest: strings.Repeat("e", 64), Files: 1, Bytes: 80}, ArtifactDigest: i.SourceDigest, ExpectedGlobHash: "0v3", UnchangedSourceDigest: strings.Repeat("a", 64)}
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	checks := []FrontendCheck{}
	out := FrontendRecord{Version: 1, Intent: FrontendIntent{Intent: i, StageID: r.Stage.ID, Continuity: r.Continuity, ReviewDigest: r.Digest, ChecksDigest: FrontendChecksDigest(checks)}, Review: r, Checks: checks, Actor: "fixture", CreatedAt: time.Now().UTC()}
	if err := out.Validate(); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestFrontendReviewBindingsGatesAndReleaseChecks(t *testing.T) {
	record := frontendRecordFixture(t)
	i := record.Intent.Intent
	p := apppolicy.Default()
	for _, pub := range []string{"private", "published"} {
		i.Baseline.Publication = pub
		for _, g := range []apppolicy.Gates{{Commit: apppolicy.Block, Publish: apppolicy.Block}, {Commit: apppolicy.Allow, Publish: apppolicy.Block}, {Commit: apppolicy.Block, Publish: apppolicy.Allow}} {
			p.Global = g
			p.Global.CreateDesk = apppolicy.Block
			if i.Check(p, i.Baseline) == nil {
				t.Fatal("frontend bypassed an effect gate", pub, g)
			}
		}
		p.Global.Commit = apppolicy.Allow
		p.Global.Publish = apppolicy.Allow
		if err := i.Check(p, i.Baseline); err != nil {
			t.Fatal(err)
		}
	}
	for _, change := range []func(*FrontendRecord){
		func(r *FrontendRecord) { r.Intent.StageID = strings.Repeat("9", 64) },
		func(r *FrontendRecord) { r.Intent.Intent.SourceDigest = strings.Repeat("9", 64) },
		func(r *FrontendRecord) { r.Intent.ShipSettingsRevision++ },
		func(r *FrontendRecord) { r.Review.Ship = "~nec" },
		func(r *FrontendRecord) { r.Review.ExpectedGlobHash = "0v4" },
		func(r *FrontendRecord) { r.Review.UnchangedSourceDigest = strings.Repeat("9", 64) },
		func(r *FrontendRecord) { r.Review.Continuity = strings.Repeat("9", 64) },
		func(r *FrontendRecord) { r.Checks = nil },
	} {
		r := record
		change(&r)
		if r.Validate() == nil {
			t.Fatal("changed frontend review accepted")
		}
	}
	review := record.Review
	review.DeskSetting.Desk.Checks = []string{"build and app smoke"}
	if err := review.Seal(); err != nil {
		t.Fatal(err)
	}
	if err := CheckFrontendEvidence(review, []FrontendCheck{}); err != nil {
		t.Fatal("private preparation requires premature release checks", err)
	}
	review.Observation.Publication = "published"
	review.DeskSetting.Desk.Publication = "source"
	if err := review.Seal(); err != nil {
		t.Fatal(err)
	}
	if CheckFrontendEvidence(review, []FrontendCheck{}) == nil {
		t.Fatal("distributed upload bypassed checks")
	}
	checks := []FrontendCheck{{Name: "build and app smoke", ClayHash: review.Observation.ClayHash, ArtifactDigest: review.ArtifactDigest, Passed: true, Evidence: "build report"}}
	if err := CheckFrontendEvidence(review, checks); err != nil {
		t.Fatal(err)
	}
	checks[0].ArtifactDigest = strings.Repeat("9", 64)
	if CheckFrontendEvidence(review, checks) == nil {
		t.Fatal("wrong artifact checks accepted")
	}
	review.DeskSetting.Desk.Workflow = "urgit"
	review.DeskSetting.Desk.SourceName = ""
	if err := review.Seal(); err != nil {
		t.Fatal(err)
	}
	if review.CheckConfiguredUpload() == nil {
		t.Fatal("Urgit workflow overwritten")
	}
}
func TestFrontendResultChecksSourceAndMetadataAndRetainsCleanupUncertainty(t *testing.T) {
	record := frontendRecordFixture(t)
	review := record.Review
	after := review.Observation
	after.Revision++
	after.ClayHash = "0v4"
	app := review.Application
	app.SourceRevision = after.Revision
	app.DocketHash = "0v5"
	app.FrontendDocketHash = "0v5"
	f := review.Frontend
	f.SourceRevision = after.Revision
	f.Declared.GlobHash = review.ExpectedGlobHash
	f.Declared.DocketHash = "0v5"
	f.Current = f.Declared
	if err := VerifyFrontendResult(review, after, app, f, review.UnchangedSourceDigest); err != nil {
		t.Fatal(err)
	}
	if VerifyFrontendResult(review, after, app, f, strings.Repeat("9", 64)) == nil {
		t.Fatal("unrelated source drift accepted")
	}
	wrong := f
	wrong.Declared.GlobHash = "0v6"
	wrong.Current = wrong.Declared
	if VerifyFrontendResult(review, after, app, wrong, review.UnchangedSourceDigest) == nil {
		t.Fatal("different glob accepted")
	}
	at := time.Now().UTC()
	receipt := Receipt{Version: 1, Intent: record.Intent.Intent, Actor: "fixture", Steps: []Step{{Kind: "upload-frontend", State: "planned", At: at}, {Kind: "upload-frontend", State: "dispatched", At: at}, {Kind: "upload-frontend", State: "acknowledged", At: at, FrontendTransfer: &FrontendTransfer{Attempted: true, Acknowledged: true, SessionCleanup: "unconfirmed"}}, {Kind: "upload-frontend", State: "observed", At: at, Observation: &after, Application: &app, Frontend: &f, FrontendSourceDigest: review.UnchangedSourceDigest, VerifiedSourceDigest: review.ArtifactDigest}}}
	if err := receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	recovered, err := receipt.Recover(at)
	if err != nil || len(recovered.Steps) != len(receipt.Steps) {
		t.Fatal("replayed successful upload", err)
	}
	receipt.Steps = receipt.Steps[:2]
	recovered, err = receipt.Recover(at)
	if err != nil || recovered.Steps[2].State != "uncertain" {
		t.Fatal("lost effect was not uncertain", err)
	}
}
