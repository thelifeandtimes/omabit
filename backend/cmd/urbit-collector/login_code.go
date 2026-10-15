package main

import (
	"context"
	"errors"
	"flag"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runLoginCode(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	if len(args) == 0 {
		return fail(errors.New("login-code requires INSTANCE and an explicit --reveal acknowledgement"), 2)
	}
	instance := args[0]
	f := flag.NewFlagSet("login-code", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	revision := f.Uint64("revision", 0, "selected revision")
	container := f.String("container-id", "", "selected full container ID")
	reveal := f.Bool("reveal", false, "explicitly allow the login code in stdout")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 || !*reveal || socket != "" || !protocol.ValidID(runner) || !protocol.ValidID(expected) || !protocol.ValidID(instance) || *revision == 0 || len(*container) != 64 {
		return fail(errors.New("login-code requires --runner, --expected-runner-id, INSTANCE, --revision, --container-id and --reveal"), 2)
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	code, err := r.LoginCode(ctx, runner, expected, instance, *revision, *container)
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(code); err != nil {
		return fail(errors.New("could not write login-code response"), 1)
	}
	return 0
}
