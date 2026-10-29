package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runLoginCode(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	if len(args) == 0 {
		return fail(errors.New("login-code requires INSTANCE and an explicit --reveal acknowledgement"), 2)
	}
	instance := ""
	if args[0] != "--help" && args[0] != "-h" {
		instance, args = args[0], args[1:]
	}
	f := flag.NewFlagSet("login-code", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	revision := f.Uint64("revision", 0, "selected revision")
	container := f.String("container-id", "", "selected full container ID")
	reveal := f.Bool("reveal", false, "explicitly allow the login code in stdout")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID login-code INSTANCE --revision N --container-id FULL_ID --reveal\n\nOperator-only: reveal the selected running ship's login code in stdout.\nUse only when the task needs this secret; keep it out of normal status reports.\nRead a fresh snapshot first: select config_revision and resources.container_id for the running instance.\nA completed Start can precede that resource observation. If resources.container_id is absent, wait and read snapshot again; do not substitute an old container ID.\nThe runner requires a fresh observation and no active operation or recovery fence.\nHelp does not contact a runner or retrieve a code.")
			f.SetOutput(stdout)
			f.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || !*reveal || socket != "" || !protocol.ValidID(runner) || !protocol.ValidID(expected) || !protocol.ValidID(instance) || *revision == 0 || len(*container) != 64 {
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
