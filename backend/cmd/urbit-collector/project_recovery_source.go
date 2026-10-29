package main

import (
	"context"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
)

func runProjectRecoverySource(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-recovery-source PROJECT

Read the saved history digest and exact member creator/original-readiness receipts
for recovery review. Requires project-recovery-source-v1; observer access suffices.
Initial launches and readiness must exist. This performs no runtime observation,
login or mutation. Saved history grants no recovery authority or current health.
For an already interrupted recovery, use project-recovery-checkpoint with its exact
parent receipt instead of replacing its preserved source with today's history.
`
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(stdout, usage)
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if len(args) != 1 || !protocol.ValidID(args[0]) || socket != "" || runner == "" || !protocol.ValidID(expected) {
		return fail(errors.New("project-recovery-source requires a pinned enrolled runner and project; use --help"), 2)
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
	v, err := c.ProjectRecoverySource(ctx, pin, args[0])
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Project %s saved recovery source (revision %d).\nReview digest: %s\nHistory digest: %s\n", v.Project.ID, v.Project.Revision, v.Project.ReviewDigest, v.HistoryDigest)
		for _, m := range v.Members {
			if err == nil {
				_, err = fmt.Fprintf(stdout, "%s: creator %s, generation %d, original readiness %s, current readiness %s\n", m.InstanceID, m.CreatorOperationID, m.ContainerGeneration, m.OriginalReadinessOperationID, m.CurrentReadinessOperationID)
			}
		}
		if err == nil {
			_, err = io.WriteString(stdout, "Saved history only; no current health check or action permission.\n")
		}
	} else {
		err = newOutput(stdout, format, false).Encode(v)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
