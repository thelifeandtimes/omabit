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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const usage = `Usage: urbit-runner COMMAND

Commands:
  state-schema               Print this binary's persistent state schema version.
  install-host --owner USER --bundle DIR --public-key FILE --acknowledge-host-changes
                             Install the owner's rootless engine, access and services.
  prepare-ssh --output DIR --public-key FILE --role operator|observer
                             Write inert loopback SSH review configuration.
  ssh-authorized-key --public-key FILE --role operator|observer
                             Print a restricted SSH grant for review.
  ssh-bridge --socket PATH --authorized-keys FILE --fingerprint ID
                             Fixed-command SSH RPC bridge (development fixtures).
  preflight                  Read local prerequisites without contacting Docker.
  setup-plan                 Show the rootless installation review plan as JSON.
  prepare-rootless --output /absolute/new/directory
                             Fetch pinned, inert rootless setup review artifacts.
  runtime-plan               Describe the pinned disposable-runtime checkpoint.
  qualify-runtime --acknowledge-disposable-run
                             Run it as the installed pier owner (never as root).
  http-plan                  Describe the host-loopback HTTP checkpoint.
  qualify-http --acknowledge-disposable-run
                             Test private HTTP publication and replacement.
  meld-plan [--format text]  Describe disposable offline-meld qualification.
  qualify-meld --acknowledge-disposable-run [--format text]
                             Test offline meld and same-pier restart.
  epochs-plan [--format text]
                             Describe disposable offline-roll qualification.
  qualify-epochs --acknowledge-disposable-run [--format text]
                             Test three rolls, restarts, and a prefix preview.
  prune-plan [--format text]
                             Describe destructive disposable-prefix qualification.
  qualify-prune --acknowledge-disposable-run [--format text]
                             Test exact prefix deletion and latest-epoch recovery.
  prefix-plan [--format text]
                             Describe snapshot-bearing and repeated-prefix tests.
  qualify-prefix --acknowledge-disposable-run [--format text]
                             Test two successive deletions/recoveries on a new fake.
  import-plan [--format text]
                             Describe bounded stopped-pier copy qualification.
  qualify-import --acknowledge-disposable-run [--format text]
                             Copy a new stopped fake pier and resume the copy twice.
  comet-plan [--format text]
                             Describe fresh disposable-comet qualification.
  qualify-comet --acknowledge-disposable-run [--format text]
                             Create a new comet and verify initialized-pier resume.
  comet-maintenance-plan [--format text]
                             Describe fresh-comet pack/meld qualification.
  qualify-comet-maintenance --acknowledge-disposable-run [--format text]
                             Test offline pack/meld and identity-preserving restarts.
  comet-epochs-plan [--format text]
                             Describe fresh-comet epoch-roll qualification.
  qualify-comet-epochs --acknowledge-disposable-run [--format text]
                             Test three rolls, exact-identity restarts and preview.
  comet-prefix-plan [--format text]
                             Describe destructive disposable-comet prefix tests.
  qualify-comet-prefix --acknowledge-disposable-run [--format text]
                             Test repeated prefix deletion and latest-epoch recovery.
  serve-fake --acknowledge-development
                             Serve the persistent fake-ship management API.
  serve-development --acknowledge-development [--resume-running]
                             Serve development management; optional resolved startup resume.
  demo --socket /private/directory/runner.sock [--operation-delay 2s]
                             Run the checkpoint 1 in-memory simulator.

install-host performs explicitly authorized host setup for an existing Unix user.
Qualification commands create new disposable ships and runtime resources.
The development management API supports operator-authorized owned-key bootstrap;
qualification commands do not accept existing production identities or piers.
No command installs system packages. qualify-comet creates a new disposable comet.
Use COMMAND --help for details.
`

const demoUsage = `Usage: urbit-runner demo --socket /private/directory/runner.sock [--operation-delay 2s]

Runs a local, memory-only simulator with two synthetic instances. No Docker,
Vere, network ports, credentials, or real piers are used. All history is lost
when the simulator exits. The socket directory must already exist, be owned by
the current user, and have no group/other permissions (normally mode 0700).
`

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "management-offline-state" {
		return runOfflineState(ctx, os.Stdin, stdout, stderr)
	}
	if len(args) == 1 && args[0] == "state-schema" {
		fmt.Fprintln(stdout, managed.StateSchemaVersion)
		return 0
	}
	if len(args) > 0 && args[0] == "owner-ssh" {
		return runOwnerSSH(ctx, args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "install-host" {
		return runInstallHost(ctx, args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "-c" {
		return runSSHShell(ctx, args, stdout, stderr)
	}
	if len(args) > 0 && (args[0] == "ssh-authorized-key" || args[0] == "ssh-bridge" || args[0] == "prepare-ssh") {
		return runSSH(ctx, args[0], args[1:], stdout, stderr)
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if len(args) > 0 && (args[0] == "preflight" || args[0] == "setup-plan" || args[0] == "prepare-rootless" || args[0] == "install-rootless") {
		return runSetup(ctx, args, stdout, stderr)
	}
	if len(args) > 0 && (args[0] == "runtime-plan" || args[0] == "qualify-runtime") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.Run)
	}
	if len(args) > 0 && (args[0] == "http-plan" || args[0] == "qualify-http") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunHTTP)
	}
	if len(args) > 0 && (args[0] == "meld-plan" || args[0] == "qualify-meld") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunMeld)
	}
	if len(args) > 0 && (args[0] == "epochs-plan" || args[0] == "qualify-epochs") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunEpochs)
	}
	if len(args) > 0 && (args[0] == "comet-epochs-plan" || args[0] == "qualify-comet-epochs") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunCometEpochs)
	}
	if len(args) > 0 && (args[0] == "comet-prefix-plan" || args[0] == "qualify-comet-prefix") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunCometPrefix)
	}
	if len(args) > 0 && (args[0] == "prune-plan" || args[0] == "qualify-prune") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunPrune)
	}
	if len(args) > 0 && (args[0] == "prefix-plan" || args[0] == "qualify-prefix") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunPrefix)
	}
	if len(args) > 0 && (args[0] == "import-plan" || args[0] == "qualify-import") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunImport)
	}
	if len(args) > 0 && (args[0] == "comet-plan" || args[0] == "qualify-comet") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunComet)
	}
	if len(args) > 0 && (args[0] == "comet-maintenance-plan" || args[0] == "qualify-comet-maintenance") {
		return runRuntime(ctx, args, stdout, stderr, runtimequalify.RunCometMaintenance)
	}
	if len(args) > 0 && args[0] == "serve-fake" {
		return runManaged(ctx, args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "serve-development" {
		return runManagedMode(ctx, args[1:], stdout, stderr, true)
	}
	if len(args) == 0 || args[0] != "demo" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, demoUsage) }
	socket := flags.String("socket", "", "explicit absolute path in a private directory")
	delay := flags.Duration("operation-delay", 2*time.Second, "simulated job duration (100ms to 1m)")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *socket == "" || flags.NArg() != 0 || *delay < 100*time.Millisecond || *delay > time.Minute {
		fmt.Fprint(stderr, demoUsage)
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
