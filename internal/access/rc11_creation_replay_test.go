package access

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11BridgeCreationReplayHeaderAndSameConnectionCapability(t *testing.T) {
	for _, path := range []string{"/v1/instances", "/v1/comets", "/v1/ships"} {
		for _, scenario := range []string{"supported", "old-server", "lost-capability", "malformed", "multiple"} {
			t.Run(path+"/"+scenario, func(t *testing.T) {
				socket, keys, grant := bridgeGrantFixture(t)
				var supports atomic.Bool
				supports.Store(scenario != "old-server")
				var mutations, handshakes atomic.Int32
				listener, err := net.Listen("unix", socket)
				if err != nil {
					t.Fatal(err)
				}
				const body = `{"request_id":"old-request","key":"synthetic-fixture","config":{"loom":29}}`
				server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get(Header) != grant.Fingerprint {
						t.Error("bridge lost authenticated actor")
					}
					w.Header().Set(protocol.Header, "1")
					if r.URL.Path == "/hello" {
						handshakes.Add(1)
						features := []string{"ssh-key-grants"}
						if supports.Load() {
							features = append(features, protocol.CreationReplayFeature)
						}
						json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "replay-runner", Mode: "development-managed", Actor: "ssh-key:" + grant.Fingerprint, AccessRole: "operator", Features: features})
						return
					}
					mutations.Add(1)
					got, err := io.ReadAll(r.Body)
					if err != nil || string(got) != body || r.URL.Path != path || r.Header.Get(protocol.CreationReplayHeader) != "true" || len(r.Header.Values(protocol.CreationReplayHeader)) != 1 {
						t.Error("bridge changed replay header/path/body", err)
					}
					json.NewEncoder(w).Encode(protocol.Operation{ID: "historical-receipt", RequestID: "old-request", Actor: "ssh-key:" + grant.Fingerprint})
				})}
				served := make(chan error, 1)
				go func() { served <- server.Serve(listener) }()
				t.Cleanup(func() { server.Close(); <-served })
				stream, peer := net.Pipe()
				defer peer.Close()
				if err := peer.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() {
					defer stream.Close()
					done <- Bridge(context.Background(), stream, stream, socket, keys, grant.Fingerprint)
				}()
				reader := bufio.NewReader(peer)
				// The retained SSH stream has already observed the same runner. A
				// subsequent inner reconnect must independently check the feature.
				if _, err := io.WriteString(peer, "GET /hello HTTP/1.1\r\nHost: runner\r\n\r\n"); err != nil {
					t.Fatal(err)
				}
				readBridgeResponse(t, reader, "GET")
				if scenario == "lost-capability" {
					supports.Store(false)
				}
				prior := handshakes.Load()
				header := protocol.CreationReplayHeader + ": true\r\n"
				if scenario == "malformed" {
					header = protocol.CreationReplayHeader + ": false\r\n"
				}
				if scenario == "multiple" {
					header += protocol.CreationReplayHeader + ": true\r\n"
				}
				writeDone := make(chan error, 1)
				go func() {
					_, err := fmt.Fprintf(peer, "POST %s HTTP/1.1\r\nHost: runner\r\n%s: 1\r\nContent-Type: application/json\r\nContent-Length: %d\r\n%sConnection: close\r\n\r\n%s", path, protocol.Header, len(body), header, body)
					writeDone <- err
				}()
				if scenario == "supported" {
					response, err := http.ReadResponse(reader, &http.Request{Method: "POST"})
					if err != nil {
						t.Fatal(err)
					}
					data, err := io.ReadAll(response.Body)
					response.Body.Close()
					if err != nil || response.StatusCode != 200 || !bytes.Contains(data, []byte("historical-receipt")) {
						t.Fatal("replay response changed", err, response.StatusCode)
					}
				}
				select {
				case err := <-done:
					if scenario == "supported" {
						if err != nil || mutations.Load() != 1 {
							t.Fatal(err, mutations.Load())
						}
					} else if err == nil || mutations.Load() != 0 {
						t.Fatal("guarded request reached unsafe runner", err, mutations.Load())
					}
					if scenario == "old-server" || scenario == "lost-capability" {
						if !strings.Contains(err.Error(), protocol.CreationReplayFeature) || handshakes.Load() != prior+1 {
							t.Fatal("refusal was not same-connection capability validation", err)
						}
					}
				case <-time.After(5 * time.Second):
					t.Fatal("bridge did not finish")
				}
				<-writeDone
			})
		}
	}
}
