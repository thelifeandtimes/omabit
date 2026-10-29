package main

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func privateFoundationFile(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../../internal/privatechain/testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	return path
}
func privateArgs(t *testing.T) []string {
	return []string{"private-chain-plan", "--label", "dev", "--foundation", privateFoundationFile(t), "--member", "dev-zod:~zod:18381", "--member", "dev-nec:~nec:18382", "--shared-administration"}
}
func TestPrivateChainPlanOfflineAndNotFakeRegistration(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, "unopened")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var first privateChainPlanView
	for _, format := range []string{"json", "text"} {
		var out, errs bytes.Buffer
		args := append([]string{"--config-dir", registry, "--format", format}, privateArgs(t)...)
		if code := run(ctx, args, &out, &errs); code != 0 || errs.Len() != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "json" {
			if err := json.Unmarshal(out.Bytes(), &first); err != nil {
				t.Fatal(err)
			}
			if first.Plan.Mode != privatechain.Mode || first.RuntimeAuthorized || first.ReservationCheck != "not-performed" {
				t.Fatal(first)
			}
			if digest, err := first.Plan.Digest(); err != nil || digest != first.ReviewDigest {
				t.Fatal("invalid review", err)
			}
		} else if !strings.Contains(out.String(), "Offline private-chain plan") || !strings.Contains(out.String(), "No resources reserved or started") {
			t.Fatal(out.String())
		}
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatal("planner opened registry", err)
	}
	plan := filepath.Join(dir, "private-plan.json")
	data, _ := json.Marshal(first)
	if err := os.WriteFile(plan, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	args := []string{"--config-dir", registry, "--runner", "absent", "--expected-runner-id", "runner-a", "project-register", "--plan", plan, "--review-digest", first.ReviewDigest, "--request-id", "attempt-a", "--confirm-reservation"}
	if code := run(ctx, args, &out, &errs); code != 2 || !strings.Contains(errs.String(), "canonical reviewed plan") {
		t.Fatal("private plan reached fake registration", code, errs.String())
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatal("registration opened registry before refusal", err)
	}
}
func TestPrivateChainPlanRefusesSelectionsAndInvalidMembers(t *testing.T) {
	for _, selection := range [][]string{{"--socket", "/absent.sock"}, {"--runner", "absent"}, {"--expected-runner-id", "runner-a"}} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), append(slices.Clone(selection), privateArgs(t)...), &out, &errs); code != 2 || !strings.Contains(errs.String(), "private-chain-plan is offline") {
			t.Fatal(code, errs.String())
		}
	}
	for name, change := range map[string]func([]string) []string{
		"bad port":         func(a []string) []string { a[6] = "dev-zod:~zod:65536"; return a },
		"extra argument":   func(a []string) []string { return append(a, "extra") },
		"unsupported ship": func(a []string) []string { a[6] = "dev-zod:~bud:18381"; return a },
		"no trust":         func(a []string) []string { return append(a, "--shared-administration=false") },
	} {
		t.Run(name, func(t *testing.T) {
			var out, errs bytes.Buffer
			if code := run(context.Background(), change(privateArgs(t)), &out, &errs); code != 2 || out.Len() != 0 || !json.Valid(errs.Bytes()) {
				t.Fatal(code, out.String(), errs.String())
			}
		})
	}
}
func TestPrivateFoundationFileRefusesUnsafeTypesAndContent(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink(privateFoundationFile(t), link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	malformed := filepath.Join(dir, "private")
	if err := os.WriteFile(malformed, []byte(`{"private_key":"do-not-reflect"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{link, fifo, dir, malformed, "relative.json"} {
		if _, err := readPrivateFoundation(p); err == nil || strings.Contains(err.Error(), "do-not-reflect") {
			t.Fatal("unsafe or private input accepted/reflected", p, err)
		}
	}
	large := filepath.Join(dir, "large")
	if err := os.WriteFile(large, bytes.Repeat([]byte(" "), privatechain.MaxFoundationBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrivateFoundation(large); err == nil {
		t.Fatal("oversize accepted")
	}
}
