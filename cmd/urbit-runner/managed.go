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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func runManaged(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fail := func(err error, code int) int {
		_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "managed_runner_error", Message: err.Error()}})
		return code
	}
	flags := flag.NewFlagSet("serve-fake", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	ack := flags.Bool("acknowledge-development", false, "authorize development fake-ship management; ships outlive this process")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, `Usage: urbit-runner serve-fake --acknowledge-development

Runs as the dedicated urbit-runner account. Serves a private Unix management API
with SQLite-backed registrations, revisions, operations, and request replay.
Only fresh fake ships, loopback HTTP, the pinned image/pill and loom 31 are
supported. No real keys, pier imports, Ames publication, meld or control socket.
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
	runner, err := managed.New(host.Directory)
	if err != nil {
		return fail(err, 1)
	}
	server := &http.Server{Handler: runner, ConnContext: managed.PeerContext, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	defer server.Close()
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	_ = json.NewEncoder(stderr).Encode(map[string]string{"mode": "development-managed-fake", "socket": host.Socket, "state_directory": host.Directory})
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
