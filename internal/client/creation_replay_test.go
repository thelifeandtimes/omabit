package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCreationReplayRechecksCapabilityAfterReconnect(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(strconv.FormatBool(supported), func(t *testing.T) {
			var hellos, mutations atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					n := hellos.Add(1)
					h := protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "same-runner", Mode: "development-managed"}
					if n == 1 || supported {
						h.Features = []string{protocol.CreationReplayFeature}
					}
					if n == 1 {
						w.Header().Set("Connection", "close")
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				mutations.Add(1)
				if r.Header.Get(protocol.CreationReplayHeader) != "true" {
					t.Error("replay-only header lost")
				}
				w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
				json.NewEncoder(w).Encode(protocol.Operation{ID: "historical-receipt"})
			})
			if _, err := c.Hello(context.Background()); err != nil {
				t.Fatal(err)
			}
			_, err := c.CreateFake(WithCreationReplayOnly(context.Background()), protocol.CreateFakeRequest{ID: "alpha", RequestID: "old-request", Config: protocol.Configuration{Loom: 29}})
			if supported {
				if err != nil || mutations.Load() != 1 {
					t.Fatal("supporting reconnect failed", err)
				}
			} else if err == nil || mutations.Load() != 0 || !strings.Contains(err.Error(), protocol.CreationReplayFeature) {
				t.Fatal("capability-losing reconnect received replay", mutations.Load(), err)
			}
			if hellos.Load() != 2 {
				t.Fatal("reconnected without handshake", hellos.Load())
			}
		})
	}
}
