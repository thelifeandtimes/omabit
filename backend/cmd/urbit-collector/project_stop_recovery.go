package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runProjectStopRecovery(ctx context.Context, command string, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID COMMAND PROJECT
  project-diagnose PROJECT --member INSTANCE --operation ORIGINAL_STOP [--probe]
  project-reconcile-stop PROJECT --review DIAGNOSIS_JSON --request-id ID
                         --confirm-action [--replay-only]

project-diagnose reads saved state only. --probe additionally makes bounded,
read-only ownership/exit observations and returns an exact review when supported.
Save the JSON diagnosis from --probe, inspect its bindings, then pass that same
file to project-reconcile-stop. Exact private bindings are operator-only data;
do not publish this review as a shareable diagnostic bundle.

Reconciliation requires the original operator and a clean exit. It changes only
the selected stop fence and records a new receipt linked to the original
interruption. It never repeats SIGTERM, starts a ship, replaces a pier or fixes
an abnormal/OOM exit. Ordinary member-start remains a separate explicit action.
Keep the original review/request after a lost reply; add --replay-only to retrieve
that exact result without observing or applying again. A stale/changed review
requires new diagnosis and review, never automatic refresh into new authority.
`
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(stdout, usage)
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if len(args) == 0 {
		return fail(errors.New("require an exact project ID; use --help"), 2)
	}
	project := args[0]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var member, operation, reviewPath, request string
	var probe, confirm, replay bool
	if command == "project-diagnose" {
		f.StringVar(&member, "member", "", "exact member ID")
		f.StringVar(&operation, "operation", "", "original interrupted stop operation")
		f.BoolVar(&probe, "probe", false, "also observe exact runtime and prepare a review")
	} else {
		f.StringVar(&reviewPath, "review", "", "saved JSON diagnosis containing the exact reviewed stop")
		f.StringVar(&request, "request-id", "", "original request ID for this reconciliation")
		f.BoolVar(&confirm, "confirm-action", false, "authorize only the reviewed fence resolution")
		f.BoolVar(&replay, "replay-only", false, "look up the original exact receipt only")
	}
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(stdout, usage)
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || runner == "" || !protocol.ValidID(expected) || !protocol.ValidID(project) {
		return fail(errors.New("require a pinned enrolled runner and exact project selection"), 2)
	}
	var intent protocol.ProjectStopReconcileRequest
	if command == "project-diagnose" {
		if !protocol.ValidID(member) || !protocol.ValidID(operation) {
			return fail(errors.New("require --member and --operation"), 2)
		}
	} else {
		if reviewPath == "" || !protocol.ValidID(request) || !confirm {
			return fail(errors.New("require --review, --request-id and --confirm-action"), 2)
		}
		file, err := os.Open(reviewPath)
		if err != nil {
			return fail(err, 1)
		}
		body, err := io.ReadAll(io.LimitReader(file, 8193))
		file.Close()
		if err != nil || len(body) > 8192 {
			return fail(errors.New("review file exceeds the bounded JSON contract"), 2)
		}
		var diagnosis protocol.ProjectMemberDiagnosis
		if json.Unmarshal(body, &diagnosis) != nil || diagnosis.Review == nil || diagnosis.ProjectID != project || diagnosis.Review.Validate() != nil {
			return fail(errors.New("require the unchanged JSON diagnosis from project-diagnose --probe"), 2)
		}
		intent = protocol.ProjectStopReconcileRequest{RequestID: request, Review: *diagnosis.Review, ConfirmAction: true}
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
	if command == "project-diagnose" {
		out, err := c.ProjectMemberDiagnosis(ctx, pin, project, member, operation, probe)
		if err != nil {
			return fail(err, 1)
		}
		if format == "text" {
			_, err = fmt.Fprintf(stdout, "%s: %s\nMember phase: %s; runtime observation: %s.\n", out.ReasonCode, out.Message, out.Phase, out.RuntimeObservation)
		} else {
			err = newOutput(stdout, format, false).Encode(out)
		}
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if replay {
		ctx = client.WithCreationReplayOnly(ctx)
	}
	out, err := c.ReconcileProjectStop(ctx, pin, project, intent)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Reconciliation %s: verified stopped state for original %s. No signal or Start dispatched.\n", out.Operation.ID, out.OriginalOperationID)
	} else {
		err = newOutput(stdout, format, false).Encode(out)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
