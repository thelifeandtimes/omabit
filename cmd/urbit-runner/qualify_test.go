package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

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

func TestHTTPPlanAndAcknowledgement(t *testing.T) {
	for _, args := range [][]string{{"http-plan"}, {"http-plan", "--help"}, {"qualify-http", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := runRuntime(context.Background(), args, &stdout, &stderr, noQualification(t)); code != 0 || stderr.Len() != 0 {
			t.Fatalf("%v: exit %d, %s", args, code, &stderr)
		}
		if len(args) == 1 {
			var plan map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil || plan["profile"] != runtimequalify.ProfileHTTP || !strings.Contains(plan["network"].(string), "127.0.0.1:ephemeral") || plan["runtime_qualified"] != false {
				t.Fatalf("HTTP plan: %s, %v", &stdout, err)
			}
		}
	}
	for _, args := range [][]string{{"qualify-http"}, {"qualify-http", "--acknowledge-disposable-run", "--publish", "0.0.0.0:8080"}} {
		if code := runRuntime(context.Background(), args, io.Discard, io.Discard, noQualification(t)); code != 2 {
			t.Fatalf("unsafe HTTP invocation: %v, exit %d", args, code)
		}
	}
	called := false
	code := runRuntime(context.Background(), []string{"qualify-http", "--acknowledge-disposable-run"}, io.Discard, io.Discard,
		func(context.Context, func(string)) (runtimequalify.Result, error) {
			called = true
			return runtimequalify.Result{Profile: runtimequalify.ProfileHTTP}, nil
		})
	if code != 0 || !called {
		t.Fatal("acknowledged HTTP command did not dispatch")
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

func TestMeldPlanAndGuardedDispatch(t *testing.T) {
	for _, args := range [][]string{{"meld-plan"}, {"meld-plan", "--format", "text"}, {"meld-plan", "--help"}, {"qualify-meld", "--help"}} {
		var stdout, stderr bytes.Buffer
		if code := runRuntime(context.Background(), args, &stdout, &stderr, noQualification(t)); code != 0 || stderr.Len() != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, &stderr)
		}
		if len(args) == 1 {
			var p map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &p); err != nil || p["profile"] != runtimequalify.ProfileMeld || p["utility"] != "/bin/urbit meld --loom 31 /urbit/fake-zod" || p["runtime_qualified"] != false {
				t.Fatalf("wrong meld plan: %s %v", &stdout, err)
			}
		} else if len(args) == 3 && (!strings.Contains(stdout.String(), "management must be stopped") || !strings.Contains(stdout.String(), "no additional swap")) {
			t.Fatalf("text plan lost review scope: %s", &stdout)
		}
	}
	for _, args := range [][]string{
		{"qualify-meld"}, {"qualify-meld", "--acknowledge-disposable-run=false"},
		{"qualify-meld", "--acknowledge-disposable-run", "--pier", "/existing"},
		{"qualify-meld", "--acknowledge-disposable-run", "--format", "invalid"},
		{"meld-plan", "--image", "other"},
	} {
		if code := runRuntime(context.Background(), args, io.Discard, io.Discard, noQualification(t)); code != 2 {
			t.Fatalf("unsafe meld arguments accepted: %v exit %d", args, code)
		}
	}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"meld-plan", "--format", "text"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "offline-meld") {
		t.Fatal("meld-plan is not routed from the top-level CLI")
	}
}

