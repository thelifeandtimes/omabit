//go:build linux && amd64

package managed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runtimeHTTPStatus(t *testing.T, r *Server, method, path string, in any) protocol.GrubberyRuntimeRecoveryStatus {
	t.Helper()
	w := deskHTTPRequest(t, r, method, path, in)
	var out protocol.GrubberyRuntimeRecoveryStatus
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Validate() != nil {
		t.Fatal("runtime recovery status refused", w.Code, w.Body.String())
	}
	return out
}

func TestGrubberyRuntimeHTTPReviewConfirmedApplySavedLookupAndOriginalReconcile(t *testing.T) {
	r, f, original, selection := managedRuntimeRecoveryFixture(t)
	g02Check(t, os.RemoveAll(filepath.Join(r.s.directory, "desk-stages-v1")))
	path := protocol.GrubberyRuntimeRecoveriesPath + "/" + original.Plan.RequestID
	reads := f.reads
	status := runtimeHTTPStatus(t, r, "GET", path, nil)
	if status.Receipt != nil || f.reads != reads {
		t.Fatal("empty saved head probed native runtime")
	}
	unknown := deskHTTPRequest(t, r, "GET", protocol.GrubberyRuntimeRecoveriesPath+"/"+strings.Repeat("2", 64), nil)
	if unknown.Code != 404 || f.reads != reads {
		t.Fatal("missing saved original was not a read-only absence", unknown.Code)
	}
	w := deskHTTPRequest(t, r, "POST", protocol.GrubberyRuntimeReviewPath, protocol.GrubberyRuntimeReviewRequest{OriginalRequestID: original.Plan.RequestID, Selection: selection})
	var review protocol.GrubberyRuntimeReview
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &review) != nil || review.Validate() != nil || !reflect.DeepEqual(review.Original, original) {
		t.Fatal("typed fresh review failed", w.Code, w.Body.String())
	}
	q := protocol.GrubberyRuntimeApplyRequest{OriginalRequestID: original.Plan.RequestID, RequestID: strings.Repeat("5", 64), Review: review.Review}
	reads = f.reads
	before := appAllRows(t, r.s.db)
	w = deskHTTPRequest(t, r, "POST", protocol.GrubberyRuntimeRecoveriesPath, q)
	if w.Code != 400 || f.reads != reads || !reflect.DeepEqual(before, appAllRows(t, r.s.db)) {
		t.Fatal("unconfirmed apply admitted or probed native state", w.Code)
	}
	q.Confirm = true
	status = runtimeHTTPStatus(t, r, "POST", protocol.GrubberyRuntimeRecoveriesPath, q)
	if status.Receipt == nil || !reflect.DeepEqual(status.Original, original) || f.writes != 1 {
		t.Fatal("audit apply resolved/changed original uncertainty")
	}
	reads = f.reads
	for _, savedPath := range []string{path, path + "/" + q.RequestID} {
		got := runtimeHTTPStatus(t, r, "GET", savedPath, nil)
		if !reflect.DeepEqual(got, status) || f.reads != reads {
			t.Fatal("saved lookup changed receipt or probed native state")
		}
	}
	phase := protocol.GrubberyAppPhaseRequest{PlanDigest: original.Plan.Digest, StepID: original.Plan.Steps[0].ID}
	got := appHTTPRecord(t, r, protocol.GrubberyAppOperationsPath+"/"+original.Plan.RequestID+"/reconcile", phase)
	if state, _ := got.State(phase.StepID); state != "observed" || f.writes != 1 || !reflect.DeepEqual(got.Events[:len(original.Events)], original.Events) {
		t.Fatal("HTTP phase did not use saved approved runtime without redispatch", state)
	}
	reads = f.reads
	r.b = nil
	r.updateFence.Held = true
	replay := runtimeHTTPStatus(t, r, "POST", protocol.GrubberyRuntimeRecoveriesPath, q)
	lookup := runtimeHTTPStatus(t, r, "GET", path+"/"+q.RequestID, nil)
	if !reflect.DeepEqual(replay.Receipt, status.Receipt) || !reflect.DeepEqual(lookup, replay) || f.reads != reads || f.writes != 1 {
		t.Fatal("lost apply replay or saved lookup required native admission")
	}
}
