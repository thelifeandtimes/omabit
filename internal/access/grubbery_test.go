package access

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestBridgeGrubberySourceBodyCapabilityAndRoleBounds(t *testing.T) {
	for _, kind := range []string{"bounded", "unsupported", "oversize", "ordinary-path"} {
		t.Run(kind, func(t *testing.T) {
			socket, keys, grant := bridgeGrantFixture(t)
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			var posts atomic.Int32
			size := 20000
			path := protocol.GrubberyAppOperationsPath
			if kind == "oversize" {
				size = protocol.GrubberyAppRequestBytes + 1
			}
			if kind == "ordinary-path" {
				path = "/v1/projects"
			}
			body := bytes.Repeat([]byte(" "), size)
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" {
					features := []string{"ssh-key-grants"}
					if kind != "unsupported" {
						features = append(features, protocol.GrubberyAppSourceFeature)
					}
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "ssh-key:" + grant.Fingerprint, AccessRole: grant.Role, Features: features})
					return
				}
				posts.Add(1)
				got, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(got, body) || r.Header.Get(Header) != grant.Fingerprint {
					t.Error("bounded body or authenticated actor changed")
				}
				io.WriteString(w, "{}")
			})}
			go srv.Serve(ln)
			defer srv.Close()
			input := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: runner\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", path, len(body), body)
			err = Bridge(t.Context(), strings.NewReader(input), io.Discard, socket, keys, grant.Fingerprint)
			if kind == "bounded" {
				if err != nil || posts.Load() != 1 {
					t.Fatal("bounded source request refused", err, posts.Load())
				}
			} else if err == nil || posts.Load() != 0 {
				t.Fatal("unsafe request forwarded", kind, err, posts.Load())
			}
		})
	}
	path := protocol.GrubberyAppOperationsPath + "/" + strings.Repeat("a", 64)
	if (Grant{Role: "observer"}).Allows("GET", path) || (Grant{Role: "observer"}).Allows("POST", path+"/advance") {
		t.Fatal("observer app authority widened")
	}
	if deskRequestTimeout(path+"/advance") != protocol.GrubberyAppTimeout || deskRequestTimeout(path+"/unknown") != 0 {
		t.Fatal("unbounded path received app deadline")
	}
}
