package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestMoveSettingsPreflight(t *testing.T) {
	for _, missing := range []string{"source", "destination"} {
		t.Run(missing, func(t *testing.T) {
			var mutations atomic.Int32
			endpoint := func(id string) *Client {
				return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(protocol.Header, "1")
					if r.Method != "GET" {
						mutations.Add(1)
						http.Error(w, "unexpected mutation", 500)
						return
					}
					if r.URL.Path == "/hello" {
						features := []string{"host-handoff"}
						if id != missing {
							features = append(features, protocol.RuntimeSettingsHandoffFeature)
						}
						json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, AccessRole: "operator", Features: features})
					} else {
						json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: id, Instances: []protocol.Instance{{ID: "fixture", ConfigRevision: 1, Config: protocol.Configuration{Loom: 34, NativeControl: true}}}})
					}
				})
			}
			source, destination := endpoint("source"), endpoint("destination")
			_, err := MoveStopped(context.Background(), source, destination, "source", "fixture", protocol.PrepareHandoffRequest{DestinationRunner: "destination", DestinationInstance: "arrived", ExpectedRevision: 1, RequestID: "move"}, 0, 0)
			if err == nil || !strings.Contains(err.Error(), "runtime-settings-handoff") || mutations.Load() != 0 {
				t.Fatal("preflight did not refuse before mutation", err, mutations.Load())
			}
		})
	}
}
