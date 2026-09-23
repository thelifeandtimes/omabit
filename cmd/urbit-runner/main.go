package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/demo"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
)

const usage = `Usage: urbit-runner demo --socket /private/directory/runner.sock [--operation-delay 2s]

Runs a local, memory-only simulator with two synthetic instances. No Docker,
Vere, network ports, credentials, or real piers are used. All history is lost
when the simulator exits. The socket directory must already exist, be owned by
the current user, and have no group/other permissions (normally mode 0700).
`

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stderr))
}

func run(ctx context.Context, args []string, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stderr, usage)
		return 0
	}
	if len(args) == 0 || args[0] != "demo" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, usage) }
	socket := flags.String("socket", "", "explicit absolute path in a private directory")
	delay := flags.Duration("operation-delay", 2*time.Second, "simulated job duration (100ms to 1m)")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *socket == "" || flags.NArg() != 0 || *delay < 100*time.Millisecond || *delay > time.Minute {
		fmt.Fprint(stderr, usage)
		return 2
	}
	listener, err := localrpc.Listen(*socket)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer listener.Close()
	runner := demo.New(*delay)
	defer runner.Close()
	server := &http.Server{
		Handler: runner, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192,
	}
	defer server.Close()
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	fmt.Fprintf(stderr, "SIMULATION ONLY: listening on %s; state resets on exit\n", *socket)
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		<-result
	}
	return 0
}
