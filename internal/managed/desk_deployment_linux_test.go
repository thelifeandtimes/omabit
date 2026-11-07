//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type commitRuntimeFixture struct {
	observation      deskwork.Observation
	files            map[string]deskwork.File
	continuity       string
	calls            int
	lostReply        bool
	failResult       bool
	corruptGenerated bool
}

func (f *commitRuntimeFixture) ListDesks(context.Context) ([]string, error) {
	return []string{"example"}, nil
}
func (f *commitRuntimeFixture) Create(context.Context, string) error {
	return errors.New("creation forbidden")
}
func (f *commitRuntimeFixture) Observe(context.Context, string) (deskwork.Observation, error) {
	return f.observation, nil
}
func (f *commitRuntimeFixture) SourceIdentity(context.Context) (string, error) {
	return f.continuity, nil
}
func (f *commitRuntimeFixture) ClayFiles(context.Context, string, uint64) (deskwork.Manifest, error) {
	files := []deskwork.File{}
	for _, v := range f.files {
		files = append(files, v)
	}
	return deskwork.NewManifest(files)
}
func canonicalCommitFixture(path string, raw []byte) deskwork.File {
	sum := sha256.Sum256(append([]byte("canonical fixture:"), raw...))
	return deskwork.File{Path: path, SHA256: hex.EncodeToString(sum[:])}
}
func (f *commitRuntimeFixture) NormalizeFiles(_ context.Context, _ string, _ uint64, files []deskwork.SourceFile) (deskwork.Manifest, error) {
	if f.failResult && f.calls > 0 {
		return deskwork.Manifest{}, errors.New("mark conversion unavailable")
	}
	out := []deskwork.File{}
	for _, v := range files {
		out = append(out, canonicalCommitFixture(v.Path, v.Bytes))
	}
	return deskwork.NewManifest(out)
}
func (f *commitRuntimeFixture) CommitFiles(_ context.Context, _ string, baseline deskwork.Observation, files []deskwork.DeltaFile) error {
	if baseline != f.observation {
		return errors.New("baseline changed")
	}
	f.calls++
	for _, file := range files {
		if file.Delete {
			delete(f.files, file.Path)
		} else {
			f.files[file.Path] = canonicalCommitFixture(file.Path, file.Bytes)
		}
	}
	if f.corruptGenerated {
		f.files["generated.txt"] = canonicalCommitFixture("generated.txt", []byte("changed externally"))
	}
	f.observation.Revision++
	f.observation.ClayHash = "0v456"
	if f.lostReply {
		return errors.New("lost acknowledgement")
	}
	return nil
}

type commitBackendFixture struct {
	*deskBackendFixture
	runtime *commitRuntimeFixture
}

