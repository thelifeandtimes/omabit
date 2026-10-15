package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestSetupCommands(t *testing.T) {
	for _, command := range []string{"preflight", "setup-plan"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(context.Background(), []string{command}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("exit %d: %s", code, &stderr)
			}
			if command == "preflight" {
				var report hostsetup.Report
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.RuntimeQualified || len(report.Checks) == 0 {
					t.Fatalf("unexpected report: %s, %v", &stdout, err)
				}
			} else {
				var plan hostsetup.SetupPlan
				if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil || plan.RuntimeQualified || !plan.RequiresAuthorization || plan.Account != "urbit-runner" {
					t.Fatalf("unexpected plan: %s, %v", &stdout, err)
				}
			}
		})
	}
}

func TestSetupHelpAndUsage(t *testing.T) {
	for _, command := range []string{"preflight", "setup-plan", "prepare-rootless"} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), []string{command, "--help"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "No installation") {
			t.Fatalf("help: exit %d, stdout %s, stderr %s", code, &stdout, &stderr)
		}
		for _, bad := range []string{"--apply", "trailing"} {
			stdout.Reset()
			stderr.Reset()
			code := run(context.Background(), []string{command, bad}, &stdout, &stderr)
			var problem protocol.ErrorResponse
			if code != 2 || stdout.Len() != 0 || json.Unmarshal(stderr.Bytes(), &problem) != nil || problem.Error.Code != "setup_error" {
				t.Fatalf("usage: exit %d, stdout %s, stderr %s", code, &stdout, &stderr)
			}
		}
	}
	for _, args := range [][]string{
		{"prepare-rootless"}, {"prepare-rootless", "--output"},
		{"prepare-rootless", "--output", "relative"},
		{"prepare-rootless", "--output", "/tmp/../tmp/new"},
	} {
		if code := run(context.Background(), args, io.Discard, io.Discard); code != 2 {
			t.Fatalf("%v: exit %d, want usage error", args, code)
		}
	}
}

func TestPrepareRefusesExistingOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"prepare-rootless", "--output", filepath.Clean(t.TempDir())}, &stdout, &stderr)
	var problem protocol.ErrorResponse
	if code != 1 || stdout.Len() != 0 || json.Unmarshal(stderr.Bytes(), &problem) != nil {
		t.Fatalf("exit %d, stdout %s, stderr %s", code, &stdout, &stderr)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("output failed") }

func TestPreflightOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"preflight"}, failingWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "output failed") {
		t.Fatalf("exit %d, stderr %s", code, &stderr)
	}
}

func TestInstallRequiresExplicitAcknowledgement(t *testing.T) {
	for _, args := range [][]string{
		{"install-rootless"}, {"install-rootless", "--bundle", "/tmp/bundle"},
		{"install-rootless", "--acknowledge-host-changes"},
		{"install-rootless", "--bundle", "relative", "--acknowledge-host-changes"},
		{"install-rootless", "--bundle", "/tmp/../tmp/bundle", "--acknowledge-host-changes"},
		{"install-rootless", "--bundle", "/tmp/bundle", "--acknowledge-host-changes", "extra"},
	} {
		var stdout, stderr bytes.Buffer
		code := runSetupWithInstaller(context.Background(), args, &stdout, &stderr, func(context.Context, string) (hostinstall.Result, error) {
			t.Fatal("installation invoked without a valid review acknowledgement")
			return hostinstall.Result{}, nil
		})
		if code != 2 || stdout.Len() != 0 {
			t.Fatalf("%v: exit %d, stdout %s stderr %s", args, code, &stdout, &stderr)
		}
	}
}

func TestInstallReportsPartialProgress(t *testing.T) {
	for _, partial := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		code := runSetupWithInstaller(context.Background(), []string{"install-rootless", "--bundle", "/tmp/bundle", "--acknowledge-host-changes"}, &stdout, &stderr, func(_ context.Context, path string) (hostinstall.Result, error) {
			if path != "/tmp/bundle" {
				t.Fatalf("unexpected path %s", path)
			}
			if partial {
				return hostinstall.Result{Partial: true, AccountCreated: true}, errors.New("partial installation")
			}
			return hostinstall.Result{RootlessVerified: true}, nil
		})
		var result hostinstall.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.RuntimeQualified || result.Partial != partial {
			t.Fatalf("invalid progress: %s, %v", &stdout, err)
		}
		if (partial && (code != 1 || !strings.Contains(stderr.String(), "partial installation"))) || (!partial && (code != 0 || stderr.Len() != 0)) {
			t.Fatalf("exit %d: %s", code, &stderr)
		}
	}
}

func TestInstallHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSetupWithInstaller(context.Background(), []string{"install-rootless", "--help"}, &stdout, &stderr, func(context.Context, string) (hostinstall.Result, error) {
		t.Fatal("help invoked installation")
		return hostinstall.Result{}, nil
	})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "DEVELOPMENT first-install") {
		t.Fatalf("help exit %d: %s %s", code, &stdout, &stderr)
	}
}
