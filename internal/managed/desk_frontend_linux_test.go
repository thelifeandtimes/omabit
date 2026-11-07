//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type frontendBackendFixture struct {
	*deskBackendFixture
	runtime  *frontendRuntimeFixture
	opens    int
	onWorker func()
}

func (b *frontendBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	b.opens++
	if b.opens == 3 && b.onWorker != nil {
		b.onWorker()
	}
	return b.runtime, nil
}

type frontendRuntimeFixture struct {
	beforeUploadFailure bool
	*commitRuntimeFixture
	app      deskwork.AppState
	frontend deskwork.FrontendState
	cleanup  string
	corrupt  bool
}

func (f *frontendRuntimeFixture) Ship() string { return "~zod" }
func (f *frontendRuntimeFixture) Application(context.Context, string) (deskwork.AppState, error) {
	return f.app, nil
}
func (f *frontendRuntimeFixture) Frontend(context.Context, string) (deskwork.FrontendState, error) {
	return f.frontend, nil
}
func (f *frontendRuntimeFixture) FrontendHash(context.Context, deskwork.FrontendManifest, []deskwork.SourceFile) (string, error) {
	return "0v3", nil
}
func (f *frontendRuntimeFixture) UploadFrontend(_ context.Context, r deskwork.FrontendReview, m deskwork.FrontendManifest, files []deskwork.SourceFile) (deskwork.FrontendTransfer, error) {
	if f.beforeUploadFailure {
		return deskwork.FrontendTransfer{SessionCleanup: "unconfirmed"}, errors.New("login response lost; no upload dispatched")
	}
	transfer := deskwork.FrontendTransfer{Attempted: true, SessionCleanup: f.cleanup}
	if _, _, err := deskwork.FrontendMultipart("example", m, files); err != nil {
		return transfer, err
	}
	f.calls++
	f.observation.Revision++
	f.observation.ClayHash = "0v4"
	f.app.SourceRevision = f.observation.Revision
	f.app.DocketHash = "0v5"
	f.app.FrontendDocketHash = "0v5"
	f.frontend.SourceRevision = f.observation.Revision
	f.frontend.Declared.DocketHash = "0v5"
	f.frontend.Declared.GlobHash = r.ExpectedGlobHash
	f.frontend.Current = f.frontend.Declared
	if f.corrupt {
		f.files["keep.txt"] = canonicalCommitFixture("keep.txt", []byte("external change"))
	}
	if f.lostReply {
		return transfer, errors.New("lost upload acknowledgement")
	}
	transfer.Acknowledged = true
	return transfer, nil
}
func frontendServerFixture(t *testing.T) (*Server, *frontendBackendFixture, deskwork.StageIntent) {
	t.Helper()
	base := &deskBackendFixture{}
	r, initial := deskServerFixture(t, base)
	record := frontendRecordFixture(t)
	f := &frontendRuntimeFixture{commitRuntimeFixture: &commitRuntimeFixture{observation: record.Review.Observation, continuity: record.Review.Continuity, files: map[string]deskwork.File{"desk.docket-0": canonicalCommitFixture("desk.docket-0", []byte("docket")), "keep.txt": canonicalCommitFixture("keep.txt", []byte("keep"))}}, app: record.Review.Application, frontend: record.Review.Frontend, cleanup: "confirmed"}
	b := &frontendBackendFixture{deskBackendFixture: base, runtime: f}
	r.b = b
	dir := t.TempDir()
	g02Check(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("frontend"), 0600))
	m, err := deskwork.Snapshot(dir)
	g02Check(t, err)
	var bundle bytes.Buffer
	g02Check(t, deskwork.WriteBundle(context.Background(), dir, m, &bundle))
	stage := deskwork.StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: initial.Target, SourceDigest: m.Digest, Files: 1, Bytes: 8}
	got := stageHTTP(t, r, stage, bundle.Bytes())
	if got.Code != 201 {
		t.Fatal(got.Code, got.Body.String())
	}
	return r, b, stage
}
func frontendSubmission(t *testing.T, r *Server, stage deskwork.StageIntent) protocol.DeskFrontendSubmission {
	t.Helper()
	q := protocol.DeskFrontendRequest{Selection: deskSelection(deskIntentFixture()), StageID: stage.ID, SettingsRevision: stage.SettingsRevision}
	got := deskHTTPRequest(t, r, "POST", protocol.DeskFrontendReviewPath, q)
	if got.Code != 200 {
		t.Fatal(got.Code, got.Body.String())
	}
	var review protocol.DeskFrontendReview
	g02Check(t, json.Unmarshal(got.Body.Bytes(), &review))
	g02Check(t, review.Validate())
	v := review.Review
	checks := []deskwork.FrontendCheck{}
	i := deskIntentFixture()
	i.Action = "upload-frontend"
	i.Baseline = v.Observation
	i.SourceDigest = v.ArtifactDigest
	i.PolicyRevision = review.PolicyRevision
	return protocol.DeskFrontendSubmission{Intent: deskwork.FrontendIntent{Intent: i, StageID: stage.ID, SettingsRevision: stage.SettingsRevision, ShipSettingsRevision: v.ShipSetting.Revision, Continuity: v.Continuity, ReviewDigest: v.Digest, ChecksDigest: deskwork.FrontendChecksDigest(checks)}, Checks: checks}
}
func TestDeskFrontendAPIEnforcesBothGatesFreshReviewAndNoReplay(t *testing.T) {
	for _, scenario := range []string{"ok", "closed", "commit-only", "publish-only", "revoked", "workflow", "source", "pier", "stage", "lost-reply", "cleanup-unknown", "pre-upload-failure"} {
		t.Run(scenario, func(t *testing.T) {
			r, b, stage := frontendServerFixture(t)
			f := b.runtime
			gates := apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Allow}
			if scenario == "closed" || scenario == "publish-only" {
				gates.Commit = apppolicy.Block
			}
			if scenario == "closed" || scenario == "commit-only" {
				gates.Publish = apppolicy.Block
			}
			p, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: gates})
			g02Check(t, err)
			input := frontendSubmission(t, r, stage)
			f.lostReply = scenario == "lost-reply"
			f.beforeUploadFailure = scenario == "pre-upload-failure"
			if scenario == "cleanup-unknown" {
				f.cleanup = "unconfirmed"
			}
			b.onWorker = func() {
				switch scenario {
				case "revoked":
					_, err = r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, ExpectedRevision: p.Revision, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
				case "workflow":
					s := deskwork.DefaultSetting(stage.Target)
					s.Desk.Instructions = "different"
					_, err = r.s.putDeskSetting("setup", deskwork.SettingUpdate{Target: stage.Target, Desk: s.Desk})
				case "source":
					f.observation.ClayHash = "0v9"
				case "pier":
					f.continuity = strings.Repeat("9", 64)
				case "stage":
					err = os.WriteFile(filepath.Join(r.s.directory, "desk-stages-v1", stage.ID, "source", "index.html"), []byte("modified"), 0600)
				}
				if err != nil {
					t.Error(err)
				}
			}
			if scenario == "stage" {
				g02Check(t, os.WriteFile(filepath.Join(r.s.directory, "desk-stages-v1", stage.ID, "source", "index.html"), []byte("modified"), 0600))
				got := deskHTTPRequest(t, r, "POST", protocol.DeskFrontendsPath, input)
				if got.Code != 409 || f.calls != 0 {
					t.Fatal("changed stage dispatched", got.Code, f.calls)
				}
				return
			}
			got := deskHTTPRequest(t, r, "POST", protocol.DeskFrontendsPath, input)
			if got.Code != 202 {
				t.Fatal(got.Code, got.Body.String())
			}
			receipt := deskWait(t, r, input.Intent.Intent.ID)
			want := "blocked"
			if scenario == "pre-upload-failure" {
				want = "rejected"
			}
			calls := 0
			if scenario == "ok" || scenario == "cleanup-unknown" {
				want = "observed"
				calls = 1
			}
			if scenario == "lost-reply" {
				want = "uncertain"
				calls = 1
			}
			if receipt.Steps[len(receipt.Steps)-1].State != want || f.calls != calls {
				t.Fatal("unexpected frontend effect", receipt, f.calls)
			}
			if scenario == "cleanup-unknown" && receipt.Steps[2].FrontendTransfer.SessionCleanup != "unconfirmed" {
				t.Fatal("cleanup failure hidden")
			}
			if scenario == "lost-reply" {
				_, err = r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, ExpectedRevision: p.Revision, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
				g02Check(t, err)
				got = deskHTTPRequest(t, r, "POST", protocol.DeskOperationsPath+"/"+input.Intent.Intent.ID+"/reconcile", struct{}{})
				if got.Code != 202 {
					t.Fatal(got.Code, got.Body.String())
				}
				r.wg.Wait()
				r.mu.Lock()
				receipt, err = loadDeskOperation(r.s.db, input.Intent.Intent.ID)
				r.mu.Unlock()
				g02Check(t, err)
				if receipt.Steps[len(receipt.Steps)-1].State != "observed" || f.calls != 1 {
					t.Fatal("lost reply reconciliation repeated or lost upload", receipt, f.calls)
				}
			}
			got = deskHTTPRequest(t, r, "POST", protocol.DeskFrontendsPath, input)
			if got.Code != 200 || f.calls != calls {
				t.Fatal("replay repeated upload", got.Code, f.calls)
			}
			owned, err := r.s.sourceOwnership(stage.Target, "example", input.Intent.Continuity)
			g02Check(t, err)
			if len(owned.Raw.Files) != 0 {
				t.Fatal("upload implicitly adopted source")
			}
		})
	}
}

