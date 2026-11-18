package client

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskExportClientRejectsChangedReplyBindingsAndBytes(t *testing.T) {
	for _, scenario := range []string{"ok", "runner", "mode", "runtime", "revision", "paths", "file-path", "file-bytes", "old", "future", "missing-feature"} {
		t.Run(scenario, func(t *testing.T) {
			q := protocol.DeskSourceExportRequest{Selection: protocol.DeskSelection{Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1}, Baseline: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, Paths: []string{"one.txt"}}
			data := []byte("source\n")
			digest := sha256.Sum256(data)
			reads := 0
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" {
					features := []string{protocol.DeskSourceExportFeature}
					if scenario == "missing-feature" {
						features = nil
					}
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "demo", Features: features})
					return
				}
				if r.Method != "POST" || r.URL.Path != protocol.DeskSourceExportPath {
					t.Error("unexpected request")
					http.Error(w, "unexpected", 400)
					return
				}
				reads++
				copyQ := q
				copyQ.Paths = append([]string{}, q.Paths...)
				out := protocol.DeskSourceExport{Version: 1, RunnerID: "runner", Mode: "demo", Request: copyQ, Files: []deskwork.ExportedFile{{Path: q.Paths[0], ContentBase64: base64.StdEncoding.EncodeToString(data), SHA256: hex.EncodeToString(digest[:])}}, ReadAt: time.Now().UTC()}
				switch scenario {
				case "runner":
					out.RunnerID = "other"
				case "mode":
					out.Mode = "other"
				case "runtime":
					out.Request.Selection.RuntimeID = strings.Repeat("c", 64)
				case "revision":
					out.Request.Baseline.Revision++
				case "paths":
					out.Request.Paths[0] = "other.txt"
				case "file-path":
					out.Files[0].Path = "other.txt"
				case "file-bytes":
					out.Files[0].ContentBase64 = base64.StdEncoding.EncodeToString([]byte("changed"))
				case "old":
					out.ReadAt = time.Now().Add(-protocol.DeskReadTimeout - time.Minute)
				case "future":
					out.ReadAt = time.Now().Add(time.Minute)
				}
				json.NewEncoder(w).Encode(out)
			})
			out, err := c.ExportDeskSource(context.Background(), Identity{RunnerID: "runner", Mode: "demo"}, q)
			if (err == nil) != (scenario == "ok") {
				t.Fatal("invalid reply accepted", scenario, err)
			}
			if scenario == "ok" {
				if out.Validate() != nil {
					t.Fatal("valid reply rejected")
				}
			} else if out.Files != nil {
				t.Fatal("unverified reply returned usable source")
			}
			if scenario == "missing-feature" && reads != 0 {
				t.Fatal("unsupported runner read source")
			}
		})
	}
}
