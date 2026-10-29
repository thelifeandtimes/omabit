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

func runProjectOperate(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-operate
       PROJECT ACTION
       --revision N --review-digest HASH --request-id ID --confirm-action
       [--member INSTANCE] [--after OPERATION] [--original-readiness OPERATION]
       [--replay-only]

prepare creates the project's private bridge and stopped namespace anchor.
start-anchor starts that prepared anchor; it does not boot a ship.
prepare-storage creates a fresh member's exclusive storage directory.
launch boots that fresh member after anchor and storage preparation.
observe records fresh identity/readiness for an already launched member; it does
not Start or repair a runtime. These three actions require --member.

Current fleet controls: member-start/member-stop --member INSTANCE --container-id FULL_ID.
member-continue --member INSTANCE --after FAILED_ADDITION continues an exact
addition before storage dispatch, or a start retired before successor creation.
Unknown dispatched outcomes are refused.

Member lifecycle actions require --member and an exact --after receipt:
  stop             --after READINESS   signal the exact observed member to stop
  observe-exit     --after STOP        record its exit without repeating Stop
  retire           --after EXIT        remove only that exited container, keep pier
  resume           --after RETIREMENT  create/start its successor from the same pier
  observe-resumed  --after RESUME      record readiness for that replacement
  observe-current  --after CREATOR --original-readiness INITIAL_OBSERVATION
                                      observe a member in the latest whole-project context
Complete initial launch before observe; current observation retains the original
initialized-pier receipt and the exact launch/resume/recovery creator. A later peer
replacement can require current observation before another lifecycle action.

Each action requires its separately advertised capability, the saved project
revision/digest and an explicit caller-chosen request ID. Observe saved receipts
with project-status; a successful action alone is not current health. Keep all
original inputs after a lost reply and add --replay-only to look up the original
receipt without starting another worker. Never retry an interrupted effect with
a new ID. Use project-recover --help for the separate coordinated anchor recovery contract.
`
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, err := io.WriteString(stdout, usage)
		if err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if len(args) < 2 {
		return fail(errors.New("project-operate requires a project ID and action; use --help"), 2)
	}
	id, kind := args[0], args[1]
	flags := flag.NewFlagSet("project-operate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	req := protocol.ProjectActionRequest{Kind: kind}
	var replay bool
	var port uint
	flags.StringVar(&req.ContainerID, "container-id", "", "selected full project runtime for member-start/member-stop")
	flags.StringVar(&req.Ship, "ship", "", "fresh fake galaxy for add-member")
	flags.UintVar(&port, "http-port", 0, "loopback HTTP port for add-member")
	flags.StringVar(&req.InstanceID, "member", "", "exact reserved member ID")
	flags.StringVar(&req.AfterOperationID, "after", "", "exact prerequisite operation ID for this action")
	flags.StringVar(&req.OriginalReadinessOperationID, "original-readiness", "", "original initialized-pier readiness for observe-current")
	flags.Uint64Var(&req.ExpectedRevision, "revision", 0, "reviewed project revision")
	flags.StringVar(&req.ReviewDigest, "review-digest", "", "reviewed canonical plan digest")
	flags.StringVar(&req.RequestID, "request-id", "", "caller-chosen durable request ID")
	flags.BoolVar(&req.ConfirmAction, "confirm-action", false, "authorize the selected action")
	flags.BoolVar(&replay, "replay-only", false, "lookup original receipt only")
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

	if port > 65535 {
		return fail(errors.New("invalid HTTP port"), 2)
	}
	req.HTTPPort = uint16(port)
	if flags.NArg() != 0 || socket != "" || runner == "" || !protocol.ValidID(expected) || !protocol.ValidID(id) || !protocol.ValidProjectActionRequest(req) {
		return fail(errors.New("project-operate requires a pinned enrolled runner and exact reviewed action inputs; use --help"), 2)
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
	result, err := c.ProjectOperate(ctx, pin, id, req)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Project %s: %s operation %s (%s).\nRequest: %s\nRead project-status %s for saved history; this receipt does not prove current health.\n", id, result.Operation.Kind, result.Operation.ID, result.Operation.Status, result.Operation.RequestID, id)
	} else {
		err = newOutput(stdout, format, false).Encode(result)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
