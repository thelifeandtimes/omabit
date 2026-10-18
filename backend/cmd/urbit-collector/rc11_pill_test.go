package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11CustomHTTPPillRefusedBeforeRunnerContact(t *testing.T) {
	for _, command := range []string{"create-fake", "create-comet", "create-ship"} {
		args := []string{"--runner=fixture", "--expected-runner-id=fixture-id", command}
		if command == "create-ship" {
			args = append(args, "--ship=~sampel-palnet", "--key-file=/missing", "--confirm-network")
		} else {
			args = append(args, "alpha", "--http-port=18080")
			if command == "create-comet" {
				args = append(args, "--confirm-network")
			}
		}
		args = append(args, "--request-id=fixture-request", "--bootstrap=recommended", "--pill=http://example.invalid/custom.pill")
		var errs bytes.Buffer
		if code := run(context.Background(), args, io.Discard, &errs); code != 2 || !strings.Contains(errs.String(), "HTTPS") {
			t.Fatal(command, code, errs.String())
		}
	}
}

func TestRC11TrustedCustomPillDigestVerifiedBeforeUpload(t *testing.T) {
	ctx := context.Background()
	body := []byte("synthetic custom pill bytes")
	path := filepath.Join(t.TempDir(), "custom.pill")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	meta := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(body)), Bytes: int64(len(body))}
	uploads, registrations := 0, 0
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hello":
			reply(t, w, protocol.Hello{ProtocolVersion: 1, AccessRole: "operator", Features: []string{"create-fake", "bootstrap-selection"}})
		case protocol.PillUploadPath:
			uploads++
			got, _ := io.ReadAll(r.Body)
			if !bytes.Equal(got, body) {
				t.Error("upload bytes changed")
			}
			reply(t, w, meta)
		default:
			registrations++
			var req protocol.CreateFakeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Bootstrap == nil || req.Bootstrap.Pill == nil || *req.Bootstrap.Pill != meta {
				t.Error("custom descriptor lost", err)
			}
			reply(t, w, protocol.Operation{})
		}
	})
	args := []string{"--socket", socket, "create-fake", "alpha", "--http-port=18080", "--request-id=fixture-request", "--pill=" + path, fmt.Sprintf("--pill-size=%d", len(body))}
	var errs bytes.Buffer
	if code := run(ctx, append(args, "--pill-sha256="+strings.Repeat("a", 64)), io.Discard, &errs); code != 1 || uploads != 0 || registrations != 0 {
		t.Fatal("untrusted bytes reached runner", code, uploads, registrations, errs.String())
	}
	if code := run(ctx, append(args, "--pill-sha256="+meta.SHA256), io.Discard, &errs); code != 0 || uploads != 1 || registrations != 1 {
		t.Fatal("custom trusted pill failed", code, uploads, registrations, errs.String())
	}
	if _, err := expectedPill(bootpill.RecommendedAlias, meta.SHA256, meta.Bytes); err == nil {
		t.Fatal("custom pin overrode release trust")
	}
}
