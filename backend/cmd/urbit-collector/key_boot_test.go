package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestBootKeyReaderIsPrivateBoundedAndDoesNotReflectContents(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ship.key")
	key := "0w" + strings.Repeat("a", 128)
	if err := os.WriteFile(p, []byte(key+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readBootKey(p)
	if err != nil || got != key {
		t.Fatal("valid private key refused")
	}
	if err := os.Chmod(p, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootKey(p); err == nil || strings.Contains(err.Error(), key) {
		t.Fatal("public or exposed key accepted")
	}
	if err := os.Chmod(p, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootKey(link); err == nil {
		t.Fatal("key symlink accepted")
	}
	if err := os.WriteFile(p, []byte(key+strings.Repeat("a", 2048)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootKey(p); err == nil {
		t.Fatal("oversized key accepted")
	}
}

func TestGeneratedMoonCLIWithAutomaticDefaultsAndSpecificTypoError(t *testing.T) {
	var posts atomic.Int32
	key := "0w" + strings.Repeat("a", 128)
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-moon", Mode: "development-managed", AccessRole: "operator", Features: []string{"key-boot-options"}})
			return
		}
		if r.Method != "POST" || r.URL.Path != "/v1/ships" {
			t.Error("unexpected request", r.URL.Path)
		}
		var req protocol.CreateShipRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Ship != "~dibpen-fogweg-dozzod-sarlev" || req.ID != "dibpen-fogweg-dozzod-sarlev" || req.PierPath != "livenet/moons/dibpen-fogweg-dozzod-sarlev/dibpen-fogweg-dozzod-sarlev" || req.Config.HTTPPort != 0 || req.Config.AmesPort != 0 || req.Key != key || !req.ConfirmNetwork {
			t.Error("moon defaults or secret transport changed")
		}
		posts.Add(1)
		reply(t, w, protocol.Operation{ID: "op-moon", InstanceID: req.ID, RequestID: req.RequestID, Kind: "register-ship", Status: "succeeded"})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	file := filepath.Join(t.TempDir(), "fixture.key")
	if err := os.WriteFile(file, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	invoke := func(args ...string) (int, string, string) {
		var out, errs bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
		return code, out.String(), errs.String()
	}
	if code, _, err := invoke("enroll", "host", "--socket", socket); code != 0 {
		t.Fatal(err)
	}
	base := []string{"--runner", "host", "--expected-runner-id", "runner-moon", "create-ship", "--ship"}
	args := append(append([]string{}, base...), "~dipben-fogweg-dozzod-sarlev", "--key-file", file, "--request-id", "moon-typo", "--confirm-network")
	if code, _, err := invoke(args...); code != 2 || !strings.Contains(err, `prefix syllable`) || posts.Load() != 0 || strings.Contains(err, key) {
		t.Fatal("missing safe validation error", code, err)
	}
	args = append(append([]string{}, base...), "~dibpen-fogweg-dozzod-sarlev", "--key-file", file, "--request-id", "moon-correct", "--confirm-network")
	code, out, err := invoke(args...)
	if code != 0 || posts.Load() != 1 || strings.Contains(out+err, key) {
		t.Fatal("valid moon registration failed or disclosed a key", code, err)
	}
}

func TestBootKeyPasteAndExplicitFilePrivacy(t *testing.T) {
	key := "0w" + strings.Repeat("a", 128)
	if got, err := readBootKeyInput(strings.NewReader("  " + key + "\n")); err != nil || got != key {
		t.Fatal("paste failed", err)
	}
	for _, value := range []string{"", "not-a-key", key + strings.Repeat("a", protocol.MaxBootKeyBytes)} {
		if _, err := readBootKeyInput(strings.NewReader(value)); err == nil || strings.Contains(err.Error(), value) && value != "" {
			t.Fatal("invalid paste accepted or disclosed")
		}
	}
	path := filepath.Join(t.TempDir(), "owned.key")
	if err := os.WriteFile(path, []byte(key), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootKeyFile(path, false); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatal("missing permission guidance", err)
	}
	if got, err := readBootKeyFile(path, true); err != nil || got != key {
		t.Fatal("explicit privacy repair failed", err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("file not private")
	}
	link := path + "-link"
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readBootKeyFile(path, true); err == nil {
		t.Fatal("hard-linked key accepted")
	}
}