func TestReadableQualificationResults(t *testing.T) {
	for _, failed := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		start := time.Date(2026, 9, 25, 17, 0, 0, 0, time.UTC)
		finish := start.Add(2 * time.Second)
		after := int64(1024)
		code := runRuntime(context.Background(), []string{"qualify-meld", "--acknowledge-disposable-run", "--format", "text"}, &stdout, &stderr,
			func(_ context.Context, progress func(string)) (runtimequalify.Result, error) {
				progress("meld")
				progress("cleanup")
				r := runtimequalify.Result{Profile: runtimequalify.ProfileMeld, RunID: "example", ReportPath: "/private/report.json", CheckpointPassed: !failed, CleanupComplete: true,
					Meld: &runtimequalify.MeldRecord{StartedAt: start, BeforePierBytes: 4096, Uncertain: failed}}
				if failed {
					r.Failure = "uncertain meld requires review"
					return r, errors.New(r.Failure)
				}
				r.Meld.FinishedAt, r.Meld.AfterPierBytes = &finish, &after
				return r, nil
			})
		if !strings.Contains(stderr.String(), "Phase: meld\nPhase: cleanup\n") || strings.Contains(stderr.String(), `"phase"`) || !strings.Contains(stdout.String(), "Report: /private/report.json") {
			t.Fatalf("unreadable/missing progress: %s %s", &stdout, &stderr)
		}
		if failed {
			if code != 1 || !strings.Contains(stdout.String(), "Meld outcome uncertain: true") || !strings.Contains(stderr.String(), "Qualification error: uncertain meld requires review") || strings.Contains(stdout.String(), "bytes after:") {
				t.Fatalf("failure or unknown after-size lost: %s %s", &stdout, &stderr)
			}
		} else if code != 0 || !strings.Contains(stdout.String(), "Checkpoint: PASSED") || !strings.Contains(stdout.String(), "bytes after:  1024") || !strings.Contains(stdout.String(), "2.00 seconds") {
			t.Fatalf("success evidence missing: %s %s", &stdout, &stderr)
		}
	}
}

func TestReadableProgressFailureCancels(t *testing.T) {
	code := runRuntime(context.Background(), []string{"qualify-meld", "--acknowledge-disposable-run", "--format", "text"}, io.Discard, failingWriter{},
		func(ctx context.Context, progress func(string)) (runtimequalify.Result, error) {
			progress("engine")
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatal("failed readable progress did not request cleanup")
			}
			return runtimequalify.Result{}, ctx.Err()
		})
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
}

func TestEpochQualificationPlanAndAcknowledgement(t *testing.T) {
	for _, args := range [][]string{{"epochs-plan"}, {"epochs-plan", "--format", "text"}, {"qualify-epochs", "--help"}} {
		var out, errout bytes.Buffer
		if code := runRuntime(context.Background(), args, &out, &errout, noQualification(t)); code != 0 || errout.Len() != 0 {
			t.Fatalf("plan/help: %d %s", code, &errout)
		}
		if len(args) == 1 {
			var p map[string]any
			if err := json.Unmarshal(out.Bytes(), &p); err != nil || p["profile"] != runtimequalify.ProfileEpochs || p["runtime_qualified"] != false {
				t.Fatalf("wrong epoch plan: %s %v", &out, err)
			}
		} else if !strings.Contains(out.String(), "no deletion") && !strings.Contains(out.String(), "no epoch is deleted") {
			t.Fatalf("plan omitted preview-only scope: %s", &out)
		}
	}
	for _, args := range [][]string{{"qualify-epochs"}, {"qualify-epochs", "--acknowledge-disposable-run=false"}, {"qualify-epochs", "--acknowledge-disposable-run", "--pier", "/existing"}, {"qualify-epochs", "--acknowledge-disposable-run", "--cutoff", "0i0"}} {
		if code := runRuntime(context.Background(), args, io.Discard, io.Discard, noQualification(t)); code != 2 {
			t.Fatalf("unsafe epoch qualifier arguments accepted: %v", args)
		}
	}
	called := false
	code := runRuntime(context.Background(), []string{"qualify-epochs", "--acknowledge-disposable-run", "--format", "text"}, io.Discard, io.Discard, func(_ context.Context, progress func(string)) (runtimequalify.Result, error) {
		called = true
		progress("epoch-preview")
		return runtimequalify.Result{Profile: runtimequalify.ProfileEpochs, RunID: "test", CheckpointPassed: true}, nil
	})
	if code != 0 || !called {
		t.Fatal("acknowledged epoch qualification not dispatched")
	}
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"epochs-plan", "--format", "text"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "epoch-lifecycle") {
		t.Fatal("top-level epoch plan route missing")
	}
}

