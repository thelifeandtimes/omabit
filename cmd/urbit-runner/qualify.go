package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func runtimePlan() map[string]any {
	return map[string]any{
		"profile": runtimequalify.ProfileLifecycle,
		"mode":    "disposable-fake-only", "account": rootless.AccountName, "home": rootless.HomeDir,
		"engine_version": rootless.EngineVersion, "image": candidate.Image, "manifest_digest": candidate.ManifestDigest,
		"config_digest": candidate.ConfigDigest, "platform": candidate.Platform, "vere_version": candidate.VereVersion,
		"pill_default_url": candidate.PillDefaultURL, "pill_generation": candidate.PillGeneration,
		"pill":                        candidate.Artifact{URL: candidate.PillURL, SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes},
		"pill_pin_basis":              "Observed HTTPS content SHA-256 on 2026-09-23; not an independent publisher signature",
		"minimum_available_ram_bytes": int64(5 << 30), "minimum_available_disk_bytes": int64(8 << 30),
		"container_memory_bytes": int64(4 << 30), "container_swap_bytes": 0, "cpus": 2, "pids": 256, "loom_exponent": 31,
		"network": "none; no published ports", "control_socket": "disabled with --no-conn", "runtime_qualified": false,
		"steps":           []string{"engine verification", "pinned image pull and identity check", "verified upstream URL pill", "live cgroup and Vere version probe", "fake boot and graceful stop", "replacement-container restart", "offline pack", "post-pack restart", "owned-container cleanup; retain report and pier"},
		"remaining_gates": []string{"desk inventory/application health", "Ames and host HTTP forwarding", "meld", "reboot", "control-socket safety", "failure recovery and production runner"},
	}
}

func httpPlan() map[string]any {
	plan := runtimePlan()
	plan["profile"] = runtimequalify.ProfileHTTP
	plan["mode"] = "disposable-fake-host-http"
	plan["network"] = "one run-owned IPv4 NAT bridge with normal outbound access; only 127.0.0.1:ephemeral -> 8080/tcp is published"
	plan["host_probe"] = "credential-free HTTP 200 with public identity ~zod at /~/host; no proxies or redirects"
	plan["steps"] = []string{"engine verification", "pinned image and pill", "isolated limits/version probe", "dedicated NAT bridge creation", "fake boot and host HTTP identity", "replacement and rediscovered host HTTP endpoint", "owned-container and network cleanup; retain data/report"}
	plan["remaining_gates"] = []string{"application authentication and desk inventory", "Ames and remote/tailnet connectivity", "meld", "reboot", "control-socket safety", "failure recovery and production runner"}
	return plan
}

func runRuntime(ctx context.Context, args []string, stdout, stderr io.Writer, qualify func(context.Context, func(string)) (runtimequalify.Result, error)) int {
	fail := func(err error, status int) int {
		_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "qualification_error", Message: err.Error()}})
		return status
	}
	command := args[0]
	isPlan := command == "runtime-plan" || command == "http-plan"
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var acknowledged bool
	if !isPlan {
		flags.BoolVar(&acknowledged, "acknowledge-disposable-run", false, "authorize bounded downloads and a new disposable fake-ship workflow")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if isPlan {
				fmt.Fprintf(stdout, "Usage: urbit-runner %s\n\nPrints pinned candidate and checkpoint scope without inspecting the host or contacting Docker.\n", command)
			} else if command == "qualify-http" {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-http --acknowledge-disposable-run

DEVELOPMENT host HTTP checkpoint. Run as the dedicated urbit-runner account,
never root. Creates a fresh fake zod, a per-run NAT bridge, and only an
ephemeral IPv4 loopback publication. Verifies public Eyre identity from the host,
replaces the container and reconnects, then cleans up owned containers/network.
The dedicated bridge permits normal outbound access; Ames remains disabled.
No real identity, existing pier, login cookie, Ames port or control socket is used.
Requires >=5 GiB available RAM and >=8 GiB free disk. Downloads ~207 MiB of pill
data plus the pinned image. Progress is JSON lines on stderr; the report is on
stdout and in private storage. Cancellation cleanup can take several minutes.
`)
			} else {
				fmt.Fprint(stdout, `Usage: urbit-runner qualify-runtime --acknowledge-disposable-run

DEVELOPMENT checkpoint. Must execute as the dedicated urbit-runner account,
never as root or the desktop account. Uses only its verified private engine.
Downloads a pinned image and ~200 MiB pill, creates a fresh network-isolated
fake zod, tests limits, boot/restart/offline pack, and cleans up owned containers.
No identity keys, existing piers, public ports, or Vere control socket are used.
Requires >=5 GiB available RAM and >=8 GiB free disk. Data and reports are retained.
Progress is JSON lines on stderr; the final report is JSON on stdout. Ctrl+C
requests cancellation with bounded cleanup, which may take several minutes.
Uncertain or incomplete cleanup requires manual review, not an automatic retry.
`)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || (!isPlan && !acknowledged) {
		return fail(errors.New("unexpected arguments or missing --acknowledge-disposable-run; review the corresponding plan first"), 2)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if isPlan {
		plan := runtimePlan()
		if command == "http-plan" {
			plan = httpPlan()
		}
		if err := encoder.Encode(plan); err != nil {
			return fail(err, 1)
		}
		return 0
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var progressErr error
	result, err := qualify(ctx, func(phase string) {
		if progressErr == nil {
			progressErr = json.NewEncoder(stderr).Encode(struct {
				Phase string `json:"phase"`
			}{phase})
			if progressErr != nil {
				cancel()
			}
		}
	})
	if writeErr := encoder.Encode(result); writeErr != nil {
		err = errors.Join(err, writeErr)
	}
	if err = errors.Join(err, progressErr); err != nil {
		return fail(err, 1)
	}
	return 0
}
