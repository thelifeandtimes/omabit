//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestDeskFrontendUploadUsesEphemeralSessionAndNeverRetries(t *testing.T) {
	for _, scenario := range []string{"ok", "logout-lost", "upload-lost", "login-lost", "source-changed", "bad-artifact", "wrong-pier"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			pier := managedPier(f, state)
			if err = os.WriteFile(filepath.Join(pier, ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(filepath.Join(pier, ".urb"), 0700); err != nil {
				t.Fatal(err)
			}
			target := apppolicy.Target{InstanceID: "alpha", Desk: "example"}
			files := []deskwork.SourceFile{{Path: "index.html", Bytes: []byte("fixture frontend")}}
			h := sha256.Sum256(files[0].Bytes)
			manifest, err := deskwork.NewManifest([]deskwork.File{{Path: "index.html", Bytes: int64(len(files[0].Bytes)), SHA256: hex.EncodeToString(h[:])}})
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := deskwork.NewFrontendManifest(manifest)
			if err != nil {
				t.Fatal(err)
			}
			ref := deskwork.FrontendReference{Mode: "glob-ames", Base: "demo", Ship: "~zod", GlobHash: "0v1", DocketHash: "0v2"}
			review := deskwork.FrontendReview{Version: 1, Ship: "~zod", Observation: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, Application: deskwork.AppState{Zest: "live", SourceRevision: 1, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, DocketAvailable: true, DocketHash: ref.DocketHash, FrontendMode: ref.Mode, FrontendState: "glob", FrontendDocketHash: ref.DocketHash}, Frontend: deskwork.FrontendState{SourceRevision: 1, DocketAvailable: true, Stage: "glob", Declared: ref, Current: ref}, ShipSetting: deskwork.DefaultSetting(target.Ship()), DeskSetting: deskwork.DefaultSetting(target), Stage: deskwork.StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: target, SourceDigest: manifest.Digest, Files: 1, Bytes: int64(len(files[0].Bytes))}, ArtifactDigest: artifact.Digest, ExpectedGlobHash: "0v3", UnchangedSourceDigest: strings.Repeat("a", 64)}
			atom := func(v any) string {
				b, _ := json.Marshal(v)
				for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
					b[i], b[j] = b[j], b[i]
				}
				out, _ := json.Marshal("0x" + hex.EncodeToString(b))
				return string(out)
			}
			bodyKey := func(q deskwork.LensRequest) string {
				b, e := q.Body()
				if e != nil {
					t.Fatal(e)
				}
				return string(b)
			}
			obs, _ := deskwork.Inventory("~zod", "example")
			app, _ := deskwork.AppInventory("~zod", "example")
			front, _ := deskwork.FrontendInventory("~zod", "example")
			requests := map[string]func() string{bodyKey(obs): func() string {
				v := review.Observation
				if scenario == "source-changed" {
					v.ClayHash = "0v9"
				}
				return atom(v)
			}, bodyKey(app): func() string { return atom(review.Application) }, bodyKey(front): func() string { return atom(review.Frontend) }}
			logins, uploads, logouts := 0, 0, 0
			f.d.executeInput = func(_ context.Context, args, _ []string, input []byte) (string, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, "lidlut-tabwed-pillex-ridrup") || strings.Contains(joined, "PRIVATE-cookie") {
					t.Fatal("credential escaped stdin")
				}
				if strings.HasSuffix(joined, "http://127.0.0.1:12322") {
					if string(input) == bodyKey(deskwork.FrontendOwnerCode()) {
						return `"lidlut-tabwed-pillex-ridrup\n"`, nil
					}
					if strings.Contains(string(input), `"dojo":"our"`) {
						return `"~zod\n"`, nil
					}
					if value, ok := requests[string(input)]; ok {
						return value(), nil
					}
					t.Fatal("unexpected Lens read")
				}
				response := func(body, headers string) string {
					return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n%s\r\n%s", len(body), headers, body)
				}
				switch args[len(args)-1] {
				case "http://127.0.0.1:8080/~/host", "http://127.0.0.1:8080/~/name":
					return response("~zod", ""), nil
				case "http://127.0.0.1:8080/~/login":
					logins++
					if scenario == "login-lost" {
						return "", errors.New("fixture loss")
					}
					return response("", "Set-Cookie: urbauth-~zod=PRIVATE-cookie; Path=/\r\n"), nil
				case "http://127.0.0.1:8080/docket/upload":
					uploads++
					if !strings.Contains(string(input), "PRIVATE-cookie") || !strings.Contains(string(input), "fixture frontend") {
						t.Fatal("missing exact artifact/session")
					}
					if scenario == "upload-lost" {
						return "", errors.New("fixture loss")
					}
					return response("successfully globbed", ""), nil
				case "http://127.0.0.1:8080/~/logout":
					logouts++
					if scenario == "logout-lost" {
						return "", errors.New("fixture loss")
					}
					return response("", ""), nil
				}
				t.Fatal("unexpected HTTP request", args[len(args)-1])
				return "", errors.New("unexpected")
			}
			runtime, err := openManagedDesk(context.Background(), state, id, f.d)
			if err != nil {
				t.Fatal(err)
			}
			review.Continuity, err = runtime.SourceIdentity(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "wrong-pier" {
				review.Continuity = strings.Repeat("9", 64)
			}
			if err = review.Seal(); err != nil {
				t.Fatal(err)
			}
			if scenario == "bad-artifact" {
				files[0].Bytes = []byte("changed")
			}
			result, err := runtime.UploadFrontend(context.Background(), review, artifact, files)
			if result.Validate() != nil {
				t.Fatal("invalid transfer receipt", result)
			}
			success := scenario == "ok" || scenario == "logout-lost"
			if (err == nil) != success || result.Acknowledged != success {
				t.Fatal("wrong upload result", result, err)
			}
			wantUploads := 0
			if success || scenario == "upload-lost" {
				wantUploads = 1
			}
			if uploads != wantUploads || logins > 1 || logouts > 1 {
				t.Fatal("retried effect", logins, uploads, logouts)
			}
			cleanup := "confirmed"
			if scenario == "logout-lost" || scenario == "login-lost" {
				cleanup = "unconfirmed"
			}
			if scenario == "bad-artifact" {
				cleanup = "not-created"
			}
			if result.SessionCleanup != cleanup {
				t.Fatal("cleanup not separately represented", result)
			}
			b, _ := json.Marshal(result)
			if strings.Contains(string(b), "PRIVATE") || err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("secret in result")
			}
		})
	}
}
