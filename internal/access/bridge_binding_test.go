package access

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func bridgeGrantFixture(t *testing.T) (string, string, Grant) {
	t.Helper()
	dir := t.TempDir()
	line, err := AuthorizedKey([]byte(publicKey(t)), "operator")
	if err != nil {
		t.Fatal(err)
	}
	grant, err := ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}
	keys := filepath.Join(dir, "keys")
	if err := os.WriteFile(keys, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "rpc.sock"), keys, grant
}

func bridgeRunnerFixture(t *testing.T, socket string, grant Grant, id, mode string, requests *atomic.Int32) *http.Server {
	t.Helper()
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	s := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: mode, Actor: "ssh-key:" + grant.Fingerprint, AccessRole: grant.Role, Features: []string{"ssh-key-grants", "host-handoff"}})
			return
		}
		requests.Add(1)
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			return // cancellation fixtures intentionally interrupt uploads
		}
		if r.Method == "GET" && protocol.HandoffDataPath(r.URL.Path) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "1")
			io.WriteString(w, "x")
			return
		}
		io.WriteString(w, "{}")
	})}
	go s.Serve(ln)
	t.Cleanup(func() { s.Close() })
	return s
}

func readBridgeResponse(t *testing.T, reader *bufio.Reader, method string) {
	t.Helper()
	response, err := http.ReadResponse(reader, &http.Request{Method: method})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err = io.Copy(io.Discard, response.Body); err != nil || response.StatusCode != 200 {
		t.Fatal(response.StatusCode, err)
	}
}

func TestBridgePinsEveryLocalHandshake(t *testing.T) {
	for _, change := range []string{"id", "mode", "same-identity", "grant-role"} {
		for _, kind := range []string{"rpc", "pill", "handoff-put", "handoff-get"} {
			t.Run(change+"/"+kind, func(t *testing.T) {
				socket, keys, grant := bridgeGrantFixture(t)
				var requests atomic.Int32
				first := bridgeRunnerFixture(t, socket, grant, "runner-a", "development-managed", &requests)
				stream, peer := net.Pipe()
				defer peer.Close()
				peer.SetDeadline(time.Now().Add(3 * time.Second))
				done := make(chan error, 1)
				go func() {
					defer stream.Close()
					done <- Bridge(context.Background(), stream, stream, socket, keys, grant.Fingerprint)
				}()
				reader := bufio.NewReader(peer)
				if _, err := io.WriteString(peer, "GET /hello HTTP/1.1\r\nHost: runner\r\n\r\n"); err != nil {
					t.Fatal(err)
				}
				readBridgeResponse(t, reader, "GET")
				if err := first.Close(); err != nil {
					t.Fatal(err)
				}
				id, mode, replacementGrant := "runner-a", "development-managed", grant
				switch change {
				case "id":
					id = "runner-b"
				case "mode":
					mode = "development-managed-fake"
				case "grant-role":
					replacementGrant.Role = "observer"
				}
				bridgeRunnerFixture(t, socket, replacementGrant, id, mode, &requests)
				method, path, body, contentType := "POST", "/v1/instances/alpha/operations", "{}", "application/json"
				if kind != "rpc" {
					contentType, body = "application/octet-stream", strings.Repeat("x", 65536)
					path = "/v1/handoffs/" + strings.Repeat("a", 32) + "/data"
					method = "PUT"
				}
				if kind == "pill" {
					method, path = "POST", protocol.PillUploadPath
				} else if kind == "handoff-get" {
					method, body = "GET", ""
				}
				writeDone := make(chan error, 1)
				go func() {
					_, err := fmt.Fprintf(peer, "%s %s HTTP/1.1\r\nHost: runner\r\n%s: 1\r\nContent-Type: %s\r\nContent-Length: %d\r\n%s: %s\r\nConnection: close\r\n\r\n%s", method, path, protocol.Header, contentType, len(body), protocol.PillHashHeader, strings.Repeat("a", 64), body)
					writeDone <- err
				}()
				if change == "same-identity" {
					readBridgeResponse(t, reader, method)
				}
				select {
				case err := <-done:
					if change == "same-identity" {
						if err != nil || requests.Load() != 1 {
							t.Fatal("same-ID restart failed", err, requests.Load())
						}
					} else if err == nil || requests.Load() != 0 {
						t.Fatal("replacement received a request/body", err, requests.Load())
					}
				case <-time.After(3 * time.Second):
					t.Fatal("bridge did not finish")
				}
				<-writeDone
			})
		}
	}
}
