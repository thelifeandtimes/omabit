package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runRepairEngineGuard(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runRepairEngineGuardWith(ctx, args, stdout, stderr, os.Geteuid, hostinstall.RepairEngineGuardWithOptions)
}

func runRepairEngineGuardWith(ctx context.Context, args []string, stdout, stderr io.Writer, euid func() int, repair func(context.Context, hostinstall.VerifyOptions) (hostinstall.GuardRepairResult, error)) int {
	format := "json"
	fail := func(message string, code int) int {
		if format == "text" {
			fmt.Fprintln(stderr, "Engine guard repair:", message)
		} else {
			_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "engine_guard_repair_error", Message: message}})
		}
		return code
	}
	f := flag.NewFlagSet("repair-engine-guard", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", "json", "output format: json or text")
	ack := f.Bool("acknowledge-host-changes", false, "authorize replacing exact known generated guards/units and reloading an active user manager")
	fetchMissing := f.Bool("fetch-missing", false, "allow native package download-only commands for missing verification artifacts; default offline")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-runner repair-engine-guard --acknowledge-host-changes [--fetch-missing] [--format json|text]\n\nRun locally as root after reviewing host-check and package authentication evidence.\nDefault: offline verification of cached native packages. --fetch-missing explicitly\npermits standard native-package download-only commands for missing verification\nartifacts. The exact installed builds must still be available from configured native\nrepositories; unavailable builds remain unresolved. It does not install or update packages or import keys.\nUpdates only recognized generated engine guards/units and reloads an already active\nuser manager. Preserves administrator edits; never starts or restarts the engine.\nThe result includes updated, reloaded and partial, including on failure.\nTime limit: 2 minutes offline; 10 minutes with --fetch-missing. An earlier caller\ndeadline still applies. This command is local-only, with no remote root RPC.")
			return 0
		}
		return fail("require --acknowledge-host-changes; optional --fetch-missing and --format json|text", 2)
	}
	if f.NArg() != 0 || !*ack || (format != "json" && format != "text") {
		return fail("require --acknowledge-host-changes; optional --fetch-missing and --format json|text", 2)
	}
	if euid() != 0 {
		return fail("engine guard repair requires explicit local root authorization", 1)
	}
	timeout := 2 * time.Minute
	if *fetchMissing {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, repairErr := repair(ctx, hostinstall.VerifyOptions{FetchMissing: *fetchMissing})
	var err error
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Engine guard: updated=%t; reloaded=%t; partial=%t\n", result.Updated, result.Reloaded, result.Partial)
	} else {
		e := json.NewEncoder(stdout)
		e.SetIndent("", "  ")
		err = e.Encode(result)
	}
	if err != nil {
		return fail("could not write engine guard repair result; review local guard/unit state before retrying", 1)
	}
	if repairErr != nil {
		return fail("engine guard repair did not complete; review local package authentication, owner and guard/unit evidence before retrying", 1)
	}
	return 0
}
