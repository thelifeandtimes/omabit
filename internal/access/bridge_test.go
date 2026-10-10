package access

import (
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
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestBridgeRejectsLegacyAndOverwritesUntrustedActor(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "delegation"}[supported], func(t *testing.T) {
			dir := t.TempDir()
			line, err := AuthorizedKey([]byte(publicKey(t)), "operator")
			if err != nil {
				t.Fatal(err)
			}
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
			var posts atomic.Int32
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(Header) != grant.Fingerprint || r.Header.Get("Authorization") != "" {
					t.Error("untrusted headers forwarded")
				}
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" {
					hello := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-1", Mode: "development-managed-fake"}
					if supported {
						hello.Features = []string{"ssh-key-grants"}
						hello.Actor = "ssh-key:" + grant.Fingerprint
						hello.AccessRole = "operator"
					}
					json.NewEncoder(w).Encode(hello)
					return
				}
				posts.Add(1)
				w.Header().Set("Set-Cookie", "secret=must-not-forward")
				json.NewEncoder(w).Encode(protocol.Operation{ID: "op-1"})
			})}
			done := make(chan error, 1)
			go func() { done <- srv.Serve(ln) }()
			defer func() { srv.Close(); <-done }()
			input := "POST /v1/instances/alpha/operations HTTP/1.1\r\nHost: untrusted\r\n" + Header + ": forged\r\nAuthorization: secret\r\nContent-Type: application/json\r\nUrbit-Runner-Protocol: 1\r\nContent-Length: 2\r\nConnection: close\r\n\r\n{}"
			var output bytes.Buffer
			err = Bridge(context.Background(), strings.NewReader(input), &output, socket, keys, grant.Fingerprint)
			if supported {
				if err != nil || posts.Load() != 1 || strings.Contains(output.String(), "must-not-forward") {
					t.Fatalf("bridge: %v posts=%d %s", err, posts.Load(), &output)
				}
			} else if err == nil || posts.Load() != 0 {
				t.Fatal("legacy runner received a delegated mutation")
			}
		})
	}
}

func TestBridgeStreamsOnlyBoundedOperatorPillUploads(t *testing.T) {
	dir := t.TempDir()
	line, _ := AuthorizedKey([]byte(publicKey(t)), "operator")
	grant, _ := ParseLine(line)
	keys := filepath.Join(dir, "keys")
	os.WriteFile(keys, []byte(line), 0600)
	socket := filepath.Join(dir, "rpc.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("pill-bytes", 8192)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	var received atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, Features: []string{"ssh-key-grants"}, Actor: "ssh-key:" + grant.Fingerprint, AccessRole: "operator"})
			return
		}
		data, _ := io.ReadAll(r.Body)
		if string(data) != body || r.Header.Get(protocol.PillHashHeader) != hash || r.Header.Get(Header) != grant.Fingerprint || r.ContentLength != int64(len(body)) {
			t.Error("pill stream identity/body mismatch")
		}
		received.Add(1)
		json.NewEncoder(w).Encode(protocol.BootPill{SHA256: hash, Bytes: int64(len(body))})
	})}
	go srv.Serve(ln)
	defer srv.Close()
	input := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: unused\r\nContent-Type: application/octet-stream\r\n%s: %s\r\n%s: 1\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", protocol.PillUploadPath, protocol.PillHashHeader, hash, protocol.Header, len(body), body)
	var output bytes.Buffer
	if err := Bridge(context.Background(), strings.NewReader(input), &output, socket, keys, grant.Fingerprint); err != nil || received.Load() != 1 {
		t.Fatal(err)
	}
	invalid := strings.Replace(input, protocol.PillUploadPath, "/v1/ships", 1)
	if err := Bridge(context.Background(), strings.NewReader(invalid), &output, socket, keys, grant.Fingerprint); err == nil || received.Load() != 1 {
		t.Fatal("ordinary RPC inherited bulk body permission")
	}
}
