package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestNewShipCLIUploadsOnlySelectedRecommendedPill(t *testing.T) {
	for _, kind := range []string{"create-fake", "create-comet"} {
		for _, selection := range []string{"bare", "recommended"} {
			t.Run(kind+"-"+selection, func(t *testing.T) {
				body := "test recommended apps pill"
				meta := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(body))), Bytes: int64(len(body))}
				downloads, uploads, registrations := 0, 0, 0
				artifact := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { downloads++; io.WriteString(w, body) }))
				defer artifact.Close()
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hello" {
						reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-pill", Mode: "development-managed", AccessRole: "operator", Features: []string{"create-fake", "create-comet", "bootstrap-selection"}})
						return
					}
					if r.URL.Path == protocol.PillUploadPath {
						uploaded, _ := io.ReadAll(r.Body)
						if string(uploaded) != body || r.ContentLength != meta.Bytes || r.Header.Get(protocol.PillHashHeader) != meta.SHA256 {
							t.Error("upload changed")
						}
						uploads++
						reply(t, w, meta)
						return
					}
					var req protocol.CreateCometRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
					}
					if req.Bootstrap == nil || req.Bootstrap.Kind != selection || selection == "bare" && req.Bootstrap.Pill != nil || selection == "recommended" && (req.Bootstrap.Pill == nil || *req.Bootstrap.Pill != meta) {
						t.Error("wrong selection", req.Bootstrap)
					}
					registrations++
					opKind := "register-fake"
					if kind == "create-comet" {
						opKind = "register-comet"
					}
					reply(t, w, protocol.Operation{ID: "op-boot", InstanceID: req.ID, RequestID: req.RequestID, Kind: opKind, Status: "succeeded"})
				})
				dir := filepath.Join(t.TempDir(), "registry")
				invoke := func(args ...string) (int, string) {
					var out, errs bytes.Buffer
					code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
					return code, errs.String()
				}
				if code, err := invoke("enroll", "target", "--socket", socket); code != 0 {
					t.Fatal(err)
				}
				args := []string{"--runner", "target", "--expected-runner-id", "runner-pill", kind, "new-ship", "--http-port", "18080", "--request-id", "boot-once", "--bootstrap", selection}
				if kind == "create-comet" {
					args = append(args, "--confirm-network")
				}
				if selection == "recommended" {
					if code, _ := invoke(args...); code != 2 || registrations != 0 || uploads != 0 {
						t.Fatal("empty recommendation fell back")
					}
					args = append(args, "--pill", artifact.URL)
				}
				if code, err := invoke(args...); code != 0 {
					t.Fatal(err)
				}
				want := 0
				if selection == "recommended" {
					want = 1
				}
				if downloads != want || uploads != want || registrations != 1 {
					t.Fatal(downloads, uploads, registrations)
				}
				if _, err := bootstrapChoice("bare", "https://example.invalid/recommended.pill"); err == nil {
					t.Fatal("bare accepted configured apps pill")
				}
				if _, err := bootstrapChoice("recommended", strings.TrimSpace(" ")); err == nil {
					t.Fatal("missing recommendation accepted")
				}
			})
		}
	}
}