func TestDeskFrontendRecoveryRequiresUnchangedOtherSource(t *testing.T) {
	r, b, stage := frontendServerFixture(t)
	_, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Allow}})
	g02Check(t, err)
	input := frontendSubmission(t, r, stage)
	b.runtime.lostReply = true
	got := deskHTTPRequest(t, r, "POST", protocol.DeskFrontendsPath, input)
	if got.Code != 202 {
		t.Fatal(got.Code, got.Body.String())
	}
	receipt := deskWait(t, r, input.Intent.Intent.ID)
	if receipt.Steps[len(receipt.Steps)-1].State != "uncertain" {
		t.Fatal(receipt)
	}
	b.runtime.files["keep.txt"] = canonicalCommitFixture("keep.txt", []byte("unrelated external change"))
	got = deskHTTPRequest(t, r, "POST", protocol.DeskOperationsPath+"/"+input.Intent.Intent.ID+"/reconcile", struct{}{})
	if got.Code != 202 {
		t.Fatal(got.Code)
	}
	r.wg.Wait()
	r.mu.Lock()
	receipt, err = loadDeskOperation(r.s.db, input.Intent.Intent.ID)
	r.mu.Unlock()
	g02Check(t, err)
	if receipt.Steps[len(receipt.Steps)-1].State != "uncertain" || b.runtime.calls != 1 {
		t.Fatal("source drift accepted or upload repeated", receipt, b.runtime.calls)
	}
}