func TestPruneQualificationScopeAndAcknowledgement(t *testing.T) {
	for _, args := range [][]string{{"prune-plan"}, {"prune-plan", "--format", "text"}, {"qualify-prune", "--help"}} {
		var out, errout bytes.Buffer
		if code := runRuntime(context.Background(), args, &out, &errout, noQualification(t)); code != 0 || errout.Len() != 0 {
			t.Fatalf("plan/help: %d %s", code, &errout)
		}
		if len(args) == 1 {
			var p map[string]any
			if err := json.Unmarshal(out.Bytes(), &p); err != nil || p["profile"] != runtimequalify.ProfilePrune || p["runtime_qualified"] != false {
				t.Fatalf("prune plan: %s %v", &out, err)
			}
		} else if !strings.Contains(strings.ToLower(out.String()), "latest") {
			t.Fatal("recovery scope missing")
		}
	}
	for _, args := range [][]string{{"qualify-prune"}, {"qualify-prune", "--acknowledge-disposable-run=false"}, {"qualify-prune", "--acknowledge-disposable-run", "--pier", "/existing"}, {"qualify-prune", "--acknowledge-disposable-run", "--cutoff", "0i66"}} {
		if code := runRuntime(context.Background(), args, io.Discard, io.Discard, noQualification(t)); code != 2 {
			t.Fatalf("unsafe prune arguments accepted: %v", args)
		}
	}
	called := false
	code := runRuntime(context.Background(), []string{"qualify-prune", "--acknowledge-disposable-run", "--format", "text"}, io.Discard, io.Discard, func(_ context.Context, p func(string)) (runtimequalify.Result, error) {
		called = true
		p("prune-prefix")
		return runtimequalify.Result{RunID: "fixture", Profile: runtimequalify.ProfilePrune}, nil
	})
	if code != 0 || !called {
		t.Fatal("acknowledged disposable prune not dispatched")
	}
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"prune-plan", "--format", "text"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "destructive-disposable-fake-only") {
		t.Fatal("prune plan route missing")
	}
}

func TestUncertainPruneOutputDoesNotClaimRecoveryOrPreservation(t *testing.T) {
	var out bytes.Buffer
	r := runtimequalify.Result{RunID: "fixture", Prune: &runtimequalify.PruneRecord{Uncertain: true}, Recovery: &runtimequalify.EpochRecoveryRecord{Uncertain: true}}
	if err := writeRuntimeResult(&out, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Retained metadata verified: false") || !strings.Contains(out.String(), "Restored checkpoint: unverified") {
		t.Fatalf("uncertainty hidden: %s", &out)
	}
}

func TestRepeatedPrefixPlanAndAcknowledgement(t *testing.T) {
	for _, args := range [][]string{{"prefix-plan"}, {"prefix-plan", "--format", "text"}, {"qualify-prefix", "--help"}} {
		var out, errout bytes.Buffer
		if code := runRuntime(context.Background(), args, &out, &errout, noQualification(t)); code != 0 || errout.Len() != 0 {
			t.Fatalf("plan/help: %d %s", code, &errout)
		}
		if len(args) == 1 {
			var p map[string]any
			if err := json.Unmarshal(out.Bytes(), &p); err != nil || p["profile"] != runtimequalify.ProfilePrefix || p["runtime_qualified"] != false {
				t.Fatalf("wrong prefix scope: %s %v", &out, err)
			}
		} else if !strings.Contains(out.String(), "five") || !strings.Contains(out.String(), "newest two") {
			t.Fatalf("missing repeated-prefix scope: %s", &out)
		}
	}
	for _, args := range [][]string{{"qualify-prefix"}, {"qualify-prefix", "--acknowledge-disposable-run=false"}, {"qualify-prefix", "--acknowledge-disposable-run", "--pier", "/existing"}, {"qualify-prefix", "--acknowledge-disposable-run", "--cutoff", "0i100"}} {
		if code := runRuntime(context.Background(), args, io.Discard, io.Discard, noQualification(t)); code != 2 {
			t.Fatalf("unsafe arguments accepted: %v", args)
		}
	}
	called := false
	code := runRuntime(context.Background(), []string{"qualify-prefix", "--acknowledge-disposable-run", "--format", "text"}, io.Discard, io.Discard, func(_ context.Context, p func(string)) (runtimequalify.Result, error) {
		called = true
		p("prefix-delete-1")
		return runtimequalify.Result{RunID: "fixture", Profile: runtimequalify.ProfilePrefix}, nil
	})
	if code != 0 || !called {
		t.Fatal("acknowledged prefix command not dispatched")
	}
	var out, errout bytes.Buffer
	if code := run(context.Background(), []string{"prefix-plan", "--format", "text"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "epoch-prefix-recovery") {
		t.Fatal("prefix plan route missing")
	}
}
