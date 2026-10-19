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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func TestHostCheckMetadataOnlyNeedsNoHosting(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("DOCKER_HOST", "tcp://untrusted.invalid:2375")
	for _, format := range []string{"json", "text"} {
		var out, errOut bytes.Buffer
		deps := hostCheckDeps{
			inspect: func() hostcompat.Report { return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64") },
			owner: func() (*ownerhost.Profile, error) {
				t.Fatal("metadata inspection required an installed owner")
				return nil, nil
			},
			probe: func(context.Context, string, string) (hostcompat.Engine, error) {
				t.Fatal("metadata inspection contacted Docker")
				return hostcompat.Engine{}, nil
			},
		}
		if code := runHostCheckWith(context.Background(), []string{"--format", format}, &out, &errOut, deps); code != 0 || errOut.Len() != 0 {
			t.Fatal(code, out.String(), errOut.String())
		}
		if format == "json" {
			var report hostcompat.Report
			if json.Unmarshal(out.Bytes(), &report) != nil || report.SchemaVersion != 1 || report.Engine != nil || report.CanMutate || report.Provenance == "verified" {
				t.Fatal("missing provider misrepresented", out.String())
			}
		} else if !strings.Contains(out.String(), "Host compatibility:") {
			t.Fatal(out.String())
		}
	}
}

func TestHostCheckLiveUsesOnlyInstalledOwnerProfile(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///untrusted/docker.sock")
	t.Setenv("HOME", "/untrusted")
	for _, scenario := range []string{"success", "missing-owner", "wrong-owner", "probe-failure"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			deps := hostCheckDeps{
				inspect: func() hostcompat.Report { return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64") },
				owner: func() (*ownerhost.Profile, error) {
					if scenario == "missing-owner" {
						return nil, nil
					}
					if scenario == "wrong-owner" {
						return nil, errors.New("secret owner error")
					}
					return &ownerhost.Profile{Version: 1, Name: "alice", UID: 1001, GID: 1001, Home: "/home/alice", Piers: "/srv/urbit/piers"}, nil
				},
				probe: func(ctx context.Context, socket, root string) (hostcompat.Engine, error) {
					calls++
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) > 3*time.Second || socket != "/run/user/1001/urbit-docker.sock" || root != "/home/alice/.local/share/urbit-runner/docker" {
						t.Fatal("unbounded or untrusted probe", socket, root)
					}
					if scenario == "probe-failure" {
						return hostcompat.Engine{}, errors.New("secret daemon response")
					}
					return hostcompat.Engine{Version: "29.9.1", APIVersion: "1.54", MinAPIVersion: "1.44", Rootless: true, CgroupDriver: "systemd", CgroupVersion: "2", DataRoot: root}, nil
				},
			}
			var out, errOut bytes.Buffer
			code := runHostCheckWith(context.Background(), []string{"--live"}, &out, &errOut, deps)
			if strings.Contains(out.String()+errOut.String(), "secret") {
				t.Fatal("raw probe error leaked")
			}
			if scenario == "success" {
				var report hostcompat.Report
				if code != 0 || json.Unmarshal(out.Bytes(), &report) != nil || report.Compatibility != "compatible" || report.Qualification != "unqualified" || calls != 1 {
					t.Fatal(code, out.String(), errOut.String())
				}
			} else if scenario == "probe-failure" {
				if code != 1 || calls != 1 || !strings.Contains(out.String(), "live-engine") {
					t.Fatal(code, out.String(), errOut.String())
				}
			} else if code != 1 || calls != 0 || out.Len() != 0 {
				t.Fatal(code, calls, out.String(), errOut.String())
			}
		})
	}
}

func TestHostCheckCLIAndDoctor(t *testing.T) {
	for _, args := range [][]string{{"host-check", "--help"}, {"doctor", "--help"}} {
		var out, errOut bytes.Buffer
		if code := run(context.Background(), args, &out, &errOut); code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "Usage:") {
			t.Fatal(code, out.String(), errOut.String())
		}
	}
	for _, args := range [][]string{{"host-check", "--socket", "/tmp/shared.sock"}, {"host-check", "--format", "xml"}, {"host-check", "--verify-host-packages"}, {"host-check", "extra"}, {"doctor", "--live"}, {"preflight", "--live"}} {
		var out, errOut bytes.Buffer
		if code := run(context.Background(), args, &out, &errOut); code != 2 || out.Len() != 0 {
			t.Fatal(args, code, out.String(), errOut.String())
		}
	}
	for _, command := range []string{"preflight", "doctor"} {
		var out, errOut bytes.Buffer
		code := run(context.Background(), []string{command}, &out, &errOut)
		var report hostsetup.Report
		if code != 0 || json.Unmarshal(out.Bytes(), &report) != nil || report.Compatibility == nil || report.Compatibility.Engine != nil || report.Compatibility.CanStop || report.Compatibility.CanMutate || report.RuntimeQualified {
			t.Fatal(code, out.String(), errOut.String())
		}
	}
}
