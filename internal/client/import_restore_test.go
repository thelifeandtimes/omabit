package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func clientRestoreResult() protocol.ImportRestoreResult {
	return protocol.ImportRestoreResult{Source: "/source/held/pier", RestoreTo: "/source/original", Destination: "/piers/fakeships/zod", RetainAt: "/piers/archive/import-id", ReviewDigest: strings.Repeat("d", 64)}
}

func TestImportRestoreClientBindsPreviewAndReceipt(t *testing.T) {
	for _, kind := range []string{"preview", "apply"} {
		for _, fault := range []string{"none", "old-runner", "observer", "runner", "instance", "revision", "digest", "paths", "stage-or-status", "request-or-size", "actor-or-files", "missing-receipt"} {
			t.Run(kind+"/"+fault, func(t *testing.T) {
				var posts atomic.Int32
				req := protocol.OperationRequest{Kind: protocol.ImportRestoreFeature, RequestID: "restore", ExpectedRevision: 2, RestoreDigest: strings.Repeat("d", 64), ConfirmRestore: true}
				c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(protocol.Header, "1")
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/hello" {
						h := protocol.Hello{ProtocolVersion: 1, RunnerID: "fixture", Mode: "development-managed", AccessRole: "operator", Actor: "operator", Features: []string{protocol.ImportRestoreFeature}}
						switch fault {
						case "old-runner":
							h.Features = nil
						case "observer":
							h.AccessRole = "observer"
						case "runner":
							h.RunnerID = "changed"
						}
						json.NewEncoder(w).Encode(h)
						return
					}
					posts.Add(1)
					result := clientRestoreResult()
					if fault == "digest" {
						result.ReviewDigest = "short"
					}
					if fault == "paths" {
						result.RetainAt = result.Source
					}
					if kind == "preview" {
						if r.URL.Path != "/v1/instances/zod/restore-import-preview" || r.Method != "POST" {
							t.Error("wrong preview route")
						}
						var body protocol.ImportRestorePreviewRequest
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ExpectedRevision != 2 {
							t.Error("unbound revision", err)
						}
						out := protocol.ImportRestorePreview{ImportRestoreResult: result, RunnerID: "fixture", InstanceID: "zod", ConfigRevision: 2, SourceFiles: 4, SourceBytes: 200, DestinationFiles: 1, DestinationBytes: 20, Stage: "review"}
						switch fault {
						case "instance":
							out.InstanceID = "other"
						case "revision":
							out.ConfigRevision++
						case "stage-or-status":
							out.Stage = "invented"
						case "request-or-size":
							out.DestinationBytes = -1
						case "actor-or-files":
							out.SourceFiles = 8193
						case "missing-receipt":
							out.Continuation = true
						}
						json.NewEncoder(w).Encode(out)
						return
					}
					if r.URL.Path != "/v1/instances/zod/operations" || r.Method != "POST" {
						t.Error("wrong operation route")
					}
					var body protocol.OperationRequest
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body != req {
						t.Error("unbound request", err)
					}
					now := time.Now()
					op := protocol.Operation{ID: "restore-operation", RequestID: req.RequestID, Kind: req.Kind, InstanceID: "zod", ConfigRevision: 2, Actor: "operator", Status: "succeeded", FinishedAt: &now, ImportRestoration: &result}
					switch fault {
					case "instance":
						op.InstanceID = "other"
					case "revision":
						op.ConfigRevision++
					case "stage-or-status":
						op.Status = "invented"
					case "request-or-size":
						op.RequestID = "other"
					case "actor-or-files":
						op.Actor = "other"
					case "missing-receipt":
						op.ImportRestoration = nil
					}
					json.NewEncoder(w).Encode(op)
				})
				var err error
				if kind == "preview" {
					_, err = c.PreviewImportRestore(context.Background(), "fixture", "zod", 2)
				} else {
					_, err = c.RestoreImport(context.Background(), "fixture", "zod", req)
				}
				if (err == nil) != (fault == "none") {
					t.Fatal("unexpected validation result", err)
				}
				if (fault == "old-runner" || fault == "observer" || fault == "runner") && posts.Load() != 0 {
					t.Fatal("sent restoration without authority")
				}
			})
		}
	}
}
