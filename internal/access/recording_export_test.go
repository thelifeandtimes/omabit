package access

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRecordingExportBridgeHasExactBoundedDeadlineAndOperatorScope(t *testing.T) {
	for _, path := range []string{protocol.RecordingExportManifestPath, protocol.RecordingExportPartPath} {
		if deskRequestTimeout(path) != protocol.RecordingExportTimeout+time.Second || (Grant{Role: "observer"}).Allows("POST", path) || !(Grant{Role: "operator"}).Allows("POST", path) {
			t.Fatal("export deadline or role incorrect", path)
		}
		for _, invalid := range []string{path + "/extra", path + "-other"} {
			if deskRequestTimeout(invalid) != 0 {
				t.Fatal("prefix obtained long request allowance")
			}
		}
	}
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
	for _, path := range []string{protocol.RecordingExportManifestPath, protocol.RecordingExportPartPath} {
		input := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: runner\r\nContent-Type: application/json\r\nContent-Length: 2\r\nConnection: close\r\n\r\n{}", path)
		if err := bridge(context.Background(), strings.NewReader(input), io.Discard, socket, keys, grant.Fingerprint, 50*time.Millisecond); err != nil {
			t.Fatal("authorized export hit ordinary framing deadline", err)
		}
	}
}
