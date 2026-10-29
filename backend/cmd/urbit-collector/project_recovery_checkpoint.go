package main

import (
	"context"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
)

func runProjectRecoveryCheckpoint(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-recovery-checkpoint PROJECT RECOVERY_OPERATION

Read the exact saved recovery checkpoint and its review/history/checkpoint digests.
Requires project-recovery-checkpoint-v1. Observer access is sufficient. This reads
stored history only: no Docker probe, login, worker or runtime action is performed.
The next-step hint grants no permission to retry an effect. An interrupted unknown
Create stays review-required; lookup is never a second Create or adoption.
Current runtime health and OS service restart recovery are not established by
this read. Use project-status to find the owning recover-project-anchor receipt.
`
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(stdout, usage)
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if len(args) != 2 || !protocol.ValidID(args[0]) || !protocol.ValidID(args[1]) || socket != "" || runner == "" || !protocol.ValidID(expected) {
		return fail(errors.New("project-recovery-checkpoint requires a pinned enrolled runner, project and recovery operation; use --help"), 2)
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
	result, err := c.ProjectRecoveryCheckpoint(ctx, pin, args[0], args[1])
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Project %s recovery %s: %s, sequence %d (%s).\nSaved next-step hint: %s %s %s.\nReview digest: %s\nHistory digest: %s\nCheckpoint digest: %s\nSaved history only; no current health check or action permission.\n", result.Project.ID, result.Operation.ID, result.Phase, result.Sequence, result.Operation.Status, result.Next.Kind, result.Next.TargetKind, result.Next.InstanceID, result.Project.ReviewDigest, result.HistoryDigest, result.CheckpointDigest)
	} else {
		err = newOutput(stdout, format, false).Encode(result)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
