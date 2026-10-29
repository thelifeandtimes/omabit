//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPublicRecoveryActionRefusesWithoutDispatch(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		t.Run(kind, func(t *testing.T) {
			base := publicRecoveryActionFixture(t, kind)
			before := append([]string(nil), base.b.calls...)
			var rows int
			g02Check(t, base.r.s.db.QueryRow("SELECT count(*) FROM operations").Scan(&rows))
			savedBackend := base.r.b
			for _, scenario := range []string{"observer", "unsupported", "confirmation", "revision", "history", "review", "quiesced", "unknown-project", "unknown-field", "bad-replay-header", "sequence", "checkpoint"} {
				t.Run(scenario, func(t *testing.T) {
					f := base
					f.req.RequestID = "refused-" + scenario
					f.r.keys = ""
					f.r.b = savedBackend
					f.r.updateFence.Held = false
					want := 409
					switch scenario {
					case "unsupported":
						f.r.b = &struct{ backend }{}
					case "confirmation":
						f.req.ConfirmAction = false
						want = 400
					case "revision":
						f.req.ExpectedRevision++
					case "history":
						f.req.HistoryDigest = strings.Repeat("a", 64)
					case "review":
						f.req.ReviewDigest = strings.Repeat("a", 64)
					case "quiesced":
						f.r.updateFence.Held = true
					case "unknown-project":
						f.project = "absent-project"
					case "observer":
						want = 403
					case "unknown-field", "bad-replay-header":
						want = 400
					case "sequence":
						if kind == "start" {
							s := uint64(0)
							f.req.ExpectedSequence = &s
							want = 400
						} else {
							s := *f.req.ExpectedSequence + 1
							f.req.ExpectedSequence = &s
						}
					case "checkpoint":
						f.req.CheckpointDigest = strings.Repeat("a", 64)
						if kind == "start" {
							want = 400
						}
					}
					raw, err := json.Marshal(f.req)
					g02Check(t, err)
					if scenario == "unknown-field" {
						raw = append(raw[:len(raw)-1], []byte(`,"surprise":true}`)...)
					}
					req := httptest.NewRequest("POST", protocol.ProjectRecoveryActionPath(f.project, f.parent, kind), bytes.NewReader(raw))
					req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
					req.Header.Set(protocol.Header, "1")
					req.Header.Set("Content-Type", "application/json")
					if scenario == "bad-replay-header" {
						req.Header.Set(protocol.CreationReplayHeader, "sometimes")
					}
					if scenario == "observer" {
						line, key := grantLine(t, "observer")
						f.r.keys = filepath.Join(privateDir(t), "keys")
						g02Check(t, os.WriteFile(f.r.keys, []byte(line+"\n"), 0600))
						req.Header.Set(access.Header, key)
					}
					w := httptest.NewRecorder()
					f.r.ServeHTTP(w, req)
					if w.Code != want || f.r.failed.Load() || f.r.updateWorkers != 0 || !reflect.DeepEqual(before, f.b.calls) {
						t.Fatal(w.Code, w.Body.String(), f.r.failed.Load())
					}
					var now int
					g02Check(t, f.r.s.db.QueryRow("SELECT count(*) FROM operations").Scan(&now))
					if now != rows {
						t.Fatal("refusal changed receipts", now, rows)
					}
				})
			}
		})
	}
}
func TestPublicRecoveryActionBeforeLaunchIsPrerequisiteRefusal(t *testing.T) {
	r, _, id, q := projectQueueFixture(t)
	r.uid = uint32(os.Geteuid())
	r.b = &queuedRecoveryBackend{}
	f := publicRecoveryFixture{r: r, project: id, req: protocol.ProjectRecoveryActionRequest{RequestID: "before-launch", ExpectedRevision: 1, ReviewDigest: q.ReviewDigest, HistoryDigest: strings.Repeat("a", 64), ConfirmAction: true}}
	w := publicRecoveryHTTP(t, f, "start", false)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "project_not_launched") || r.failed.Load() || r.updateWorkers != 0 {
		t.Fatal(w.Code, w.Body.String(), r.failed.Load())
	}
}
