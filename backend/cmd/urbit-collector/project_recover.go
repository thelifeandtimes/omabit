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

func runProjectRecover(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-recover
       PROJECT start|observe|continue --revision N --review-digest HASH
       --history-digest HASH --request-id ID --confirm-action
       [--recovery OPERATION --sequence N --checkpoint-digest HASH] [--replay-only]

start reviews current runtime ownership, then coordinates graceful member drain,
namespace recovery and same-pier replacements. This may interrupt every member.
Use the exact project-recovery-source history digest for this new operation.

observe requires the exact interrupted parent and checkpoint; it observes one
uncertain outcome and journals evidence without repeating the effect.
continue requires that exact parent/checkpoint and separately authorizes new work.
Use project-recovery-checkpoint for both. Unknown-ID Create remains review-required;
no action adopts an unknown container or repeats an uncertain Create.

Each action requires its own capability and operator access. Saved digests/hints
do not grant authority. After a lost reply, preserve every input and add only
--replay-only to look up the same receipt without scheduling another worker.
An interrupted operation is not resumed by replay. Do not mint an ID to retry it.
`
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(stdout, usage)
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if len(args) < 2 {
		return fail(errors.New(usage), 2)
	}
	project, kind := args[0], args[1]
	flags := flag.NewFlagSet("project-recover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var req protocol.ProjectRecoveryActionRequest
	var parent string
	var seq uint64
	var replay bool
	flags.StringVar(&parent, "recovery", "", "exact retained parent for observe/continue")
	flags.Uint64Var(&seq, "sequence", 0, "exact reviewed checkpoint sequence")
	flags.StringVar(&req.CheckpointDigest, "checkpoint-digest", "", "exact reviewed checkpoint digest")
	flags.Uint64Var(&req.ExpectedRevision, "revision", 0, "reviewed project revision")
	flags.StringVar(&req.ReviewDigest, "review-digest", "", "reviewed plan digest")
	flags.StringVar(&req.HistoryDigest, "history-digest", "", "reviewed saved history digest")
	flags.StringVar(&req.RequestID, "request-id", "", "new explicit request or original lookup request")
	flags.BoolVar(&req.ConfirmAction, "confirm-action", false, "authorize only the selected recovery action")
	flags.BoolVar(&replay, "replay-only", false, "look up original receipt without work")
	if err := flags.Parse(args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(stdout, usage)
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "sequence" {
			req.ExpectedSequence = &seq
		}
	})
	if flags.NArg() != 0 || socket != "" || runner == "" || !protocol.ValidID(expected) || !protocol.ValidID(project) || !protocol.ValidProjectRecoveryAction(kind, parent, req) {
		return fail(errors.New("project-recover requires a pinned enrolled runner and exact explicitly reviewed action; use --help"), 2)
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
	if replay {
		ctx = client.WithCreationReplayOnly(ctx)
	}
	v, err := c.ProjectRecover(ctx, pin, project, parent, kind, req)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Project %s: %s receipt %s (%s), recovery parent %s.\nSaved receipt only; use project-recovery-checkpoint for the retained checkpoint.\n", project, v.Operation.Kind, v.Operation.ID, v.Operation.Status, v.RecoveryOperationID)
	} else {
		err = newOutput(stdout, format, false).Encode(v)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
