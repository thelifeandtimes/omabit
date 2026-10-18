package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runPublishAmes(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet("publish-ames", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var id string
	var port uint
	var req protocol.PublishAmesRequest
	if len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
		id, args = args[0], args[1:]
	}
	f.Uint64Var(&req.ExpectedRevision, "revision", 0, "required selected configuration revision")
	f.StringVar(&req.RequestID, "request-id", "", "required actor-scoped request ID; reuse only for exact replay")
	f.UintVar(&port, "ames-port", 0, "required host UDP port, 1024..65535")
	f.BoolVar(&req.ConfirmNetwork, "confirm-network", false, "publish Ames UDP on the next explicit Start; the ship remains stopped now")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID publish-ames INSTANCE --revision N --ames-port PORT --request-id ID --confirm-network")
			f.SetOutput(stdout)
			f.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || !validID(runner) || !validID(expected) || !validID(id) || !validID(req.RequestID) || req.ExpectedRevision == 0 || req.ExpectedRevision >= math.MaxInt64 || port < 1024 || port > 65535 || !req.ConfirmNetwork {
		return fail(errors.New("publish-ames requires a pinned enrolled runner, instance/request IDs, --revision, --ames-port in 1024..65535 and --confirm-network"), 2)
	}
	req.AmesPort = uint16(port)
	registry, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := registry.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	if entry.RunnerID != expected {
		return fail(errors.New("runner selection changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	op, err := c.PublishAmes(ctx, id, req)
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(op); err != nil {
		return fail(err, 1)
	}
	return 0
}
