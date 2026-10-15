package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runImports(ctx context.Context, command string, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var id, source, digest, request, label string
	var port uint
	var confirm bool
	if command == "import-fake" {
		if len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
			id, args = args[0], args[1:]
		}
		flags.StringVar(&source, "source-run", "", "selected completed qualification source ID")
		flags.StringVar(&digest, "expected-digest", "", "selected source content digest")
		flags.StringVar(&request, "request-id", "", "actor-scoped request identity")
		flags.StringVar(&label, "label", "", "optional instance label")
		flags.UintVar(&port, "http-port", 0, "reserved host HTTP port, 1024..65535")
		flags.BoolVar(&confirm, "confirm-stopped", false, "acknowledge the selected stopped source")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID "+command)
			if command == "import-fake" {
				fmt.Fprintln(stdout, "  INSTANCE --source-run RUN_ID --expected-digest HASH --http-port PORT --request-id ID --confirm-stopped [--label TEXT]")
			}
			fmt.Fprintln(stdout, "Only completed pinned disposable-fake qualification sources are eligible; no external path or real pier is accepted.")
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || socket != "" || !validID(runner) || !validID(expected) {
		return fail(errors.New("import commands require an enrolled runner and its expected identity"), 2)
	}
	if command == "import-fake" {
		_, sourceErr := hex.DecodeString(source)
		_, digestErr := hex.DecodeString(digest)
		if !validID(id) || !validID(request) || !confirm || port < 1024 || port > 65535 || len(source) != 32 || sourceErr != nil || source != strings.ToLower(source) || len(digest) != 64 || digestErr != nil || digest != strings.ToLower(digest) {
			return fail(errors.New("require valid instance/request IDs, source ID, expected SHA-256, HTTP port and --confirm-stopped"), 2)
		}
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := r.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	if entry.RunnerID != expected {
		return fail(errors.New("selected runner enrollment changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	if hello.AccessRole != "operator" || !slices.Contains(hello.Features, "import-qualified-fake") {
		return fail(errors.New("runner requires operator access and import-qualified-fake capability"), 1)
	}
	var result any
	if command == "import-sources" {
		result, err = c.ImportSources(ctx, expected)
	} else {
		result, err = c.ImportFake(ctx, protocol.ImportFakeRequest{ID: id, Label: label, Config: protocol.Configuration{HTTPPort: uint16(port), AmesPort: 34343, Loom: 31}, RequestID: request, SourceRunID: source, ExpectedDigest: digest, ConfirmStopped: confirm})
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
