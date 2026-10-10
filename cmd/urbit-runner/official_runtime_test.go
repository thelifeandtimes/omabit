package main

import (
	"bytes"
	"context"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestExperimentalRuntimeCommandsAreUnavailable(t *testing.T) {
	for _, command := range []string{
		"loom-plan", "qualify-loom", "loom-fixed-plan", "qualify-loom-fixed",
		"loom-comet-plan", "qualify-loom-comet", "transition-fake-plan",
		"qualify-transition-fake", "transition-comet-plan", "qualify-transition-comet",
		"_qualification-loom-child",
	} {
		t.Run(command, func(t *testing.T) {
			var out, errs bytes.Buffer
			if code := run(context.Background(), []string{command}, &out, &errs); code != 2 {
				t.Fatalf("unsupported runtime command dispatched: %s (%d)", command, code)
			}
		})
	}
}

func TestOfficialRuntimeCannotOptIntoExperimentalOverrides(t *testing.T) {
	for _, option := range []string{"--acknowledge-control-socket", "--acknowledge-stock-rejection", "--acknowledge-transition-rejection", "--binary=/unreviewed/urbit", "--runtime=edge"} {
		var out, errs bytes.Buffer
		called := false
		qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
			called = true
			return runtimequalify.Result{}, nil
		}
		code := runRuntime(context.Background(), []string{"qualify-runtime", "--acknowledge-disposable-run", option}, &out, &errs, qualify)
		if code != 2 || called {
			t.Fatalf("unsupported override dispatched: %s (%d)", option, code)
		}
	}
}
