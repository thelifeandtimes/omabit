package access

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestBridgeStreamsHandoffInBothDirectionsAndRejectsTrailingRPC(t *testing.T) {
	dir := t.TempDir()
	line, _ := AuthorizedKey([]byte(publicKey(t)), "operator")
	grant, _ := ParseLine(line)
	keys := filepath.Join(dir, "keys")
	if err := os.WriteFile(keys, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "rpc.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("private-pier-data\x00", 100000)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	path := "/v1/handoffs/" + strings.Repeat("a", 32) + "/data"
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.Header.Get(Header) != grant.Fingerprint {
			t.Error("unbound actor")
		}
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, Features: []string{"ssh-key-grants", "host-handoff"}, Actor: "ssh-key:" + grant.Fingerprint, AccessRole: "operator"})
			return
		}
		if r.URL.Path != path {
			t.Error("trailing RPC forwarded")
		}
		w.Header().Set("Set-Cookie", "must-not-forward")
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.Header().Set(protocol.HandoffHashHeader, hash)
			io.WriteString(w, body)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != body || r.Header.Get(protocol.HandoffHashHeader) != hash {
			t.Error("stream changed", err)
		}
		w.WriteHeader(202)
		io.WriteString(w, `{"state":"receiving"}`)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	for _, method := range []string{"GET", "PUT"} {
		payload := ""
		length := 0
		if method == "PUT" {
			payload = body
			length = len(body)
		}
		input := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: runner\r\n%s: 1\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n%s: %s\r\n\r\n%sPOST /unexpected HTTP/1.1\r\nHost: runner\r\nContent-Length: 0\r\n\r\n", method, path, protocol.Header, length, protocol.HandoffHashHeader, hash, payload)
		var output bytes.Buffer
		if err = Bridge(context.Background(), strings.NewReader(input), &output, socket, keys, grant.Fingerprint); err != nil {
			t.Fatal(method, err)
		}
		resp, err := http.ReadResponse(bufio.NewReader(&output), &http.Request{Method: method})
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.Header.Get("Set-Cookie") != "" || !resp.Close {
			t.Fatal("response framing", err)
		}
		if method == "GET" && (string(data) != body || resp.Header.Get(protocol.HandoffHashHeader) != hash) {
			t.Fatal("outbound data changed")
		}
	}
}
