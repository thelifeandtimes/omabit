package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func guidedImportFixture() protocol.PierImportPreview {
	return protocol.PierImportPreview{RunnerID: "fixture", Source: pierstore.ExternalSource{Path: "/source/pier", Fingerprint: strings.Repeat("d", 64), Bytes: 512 << 30, Files: 4}, StorageRoot: "default", InstanceID: "zod", PierIdentity: protocol.PierIdentity{Ship: "~zod", Kind: "fake", Network: "fakenet"}, Destination: "fakeships/zod/zod", HTTPPort: 18080}
}

func TestLocalImportPreviewByteBoundaries(t *testing.T) {
	for _, kind := range []string{"guided", "restore-source", "restore-destination"} {
		for _, size := range []int64{-1, 0, protocol.MaxPierBytes, protocol.MaxPierBytes + 1, 512 << 30, pierstore.MaxLocalPierBytes, pierstore.MaxLocalPierBytes + 1} {
			t.Run(kind+"/"+strconv.FormatInt(size, 10), func(t *testing.T) {
				c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(protocol.Header, "1")
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/hello" {
						json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "fixture", AccessRole: "operator", Actor: "operator", Features: []string{protocol.ImportRestoreFeature}})
						return
					}
					if kind == "guided" {
						if r.Method != "POST" || r.URL.Path != "/v1/pier-import-preview" {
							t.Error("unexpected mutation or route", r.URL.Path)
						}
						p := guidedImportFixture()
						p.Source.Bytes = size
						json.NewEncoder(w).Encode(p)
					} else {
						if r.Method != "POST" || r.URL.Path != "/v1/instances/zod/restore-import-preview" {
							t.Error("unexpected mutation or route", r.URL.Path)
						}
						p := protocol.ImportRestorePreview{ImportRestoreResult: clientRestoreResult(), RunnerID: "fixture", InstanceID: "zod", ConfigRevision: 2, SourceFiles: 4, DestinationFiles: 4, SourceBytes: 200, DestinationBytes: 200, Stage: "review"}
						if kind == "restore-source" {
							p.SourceBytes = size
						} else {
							p.DestinationBytes = size
						}
						json.NewEncoder(w).Encode(p)
					}
				})
				var err error
				if kind == "guided" {
					_, err = c.PreviewGuidedPierImport(context.Background(), "fixture", "/source/pier", "default")
				} else {
					_, err = c.PreviewImportRestore(context.Background(), "fixture", "zod", 2)
				}
				want := size >= 0 && size <= pierstore.MaxLocalPierBytes
				if (err == nil) != want {
					t.Fatalf("size %d acceptance = %v; want %v", size, err, want)
				}
				if size > pierstore.MaxLocalPierBytes && (err == nil || !strings.Contains(err.Error(), "8 TiB")) {
					t.Fatalf("missing local size diagnostic: %v", err)
				}
			})
		}
	}
}

func TestLargeGuidedImportRetainsPlanGuards(t *testing.T) {
	faults := map[string]func(*protocol.PierImportPreview){
		"runner":               func(p *protocol.PierImportPreview) { p.RunnerID = "other" },
		"source":               func(p *protocol.PierImportPreview) { p.Source.Path = "/other" },
		"root":                 func(p *protocol.PierImportPreview) { p.StorageRoot = "other" },
		"instance":             func(p *protocol.PierImportPreview) { p.InstanceID = "../bad" },
		"ship":                 func(p *protocol.PierImportPreview) { p.Ship = "invalid" },
		"path":                 func(p *protocol.PierImportPreview) { p.Destination = "../outside" },
		"destination-root":     func(p *protocol.PierImportPreview) { p.Destination = "@other/fakeships/zod/zod" },
		"kind":                 func(p *protocol.PierImportPreview) { p.Kind = "moon" },
		"port":                 func(p *protocol.PierImportPreview) { p.HTTPPort = 1 },
		"network":              func(p *protocol.PierImportPreview) { p.Network = "unknown" },
		"digest":               func(p *protocol.PierImportPreview) { p.Source.Fingerprint = "bad" },
		"negative-files":       func(p *protocol.PierImportPreview) { p.Source.Files = -1 },
		"too-many-files":       func(p *protocol.PierImportPreview) { p.Source.Files = 8193 },
		"destination-ship":     func(p *protocol.PierImportPreview) { p.Destination = "fakeships/zod/nec" },
		"destination-instance": func(p *protocol.PierImportPreview) { p.Destination = "fakeships/other/zod" },
		"live-class":           func(p *protocol.PierImportPreview) { p.Network = "livenet" },
	}
	for name, change := range faults {
		t.Run(name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/v1/pier-import-preview" {
					t.Error("unexpected route", r.URL.Path)
				}
				p := guidedImportFixture()
				change(&p)
				json.NewEncoder(w).Encode(p)
			})
			if _, err := c.PreviewGuidedPierImport(context.Background(), "fixture", "/source/pier", "default"); err == nil {
				t.Fatal("changed plan accepted")
			}
		})
	}
}
