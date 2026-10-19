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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runVerifyHostPackages(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runVerifyHostPackagesWith(ctx, args, stdout, stderr, os.Geteuid, hostcompat.VerifyCurrentWithOptions)
}

func runVerifyHostPackagesWith(ctx context.Context, args []string, stdout, stderr io.Writer, euid func() int, verify func(context.Context, hostcompat.VerificationOptions) (hostcompat.Report, error)) int {
	format := "json"
	fail := func(message string, code int) int {
		if format == "text" {
			fmt.Fprintln(stderr, "Package verification:", message)
		} else {
			_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "host_verification_error", Message: message}})
		}
		return code
	}
	f := flag.NewFlagSet("verify-host-packages", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", "json", "output format: json or text")
	ack := f.Bool("acknowledge-local-verification", false, "authorize native trust/payload checks and writing local root-owned verification receipts")
	fetchMissing := f.Bool("fetch-missing", false, "allow native package download-only commands for missing verification artifacts; default offline")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-runner verify-host-packages --acknowledge-local-verification [--fetch-missing] [--format json|text]\n\nRun locally as root after reviewing the host metadata. Default: offline verification\nof cached native packages against distribution trust and installed executable bytes.\n--fetch-missing explicitly permits standard native-package download-only commands\nfor missing verification artifacts. The exact installed builds must still be\navailable from configured native repositories; unavailable builds remain unresolved.\nIt does not install or update packages, import keys, restart services, or qualify\na runtime. Verification writes local root-owned receipts and reports partial evidence\non failure. Time limit: 2 minutes offline; 10 minutes with --fetch-missing. An earlier\ncaller deadline still applies. This command is local-only, with no remote root RPC.")
			return 0
		}
		return fail("require --acknowledge-local-verification; optional --fetch-missing and --format json|text", 2)
	}
	if f.NArg() != 0 || !*ack || (format != "json" && format != "text") {
		return fail("require --acknowledge-local-verification; optional --fetch-missing and --format json|text", 2)
	}
	if euid() != 0 {
		return fail("package verification requires explicit local root authorization", 1)
	}
	timeout := 2 * time.Minute
	if *fetchMissing {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	report, err := verify(ctx, hostcompat.VerificationOptions{FetchMissing: *fetchMissing})
	if report.SchemaVersion != 0 {
		if outputErr := writeLocalHostReport(stdout, format, report); outputErr != nil {
			return fail("could not write package verification report", 1)
		}
	}
	if err != nil {
		message := "package authentication is incomplete or unavailable; review the reported evidence and local native package cache"
		if *fetchMissing {
			message = "package authentication is incomplete or unavailable after download-enabled verification; review native repository availability and the reported evidence"
		}
		return fail(message, 1)
	}
	return 0
}
