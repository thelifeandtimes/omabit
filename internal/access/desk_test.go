package access

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func stageDescriptor(t *testing.T) string {
	t.Helper()
	i := deskwork.StageIntent{Version: 1, ID: strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), Target: apppolicy.Target{InstanceID: "alpha", Desk: "test"}, Files: 1, Bytes: 50000}
	if err := i.Validate(); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(i)
	return base64.RawURLEncoding.EncodeToString(b)
}

func TestBridgeStreamsDeskSourceAndClosesAfterOneUpload(t *testing.T) {
	for _, framing := range []string{"length", "chunked", "unsupported", "observer"} {
		t.Run(framing, func(t *testing.T) {
			socket, keys, grant := bridgeGrantFixture(t)
			if framing == "observer" {
				line, _ := AuthorizedKey([]byte(publicKey(t)), "observer")
				grant, _ = ParseLine(line)
				if err := os.WriteFile(keys, []byte(line), 0600); err != nil {
					t.Fatal(err)
				}
			}
			body, descriptor := strings.Repeat("source bytes", 8192), stageDescriptor(t)
			var effects atomic.Int32
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" {
					features := []string{"ssh-key-grants"}
					if framing != "unsupported" {
						features = append(features, protocol.DeskStageFeature)
					}
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Actor: "ssh-key:" + grant.Fingerprint, AccessRole: grant.Role, Features: features})
					return
				}
				effects.Add(1)
				data, err := io.ReadAll(r.Body)
				if err != nil || string(data) != body || r.URL.Path != protocol.DeskStagesPath || r.Header.Get(protocol.DeskStageHeader) != descriptor || r.Header.Get(Header) != grant.Fingerprint || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Error("source stream or trusted headers changed", err)
				}
				w.Header().Set("Set-Cookie", "private-response")
				io.WriteString(w, "{}")
			})}
			go srv.Serve(ln)
			defer srv.Close()
			req, _ := http.NewRequest("POST", "http://runner"+protocol.DeskStagesPath, strings.NewReader(body))
			if framing == "chunked" {
				req.ContentLength = -1
			}
			req.Header.Set(protocol.Header, "1")
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set(protocol.DeskStageHeader, descriptor)
			req.Header.Set(Header, "forged")
			req.Header.Set("Authorization", "private-client")
			req.Header.Set("Cookie", "private-client")
			var input, output bytes.Buffer
			if err := req.Write(&input); err != nil {
				t.Fatal(err)
			}
			input.WriteString("POST /v1/instances/alpha/operations HTTP/1.1\r\nHost: runner\r\nContent-Length: 2\r\n\r\n{}")
			err = Bridge(context.Background(), &input, &output, socket, keys, grant.Fingerprint)
			if framing == "unsupported" || framing == "observer" {
				if effects.Load() != 0 || framing == "unsupported" && err == nil {
					t.Fatal("unauthorized upload forwarded", err, effects.Load())
				}
			} else if err != nil || effects.Load() != 1 || strings.Contains(output.String(), "private-") || !strings.Contains(output.String(), "Connection: close") {
				t.Fatal("upload not forwarded exactly once", err, effects.Load(), output.String())
			}
		})
	}
}

func TestBridgeRejectsMalformedDeskSourceBeforeForwarding(t *testing.T) {
	descriptor := stageDescriptor(t)
	for _, kind := range []string{"missing", "duplicate", "oversize", "base64", "unknown-field", "invalid-intent", "trailing-json", "zero", "too-large", "content-type", "trailer"} {
		t.Run(kind, func(t *testing.T) {
			socket, keys, grant := bridgeGrantFixture(t) // no server: rejection precedes forwarding
			header, contentType, length, extra := descriptor, "application/octet-stream", "1", ""
			switch kind {
			case "missing":
				header = ""
			case "duplicate":
				extra = protocol.DeskStageHeader + ": " + descriptor + "\r\n"
			case "oversize":
				header = strings.Repeat("a", 8193)
			case "base64":
				header = "!"
			case "unknown-field", "trailing-json":
				data, _ := base64.RawURLEncoding.DecodeString(descriptor)
				if kind == "unknown-field" {
					data = append(data[:len(data)-1], []byte(",\"unknown\":true}")...)
				} else {
					data = append(data, []byte("{}")...)
				}
				header = base64.RawURLEncoding.EncodeToString(data)
			case "invalid-intent":
				header = base64.RawURLEncoding.EncodeToString([]byte("{}"))
			case "zero":
				length = "0"
			case "too-large":
				length = fmt.Sprint(deskwork.MaxBundleBytes + 1)
			case "content-type":
				contentType = "application/json"
			case "trailer":
				extra = "Transfer-Encoding: chunked\r\nTrailer: Extra\r\n"
			}
			input := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: runner\r\n%s: 1\r\n%s: %s\r\nContent-Type: %s\r\nContent-Length: %s\r\n%s\r\nx", protocol.DeskStagesPath, protocol.Header, protocol.DeskStageHeader, header, contentType, length, extra)
			err := Bridge(context.Background(), strings.NewReader(input), io.Discard, socket, keys, grant.Fingerprint)
			if err == nil || !strings.Contains(err.Error(), "invalid source") {
				t.Fatal("framing not rejected before local connection", err)
			}
		})
	}
}

func TestBridgeDeskRequestUsesApplicationDeadline(t *testing.T) {
	socket, keys, grant := bridgeGrantFixture(t)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "development-managed", Actor: "ssh-key:" + grant.Fingerprint, AccessRole: grant.Role, Features: []string{"ssh-key-grants"}})
			return
		}
		time.Sleep(150 * time.Millisecond)
		io.WriteString(w, "{}")
	})}
	go srv.Serve(ln)
	defer srv.Close()
	input := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: runner\r\nConnection: close\r\n\r\n", protocol.DeskDiscoveryPath)
	if err := bridge(context.Background(), strings.NewReader(input), io.Discard, socket, keys, grant.Fingerprint, 50*time.Millisecond); err != nil {
		t.Fatal("valid desk request hit framing deadline", err)
	}
}
