package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func runManaged(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runManagedMode(ctx, args, stdout, stderr, false)
}

func runManagedMode(ctx context.Context, args []string, stdout, stderr io.Writer, comets bool) int {
	fail := func(err error, code int) int {
		_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "managed_runner_error", Message: err.Error()}})
		return code
	}
	command := "serve-fake"
	runtimeMode := "development-managed-fake"
	if comets {
		command = "serve-development"
		runtimeMode = managed.DevelopmentMode
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	ack := flags.Bool("acknowledge-development", false, "authorize the selected development management mode; ships outlive this process")
	var resume bool
	if comets {
		flags.BoolVar(&resume, "resume-running", false, "resume eligible initialized ships after host reboot, once per host boot")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if comets {
				fmt.Fprint(stdout, `Usage: urbit-runner serve-development --acknowledge-development [--resume-running]

Serve the installed pier-owner development runner with fake/comet creation, owned-key
bootstrap, stopped-pier enrollment, maintenance and host handoff. Comets generate
their identity internally. HTTP is loopback-only; owned ships publish configured Ames UDP.
Collector enrollment must explicitly accept mode development-managed while
retaining the same runner ID. Private state has explicit schema/downgrade guards.
Start/stop are explicit; incomplete bootstrap is quarantined, never retried.
With --resume-running, startup may replace terminal runtime containers for
initialized ships whose persisted desired state is running. Each ship gets at most
one journaled attempt per host boot. Nonzero exits require a recorded healthy prior
boot; intentionally stopped, interrupted and independently uncertain ships are skipped.
`)
				return 0
			}
			fmt.Fprint(stdout, `Usage: urbit-runner serve-fake --acknowledge-development

Runs as the dedicated urbit-runner account. Serves a private Unix management API
with SQLite-backed registrations, revisions, operations, and request replay.
Supports fake ships and qualified-fake imports, explicit maintenance, loopback
HTTP, the pinned image/pill and loom 31. Comets require serve-development.
Stopping this management process does NOT stop its independently running ships.
Use urbit-collector operate INSTANCE stop before ending a development session.
Interrupted work is recorded for reconciliation, never blindly re-executed.
`)
			return 0
		}
		return fail(err, 2)
	}
	if !*ack || flags.NArg() != 0 {
		return fail(errors.New("require --acknowledge-development with no additional arguments"), 2)
	}
	host, err := runtimequalify.OpenManagedHost()
	if err != nil {
		return fail(err, 1)
	}
	defer host.Close()
	if err := removeStaleManagedSocket(host.Socket); err != nil {
		return fail(err, 1)
	}
	listener, err := localrpc.Listen(host.Socket)
	if err != nil {
		return fail(err, 1)
	}
	defer listener.Close()
	if err := os.Chmod(host.Socket, os.ModeSticky|0600); err != nil {
		return fail(err, 1)
	}
	var runner *managed.Server
	if comets {
		runner, err = managed.NewDevelopmentService(host.Directory, access.DefaultKeys, resume)
	} else {
		runner, err = managed.NewWithAccess(host.Directory, access.DefaultKeys)
	}
	if err != nil {
		return fail(err, 1)
	}
	server := &http.Server{Handler: runner, ConnContext: managed.PeerContext, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	defer server.Close()
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	_ = json.NewEncoder(stderr).Encode(map[string]string{"mode": runtimeMode, "socket": host.Socket, "state_directory": host.Directory})
	select {
	case err = <-result:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Shutdown(shutdown)
		cancel()
		if err != nil {
			_ = server.Close()
		}
		<-result
	}
	err = errors.Join(err, runner.Close())
	if err != nil {
		return fail(err, 1)
	}
	return 0
}

// Called only while the managed-host state lock is held. Do not replace a live
// listener, unknown file, symlink or another user's socket after a crash.
func removeStaleManagedSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode().Type() != os.ModeSocket || info.Mode().Perm() != 0600 || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 {
		return errors.New("refusing unexpected existing management socket")
	}
	c, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err == nil {
		c.Close()
		return errors.New("management socket already has an active listener")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return errors.New("management socket status is uncertain; refusing removal")
	}
	return os.Remove(path)
}
