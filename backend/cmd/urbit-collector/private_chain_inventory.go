package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runPrivateChainProjects(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID private-chain-projects

Read the selected runner's saved private-chain project reservations. Observer access
is sufficient. Requires private-chain-reservations-v1. This does not inspect a live
chain or Docker, read keys, probe ships, or grant runtime authority. Planned ports
are reservations only. Private-chain lifecycle is not implemented yet.
`
	flags := flag.NewFlagSet("private-chain-projects", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(stdout, usage)
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || runner == "" || socket != "" || !protocol.ValidID(expected) {
		return fail(errors.New("private-chain-projects requires --runner and --expected-runner-id; use --help"), 2)
	}
	registry, err := openRegistry(config)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := registry.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	pin := entry.Identity()
	if pin.RunnerID != expected {
		return fail(client.ErrIdentityMismatch, 1)
	}
	c := entry.Client(true)
	defer c.Close()
	result, err := c.PrivateProjects(ctx, pin)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Saved private-chain reservations on %s. Runtime observation: not performed.\n", result.RunnerID)
		for _, p := range result.Projects {
			if err != nil {
				break
			}
			_, err = fmt.Fprintf(stdout, "%s  %s  revision %d, generation %d (%s)\n", p.Binding.ProjectID, p.Plan.Label, p.Revision, p.Binding.Generation, p.Phase)
		}
		if err == nil && len(result.Projects) == 0 {
			_, err = io.WriteString(stdout, "No private-chain projects reserved.\n")
		}
	} else {
		err = newOutput(stdout, format, false).Encode(result)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