func (f *commitBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return f.runtime, nil
}
func (f *commitBackendFixture) OpenProjectDesk(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string, string) (deskRuntime, error) {
	return f.runtime, nil
}
func (f *commitBackendFixture) OpenExpandedProjectDesk(context.Context, runtimequalify.ProjectExpansion, string, string) (deskRuntime, error) {
	return f.runtime, nil
}
func commitServerFixture(t *testing.T) (*Server, *commitRuntimeFixture, deskwork.StageIntent) {
	t.Helper()
	base := &deskBackendFixture{}
	r, original := deskServerFixture(t, base)
	f := &commitRuntimeFixture{observation: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, continuity: strings.Repeat("f", 64), files: map[string]deskwork.File{"generated.txt": canonicalCommitFixture("generated.txt", []byte("keep"))}}
	r.b = &commitBackendFixture{base, f}
	stage, body := stageBundleFixture(t, original)
	uploaded := stageHTTP(t, r, stage, body)
	if uploaded.Code != 201 {
		t.Fatal(uploaded.Code, uploaded.Body.String())
	}
	return r, f, stage
}
func reviewCommitFixture(t *testing.T, r *Server, stage deskwork.StageIntent) protocol.DeskDeploymentReview {
	t.Helper()
	request := protocol.DeskDeploymentRequest{Selection: deskSelection(deskIntentFixture()), StageID: stage.ID, SettingsRevision: stage.SettingsRevision}
	got := deskHTTPRequest(t, r, "POST", protocol.DeskDeploymentReviewPath, request)
	var out protocol.DeskDeploymentReview
	if got.Code != 200 || json.Unmarshal(got.Body.Bytes(), &out) != nil || out.Validate() != nil {
		t.Fatal(got.Code, got.Body.String())
	}
	return out
}
func commitIntentFromReview(v protocol.DeskDeploymentReview) deskwork.CommitIntent {
	return deskwork.CommitIntent{Intent: deskwork.Intent{Version: 1, ID: strings.Repeat("a", 64), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "commit", Baseline: v.Observation, SourceDigest: v.SourceDigest}, StageID: v.Request.StageID, SettingsRevision: v.Request.SettingsRevision, SourceName: v.SourceName, Continuity: v.Continuity, PlanDigest: v.PlanDigest}
}
func TestDeskCommitAPIDefaultGatePublishedGateAndVerifiedCompletion(t *testing.T) {
	for _, scenario := range []string{"blocked-default", "allow-private", "block-published", "allow-published", "lost-reply", "failed-verification", "generated-drift"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, stage := commitServerFixture(t)
			if scenario != "blocked-default" {
				gates := apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}
				if scenario == "allow-published" {
					gates.Publish = apppolicy.Allow
				}
				_, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: gates})
				g02Check(t, err)
			}
			if strings.HasSuffix(scenario, "published") {
				f.observation.Publication = "published"
			}
			f.lostReply = scenario == "lost-reply"
			f.failResult = scenario == "failed-verification"
			f.corruptGenerated = scenario == "generated-drift"
			v := reviewCommitFixture(t, r, stage)
			if v.ChangeCount != 1 || v.PreservedCount != 1 || f.calls != 0 {
				t.Fatal("review mutated runtime or lost generated files", v)
			}
			input := commitIntentFromReview(v)
			posted := deskHTTPRequest(t, r, "POST", protocol.DeskCommitPath, input)
			if posted.Code != 202 {
				t.Fatal(posted.Code, posted.Body.String())
			}
			r.wg.Wait()
			receipt, err := loadDeskOperation(r.s.db, input.Intent.ID)
			g02Check(t, err)
			state := receipt.Steps[len(receipt.Steps)-1].State
			want := "observed"
			calls := 1
			switch scenario {
			case "blocked-default", "block-published":
				want = "blocked"
				calls = 0
			case "lost-reply", "failed-verification", "generated-drift":
				want = "uncertain"
			}
			if state != want || f.calls != calls {
				t.Fatal(scenario, state, f.calls)
			}
			replay := deskHTTPRequest(t, r, "POST", protocol.DeskCommitPath, input)
			if replay.Code != 200 || f.calls != calls {
				t.Fatal("replay redispatched", replay.Code, f.calls)
			}
			owned, err := r.s.sourceOwnership(stage.Target, "example", f.continuity)
			g02Check(t, err)
			if (len(owned.Raw.Files) == 1) != (want == "observed") {
				t.Fatal("unverified ownership stored", owned)
			}
			if scenario == "lost-reply" || scenario == "failed-verification" {
				f.failResult = false
				reconciled := deskHTTPRequest(t, r, "POST", protocol.DeskOperationsPath+"/"+input.Intent.ID+"/reconcile", struct{}{})
				if reconciled.Code != 202 {
					t.Fatal(reconciled.Code, reconciled.Body.String())
				}
				r.wg.Wait()
				receipt, err = loadDeskOperation(r.s.db, input.Intent.ID)
				g02Check(t, err)
				if receipt.Steps[len(receipt.Steps)-1].State != "observed" || f.calls != 1 {
					t.Fatal("reconciliation failed or repeated effect", receipt, f.calls)
				}
			}
			if want == "observed" {
				discarded := deskHTTPRequest(t, r, "DELETE", protocol.DeskStagesPath+"/"+stage.ID, protocol.DeskStageDiscardRequest{SourceDigest: stage.SourceDigest})
				if discarded.Code != 200 {
					t.Fatal(discarded.Code, discarded.Body.String())
				}
				if replay := deskHTTPRequest(t, r, "POST", protocol.DeskCommitPath, input); replay.Code != 200 || f.calls != 1 {
					t.Fatal("completed replay depended on discarded cache", replay.Code)
				}
			}
		})
	}
}
func TestDeskCommitAPIRejectsStaleOrConflictingReviewBeforeDispatch(t *testing.T) {
	for _, scenario := range []string{"desk-changed", "source-changed", "settings-changed", "pier-changed", "plan-changed", "policy-changed", "no-op", "conflict"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, stage := commitServerFixture(t)
			if scenario == "no-op" {
				f.files["hello.txt"] = canonicalCommitFixture("hello.txt", []byte("hello\n"))
			}
			if scenario == "conflict" {
				f.files["hello.txt"] = canonicalCommitFixture("hello.txt", []byte("external"))
			}
			v := reviewCommitFixture(t, r, stage)
			input := commitIntentFromReview(v)
			switch scenario {
			case "desk-changed":
				f.observation.Revision++
			case "source-changed":
				g02Check(t, os.WriteFile(filepath.Join(r.s.directory, "desk-stages-v1", stage.ID, "source", "hello.txt"), []byte("edited"), 0600))
			case "settings-changed":
				input.SettingsRevision++
			case "pier-changed":
				f.continuity = strings.Repeat("e", 64)
			case "plan-changed":
				input.PlanDigest = strings.Repeat("e", 64)
			case "policy-changed":
				input.Intent.PolicyRevision++
			}
			posted := deskHTTPRequest(t, r, "POST", protocol.DeskCommitPath, input)
			if posted.Code != 409 || f.calls != 0 {
				t.Fatal("stale commit accepted", scenario, posted.Code, posted.Body.String())
			}
			var count int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM desk_operations").Scan(&count))
			if count != 0 {
				t.Fatal("refused review created effect")
			}
		})
	}
}

