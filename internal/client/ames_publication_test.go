package client

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPublishAmesCapabilityAndReceiptBinding(t *testing.T) {
	for _, scenario := range []string{"success", "old-runner", "observer", "mode", "instance", "request", "revision", "actor", "kind", "unfinished"} {
		t.Run(scenario, func(t *testing.T) {
			var posts atomic.Int32
			req := protocol.PublishAmesRequest{RequestID: "publish", ExpectedRevision: 4, AmesPort: 34567, ConfirmNetwork: true}
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "fixture", Mode: "development-managed", Actor: "operator", AccessRole: "operator", Features: []string{protocol.AmesPublicationFeature}}
					switch scenario {
					case "old-runner":
						h.Features = nil
					case "observer":
						h.AccessRole = "observer"
					case "mode":
						h.Mode = "development-managed-fake"
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				posts.Add(1)
				if r.Method != "POST" || r.URL.Path != "/v1/instances/legacy/publish-ames" {
					t.Error("wrong publication route", r.Method, r.URL.Path)
				}
				var got protocol.PublishAmesRequest
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got != req {
					t.Error("publication request changed", got, err)
				}
				now := time.Now().UTC()
				op := protocol.Operation{ID: "op-publication", InstanceID: "legacy", RequestID: req.RequestID, Actor: "operator", Kind: protocol.AmesPublicationFeature, Status: "succeeded", ConfigRevision: 5, FinishedAt: &now}
				switch scenario {
				case "instance":
					op.InstanceID = "other"
				case "request":
					op.RequestID = "other"
				case "revision":
					op.ConfigRevision = 4
				case "actor":
					op.Actor = "other"
				case "kind":
					op.Kind = "start"
				case "unfinished":
					op.FinishedAt = nil
				}
				json.NewEncoder(w).Encode(op)
			})
			_, err := c.PublishAmes(context.Background(), "legacy", req)
			if (err == nil) != (scenario == "success") {
				t.Fatal("incorrect capability/receipt validation", err)
			}
			want := int32(1)
			if scenario == "old-runner" || scenario == "observer" || scenario == "mode" {
				want = 0
			}
			if posts.Load() != want {
				t.Fatal("unexpected mutation count", posts.Load())
			}
		})
	}
}
