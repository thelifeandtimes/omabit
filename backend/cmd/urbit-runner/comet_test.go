package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestCometPlanAndAcknowledgement(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"comet-plan", "--format", "text"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "fresh-comet-bootstrap") || !strings.Contains(out.String(), "outbound") || !strings.Contains(out.String(), "no host Ames/UDP publication") {
		t.Fatalf("plan: %d %s %s", code, &out, &errs)
	}
	called := false
	qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
		called = true
		return runtimequalify.Result{}, nil
	}
	for _, args := range [][]string{{"qualify-comet"}, {"qualify-comet", "--acknowledge-disposable-run", "--key", "/private/key"}, {"qualify-comet", "--acknowledge-disposable-run", "--pier", "/existing/pier"}} {
		out.Reset()
		errs.Reset()
		if code := runRuntime(context.Background(), args, &out, &errs, qualify); code != 2 || called {
			t.Fatal("comet input boundary failed")
		}
	}
	out.Reset()
	errs.Reset()
	if code := runRuntime(context.Background(), []string{"qualify-comet", "--help"}, &out, &errs, qualify); code != 0 || called || !strings.Contains(out.String(), "NEW DISPOSABLE COMET") {
		t.Fatal("missing clear comet scope")
	}
}
