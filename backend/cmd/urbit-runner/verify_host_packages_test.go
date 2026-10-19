package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func assertLocalHostDeadline(t *testing.T, ctx context.Context, want time.Duration) {
	t.Helper()
	deadline, ok := ctx.Deadline()
	remaining := time.Until(deadline)
	if !ok || remaining > want || remaining < want-time.Second {
		t.Fatalf("remaining deadline = %s; want approximately %s", remaining, want)
	}
}

func TestVerifyHostPackagesRequiresLocalRootAndAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		uid, code int
	}{
		{nil, 0, 2},
		{[]string{"--format", "json"}, 0, 2},
		{[]string{"--fetch-missing"}, 0, 2},
		{[]string{"--fetch-missing", "--acknowledge-local-verification=false"}, 0, 2},
		{[]string{"--acknowledge-local-verification", "--runner", "remote"}, 0, 2},
		{[]string{"--acknowledge-local-verification", "--format", "xml"}, 0, 2},
		{[]string{"--acknowledge-local-verification"}, 1000, 1},
		{[]string{"--acknowledge-local-verification", "--fetch-missing"}, 1000, 1},
		{[]string{"--help"}, 0, 0},
	} {
		var out, errOut bytes.Buffer
		code := runVerifyHostPackagesWith(context.Background(), tc.args, &out, &errOut, func() int { return tc.uid }, func(context.Context, hostcompat.VerificationOptions) (hostcompat.Report, error) {
			t.Fatal("verification dispatched without explicit root acknowledgement")
			return hostcompat.Report{}, nil
		})
		if code != tc.code {
			t.Fatal(tc.args, code, out.String(), errOut.String())
		}
	}
}

func TestVerifyHostPackagesBoundedPartialEvidenceAndSanitizedError(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		var out, errOut bytes.Buffer
		code := runVerifyHostPackagesWith(context.Background(), []string{"--acknowledge-local-verification", "--format", format}, &out, &errOut, func() int { return 0 }, func(ctx context.Context, options hostcompat.VerificationOptions) (hostcompat.Report, error) {
			if options.FetchMissing {
				t.Fatal("default verification fetched packages")
			}
			assertLocalHostDeadline(t, ctx, 2*time.Minute)
			return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), errors.New("secret signature tool stderr")
		})
		if code != 1 || out.Len() == 0 || !strings.Contains(errOut.String(), "incomplete or unavailable") || strings.Contains(out.String()+errOut.String(), "secret") {
			t.Fatal(code, out.String(), errOut.String())
		}
	}
}

func TestVerifyHostPackagesFetchMissingExplicitlyForwarded(t *testing.T) {
	for _, flag := range []string{"", "--fetch-missing", "--fetch-missing=false"} {
		for _, partial := range []bool{false, true} {
			var out, errOut bytes.Buffer
			args := []string{"--acknowledge-local-verification"}
			if flag != "" {
				args = append(args, flag)
			}
			calls := 0
			code := runVerifyHostPackagesWith(context.Background(), args, &out, &errOut, func() int { return 0 }, func(ctx context.Context, options hostcompat.VerificationOptions) (hostcompat.Report, error) {
				calls++
				if options.FetchMissing != (flag == "--fetch-missing") {
					t.Fatal("fetch permission changed", options)
				}
				wantDeadline := 2 * time.Minute
				if flag == "--fetch-missing" {
					wantDeadline = 10 * time.Minute
				}
				assertLocalHostDeadline(t, ctx, wantDeadline)
				r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
				if partial {
					return r, errors.New("secret native download stderr")
				}
				return r, nil
			})
			wantCode := 0
			if partial {
				wantCode = 1
			}
			var report hostcompat.Report
			if code != wantCode || calls != 1 || json.Unmarshal(out.Bytes(), &report) != nil || report.SchemaVersion != 1 || report.Engine != nil || report.Qualification != "unqualified" || strings.Contains(errOut.String(), "secret") {
				t.Fatal(flag, code, calls, out.String(), errOut.String())
			}
			if partial && flag == "--fetch-missing" && !strings.Contains(errOut.String(), "download-enabled verification") {
				t.Fatal(errOut.String())
			}
		}
	}
}

func TestVerifyHostPackagesHelpDescribesOfflineAndDownloadOnly(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"verify-host-packages", "--help"}, &out, &errOut); code != 0 {
		t.Fatal(code, errOut.String())
	}
	for _, phrase := range []string{"--fetch-missing", "Default: offline", "download-only", "does not install or update packages", "local root-owned", "exact installed builds", "unavailable builds remain unresolved", "2 minutes offline; 10 minutes with --fetch-missing", "no remote root RPC"} {
		if !strings.Contains(out.String(), phrase) {
			t.Fatal("missing verification constraint", phrase, out.String())
		}
	}
}

func TestVerifyHostPackagesPreservesEarlierCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	parentDeadline, _ := ctx.Deadline()
	for _, fetch := range []bool{false, true} {
		args := []string{"--acknowledge-local-verification"}
		if fetch {
			args = append(args, "--fetch-missing")
		}
		var out, errOut bytes.Buffer
		code := runVerifyHostPackagesWith(ctx, args, &out, &errOut, func() int { return 0 }, func(child context.Context, options hostcompat.VerificationOptions) (hostcompat.Report, error) {
			deadline, ok := child.Deadline()
			if !ok || !deadline.Equal(parentDeadline) || options.FetchMissing != fetch {
				t.Fatal("caller deadline or fetch selection lost", deadline, options)
			}
			return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil
		})
		if code != 0 {
			t.Fatal(code, errOut.String())
		}
	}
}
