package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const scope = `Development management client; inspect hello.mode/features for server capabilities.
This is not a full collector daemon or production runtime manager.
Select --socket with an absolute path or --runner with an enrolled name before the subcommand.
Runner requests check Hello compatibility; listing and forgetting enrollment are local.
Revisions and request IDs are never inferred; mutations are never automatically retried.
Results are indented JSON; watch commands emit compact newline-delimited JSON (NDJSON).
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	format := "json"
	fail := func(err error, code int) int {
		problem := protocol.APIError{Code: "client_error", Message: err.Error()}
		var remote *client.RemoteError
		if errors.As(err, &remote) {
			problem = remote.Problem
		} else if errors.Is(err, client.ErrIdentityMismatch) {
			problem = protocol.APIError{Code: "runner_identity_mismatch", Message: client.ErrIdentityMismatch.Error()}
		}
		encoder := newOutput(stderr, format, false)
		_ = encoder.Encode(protocol.ErrorResponse{Error: problem})
		return code
	}

	globals := flag.NewFlagSet("urbit-collector", flag.ContinueOnError)
	globals.SetOutput(io.Discard)
	globals.Usage = func() {}
	socket := globals.String("socket", "", "absolute path to the runner Unix socket")
	runnerName := globals.String("runner", "", "enrolled runner name (exclusive with --socket)")
	configDir := globals.String("config-dir", "", "private collector enrollment directory (default: XDG config/urbit-collector)")
	globals.StringVar(&format, "format", "json", "output format: json or text (before subcommand)")
	if err := globals.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, `Usage: urbit-collector [--config-dir /private/directory] [--socket /absolute/path/runner.sock | --runner NAME] COMMAND

Commands:
  enroll NAME --socket /absolute/path/runner.sock
  enroll NAME --ssh HOST_ALIAS [--ssh-config /absolute/config]
              Verify and save a named runner connection with its identity.
  runners     List saved connections without contacting runners.
  forget NAME --runner-id ID
              Remove only local enrollment, requiring its expected runner ID.
  inventory [--watch] [--interval 1s]
              Observe all enrolled runners; watch emits independent NDJSON updates.
  hello       Verify compatibility and show runner capabilities.
  snapshot    Show the current runner snapshot.
  create-fake INSTANCE --http-port PORT --request-id ID [--label TEXT]
              Register a fresh fake instance (does not start it).
  configure INSTANCE --revision N --http-port PORT --ames-port PORT --loom EXPONENT
              Replace configuration; all flags are required. Demo loom: 31..33,
              not universal runtime bounds.
  operate INSTANCE start|stop --revision N --request-id ID
              Submit one operation; both flags are required.
  watch [--interval 1s]
               Poll snapshots until canceled; minimum interval: 100ms.
  ship-enroll NAME --url ORIGIN --ship ~SHIP [--credentials secret-service|file]
  ships
  ship-login NAME --code-stdin
  ship-status NAME
  ship-scry NAME --app APP --path /PATH
  ship-watch NAME --app APP --path /PATH [--events N] [--duration 30s]
  ship-logout NAME
  ship-forget NAME --url EXPECTED_ORIGIN
               Manage independent, origin-bound Eyre application sessions.

Use --format text for readable tables, including watch output.
Use COMMAND --help for command flags and constraints.

