package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

// Keep the secret out of QML, argv, environment, logs, state and generic output.
// This process owns a foreground clipboard provider. Exiting it releases only
// its own selection; it never runs `wl-copy --clear` over a newer clipboard.
func copyCode(ctx context.Context, r *collection.Registry, args []string, output func(any) int, fail func(error) int) int {
	flags := flag.NewFlagSet("copy-code", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("runner", "", "enrolled runner name")
	id := flags.String("expected-runner-id", "", "selected runner identity")
	instance := flags.String("instance", "", "registered instance")
	revision := flags.Uint64("revision", 0, "selected configuration revision")
	container := flags.String("container-id", "", "selected full container identity")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return fail(errors.New("invalid copy-code selection"))
	}
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fail(errors.New("copy-code requires a Wayland session"))
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return fail(errors.New("wl-copy is required for code handoff"))
	}
	request, cancel := context.WithTimeout(ctx, 5*time.Second)
	code, err := r.LoginCode(request, *name, *id, *instance, *revision, *container)
	cancel()
	if err != nil {
		return fail(err)
	}
	clipboard, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	cmd := exec.CommandContext(clipboard, "wl-copy", "--foreground", "--sensitive", "--type", "text/plain")
	protectClipboardChild(cmd)
	cmd.Stdin = strings.NewReader(code.Code)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return fail(errors.New("clipboard provider could not start"))
	}
	code.Code = ""
	if result := output(map[string]any{"status": "clipboard_handoff_started", "expires_after_seconds": 30}); result != 0 {
		stop()
		_ = cmd.Wait()
		return result
	}
	err = cmd.Wait()
	if clipboard.Err() != nil || err == nil {
		return 0
	}
	return fail(errors.New("clipboard handoff failed"))
}
