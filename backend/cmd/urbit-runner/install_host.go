package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
)

func runInstallHost(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runInstallHostWith(ctx, args, stdout, stderr, hostinstall.InstallHost)
}

func runInstallHostWith(ctx context.Context, args []string, stdout, stderr io.Writer, install func(context.Context, string, []byte, func(string)) (hostinstall.HostResult, error)) int {
	flags := flag.NewFlagSet("install-host", flag.ContinueOnError)
	flags.SetOutput(stderr)
	bundle := flags.String("bundle", "", "verified rootless preparation directory")
	public := flags.String("public-key", "", "desktop Ed25519 public key file")
	ack := flags.Bool("acknowledge-host-changes", false, "authorize dedicated account, access and service setup")
	flags.Usage = func() {
		fmt.Fprintln(stdout, "Usage: urbit-runner install-host --bundle DIR --public-key FILE --acknowledge-host-changes\n\nOS-authorized host step used by omabit setup. Installs or verifies the dedicated\nrootless engine, current runner, restricted SSH access and boot services. Preserves\nship data, runner identity, existing grants and SSH host identity. Outputs bounded\nnon-secret JSON progress; package prerequisites must already be installed.")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !*ack || flags.NArg() != 0 || !filepath.IsAbs(*bundle) || filepath.Clean(*bundle) != *bundle || !filepath.IsAbs(*public) || filepath.Clean(*public) != *public {
		fmt.Fprintln(stderr, "Require an absolute bundle/public-key path and --acknowledge-host-changes.")
		return 2
	}
	f, err := os.OpenFile(*public, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Fprintln(stderr, "Desktop public key is unavailable.")
		return 1
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 8192 {
		fmt.Fprintln(stderr, "Desktop public key must be a bounded regular file.")
		return 1
	}
	data, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(data) > 8192 {
		return 1
	}
	encoder := json.NewEncoder(stdout)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var outputErr error
	result, err := install(ctx, *bundle, data, func(step string) {
		if outputErr == nil {
			outputErr = encoder.Encode(map[string]string{"step": step, "status": "running"})
			if outputErr != nil {
				cancel()
			}
		}
	})
	if err = errors.Join(err, outputErr); err != nil {
		message := err.Error()
		if len(message) > 8192 {
			message = message[:8192] + "…"
		}
		_ = encoder.Encode(map[string]any{"step": "host", "status": "failed", "message": message, "partial": result.Partial})
		return 1
	}
	if encoder.Encode(map[string]any{"step": "host", "status": "complete", "host": result}) != nil {
		return 1
	}
	return 0
}
