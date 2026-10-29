package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPlanCLIIsOfflineCanonicalAndExplicit(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, "unopened")
	var first projectPlanView
	for i, members := range [][]string{{"demo-zod:~zod:18381", "demo-nec:~nec:18382"}, {"demo-nec:~nec:18382", "demo-zod:~zod:18381"}} {
		args := []string{"--config-dir", registry, "project-plan", "--label", "demo", "--shared-administration"}
		for _, m := range members {
			args = append(args, "--member", m)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var out, errs bytes.Buffer
		if code := run(ctx, args, &out, &errs); code != 0 || errs.Len() != 0 {
			t.Fatal(code, errs.String())
		}
		var v projectPlanView
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v.SchemaVersion != 1 || v.RuntimeAuthorized || v.ReservationCheck != "not-performed" || len(v.ReviewDigest) != 64 || len(v.Plan.Members) != 2 || v.Plan.Members[0].Ship != "~nec" || v.Plan.Members[1].InternalHTTPPort != 8081 {
			t.Fatal(v)
		}
		if i == 0 {
			first = v
		} else if v.ReviewDigest != first.ReviewDigest {
			t.Fatal("member argument order changed canonical plan")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("offline planner opened registry", entries, err)
	}
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"--format", "text", "project-plan", "--label", "demo", "--member", "demo-zod:~zod:18381", "--shared-administration"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "Offline project plan") || !strings.Contains(out.String(), "No resources reserved or started") {
		t.Fatal(code, out.String(), errs.String())
	}
}

func TestProjectPlanCLIRejectsAmbiguousOrUnsafeInputs(t *testing.T) {
	base := []string{"project-plan", "--label", "demo", "--shared-administration"}
	for _, tail := range [][]string{
		{"--member", "zod:~zod:0"}, {"--member", "zod:~zod:65536"}, {"--member", "zod:~zod:-1"}, {"--member", "zod:~zod:1023"},
		{"--member", "zod:~zod:18381", "extra"}, {"--member", "zod:~zod:18381", "--member", "nec:~nec:18381"},
		{"--member", "zod:~zod:18381", "--member", "another:~zod:18382"}, {"--member", "zod:zod:18381"}, {"--member", "missing-parts"},
		{"--member", "zod:~zod:18381", "--shared-administration=false"},
	} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), append(append([]string{}, base...), tail...), &out, &errs); code != 2 || out.Len() != 0 || !json.Valid(errs.Bytes()) {
			t.Fatalf("accepted invalid plan %v: %d %s %s", tail, code, out.String(), errs.String())
		}
	}
	for _, selection := range [][]string{{"--socket", "/absent/runner.sock"}, {"--runner", "absent"}, {"--expected-runner-id", "runner-example"}} {
		args := append(append([]string{}, selection...), base...)
		args = append(args, "--member", "zod:~zod:18381")
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, &out, &errs); code != 2 || !strings.Contains(errs.String(), "project-plan is offline") {
			t.Fatal(code, errs.String())
		}
	}
}
