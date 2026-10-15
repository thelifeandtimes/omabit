package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestImportPlanAndFixedQualificationBoundary(t *testing.T) {
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"import-plan", "--format", "text"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "stopped-pier-copy") || !strings.Contains(out.String(), "source/") {
		t.Fatalf("plan: %d %s %s", code, &out, &errout)
	}
	called := false
	qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
		called = true
		return runtimequalify.Result{}, nil
	}
	for _, args := range [][]string{{"qualify-import"}, {"qualify-import", "--acknowledge-disposable-run", "--source", "/existing/pier"}, {"import-plan", "--acknowledge-disposable-run"}} {
		out.Reset()
		errout.Reset()
		if code := runRuntime(context.Background(), args, &out, &errout, qualify); code != 2 || called {
			t.Fatalf("unsafe arguments dispatched: %v", args)
		}
	}
	out.Reset()
	errout.Reset()
	if code := runRuntime(context.Background(), []string{"qualify-import", "--help"}, &out, &errout, qualify); code != 0 || called || !strings.Contains(out.String(), "No existing-pier") {
		t.Fatal("help altered qualification boundary")
	}
}
