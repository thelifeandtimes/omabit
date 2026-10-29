package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
	"strconv"
	"strings"
	"time"
)

func runProjectAdd(ctx context.Context, args []string, config, socket, runner, expected, format string, out io.Writer, fail func(error, int) int) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "help" || len(args) > 1 && args[1] == "--help") {
		fmt.Fprintln(out, "Usage: urbit-collector project-add PROJECT --member INSTANCE:~GALAXY:HTTP_PORT --request-id ID --confirm-start [--resume]\nRequires --runner NAME --expected-runner-id ID. Adds one fake galaxy without restarting current project members. The same identity may exist in another project. --resume only retrieves the same durable request; unresolved dispatch outcomes need review.")
		return 0
	}
	if len(args) < 1 {
		return fail(errors.New("project-add requires PROJECT --member INSTANCE:~GALAXY:PORT --request-id ID --confirm-start"), 2)
	}
	project := args[0]
	f := flag.NewFlagSet("project-add", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var member, request string
	var confirm, resume bool
	f.StringVar(&member, "member", "", "new member INSTANCE:~GALAXY:HTTP_PORT")
	f.StringVar(&request, "request-id", "", "durable request identity")
	f.BoolVar(&confirm, "confirm-start", false, "boot a fresh member in this project")
	f.BoolVar(&resume, "resume", false, "look up the exact prior addition")
	if err := f.Parse(args[1:]); err != nil {
		return fail(err, 2)
	}
	parts := strings.Split(member, ":")
	if len(parts) != 3 || !protocol.ValidID(project) || !protocol.ValidID(request) || !confirm || socket != "" || runner == "" || f.NArg() != 0 {
		return fail(errors.New("invalid project-add selection"), 2)
	}
	port, err := strconv.ParseUint(parts[2], 10, 16)
	if err != nil || port < 1024 {
		return fail(errors.New("invalid member HTTP port"), 2)
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
		return fail(errors.New("runner selection changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	status, err := c.ProjectStatus(ctx, pin, project)
	if err != nil {
		return fail(err, 1)
	}
	req := protocol.ProjectActionRequest{Kind: "add-member", InstanceID: parts[0], Ship: parts[1], HTTPPort: uint16(port), RequestID: request, ExpectedRevision: status.Project.Revision, ReviewDigest: status.Project.ReviewDigest, ConfirmAction: true}
	submitCtx := ctx
	if resume {
		submitCtx = client.WithCreationReplayOnly(ctx)
	}
	result, err := c.ProjectOperate(submitCtx, pin, project, req)
	if err != nil {
		return fail(err, 1)
	}
	emit := func(phase, step string) error {
		return json.NewEncoder(out).Encode(projectCreateProgress{Phase: phase, Step: step, ProjectID: project, InstanceID: req.InstanceID, OperationID: result.Operation.ID})
	}
	if err = emit("working", "joining-project"); err != nil {
		return fail(err, 1)
	}
	for {
		status, err = c.ProjectStatus(ctx, pin, project)
		if err != nil {
			return fail(err, 1)
		}
		found := false
		for _, op := range status.Operations {
			if op.ID == result.Operation.ID && op.RequestID == request {
				found = true
				if op.Status == "succeeded" {
					if err = emit("ready", "member-ready"); err != nil {
						return fail(err, 1)
					}
					return 0
				}
				if op.Status != "queued" && op.Status != "running" {
					return fail(errors.New("member addition paused; its pier and journal remain for recovery"), 1)
				}
			}
		}
		if !found {
			return fail(errors.New("addition receipt is absent; inspect project history before retrying"), 1)
		}
		select {
		case <-ctx.Done():
			return fail(errors.New("addition is retained; inspect project history before retrying"), 1)
		case <-time.After(time.Second):
		}
	}
}