`+scope)
			return 0
		}
		return fail(err, 2)
	}
	args = globals.Args()
	if format != "json" && format != "text" {
		return fail(errors.New("--format must be json or text"), 2)
	}
	if len(args) == 0 {
		return fail(errors.New("a subcommand is required; use --help for usage"), 2)
	}
	command := args[0]
	args = args[1:]
	if slices.Contains([]string{"ship-enroll", "ships", "ship-login", "ship-status", "ship-scry", "ship-watch", "ship-logout", "ship-forget"}, command) {
		return runShips(ctx, command, args, *configDir, *socket, *runnerName, format, os.Stdin, stdout, fail)
	}
	if command == "enroll" || command == "runners" || command == "forget" || command == "inventory" {
		return runCollection(ctx, command, args, *configDir, *socket, *runnerName, format, stdout, fail)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var revision uint64
	var httpPort, amesPort, loom uint
	var requestID string
	var label string
	var interval time.Duration
	usage := command
	switch command {
	case "hello", "snapshot":
	case "create-fake":
		usage += " INSTANCE --http-port PORT --request-id ID [--label TEXT]"
		flags.UintVar(&httpPort, "http-port", 0, "required loopback host HTTP port, 1024..65535")
		flags.StringVar(&requestID, "request-id", "", "required caller-chosen idempotency key")
		flags.StringVar(&label, "label", "", "optional display label")
	case "configure":
		usage += " INSTANCE --revision N --http-port PORT --ames-port PORT --loom EXPONENT"
		flags.Uint64Var(&revision, "revision", 0, "required positive expected configuration revision")
		flags.UintVar(&httpPort, "http-port", 0, "required HTTP port, 1..65535")
		flags.UintVar(&amesPort, "ames-port", 0, "required Ames port, 1..65535")
		flags.UintVar(&loom, "loom", 0, "required demo loom exponent, 31..33 (not universal runtime bounds)")
	case "operate":
		usage += " INSTANCE start|stop --revision N --request-id ID"
		flags.Uint64Var(&revision, "revision", 0, "required positive expected configuration revision")
		flags.StringVar(&requestID, "request-id", "", "required caller-chosen ID: [A-Za-z0-9_-], 1..64 characters")
	case "watch":
		usage += " [--interval 1s]"
		flags.DurationVar(&interval, "interval", time.Second, "snapshot polling interval, at least 100ms; cancellation exits successfully")
	default:
		return fail(errors.New("unknown subcommand; use --help for usage"), 2)
	}

	var instanceID, kind string
	if command == "configure" || command == "operate" || command == "create-fake" {
		if len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
			instanceID, args = args[0], args[1:]
		}
		if command == "operate" && len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
			kind, args = args[0], args[1:]
		}
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector --socket /absolute/path/runner.sock %s\n\n%s\n", usage, scope)
			flags.SetOutput(stdout)
			flags.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 {
		return fail(errors.New("unexpected trailing arguments; command flags must follow positional arguments"), 2)
	}
	if (*socket == "") == (*runnerName == "") || (*socket != "" && !collection.ValidSocket(*socket)) || (*runnerName != "" && !validID(*runnerName)) {
		return fail(errors.New("select either --socket with an absolute clean path or --runner with an enrolled name before the subcommand"), 2)
	}
	if *configDir != "" && (!filepath.IsAbs(*configDir) || filepath.Clean(*configDir) != *configDir) {
		return fail(errors.New("--config-dir must be absolute and clean"), 2)
	}
	if command == "configure" || command == "operate" {
		if !validID(instanceID) {
			return fail(errors.New("instance ID is required: use 1..64 ASCII letters, digits, underscores, or hyphens"), 2)
		}
		if revision == 0 {
			return fail(errors.New("--revision is required and must be positive"), 2)
		}
	}
	switch command {
	case "create-fake":
		if !validID(instanceID) || !validID(requestID) || httpPort < 1024 || httpPort > 65535 {
			return fail(errors.New("create-fake requires valid instance/request IDs and --http-port in 1024..65535"), 2)
		}
	case "configure":
		if httpPort == 0 || httpPort > 65535 || amesPort == 0 || amesPort > 65535 {
			return fail(errors.New("--http-port and --ames-port are required and must be in 1..65535"), 2)
		}
		if loom < 31 || loom > 33 {
			return fail(errors.New("--loom is required and must be in 31..33 for this demo, not universal runtime bounds"), 2)
		}
	case "operate":
		if kind != "start" && kind != "stop" {
			return fail(errors.New("operation must be start or stop"), 2)
		}
		if !validID(requestID) {
			return fail(errors.New("--request-id is required: use 1..64 ASCII letters, digits, underscores, or hyphens"), 2)
		}
	case "watch":
		if interval < 100*time.Millisecond {
			return fail(errors.New("--interval must be at least 100ms"), 2)
		}
	}

	var c *client.Client
	var identity *client.Identity
	if *runnerName != "" {
		registry, err := openRegistry(*configDir)
		if err != nil {
			return fail(err, 1)
		}
		entry, err := registry.Get(ctx, *runnerName)
		if err != nil {
			if command == "watch" && ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return 0
			}
			return fail(err, 1)
		}
		pin := entry.Identity()
		identity = &pin
		c = entry.Client(true)
	} else {
		c = client.New(*socket)
	}
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err == nil && identity != nil {
		err = identity.Check(hello.RunnerID, hello.Mode)
	}
	if err != nil {
		if command == "watch" && ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return 0
		}
		return fail(err, 1)
	}
	encoder := newOutput(stdout, format, command == "watch")
	var result any
	switch command {
	case "hello":
		result = hello
	case "snapshot":
		result, err = checkedSnapshot(ctx, c, identity)
	case "create-fake":
		if !slices.Contains(hello.Features, "create-fake") {
			return fail(errors.New("this runner does not support create-fake"), 1)
		}
		result, err = c.CreateFake(ctx, protocol.CreateFakeRequest{ID: instanceID, Label: label, RequestID: requestID, Config: protocol.Configuration{HTTPPort: uint16(httpPort), AmesPort: 34343, Loom: 31}})
	case "configure":
		result, err = c.Configure(ctx, instanceID, protocol.ConfigureRequest{
			ExpectedRevision: revision,
			Config: protocol.Configuration{
				HTTPPort: uint16(httpPort), AmesPort: uint16(amesPort), Loom: uint8(loom),
			},
		})
	case "operate":
		result, err = c.Operate(ctx, instanceID, protocol.OperationRequest{
			RequestID: requestID, Kind: kind, ExpectedRevision: revision,
		})
	case "watch":
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return 0
			}
			snapshot, err := checkedSnapshot(ctx, c, identity)
			if err != nil {
				if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
					return 0
				}
				return fail(err, 1)
			}
			if err := encoder.Encode(snapshot); err != nil {
				return fail(err, 1)
			}
			select {
			case <-ctx.Done():
				return 0
			case <-ticker.C:
			}
		}
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := encoder.Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}

func checkedSnapshot(ctx context.Context, c *client.Client, identity *client.Identity) (protocol.Snapshot, error) {
	snapshot, err := c.Snapshot(ctx)
	if err == nil && identity != nil {
		err = identity.Check(snapshot.RunnerID, snapshot.Mode)
	}
	return snapshot, err
}

func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-') {
			return false
		}
	}
	return true
}
