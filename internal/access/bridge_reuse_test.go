package access

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPersistentBridgeRechecksRevocationBeforeEveryRequest(t *testing.T) {
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
	if err = os.WriteFile(keys, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, Features: []string{"ssh-key-grants"}, Actor: "ssh-key:" + grant.Fingerprint, AccessRole: "operator"})
			return
		}
		requests.Add(1)
		if err := os.WriteFile(keys, nil, 0600); err != nil {
			t.Error(err)
		}
		io.WriteString(w, `{"runner_id":"runner"}`)
	})}
	go server.Serve(listener)
	defer server.Close()
	request := "GET /v1/snapshot HTTP/1.1\r\nHost: runner\r\nUrbit-Runner-Protocol: 1\r\n\r\n"
	var output bytes.Buffer
	if err = Bridge(context.Background(), strings.NewReader(request+request), &output, socket, keys, grant.Fingerprint); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(&output)
	for _, want := range []int{200, 403} {
		response, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode != want {
			t.Fatal(response.StatusCode, want)
		}
	}
	if requests.Load() != 1 {
		t.Fatal("revoked request reached runner", requests.Load())
	}
}