func TestDeskCommitRechecksQueuedPlanBeforeEffect(t *testing.T) {
	for _, scenario := range []string{"policy", "settings", "source", "pier", "runtime"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, stage := commitServerFixture(t)
			policy, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: stage.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
			g02Check(t, err)
			review := reviewCommitFixture(t, r, stage)
			input := commitIntentFromReview(review)
			func() {
				r.mu.Lock()
				defer r.mu.Unlock()
				data, _ := json.Marshal(input)
				req := httptest.NewRequest("POST", protocol.DeskCommitPath, bytes.NewReader(data))
				req.Header.Set("Content-Type", "application/json")
				out := httptest.NewRecorder()
				r.serveDeskCommit(out, req, fmt.Sprintf("local-uid:%d", r.uid), "operator")
				if out.Code != 202 {
					t.Fatal(out.Code, out.Body.String())
				}
				switch scenario {
				case "policy":
					_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: stage.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
					g02Check(t, err)
				case "settings":
					setting := deskwork.DefaultSetting(stage.Target)
					setting.Desk.Instructions = "updated release checks"
					_, err = r.s.putDeskSetting("setup", deskwork.SettingUpdate{Target: stage.Target, Desk: setting.Desk})
					g02Check(t, err)
				case "source":
					g02Check(t, os.WriteFile(filepath.Join(r.s.directory, "desk-stages-v1", stage.ID, "source", "hello.txt"), []byte("edited"), 0600))
				case "pier":
					f.continuity = strings.Repeat("e", 64)
				case "runtime":
					v, err := loadInstance(r.s.db, "alpha")
					g02Check(t, err)
					v.Public.ObservedState = "stopped"
					saveRecoveryFixture(t, r.s, v)
				}
			}()
			r.wg.Wait()
			receipt, err := loadDeskOperation(r.s.db, input.Intent.ID)
			g02Check(t, err)
			if f.calls != 0 || receipt.Steps[len(receipt.Steps)-1].State != "blocked" {
				t.Fatal("queued commit ignored changed binding", scenario, receipt, f.calls)
			}
		})
	}
}
