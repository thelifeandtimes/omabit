package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestCometMaintenancePlanAndFixedInputBoundary(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"comet-maintenance-plan", "--format", "text"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "comet-offline-maintenance") || !strings.Contains(out.String(), "network=none") {
		t.Fatalf("plan: %d %s %s", code, &out, &errs)
	}
	called := false
	qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
		called = true
		return runtimequalify.Result{}, nil
	}
	for _, args := range [][]string{{"qualify-comet-maintenance"}, {"qualify-comet-maintenance", "--acknowledge-disposable-run", "--pier", "/existing"}, {"qualify-comet-maintenance", "--acknowledge-disposable-run", "--key", "/private/key"}} {
		out.Reset()
		errs.Reset()
		if code := runRuntime(context.Background(), args, &out, &errs, qualify); code != 2 || called {
			t.Fatal("qualification boundary failed")
		}
	}
}
