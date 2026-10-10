package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runPiers(ctx context.Context, command string, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var req protocol.EnrollPierRequest
	var port uint
	if command != "piers" {
		if len(args) > 0 && args[0] != "--help" {
			req.ID, args = args[0], args[1:]
		}
		f.StringVar(&req.Path, "path", "", "convention-relative pier directory")
		f.StringVar(&req.RequestID, "request-id", "", "required actor-scoped request identity")
		f.Uint64Var(&req.ExpectedRevision, "revision", 0, "selected instance revision for move/relink")
		if command == "enroll-pier" {
			f.StringVar(&req.Ship, "ship", "", "expected public Urbit identity")
			f.StringVar(&req.Fingerprint, "fingerprint", "", "fingerprint from current discovery")
			f.StringVar(&req.Label, "label", "", "instance label")
			f.UintVar(&port, "http-port", 0, "unprivileged host HTTP port")
			f.BoolVar(&req.ConfirmStopped, "confirm-stopped", false, "source is stopped and will not run elsewhere")
			f.BoolVar(&req.Relink, "relink", false, "reconnect the same filesystem object after a move")
		}
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID %s [INSTANCE flags]\n", command)
			f.SetOutput(stdout)
			f.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || !validID(runner) || !validID(expected) {
		return fail(errors.New("pier commands require a pinned enrolled runner"), 2)
	}
	if command != "piers" && (!validID(req.ID) || !validID(req.RequestID) || !pierstore.ValidPath(req.Path)) {
		return fail(errors.New("require instance/request IDs and a convention-relative --path"), 2)
	}
	if command == "move-pier" && req.ExpectedRevision == 0 {
		return fail(errors.New("move-pier requires --revision"), 2)
	}
	if command == "enroll-pier" {
		if !strings.HasPrefix(req.Ship, "~") {
			req.Ship = "~" + req.Ship
		}
		if !protocol.ValidFakeShip(req.Ship) || !epochs.ValidDigest(req.Fingerprint) || !req.ConfirmStopped || req.Relink && req.ExpectedRevision == 0 || !req.Relink && (port < 1024 || port > 65535) {
			return fail(errors.New("require expected ~ship, discovery fingerprint, stopped-source confirmation and HTTP port (or --relink --revision)"), 2)
		}
		req.HTTPPort = uint16(port)
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
		return fail(errors.New("runner selection changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	h, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	if !slices.Contains(h.Features, "pier-storage") || command != "piers" && h.AccessRole != "operator" {
		return fail(errors.New("require pier-storage capability and operator access for enrollment/moves"), 1)
	}
	var result any
	switch command {
	case "piers":
		s, e := c.Snapshot(ctx)
		err = e
		if err == nil {
			if s.RunnerID != expected || s.Piers == nil {
				return fail(errors.New("pier discovery identity or catalogue unavailable"), 1)
			}
			result = s.Piers
		}
	case "move-pier":
		result, err = c.Operate(ctx, req.ID, protocol.OperationRequest{Kind: "move-pier", PierPath: req.Path, RequestID: req.RequestID, ExpectedRevision: req.ExpectedRevision})
	case "enroll-pier":
		result, err = c.EnrollPier(ctx, req)
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
