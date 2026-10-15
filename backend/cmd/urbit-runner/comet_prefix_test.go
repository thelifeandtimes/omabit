package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestCometPrefixPlanDestructiveScopeAndFixedInputs(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"comet-prefix-plan"}, &out, &errs); code != 0 {
		t.Fatal(code, &errs)
	}
	var plan map[string]any
	if json.Unmarshal(out.Bytes(), &plan) != nil || plan["profile"] != runtimequalify.ProfileCometPrefix || plan["runtime_qualified"] != false {
		t.Fatal("wrong profile", &out)
	}
	out.Reset()
	if code := run(context.Background(), []string{"qualify-comet-prefix", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "DESTRUCTIVE") || !strings.Contains(out.String(), "NEW DISPOSABLE COMET") || !strings.Contains(out.String(), "Seventeen containers") {
		t.Fatal("missing destructive scope", &out, &errs)
	}
	called := false
	qualify := func(context.Context, func(string)) (runtimequalify.Result, error) {
		called = true
		return runtimequalify.Result{}, nil
	}
	for _, extra := range [][]string{nil, {"--acknowledge-disposable-run", "--pier", "/existing"}, {"--acknowledge-disposable-run", "--key", "/private/key"}, {"--acknowledge-disposable-run", "--cutoff", "0i0"}} {
		out.Reset()
		errs.Reset()
		if code := runRuntime(context.Background(), append([]string{"qualify-comet-prefix"}, extra...), &out, &errs, qualify); code != 2 || called {
			t.Fatal("destructive input boundary failed")
		}
	}
}
