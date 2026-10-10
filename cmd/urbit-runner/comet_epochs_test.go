package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestCometEpochPlanHelpAndNoExistingInput(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"comet-epochs-plan"}, &out, &errs); code != 0 {
		t.Fatal(code, &errs)
	}
	var plan map[string]any
	if json.Unmarshal(out.Bytes(), &plan) != nil || plan["profile"] != runtimequalify.ProfileCometEpochs || plan["runtime_qualified"] != false {
		t.Fatal("wrong qualification profile", &out)
	}
	out.Reset()
	if code := run(context.Background(), []string{"qualify-comet-epochs", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "NEW DISPOSABLE COMET") || !strings.Contains(out.String(), "No epoch is deleted") {
		t.Fatal("missing scope in help", code, &out, &errs)
	}
	called := false
	qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
		called = true
		return runtimequalify.Result{}, nil
	}
	for _, extra := range [][]string{nil, {"--acknowledge-disposable-run", "--pier", "/existing"}, {"--acknowledge-disposable-run", "--key", "/private/key"}, {"--acknowledge-disposable-run", "--cutoff", "0i0"}} {
		out.Reset()
		errs.Reset()
		if code := runRuntime(context.Background(), append([]string{"qualify-comet-epochs"}, extra...), &out, &errs, qualify); code != 2 || called {
			t.Fatal("qualification input boundary failed")
		}
	}
}
