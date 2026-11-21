package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectflow"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Legacy parsing/progress shape remains here; orchestration is shared with MCP.
type projectCreateClient = projectflow.Client
type projectCreateProgress = projectflow.Progress
type projectCreateStep struct{ kind, member string }

func projectCreateSteps(plan projectnetwork.Plan) []projectCreateStep {
	var out []projectCreateStep
	for _, step := range projectflow.Steps(plan) {
		out = append(out, projectCreateStep{step.Kind, step.Member})
	}
	return out
}
func projectCreateRequestID(parent string, step projectCreateStep) string {
	return projectflow.RequestID(parent, projectflow.Step{Kind: step.kind, Member: step.member})
}
func createProject(ctx context.Context, c projectCreateClient, pin client.Identity, actor string, plan projectnetwork.Plan, requestID string, resume bool, emit func(projectCreateProgress) error, wait func(context.Context) error) error {
	return projectflow.Create(ctx, c, pin, actor, plan, requestID, resume, emit, wait)
}
func runProjectCreate(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-create
 --label LABEL --member INSTANCE:~GALAXY:HTTP_PORT [--member ...]
 --request-id ID --shared-administration --confirm-start [--resume]

Create and boot one private fake-ship network with 0–16 fresh galaxies. Members
share administrative loopback access. Omit --member to create an empty network.
This guided flow reserves ports, downloads/verifies the pinned Vere image and core
bootstrap pill, prepares the network/storage, boots every member,
and waits for identity/readiness. It emits one JSON progress object per line.
It does not import existing piers or move ordinary fleet ships into a new network.

Keep these exact inputs and request ID. After a lost reply, --resume looks up the
original reservation and continues only its original successful/pending steps.
Failed/interrupted actions stop the workflow; no replacement IDs or destructive
cleanup are attempted. Existing project-operate controls remain available.
`
	f := flag.NewFlagSet("project-create", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var req projectnetwork.Request
	var request string
	var confirm, resume bool
	f.StringVar(&req.Label, "label", "", "project name")
	f.StringVar(&request, "request-id", "", "durable creation request ID")
	f.BoolVar(&req.ConfirmSharedAdministration, "shared-administration", false, "members share administrative loopback")
	f.BoolVar(&confirm, "confirm-start", false, "create and boot this complete fresh project")
	f.BoolVar(&resume, "resume", false, "look up the original reservation before continuing")
	f.Func("member", "repeat INSTANCE:~GALAXY:HTTP_PORT", func(value string) error {
		p := strings.Split(value, ":")
		if len(p) != 3 {
			return errors.New("member must be INSTANCE:~GALAXY:HTTP_PORT")
		}
		port, e := strconv.ParseUint(p[2], 10, 16)
		if e != nil {
			return e
		}
		req.Members = append(req.Members, projectnetwork.MemberRequest{InstanceID: p[0], Ship: p[1], HTTPPort: uint16(port)})
		return nil
	})
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, e := io.WriteString(stdout, usage)
			if e != nil {
				return fail(e, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" || runner == "" || !protocol.ValidID(expected) || !protocol.ValidID(request) || !confirm || len(req.Members) > 16 {
		return fail(errors.New("project-create requires a pinned runner, 1–16 fresh members, a request ID and explicit consent; use --help"), 2)
	}
	plan, err := projectnetwork.Build(req, projectnetwork.Reservations{})
	if err != nil {
		return fail(err, 2)
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
	hello, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	if err = pin.Check(hello.RunnerID, hello.Mode); err != nil {
		return fail(err, 1)
	}
	if !slices.Contains(hello.Features, protocol.ProjectDependenciesFeature) {
		return fail(errors.New("runner needs project-dependencies-v1 before guided project creation"), 1)
	}
	for _, step := range projectCreateSteps(plan) {
		feature, _, _ := protocol.ProjectAction(step.kind)
		if !slices.Contains(hello.Features, feature) {
			return fail(fmt.Errorf("runner needs %s before creating this project", feature), 1)
		}
	}
	if hello.AccessRole != "operator" || hello.Actor == "" {
		return fail(errors.New("project creation requires operator access"), 1)
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	emit := func(p projectCreateProgress) error {
		if format == "text" {
			_, e := fmt.Fprintf(stdout, "%s · %s %s %s\n", p.Phase, p.Step, p.InstanceID, p.Message)
			return e
		}
		return json.NewEncoder(stdout).Encode(p)
	}
	wait := func(ctx context.Context) error {
		t := time.NewTimer(time.Second)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			return nil
		}
	}
	if err = createProject(ctx, c, pin, hello.Actor, plan, request, resume, emit, wait); err != nil {
		return fail(err, 1)
	}
	return 0
}
