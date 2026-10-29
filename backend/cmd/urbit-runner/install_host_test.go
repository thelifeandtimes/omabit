package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
)

func TestInstallHostRequiresAcknowledgementAndBoundedPublicInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.pub")
	if err := os.WriteFile(path, []byte("public fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	install := func(context.Context, string, []byte, string, func(string)) (hostinstall.HostResult, error) {
		called = true
		return hostinstall.HostResult{Ready: true}, nil
	}
	for _, args := range [][]string{{}, {"--bundle", "relative", "--public-key", path, "--acknowledge-host-changes"}, {"--bundle", "/bundle", "--public-key", path}} {
		var out, errs bytes.Buffer
		if code := runInstallHostWith(context.Background(), args, &out, &errs, install); code != 2 || called {
			t.Fatal("unauthorized install", args, code)
		}
	}
	var out, errs bytes.Buffer
	if code := runInstallHostWith(context.Background(), []string{"--bundle", "/bundle", "--public-key", path, "--acknowledge-host-changes"}, &out, &errs, install); code != 0 || !called {
		t.Fatal(code, &errs)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"status":"complete"`)) {
		t.Fatal("missing structured result", &out)
	}
}

func TestGuidedInstallForwardsExpectedRunnerIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.pub")
	if err := os.WriteFile(path, []byte("public fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	called := false
	code := runInstallHostOptionsWith(context.Background(), []string{"--owner", "desktop", "--bundle", "/bundle", "--public-key", path, "--expected-runner-id", "runner-original", "--acknowledge-host-changes"}, &out, &errs, func(_ context.Context, _ string, _ []byte, owner string, options hostinstall.HostOptions, _ func(string)) (hostinstall.HostResult, error) {
		called = true
		if owner != "desktop" || options.ExpectedRunnerID != "runner-original" || options.OfflineUpgrade {
			t.Fatal(owner, options)
		}
		return hostinstall.HostResult{Ready: true}, nil
	})
	if code != 0 || !called {
		t.Fatal(code, &errs)
	}
}
