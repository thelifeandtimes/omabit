package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runLogs(ctx context.Context, args []string, dir, socket, runner, expected, format string, out io.Writer, fail func(error, int) int) int {
	if len(args) == 0 {
		return fail(errors.New("logs requires INSTANCE --revision N and an identity-pinned runner"), 2)
	}
	id := args[0]
	f := flag.NewFlagSet("logs", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	rev := f.Uint64("revision", 0, "selected configuration revision")
	watch := f.Bool("watch", false, "reuse one pinned connection and emit bounded NDJSON updates until canceled")
	interval := f.Duration("interval", 3*time.Second, "log watch interval, at least 1s")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || *rev == 0 || socket != "" || !protocol.ValidID(id) || !protocol.ValidID(runner) || !protocol.ValidID(expected) {
		return fail(errors.New("logs requires --runner, --expected-runner-id, INSTANCE and --revision"), 2)
	}
	if *interval < time.Second || *watch && format != "json" {
		return fail(errors.New("log watch requires --format json and --interval of at least 1s"), 2)
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
	if *watch {
		encoder := newOutput(out, format, true)
		err = collection.WatchLogs(ctx, entry, id, *rev, *interval, func(update collection.LogUpdate) error {
			return encoder.Encode(update)
		})
		if ctx.Err() != nil {
			return 0
		}
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	c := entry.Client(true)
	defer c.Close()
	logs, err := c.RuntimeLogs(ctx, expected, id, *rev)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		for _, line := range logs.Lines {
			if _, err := fmt.Fprintln(out, cell(line)); err != nil {
				return fail(errors.New("could not write runtime logs"), 1)
			}
		}
		return 0
	}
	if err = newOutput(out, format, false).Encode(logs); err != nil {
		return fail(errors.New("could not write runtime logs"), 1)
	}
	return 0
}
