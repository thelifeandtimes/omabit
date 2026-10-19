package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
)

func TestRepairEngineGuardRequiresAcknowledgedLocalRoot(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		uid, code int
	}{
		{nil, 0, 2},
		{[]string{"--format", "json"}, 0, 2},
		{[]string{"--fetch-missing"}, 0, 2},
		{[]string{"--fetch-missing", "--acknowledge-host-changes=false"}, 0, 2},
		{[]string{"--acknowledge-host-changes", "--runner", "remote"}, 0, 2},
		{[]string{"--acknowledge-host-changes", "--format", "xml"}, 0, 2},
		{[]string{"--acknowledge-host-changes"}, 1000, 1},
		{[]string{"--acknowledge-host-changes", "--fetch-missing"}, 1000, 1},
		{[]string{"--help"}, 0, 0},
	} {
		var out, errOut bytes.Buffer
		code := runRepairEngineGuardWith(context.Background(), tc.args, &out, &errOut, func() int { return tc.uid }, func(context.Context, hostinstall.VerifyOptions) (hostinstall.GuardRepairResult, error) {
			t.Fatal("repair invoked without explicit local root acknowledgement")
			return hostinstall.GuardRepairResult{}, nil
		})
		if code != tc.code {
			t.Fatal(tc.args, code, out.String(), errOut.String())
		}
	}
}

func TestRepairEngineGuardReturnsProgressAndFixedError(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		for _, partial := range []bool{false, true} {
			var out, errOut bytes.Buffer
			want := hostinstall.GuardRepairResult{Updated: true, Reloaded: !partial, Partial: partial}
			code := runRepairEngineGuardWith(context.Background(), []string{"--acknowledge-host-changes", "--format", format}, &out, &errOut, func() int { return 0 }, func(ctx context.Context, options hostinstall.VerifyOptions) (hostinstall.GuardRepairResult, error) {
				if options.FetchMissing {
					t.Fatal("default repair fetched packages")
				}
				assertLocalHostDeadline(t, ctx, 2*time.Minute)
				if partial {
					return want, errors.New("private-secret systemctl error")
				}
				return want, nil
			})
			if (code == 1) != partial || strings.Contains(out.String()+errOut.String(), "private-secret") {
				t.Fatal(code, out.String(), errOut.String())
			}
			if format == "json" {
				var got hostinstall.GuardRepairResult
				if json.Unmarshal(out.Bytes(), &got) != nil || got != want {
					t.Fatal(out.String())
				}
			} else if !strings.Contains(out.String(), "updated=true") || !strings.Contains(out.String(), "reloaded=") || !strings.Contains(out.String(), "partial=") {
				t.Fatal(out.String())
			}
		}
	}
}

func TestRepairEngineGuardFetchMissingExplicitlyForwarded(t *testing.T) {
	for _, flag := range []string{"", "--fetch-missing", "--fetch-missing=false"} {
		var out, errOut bytes.Buffer
		args := []string{"--acknowledge-host-changes"}
		if flag != "" {
			args = append(args, flag)
		}
		calls := 0
		want := hostinstall.GuardRepairResult{Updated: true, Partial: true}
		code := runRepairEngineGuardWith(context.Background(), args, &out, &errOut, func() int { return 0 }, func(ctx context.Context, options hostinstall.VerifyOptions) (hostinstall.GuardRepairResult, error) {
			calls++
			if options.FetchMissing != (flag == "--fetch-missing") {
				t.Fatal("repair fetch permission changed", options)
			}
			wantDeadline := 2 * time.Minute
			if flag == "--fetch-missing" {
				wantDeadline = 10 * time.Minute
			}
			assertLocalHostDeadline(t, ctx, wantDeadline)
			return want, errors.New("secret native download/repair error")
		})
		var got hostinstall.GuardRepairResult
		if code != 1 || calls != 1 || json.Unmarshal(out.Bytes(), &got) != nil || got != want || strings.Contains(errOut.String(), "secret") {
			t.Fatal(flag, code, calls, out.String(), errOut.String())
		}
	}
}

func TestHostCommandTopLevelHelpAndRepairDispatch(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"--help"}, &out, &errOut); code != 0 {
		t.Fatal(code, errOut.String())
	}
	for _, command := range []string{"host-check", "verify-host-packages", "repair-engine-guard", "doctor"} {
		if !strings.Contains(out.String(), command) {
			t.Fatal("missing help command", command)
		}
	}
	out.Reset()
	if code := run(context.Background(), []string{"repair-engine-guard", "--help"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "--acknowledge-host-changes") {
		t.Fatal(code, out.String(), errOut.String())
	}
	for _, phrase := range []string{"--fetch-missing", "Default: offline", "download-only", "does not install or update packages or import keys", "never starts or restarts the engine", "exact installed builds", "unavailable builds remain unresolved", "2 minutes offline; 10 minutes with --fetch-missing", "no remote root RPC"} {
		if !strings.Contains(out.String(), phrase) {
			t.Fatal("missing repair constraint", phrase, out.String())
		}
	}
	// Never call the real repair seam: this command must fail before dispatch.
	out.Reset()
	if code := run(context.Background(), []string{"repair-engine-guard"}, &out, &errOut); code != 2 || out.Len() != 0 {
		t.Fatal(code, out.String(), errOut.String())
	}
}

func TestRepairEngineGuardPreservesEarlierCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	parentDeadline, _ := ctx.Deadline()
	for _, fetch := range []bool{false, true} {
		args := []string{"--acknowledge-host-changes"}
		if fetch {
			args = append(args, "--fetch-missing")
		}
		var out, errOut bytes.Buffer
		code := runRepairEngineGuardWith(ctx, args, &out, &errOut, func() int { return 0 }, func(child context.Context, options hostinstall.VerifyOptions) (hostinstall.GuardRepairResult, error) {
			deadline, ok := child.Deadline()
			if !ok || !deadline.Equal(parentDeadline) || options.FetchMissing != fetch {
				t.Fatal("caller deadline or fetch selection lost", deadline, options)
			}
			return hostinstall.GuardRepairResult{}, nil
		})
		if code != 0 {
			t.Fatal(code, errOut.String())
		}
	}
}
