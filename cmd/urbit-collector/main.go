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
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
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
	expectedRunner := globals.String("expected-runner-id", "", "optional identity from the selected UI snapshot; requires --runner")
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
  import-sources
              List completed, runner-owned qualified fake-pier sources.
  import-fake INSTANCE --source-run RUN_ID --expected-digest HASH --http-port PORT
              --request-id ID --confirm-stopped [--label TEXT]
              Durably copy/register a qualified stopped fake; does not start it.
  runner-edit NAME --runner-id ID --revision N --name NEW_NAME
              (--socket /absolute/path | --ssh HOST_ALIAS [--ssh-config /absolute/config])
              Verify the same runner at a new endpoint and atomically update enrollment.
  runner-accept-mode NAME --runner-id ID --revision N --mode development-managed
              Explicitly accept mixed development mode at the same pinned endpoint.
  forget NAME --runner-id ID
              Remove only local enrollment, requiring its expected runner ID.
  inventory [--watch] [--interval 1s]
              Observe all enrolled runners; watch emits independent NDJSON updates.
  hello       Verify compatibility and show runner capabilities.
  login-code INSTANCE --revision N --container-id FULL_ID --reveal
              Operator-only secret response; requires pinned --runner/--expected-runner-id.
  snapshot    Show the current runner snapshot.
  logs INSTANCE --revision N
              Read a bounded operator-only runtime log tail from the pinned runner.
  piers       Discover enrolled, moved and unenrolled piers (archive is excluded).
  create-ship --ship ~SHIP --key-file FILE --request-id ID --confirm-network
               [--http-port PORT --ames-port PORT --bootstrap bare|recommended --pill URL_OR_FILE]
              Derive instance/path, allocate omitted ports, and stage boot material.
              Blank pill uses the runtime canonical default; explicit Start boots it.
  enroll-pier INSTANCE --path RELATIVE --ship ~SHIP --fingerprint HASH
              --http-port PORT --request-id ID --confirm-stopped
              Enroll a discovered stopped pier without bootstrapping or starting it.
              Use --relink --revision N instead of --http-port after moving a registered pier.
  move-pier INSTANCE --path RELATIVE --revision N --request-id ID
               Move a stopped pier into the storage convention without replacing a destination.
  move-host INSTANCE --to-runner NAME --to-runner-id ID --to-instance ID
               --revision N --request-id ID [--http-port PORT --ames-port PORT]
               Transfer a cleanly stopped pier; permanently fence its source.
               Destination stays stopped. Continue interruptions with a new request ID.
  routine INSTANCE
              Read the selected instance's explicit UTC maintenance policy.
  routine-set INSTANCE --revision N --instance-revision N --enabled=true|false
              --action pack|meld --days 0,1,... --hour-utc N --minute-utc N
              --window-minutes N --stop-running=true|false --resume=true|false
              Save a revision-bound routine; no retroactive or immediate execution.
  create-fake INSTANCE --http-port PORT --request-id ID [--label TEXT]
               [--bootstrap bare|recommended --pill URL_OR_FILE]
               Register a fresh fake instance (does not start it).
  create-comet INSTANCE --http-port PORT --request-id ID --confirm-network [--label TEXT]
               [--bootstrap bare|recommended --pill URL_OR_FILE]
              Register a new comet; first explicit Start generates its identity.
  configure INSTANCE --revision N --http-port PORT --ames-port PORT --loom EXPONENT
              Replace configuration; all flags are required. Demo loom: 31..33,
              not universal runtime bounds.
  operate INSTANCE start|stop|pack|meld|roll|inspect-epochs --revision N --request-id ID
              Submit one operation; both flags are required.
  epoch-preview INSTANCE --revision N --expected-digest HASH --cutoff 0iEVENT
              Preview an oldest prefix from a recent stopped-pier inspection.
  epoch-prune INSTANCE --revision N --expected-digest HASH --cutoff 0iEVENT
              --request-id ID --confirm-history-loss
              Explicitly remove the selected oldest prefix, retaining newest two.
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
	if *expectedRunner != "" && (*runnerName == "" || !validID(*expectedRunner)) {
		return fail(errors.New("--expected-runner-id requires --runner and a valid runner ID"), 2)
	}
	if command == "login-code" {
		return runLoginCode(ctx, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "logs" {
		return runLogs(ctx, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "move-host" {
		return runHandoff(ctx, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "piers" || command == "enroll-pier" || command == "move-pier" {
		return runPiers(ctx, command, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "create-ship" {
		return runKeyBoot(ctx, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "routine" || command == "routine-set" {
		return runRoutine(ctx, command, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if command == "import-sources" || command == "import-fake" {
		return runImports(ctx, command, args, *configDir, *socket, *runnerName, *expectedRunner, format, stdout, fail)
	}
	if slices.Contains([]string{"ship-enroll", "ships", "ship-login", "ship-status", "ship-scry", "ship-watch", "ship-logout", "ship-forget"}, command) {
		return runShips(ctx, command, args, *configDir, *socket, *runnerName, format, os.Stdin, stdout, fail)
	}
	if command == "enroll" || command == "runner-edit" || command == "runner-accept-mode" || command == "runners" || command == "forget" || command == "inventory" {
		return runCollection(ctx, command, args, *configDir, *socket, *runnerName, format, stdout, fail)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var revision uint64
	var httpPort, amesPort, loom uint
	var requestID string
	var label string
	var fakeShip, pierName string
	var bootstrap, pillSource string
	var pierPath string
	var fakeOptions bool
	var cutoff, digest string
	var confirmHistoryLoss bool
	var confirmNetwork bool
	var interval time.Duration
	usage := command
	switch command {
	case "hello", "snapshot":
	case "create-fake", "create-comet":
		usage += " INSTANCE --http-port PORT --request-id ID [--label TEXT]"
		flags.UintVar(&httpPort, "http-port", 0, "required loopback host HTTP port, 1024..65535")
		flags.StringVar(&requestID, "request-id", "", "required caller-chosen idempotency key")
		flags.StringVar(&label, "label", "", "optional display label")
		flags.StringVar(&pierPath, "pier-path", "", "permanent pier path, e.g. fakeships/project/zod/pier or livenet/comets/testing-comet-1/pier")
		flags.StringVar(&bootstrap, "bootstrap", "", "bare or recommended; recommended requires --pill")
		flags.StringVar(&pillSource, "pill", "", "selected pill URL or local file for recommended bootstrap")
		usage += " [--bootstrap bare|recommended --pill URL_OR_FILE]"
		if command == "create-fake" {
			flags.StringVar(&fakeShip, "ship", "~zod", "fake Urbit identity, e.g. ~bus (no real keys or Ames)")
			flags.StringVar(&pierName, "pier-name", "", "directory basename within instance storage; defaults to the instance name when options are supplied")
			usage += " [--ship ~SHIP --pier-name DIRECTORY]"
		}
		if command == "create-comet" {
			flags.BoolVar(&confirmNetwork, "confirm-network", false, "acknowledge internally generated comet identity and outbound networking on explicit Start")
			usage += " --confirm-network"
		}
	case "configure":
		usage += " INSTANCE --revision N --http-port PORT --ames-port PORT --loom EXPONENT"
		flags.Uint64Var(&revision, "revision", 0, "required positive expected configuration revision")
		flags.UintVar(&httpPort, "http-port", 0, "required HTTP port, 1..65535")
		flags.UintVar(&amesPort, "ames-port", 0, "required Ames port, 1..65535")
		flags.UintVar(&loom, "loom", 0, "required demo loom exponent, 31..33 (not universal runtime bounds)")
	case "operate":
		usage += " INSTANCE start|stop|pack|meld|roll|inspect-epochs --revision N --request-id ID"
		flags.Uint64Var(&revision, "revision", 0, "required positive expected configuration revision")
		flags.StringVar(&requestID, "request-id", "", "required caller-chosen ID: [A-Za-z0-9_-], 1..64 characters")
	case "epoch-preview", "epoch-prune":
		usage += " INSTANCE --revision N --expected-digest HASH --cutoff 0iEVENT"
		flags.Uint64Var(&revision, "revision", 0, "required expected configuration revision")
		flags.StringVar(&digest, "expected-digest", "", "required inventory SHA-256 from the selected snapshot")
		flags.StringVar(&cutoff, "cutoff", "", "selected epoch and all older epochs; newest two are protected")
		if command == "epoch-prune" {
			usage = "epoch-prune INSTANCE --revision N --expected-digest HASH --cutoff 0iEVENT --request-id ID --confirm-history-loss"
			flags.StringVar(&requestID, "request-id", "", "required caller-chosen idempotency key")
			flags.BoolVar(&confirmHistoryLoss, "confirm-history-loss", false, "confirm irreversible removal of the selected epoch history")
		}
	case "watch":
		usage += " [--interval 1s]"
		flags.DurationVar(&interval, "interval", time.Second, "snapshot polling interval, at least 100ms; cancellation exits successfully")
	default:
		return fail(errors.New("unknown subcommand; use --help for usage"), 2)
	}

	var instanceID, kind string
	if command == "configure" || command == "operate" || command == "create-fake" || command == "create-comet" || command == "epoch-preview" || command == "epoch-prune" {
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
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "ship" || f.Name == "pier-name" {
			fakeOptions = true
		}
	})
	if fakeOptions {
		if !strings.HasPrefix(fakeShip, "~") {
			fakeShip = "~" + fakeShip
		}
		if pierName == "" {
			pierName = instanceID
		}
		if !protocol.ValidFakeShip(fakeShip) || len(fakeShip) > 28 || !protocol.ValidPierName(pierName) {
			return fail(errors.New("require a valid fake Urbit name and a pier directory basename (letters, digits, hyphens, underscores; start with a letter or digit)"), 2)
		}
	}
	if (*socket == "") == (*runnerName == "") || (*socket != "" && !collection.ValidSocket(*socket)) || (*runnerName != "" && !validID(*runnerName)) {
		return fail(errors.New("select either --socket with an absolute clean path or --runner with an enrolled name before the subcommand"), 2)
	}
	if *configDir != "" && (!filepath.IsAbs(*configDir) || filepath.Clean(*configDir) != *configDir) {
		return fail(errors.New("--config-dir must be absolute and clean"), 2)
	}
	if command == "configure" || command == "operate" || command == "epoch-preview" || command == "epoch-prune" {
		if !validID(instanceID) {
			return fail(errors.New("instance ID is required: use 1..64 ASCII letters, digits, underscores, or hyphens"), 2)
		}
		if revision == 0 {
			return fail(errors.New("--revision is required and must be positive"), 2)
		}
	}
	switch command {
	case "create-fake", "create-comet":
		if _, err := bootstrapChoice(bootstrap, pillSource); err != nil {
			return fail(err, 2)
		}
		if pierPath != "" && (!pierstore.ValidPath(pierPath) || (command == "create-fake" && pierstore.Kind(pierPath) != "fake") || (command == "create-comet" && pierstore.Kind(pierPath) != "comet")) {
			return fail(errors.New("invalid pier convention path for this ship type"), 2)
		}
		if !validID(instanceID) || !validID(requestID) || httpPort < 1024 || httpPort > 65535 {
			return fail(errors.New("create-fake requires valid instance/request IDs and --http-port in 1024..65535"), 2)
		}
		if command == "create-comet" && (!confirmNetwork || *runnerName == "" || *expectedRunner == "") {
			return fail(errors.New("create-comet requires a pinned --runner/--expected-runner-id and --confirm-network"), 2)
		}
	case "configure":
		if httpPort == 0 || httpPort > 65535 || amesPort == 0 || amesPort > 65535 {
			return fail(errors.New("--http-port and --ames-port are required and must be in 1..65535"), 2)
		}
		if loom < 31 || loom > 33 {
			return fail(errors.New("--loom is required and must be in 31..33 for this demo, not universal runtime bounds"), 2)
		}
	case "operate":
		if kind != "start" && kind != "stop" && kind != "pack" && kind != "meld" && kind != "roll" && kind != "inspect-epochs" {
			return fail(errors.New("operation must be start, stop, pack, meld, roll, or inspect-epochs"), 2)
		}
		if !validID(requestID) {
			return fail(errors.New("--request-id is required: use 1..64 ASCII letters, digits, underscores, or hyphens"), 2)
		}
	case "epoch-preview", "epoch-prune":
		if _, err := epochs.Number(cutoff); err != nil {
			return fail(err, 2)
		}
		if !epochs.ValidDigest(digest) {
			return fail(errors.New("--expected-digest requires the inventory SHA-256"), 2)
		}
		if command == "epoch-prune" && (!validID(requestID) || !confirmHistoryLoss) {
			return fail(errors.New("epoch-prune requires a request ID and --confirm-history-loss"), 2)
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
	if err == nil && *expectedRunner != "" && hello.RunnerID != *expectedRunner {
		err = client.ErrIdentityMismatch
	}
	if err != nil {
		if command == "watch" && ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return 0
		}
		return fail(err, 1)
	}
	encoder := newOutput(stdout, format, command == "watch")
	if pierPath != "" && !slices.Contains(hello.Features, "pier-storage") {
		return fail(errors.New("update this runner for convention-based pier storage"), 1)
	}
	var boot *protocol.Bootstrap
	if command == "create-fake" || command == "create-comet" {
		if command == "create-comet" && (hello.Mode != "development-managed" || hello.AccessRole != "operator" || !slices.Contains(hello.Features, "create-comet")) {
			return fail(errors.New("comet creation requires an operator and the explicit development-managed mode/capability"), 1)
		}
		if command == "create-fake" && !slices.Contains(hello.Features, "create-fake") {
			return fail(errors.New("this runner does not support create-fake"), 1)
		}
		supported := slices.Contains(hello.Features, "bootstrap-selection")
		if bootstrap != "" || pillSource != "" {
			if !supported || hello.AccessRole != "operator" {
				return fail(errors.New("bootstrap selection requires an updated operator runner"), 1)
			}
		}
		if supported {
			choice, _ := bootstrapChoice(bootstrap, pillSource)
			p, err := stageSelectedPill(ctx, c, pillSource)
			if err != nil {
				return fail(err, 1)
			}
			boot = &protocol.Bootstrap{Kind: choice, Pill: p}
		}
	}
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
		req := protocol.CreateFakeRequest{Bootstrap: boot, ID: instanceID, Label: label, RequestID: requestID, PierPath: pierPath, Config: protocol.Configuration{HTTPPort: uint16(httpPort), AmesPort: 34343, Loom: 31}}
		if fakeOptions {
			if !slices.Contains(hello.Features, "fake-ship-options") {
				return fail(errors.New("update this runner to choose a fake identity and pier directory"), 1)
			}
			req.FakeShip, req.PierName = fakeShip, pierName
		}
		result, err = c.CreateFake(ctx, req)
	case "create-comet":
		if hello.Mode != "development-managed" || hello.AccessRole != "operator" || !slices.Contains(hello.Features, "create-comet") {
			return fail(errors.New("comet creation requires an operator and the explicit development-managed mode/capability"), 1)
		}
		result, err = c.CreateComet(ctx, protocol.CreateCometRequest{CreateFakeRequest: protocol.CreateFakeRequest{Bootstrap: boot, ID: instanceID, Label: label, RequestID: requestID, PierPath: pierPath, Config: protocol.Configuration{HTTPPort: uint16(httpPort), AmesPort: 34343, Loom: 31}}, ConfirmNetwork: confirmNetwork})
	case "configure":
		result, err = c.Configure(ctx, instanceID, protocol.ConfigureRequest{
			ExpectedRevision: revision,
			Config: protocol.Configuration{
				HTTPPort: uint16(httpPort), AmesPort: uint16(amesPort), Loom: uint8(loom),
			},
		})
	case "operate":
		if (kind == "pack" || kind == "meld" || kind == "roll") && !slices.Contains(hello.Features, "offline-"+kind) {
			return fail(fmt.Errorf("this runner does not support offline %s", kind), 1)
		}
		if kind == "inspect-epochs" && !slices.Contains(hello.Features, "epoch-inventory") {
			return fail(errors.New("this runner does not support epoch inventory"), 1)
		}
		result, err = c.Operate(ctx, instanceID, protocol.OperationRequest{
			RequestID: requestID, Kind: kind, ExpectedRevision: revision,
		})
	case "epoch-preview":
		if !slices.Contains(hello.Features, "epoch-inventory") {
			return fail(errors.New("this runner does not support epoch inventory"), 1)
		}
		snapshot, readErr := checkedSnapshot(ctx, c, identity)
		if readErr != nil {
			return fail(readErr, 1)
		}
		if snapshot.RunnerID != hello.RunnerID || snapshot.Mode != hello.Mode {
			return fail(client.ErrIdentityMismatch, 1)
		}
		var selected *protocol.Instance
		for i := range snapshot.Instances {
			if snapshot.Instances[i].ID == instanceID {
				selected = &snapshot.Instances[i]
				break
			}
		}
		if selected == nil {
			return fail(errors.New("instance is absent from the snapshot"), 1)
		}
		if selected.Mode == "comet" && !slices.Contains(hello.Features, "comet-epochs") {
			return fail(errors.New("this runner does not support comet epoch previews"), 1)
		}
		if selected.ConfigRevision != revision || selected.ActiveOperationID != "" || selected.RecoveryRequired || selected.DesiredState != "stopped" || selected.ObservedState != "stopped" {
			return fail(errors.New("epoch preview requires the selected revision and a clean stopped instance without active work"), 1)
		}
		preview, planErr := epochs.Plan(selected.Epochs, cutoff, digest, time.Now().UTC())
		if planErr != nil {
			return fail(planErr, 1)
		}
		preview.RunnerID, preview.InstanceID, preview.ConfigRevision = snapshot.RunnerID, instanceID, revision
		prefixSupported := selected.Mode == "comet" && slices.Contains(hello.Features, "comet-prune-prefix") || selected.Mode != "comet" && slices.Contains(hello.Features, "epoch-prune-prefix")
		if prefixSupported {
			if _, scopeErr := epochs.PrunePlan(selected.Epochs, cutoff, digest, time.Now().UTC()); scopeErr == nil {
				preview.DeletionSupported = true
			}
		} else if selected.Mode != "comet" && slices.Contains(hello.Features, "epoch-prune-genesis") {
			if _, scopeErr := epochs.GenesisPlan(selected.Epochs, cutoff, digest, time.Now().UTC()); scopeErr == nil {
				preview.DeletionSupported = true
			}
		}
		result = preview
	case "epoch-prune":
		if !slices.Contains(hello.Features, "comet-prune-prefix") && !slices.Contains(hello.Features, "epoch-prune-prefix") && (!slices.Contains(hello.Features, "epoch-prune-genesis") || cutoff != "0i0") {
			return fail(errors.New("this runner does not support the requested prefix pruning scope"), 1)
		}
		result, err = c.Operate(ctx, instanceID, protocol.OperationRequest{RequestID: requestID, Kind: "prune-epochs", ExpectedRevision: revision, EpochCutoff: cutoff, InventoryDigest: digest, ConfirmHistoryLoss: confirmHistoryLoss})
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
