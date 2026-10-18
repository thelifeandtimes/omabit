package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPublishAmesCLIUsesPinnedExplicitPublication(t *testing.T) {
	for _, scenario := range []string{"success", "old-runner", "observer", "replaced-runner", "missing-port", "missing-confirmation", "missing-revision", "missing-request", "wrong-pin"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			var posts atomic.Int32
			var changed atomic.Bool
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "ames-fixture", Mode: "development-managed", Actor: "operator", AccessRole: "operator", Features: []string{protocol.AmesPublicationFeature}}
					if scenario == "old-runner" {
						h.Features = []string{"snapshot", "configuration-revisions"}
					}
					if scenario == "observer" {
						h.AccessRole = "observer"
					}
					if scenario == "replaced-runner" && changed.Load() {
						h.RunnerID = "replacement"
					}
					reply(t, w, h)
					return
				}
				posts.Add(1)
				if r.Method != "POST" || r.URL.Path != "/v1/instances/legacy/publish-ames" {
					t.Error("unexpected mutation", r.Method, r.URL.Path)
				}
				var got protocol.PublishAmesRequest
				want := protocol.PublishAmesRequest{RequestID: "publication", ExpectedRevision: 3, AmesPort: 34567, ConfirmNetwork: true}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got != want {
					t.Error("publication request lost explicit input", got, err)
				}
				now := time.Now().UTC()
				reply(t, w, protocol.Operation{ID: "op-publication", InstanceID: "legacy", RequestID: "publication", Kind: protocol.AmesPublicationFeature, Status: "succeeded", Actor: "operator", ConfigRevision: 4, FinishedAt: &now})
			})
			dir := filepath.Join(t.TempDir(), "collector")
			var out, errs bytes.Buffer
			if code := run(ctx, []string{"--config-dir", dir, "enroll", "fixture", "--socket", socket}, &out, &errs); code != 0 {
				t.Fatal("fixture enrollment failed", code, errs.String())
			}
			changed.Store(true)
			out.Reset()
			errs.Reset()
			pin := "ames-fixture"
			if scenario == "wrong-pin" {
				pin = "other-fixture"
			}
			args := []string{"--config-dir", dir, "--runner", "fixture", "--expected-runner-id", pin, "publish-ames", "legacy"}
			for _, field := range []struct{ scenario, flag string }{{"missing-port", "--ames-port=34567"}, {"missing-confirmation", "--confirm-network"}, {"missing-revision", "--revision=3"}, {"missing-request", "--request-id=publication"}} {
				if scenario != field.scenario {
					args = append(args, field.flag)
				}
			}
			code := run(ctx, args, &out, &errs)
			if scenario == "success" {
				if code != 0 || posts.Load() != 1 {
					t.Fatal("publication failed", code, errs.String(), posts.Load())
				}
				var op protocol.Operation
				if err := json.Unmarshal(out.Bytes(), &op); err != nil || op.ConfigRevision != 4 {
					t.Fatal("CLI omitted new revision receipt", err)
				}
			} else if code == 0 || posts.Load() != 0 {
				t.Fatal("guarded CLI sent a mutation", code, posts.Load(), errs.String())
			}
		})
	}
}
