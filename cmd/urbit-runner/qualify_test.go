package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func noQualification(t *testing.T) func(context.Context, func(string)) (runtimequalify.Result, error) {
	t.Helper()
	return func(context.Context, func(string)) (runtimequalify.Result, error) {
		t.Fatal("qualification must not run for plan/help/invalid arguments")
		return runtimequalify.Result{}, nil
	}
}

func TestRuntimePlanAndHelp(t *testing.T) {
	for _, args := range [][]string{{"runtime-plan"}, {"runtime-plan", "--help"}, {"qualify-runtime", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := runRuntime(context.Background(), args, &stdout, &stderr, noQualification(t)); code != 0 || stderr.Len() != 0 {
			t.Fatalf("%v: exit %d, stderr %s", args, code, &stderr)
		}
		if len(args) == 1 {
			var plan map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil || plan["image"] != candidate.Image || plan["runtime_qualified"] != false || plan["network"] != "none; no published ports" {
				t.Fatalf("unexpected plan: %s, %v", &stdout, err)
			}
		}
	}
}

func TestRuntimeUsage(t *testing.T) {
	for _, args := range [][]string{
		{"qualify-runtime"}, {"qualify-runtime", "--acknowledge-disposable-run=false"},
		{"qualify-runtime", "--acknowledge-disposable-run", "extra"},
		{"qualify-runtime", "--acknowledge-disposable-run", "--pier", "/existing"},
		{"qualify-runtime", "--image", "other"}, {"runtime-plan", "extra"},
	} {
		var stdout, stderr bytes.Buffer
		code := runRuntime(context.Background(), args, &stdout, &stderr, noQualification(t))
		var response protocol.ErrorResponse
		if code != 2 || stdout.Len() != 0 || json.Unmarshal(stderr.Bytes(), &response) != nil || response.Error.Code != "qualification_error" {
			t.Fatalf("%v: exit %d stdout %s stderr %s", args, code, &stdout, &stderr)
		}
	}
}

func TestRuntimeProgressAndFailureReport(t *testing.T) {
	for _, failed := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		code := runRuntime(context.Background(), []string{"qualify-runtime", "--acknowledge-disposable-run"}, &stdout, &stderr,
			func(_ context.Context, progress func(string)) (runtimequalify.Result, error) {
				progress("engine")
				progress("cleanup")
				if failed {
					return runtimequalify.Result{RunID: "example", CleanupComplete: false}, errors.New("manual review required")
				}
				return runtimequalify.Result{RunID: "example", CheckpointPassed: true, CleanupComplete: true}, nil
			})
		var result runtimequalify.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.RuntimeQualified || result.RunID != "example" {
			t.Fatalf("invalid report: %s, %v", &stdout, err)
		}
		decoder := json.NewDecoder(&stderr)
		for _, phase := range []string{"engine", "cleanup"} {
			var event map[string]string
			if err := decoder.Decode(&event); err != nil || event["phase"] != phase {
				t.Fatalf("missing progress %s: %v", phase, err)
			}
		}
		if failed {
			var response protocol.ErrorResponse
			if err := decoder.Decode(&response); err != nil || code != 1 || !strings.Contains(response.Error.Message, "manual review") {
				t.Fatalf("failure lost: exit %d, %+v, %v", code, response, err)
			}
		} else if code != 0 || result.CheckpointPassed != true {
			t.Fatalf("success exit %d: %+v", code, result)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			t.Fatalf("unexpected stderr: %v, %v", extra, err)
		}
	}
}

func TestRuntimeProgressFailureCancels(t *testing.T) {
	var stdout bytes.Buffer
	code := runRuntime(context.Background(), []string{"qualify-runtime", "--acknowledge-disposable-run"}, &stdout, failingWriter{},
		func(ctx context.Context, progress func(string)) (runtimequalify.Result, error) {
			progress("engine")
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatal("lost progress output did not request bounded cancellation/cleanup")
			}
			return runtimequalify.Result{}, ctx.Err()
		})
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
}
