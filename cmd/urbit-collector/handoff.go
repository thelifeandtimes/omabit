package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
)

func runHandoff(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet("move-host", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var instance, destination string
	var req protocol.PrepareHandoffRequest
	var httpPort, amesPort uint
	if len(args) > 0 && args[0] != "--help" {
		instance, args = args[0], args[1:]
	}
	f.StringVar(&destination, "to-runner", "", "destination enrollment name")
	f.StringVar(&req.DestinationRunner, "to-runner-id", "", "pinned destination runner identity")
	f.StringVar(&req.DestinationInstance, "to-instance", "", "new destination instance ID")
	f.StringVar(&req.RequestID, "request-id", "", "explicit attempt ID (maximum 40 characters)")
	f.Uint64Var(&req.ExpectedRevision, "revision", 0, "source configuration revision")
	f.UintVar(&httpPort, "http-port", 0, "destination HTTP port (0 allocates)")
	f.UintVar(&amesPort, "ames-port", 0, "destination Ames port if published (0 allocates)")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner SOURCE --expected-runner-id ID move-host INSTANCE --to-runner NAME --to-runner-id ID --to-instance ID --revision N --request-id ID\nStop first. A move permanently fences the source; destination remains stopped. Repeat with a new request ID to continue an interrupted move.")
			f.SetOutput(stdout)
			f.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || !validID(runner) || !validID(expected) || !validID(instance) || !validID(destination) || !validID(req.DestinationRunner) || expected == req.DestinationRunner || !validID(req.DestinationInstance) || !validID(req.RequestID) || len(req.RequestID) > 40 || req.ExpectedRevision == 0 || httpPort > 65535 || amesPort > 65535 || httpPort != 0 && httpPort < 1024 || amesPort != 0 && amesPort < 1024 {
		return fail(errors.New("move-host requires two pinned runners, source revision, destination instance, valid ports and an attempt ID of at most 40 characters"), 2)
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	a, err := r.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	b, err := r.Get(ctx, destination)
	if err != nil {
		return fail(err, 1)
	}
	if a.RunnerID != expected || b.RunnerID != req.DestinationRunner {
		return fail(client.ErrIdentityMismatch, 1)
	}
	source, target := a.Client(true), b.Client(true)
	defer source.Close()
	defer target.Close()
	h, err := client.MoveStopped(ctx, source, target, expected, instance, req, uint16(httpPort), uint16(amesPort))
	if err != nil {
		return fail(err, 1)
	}
	if err = newOutput(stdout, format, false).Encode(h); err != nil {
		return fail(err, 1)
	}
	return 0
}
