package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
)

func TestRC11OfflineInstallFlagIsExplicitAndPreservesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator.pub")
	if err := os.WriteFile(path, []byte("inert public fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, offline := range []bool{false, true} {
		called := false
		install := func(_ context.Context, bundle string, key []byte, owner string, options hostinstall.HostOptions, _ func(string)) (hostinstall.HostResult, error) {
			called = true
			if options.OfflineUpgrade != offline || bundle != "/bundle" || owner != "desktop" {
				t.Fatal(options, bundle, owner)
			}
			return hostinstall.HostResult{Ready: true}, nil
		}
		args := []string{"--owner", "desktop", "--bundle", "/bundle", "--public-key", path, "--acknowledge-host-changes"}
		if offline {
			args = append(args, "--offline-upgrade")
		}
		var out, errs bytes.Buffer
		if code := runInstallHostOptionsWith(context.Background(), args, &out, &errs, install); code != 0 || !called {
			t.Fatal(code, errs.String())
		}
	}
}
