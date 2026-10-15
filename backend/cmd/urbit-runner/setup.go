package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func runSetup(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "install-rootless" {
		fmt.Fprintln(stderr, "Use install-host --owner USER --bundle DIR --public-key FILE --acknowledge-host-changes for owner-user installation.")
		return 2
	}
	return runSetupWithInstaller(ctx, args, stdout, stderr, hostinstall.Install)
}

func runSetupWithInstaller(ctx context.Context, args []string, stdout, stderr io.Writer, install func(context.Context, string) (hostinstall.Result, error)) int {
	fail := func(err error, code int) int {
		encoder := json.NewEncoder(stderr)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "setup_error", Message: err.Error()}})
		return code
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var output string
	var bundle string
	var acknowledged bool
	if command == "prepare-rootless" {
		flags.StringVar(&output, "output", "", "new absolute directory under an existing user-owned, non-writable-by-others parent")
	}
	if command == "install-rootless" {
		flags.StringVar(&bundle, "bundle", "", "prepared bundle; reverified against this executable's compiled pins/templates")
		flags.BoolVar(&acknowledged, "acknowledge-host-changes", false, "authorize the reviewed first-install changes; requires OS elevation separately")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if command == "install-rootless" {
				fmt.Fprint(stdout, `Usage: urbit-runner install-rootless --bundle /absolute/bundle --acknowledge-host-changes

DEVELOPMENT first-install helper. Run only after reviewing setup-plan and the
checkpoint instructions. Requires root through explicit OS authorization.
Installs verified files, creates the locked urbit-runner account and subordinate
IDs, enables lingering and its private user engine, then checks rootless mode.
Existing resources are refused. Dependencies must already be installed.
No packages, system Docker changes, image pulls, or ships are involved.
Failures may leave partial setup; inspect the result and do not force a retry.
`)
				return 0
			}
			fmt.Fprintf(stdout, "Usage: urbit-runner %s", command)
			if command == "prepare-rootless" {
				fmt.Fprint(stdout, " --output /absolute/new/directory")
			}
			fmt.Fprint(stdout, "\n\nNo installation or Docker activation is performed. Exit 0 means report/bundle\ngeneration succeeded, not that this host or its runtime is qualified.\n")
			if command == "prepare-rootless" {
				fmt.Fprintf(stdout, "\nRequires a non-root user. Downloads pinned HTTPS artifacts for Docker %s\nwith hash verification; scripts remain non-executable. Existing output is refused.\n", rootless.EngineVersion)
			} else {
				fmt.Fprint(stdout, "\nReads local metadata only. Missing or unknown prerequisites are included in JSON.\nNo programs, network requests, socket connections, or writes are performed.\n")
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 {
		return fail(errors.New("unexpected trailing arguments"), 2)
	}
	var result any
	var err error
	if command == "install-rootless" {
		if !acknowledged || !filepath.IsAbs(bundle) || filepath.Clean(bundle) != bundle {
			return fail(errors.New("require --acknowledge-host-changes and an absolute, clean --bundle path; review setup-plan first"), 2)
		}
		// Progress is returned even on failure so partially changed hosts cannot
		// be confused with a refused, untouched installation.
		progress, installErr := install(ctx, bundle)
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(progress); err != nil {
			return fail(errors.Join(err, installErr), 1)
		}
		if installErr != nil {
			return fail(installErr, 1)
		}
		return 0
	} else if command == "prepare-rootless" {
		if !filepath.IsAbs(output) || filepath.Clean(output) != output {
			return fail(errors.New("--output is required and must be an absolute, clean path"), 2)
		}
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
			return fail(errors.New("rootless preparation currently supports Linux x86-64 only"), 1)
		}
		result, err = rootless.Prepare(ctx, output)
	} else {
		report := hostsetup.InspectCurrentOwner()
		result = report
		if command == "setup-plan" {
			result = hostsetup.Plan(report)
		}
	}
	if err != nil {
		return fail(err, 1)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
