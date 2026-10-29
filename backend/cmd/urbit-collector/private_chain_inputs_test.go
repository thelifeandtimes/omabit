package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func TestPrivateChainInputHelpIsOffline(t *testing.T) {
	registry := filepath.Join(t.TempDir(), "unopened")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tail := range [][]string{{"private-chain-inputs", "--help"}, {"--format", "text", "help", "projects"}, {"--help"}} {
		var out, errs bytes.Buffer
		args := append([]string{"--config-dir", registry, "--runner", "absent", "--socket", "/absent.sock", "--expected-runner-id", "invalid selection"}, tail...)
		if code := run(ctx, args, &out, &errs); code != 0 || errs.Len() != 0 || !strings.Contains(out.String(), "private-chain-inputs") {
			t.Fatal(code, out.String(), errs.String())
		}
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatal("help opened registry", err)
	}
}

func TestPrivateChainInputInvalidSelectionAndCancellationHaveNoEffects(t *testing.T) {
	registry := filepath.Join(t.TempDir(), "unopened")
	command := []string{"private-chain-inputs", "--directory", "/absent/sealed-inputs", "--manifest-sha256", strings.Repeat("a", 64), "--foundation-digest", strings.Repeat("b", 64)}
	for _, selection := range [][]string{{"--runner", "absent"}, {"--socket", "/absent.sock"}, {"--expected-runner-id", "runner-a"}} {
		var out, errs bytes.Buffer
		args := append([]string{"--config-dir", registry}, selection...)
		args = append(args, command...)
		if code := run(context.Background(), args, &out, &errs); code != 2 || out.Len() != 0 || !strings.Contains(errs.String(), "local and offline") {
			t.Fatal(code, out.String(), errs.String())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, errs bytes.Buffer
	if code := run(ctx, append([]string{"--config-dir", registry}, command...), &out, &errs); code != 1 || out.Len() != 0 || !strings.Contains(errs.String(), "context canceled") {
		t.Fatal(code, out.String(), errs.String())
	}
	for _, args := range [][]string{{"private-chain-inputs"}, {"private-chain-inputs", "extra"}, append(append([]string{}, command...), "extra")} {
		out.Reset()
		errs.Reset()
		if code := run(context.Background(), append([]string{"--config-dir", registry}, args...), &out, &errs); code != 2 || out.Len() != 0 || !json.Valid(errs.Bytes()) {
			t.Fatal(code, out.String(), errs.String())
		}
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatal("input inspection opened registry", err)
	}
}

func TestPrivateChainInputViewsPreserveNoAuthority(t *testing.T) {
	result := privateChainInputsView{Format: privatechain.RuntimeInputFormat, ManifestSHA256: strings.Repeat("a", 64), FoundationDigest: strings.Repeat("b", 64), Files: 10, Bytes: 100}
	for _, format := range []string{"json", "text"} {
		var out bytes.Buffer
		if err := newOutput(&out, format, false).Encode(result); err != nil {
			t.Fatal(err)
		}
		if format == "json" {
			var got privateChainInputsView
			if err := json.Unmarshal(out.Bytes(), &got); err != nil || got != result || got.RuntimeAuthorized {
				t.Fatal(err, out.String())
			}
		}
		if format == "text" && (!strings.Contains(out.String(), "Runtime authorized: no") || !strings.Contains(out.String(), "No inputs executed")) {
			t.Fatal(out.String())
		}
	}
}
