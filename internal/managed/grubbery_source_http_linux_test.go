//go:build linux && amd64

package managed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func appHTTPRecord(t *testing.T, r *Server, path string, in any) grubbery.Record {
	t.Helper()
	w := deskHTTPRequest(t, r, "POST", path, in)
	var out protocol.GrubberyAppOperation
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Validate() != nil {
		t.Fatal("app operation refused", w.Code, w.Body.String())
	}
	return out.Record
}

func TestGrubberySourceHTTPOriginalPhaseRecoveryWithoutCacheOrCommit(t *testing.T) {
	r, f, p, _ := managedGrubberyFixture(t)
	q := protocol.GrubberyAppReviewRequest{Selection: protocol.DeskSelection{Target: p.Binding.Target, RuntimeID: p.Binding.RuntimeID, ConfigRevision: p.Binding.ConfigRevision}, FrameworkReceiptID: p.Binding.FrameworkReceiptID, AppID: p.Binding.AppID, StageID: p.StageID, RequestID: p.RequestID}
	w := deskHTTPRequest(t, r, "POST", protocol.GrubberyAppReviewPath, q)
	var review protocol.GrubberyAppReview
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &review) != nil || review.Validate() != nil || review.Plan.Binding.Actor != p.Binding.Actor || f.writes != 0 {
		t.Fatal("review fabricated authority or wrote app", w.Code, w.Body.String())
	}
	p = review.Plan
	if _, err := (grubberyJournal{r.s}).Load(t.Context(), p.RequestID); err != grubbery.ErrMissing {
		t.Fatal("review wrote journal", err)
	}
	begin := protocol.GrubberyAppBeginRequest{Plan: p}
	original := appHTTPRecord(t, r, protocol.GrubberyAppOperationsPath, begin)
	if original.Revision != 0 || f.writes != 0 {
		t.Fatal("begin dispatched effect")
	}
	path := protocol.GrubberyAppOperationsPath + "/" + p.RequestID
	phase := protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}
	skipped := phase
	skipped.StepID = p.Steps[1].ID
	if got := deskHTTPRequest(t, r, "POST", path+"/advance", skipped); got.Code != 400 {
		t.Fatal("skipped original phase", got.Code)
	}
	f.lost = true
	original = appHTTPRecord(t, r, path+"/advance", phase)
	state, _ := original.State(phase.StepID)
	if state != "uncertain" || f.writes != 1 {
		t.Fatal("lost reply not retained", state, f.writes)
	}
	reads := f.reads
	again := appHTTPRecord(t, r, path+"/advance", phase)
	if !reflect.DeepEqual(original, again) || f.reads != reads || f.writes != 1 {
		t.Fatal("uncertain advance probed or redispatched")
	}
	wrong := phase
	wrong.PlanDigest = strings.Repeat("0", 64)
	if got := deskHTTPRequest(t, r, "POST", path+"/reconcile", wrong); got.Code != 409 || f.reads != reads {
		t.Fatal("changed plan accessed native state")
	}
	cache := filepath.Join(r.s.directory, "desk-stages-v1", p.StageID)
	g02Check(t, os.Rename(cache, cache+"-unavailable"))
	policy, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: p.Binding.Target, Gates: apppolicy.Gates{Commit: apppolicy.Block, Publish: apppolicy.Block, CreateDesk: apppolicy.Inherit}})
	g02Check(t, err)
	// Original begin replay succeeds without cache, Commit or native adapter.
	backend := r.b
	r.b = nil
	again = appHTTPRecord(t, r, protocol.GrubberyAppOperationsPath, begin)
	r.b = backend
	if !reflect.DeepEqual(original, again) || f.reads != reads {
		t.Fatal("begin replay required native admission")
	}
	resolved := appHTTPRecord(t, r, path+"/reconcile", phase)
	state, _ = resolved.State(phase.StepID)
	if state != "observed" || f.writes != 1 || !reflect.DeepEqual(resolved.Events[:len(original.Events)], original.Events) {
		t.Fatal("reconciliation rewrote uncertainty or repeated effect")
	}
	reads = f.reads
	again = appHTTPRecord(t, r, path+"/advance", phase)
	if !reflect.DeepEqual(resolved, again) || f.reads != reads || f.writes != 1 {
		t.Fatal("observed replay selected next phase")
	}
	next := phase
	next.StepID = p.Steps[1].ID
	if got := deskHTTPRequest(t, r, "POST", path+"/advance", next); got.Code == 200 || f.writes != 1 {
		t.Fatal("next phase bypassed cache/policy")
	}
}

func TestGrubberySourceHTTPPristineAbandonAndActorBinding(t *testing.T) {
	r, f, p, _ := managedGrubberyFixture(t)
	begin := protocol.GrubberyAppBeginRequest{Plan: p}
	original := appHTTPRecord(t, r, protocol.GrubberyAppOperationsPath, begin)
	// A caller cannot replace the authenticated original actor by editing JSON.
	forged := begin
	forged.Plan.Binding.Actor = "other"
	if got := deskHTTPRequest(t, r, "POST", protocol.GrubberyAppOperationsPath, forged); got.Code == 200 {
		t.Fatal("caller actor accepted")
	}
	reads := f.reads
	r.b = nil
	phase := protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}
	path := protocol.GrubberyAppOperationsPath + "/" + p.RequestID + "/abandon"
	abandoned := appHTTPRecord(t, r, path, phase)
	if !abandoned.Abandoned() || abandoned.Revision != original.Revision+1 || f.writes != 0 || f.reads != reads {
		t.Fatal("pristine abandonment touched native app")
	}
	again := appHTTPRecord(t, r, path, phase)
	if !reflect.DeepEqual(abandoned, again) {
		t.Fatal("abandonment replay changed history")
	}
}
