//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func stageBundleFixture(t *testing.T, intent deskwork.Intent) (deskwork.StageIntent, []byte) {
	t.Helper()
	dir := t.TempDir()
	g02Check(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0600))
	m, err := deskwork.Snapshot(dir)
	g02Check(t, err)
	var data bytes.Buffer
	g02Check(t, deskwork.WriteBundle(context.Background(), dir, m, &data))
	return deskwork.StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: intent.Target, SourceDigest: m.Digest, Files: 1, Bytes: 6}, data.Bytes()
}
func stageHTTP(t *testing.T, r *Server, intent deskwork.StageIntent, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	header, _ := json.Marshal(intent)
	req := httptest.NewRequest("POST", protocol.DeskStagesPath, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set(protocol.Header, "1")
	req.Header.Set(protocol.DeskStageHeader, base64.RawURLEncoding.EncodeToString(header))
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
}
func TestDeskStageAPIOnlyWritesFrozenCacheAndReconciles(t *testing.T) {
	b := &deskBackendFixture{}
	r, intent := deskServerFixture(t, b)
	v, err := loadInstance(r.s.db, "alpha")
	g02Check(t, err)
	v.Public.ObservedState = "stopped"
	saveRecoveryFixture(t, r.s, v)
	before, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	plan, body := stageBundleFixture(t, intent)
	posted := stageHTTP(t, r, plan, body)
	var out protocol.DeskStage
	if posted.Code != 201 || json.Unmarshal(posted.Body.Bytes(), &out) != nil || out.Receipt.Intent != plan || out.Receipt.Validate() != nil {
		t.Fatal(posted.Code, posted.Body.String())
	}
	replay := stageHTTP(t, r, plan, body)
	var replayOut protocol.DeskStage
	if replay.Code != 201 || json.Unmarshal(replay.Body.Bytes(), &replayOut) != nil || !reflect.DeepEqual(out.Receipt, replayOut.Receipt) {
		t.Fatal("replay changed receipt", replay.Code, replay.Body.String())
	}
	read := deskHTTPRequest(t, r, "GET", protocol.DeskStagesPath+"/"+plan.ID, nil)
	if read.Code != 200 {
		t.Fatal(read.Code, read.Body.String())
	}
	if bytes.Contains(read.Body.Bytes(), []byte(r.s.directory)) || bytes.Contains(read.Body.Bytes(), []byte("hello.txt")) {
		t.Fatal("receipt revealed path or source manifest")
	}
	after, err := loadAppPolicy(r.s.db)
	g02Check(t, err)
	if !reflect.DeepEqual(before, after) || b.calls.Load() != 0 || b.reads.Load() != 0 {
		t.Fatal("stage changed policy or touched runtime")
	}
	entries, err := os.ReadDir(filepath.Join(r.s.directory, "desk-stages-v1"))
	g02Check(t, err)
	if len(entries) != 1 {
		t.Fatal("duplicate stages", entries)
	}
	denied := httptest.NewRecorder()
	r.serveDeskStage(denied, httptest.NewRequest("GET", protocol.DeskStagesPath+"/"+plan.ID, nil), "another-operator", "operator")
	if denied.Code != 409 {
		t.Fatal("stage crossed operators", denied.Code)
	}
	g02Check(t, os.WriteFile(filepath.Join(r.s.directory, "desk-stages-v1", plan.ID, "source", "hello.txt"), []byte("edited"), 0600))
	if got := deskHTTPRequest(t, r, "GET", protocol.DeskStagesPath+"/"+plan.ID, nil); got.Code != 409 {
		t.Fatal("corrupt source acknowledged", got.Code)
	}
}
func TestDeskStageAPIRejectsInvalidOrStaleUpload(t *testing.T) {
	for _, scenario := range []string{"changed-settings", "unknown-target", "wrong-generation", "truncated", "busy"} {
		t.Run(scenario, func(t *testing.T) {
			b := &deskBackendFixture{}
			r, intent := deskServerFixture(t, b)
			plan, body := stageBundleFixture(t, intent)
			switch scenario {
			case "changed-settings":
				plan.SettingsRevision = 1
			case "unknown-target":
				plan.Target.InstanceID = "other"
			case "wrong-generation":
				plan.Target.ProjectID = "other"
				plan.Target.Generation = 2
				plan.Target.Creator = "owner"
			case "truncated":
				body = body[:len(body)-1]
			case "busy":
				r.deskStageMu.Lock()
				defer r.deskStageMu.Unlock()
			}
			posted := stageHTTP(t, r, plan, body)
			if posted.Code < 400 {
				t.Fatal("invalid stage accepted", posted.Code)
			}
			if got := deskHTTPRequest(t, r, "GET", protocol.DeskStagesPath+"/"+plan.ID, nil); got.Code != 404 {
				t.Fatal("failed upload published stage", got.Code, got.Body.String())
			}
			if b.calls.Load() != 0 || b.reads.Load() != 0 {
				t.Fatal("failed stage touched runtime")
			}
		})
	}
}

func TestDeskStageDiscardRequiresOwnerDigestAndResolvedEffects(t *testing.T) {
	for _, scenario := range []string{"ok", "actor", "digest", "planned", "dispatched", "acknowledged", "uncertain", "blocked", "other-target", "observer", "busy"} {
		t.Run(scenario, func(t *testing.T) {
			b := &deskBackendFixture{}
			r, intent := deskServerFixture(t, b)
			plan, body := stageBundleFixture(t, intent)
			posted := stageHTTP(t, r, plan, body)
			if posted.Code != 201 {
				t.Fatal(posted.Code, posted.Body.String())
			}
			var uploaded protocol.DeskStage
			g02Check(t, json.Unmarshal(posted.Body.Bytes(), &uploaded))
			actor, role, digest := uploaded.Receipt.Actor, "operator", plan.SourceDigest
			switch scenario {
			case "actor":
				actor = "someone-else"
			case "digest":
				digest = strings.Repeat("f", 64)
			case "observer":
				role = "observer"
			case "busy":
				r.deskStageMu.Lock()
				defer r.deskStageMu.Unlock()
			case "planned", "dispatched", "acknowledged", "uncertain", "blocked", "other-target":
				if scenario == "other-target" {
					intent.Target.Desk = "another"
				}
				op, _, err := r.s.createDeskOperation("another-operator", intent)
				g02Check(t, err)
				if scenario == "blocked" {
					_, err = r.s.appendDeskStep(op, deskStep(op, "blocked"))
					g02Check(t, err)
				}
				if scenario == "dispatched" || scenario == "acknowledged" || scenario == "uncertain" {
					op, err = r.s.appendDeskStep(op, deskStep(op, "dispatched"))
					g02Check(t, err)
					if scenario != "dispatched" {
						_, err = r.s.appendDeskStep(op, deskStep(op, scenario))
						g02Check(t, err)
					}
				}
			}
			raw, _ := json.Marshal(protocol.DeskStageDiscardRequest{SourceDigest: digest})
			req := httptest.NewRequest("DELETE", protocol.DeskStagesPath+"/"+plan.ID, bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			out := httptest.NewRecorder()
			r.serveDeskStage(out, req, actor, role)
			allowed := scenario == "ok" || scenario == "blocked" || scenario == "other-target"
			if (out.Code == 200) != allowed {
				t.Fatal(scenario, out.Code, out.Body.String())
			}
			_, err := os.Stat(filepath.Join(r.s.directory, "desk-stages-v1", plan.ID))
			if allowed != os.IsNotExist(err) {
				t.Fatal("incorrect cache disposition", err)
			}
			if b.calls.Load() != 0 || b.reads.Load() != 0 {
				t.Fatal("discard touched runtime")
			}
		})
	}
}
