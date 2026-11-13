package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestShipRoutesCLIRejectsBadInputBeforeOpeningState(t *testing.T) {
	for _, args := range [][]string{
		{"ship-routes"},
		{"ship-routes", "--instance", "alpha", "--request-json", "{}"},
		{"ship-routes-configure", "--request-json", "{}"},
		{"ship-routes-configure", "--instance", "alpha", "--request-json", "{}"},
		{"ship-routes-configure", "--request-json", "{} {}"},
		{"ship-routes-configure", "--request-file", "/no-such-file"},
	} {
		t.Run(args[0], func(t *testing.T) {
			root := t.TempDir()
			config := filepath.Join(root, "not-created")
			argv := append([]string{"--config-dir", config, args[0], "--runner", "fixture", "--expected-runner-id", "runner-fixture"}, args[1:]...)
			if run(context.Background(), argv) == 0 {
				t.Fatal("invalid input accepted")
			}
			if _, err := os.Stat(config); !os.IsNotExist(err) {
				t.Fatal("invalid request opened state", err)
			}
		})
	}
}
