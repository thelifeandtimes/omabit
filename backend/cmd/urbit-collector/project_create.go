package main

import (
	"context"
	"crypto/sha256"
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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This is a coordinator over the existing durable runner operations. It never
// calls Docker or retries a failed/ambiguous mutation. Reopening the same intent
// uses the original request IDs, including after a collector process exits.
type projectCreateClient interface {
	PrepareProjectDependencies(context.Context, client.Identity) error
	RegisterProject(context.Context, client.Identity, protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error)
	ProjectOperate(context.Context, client.Identity, string, protocol.ProjectActionRequest) (protocol.ProjectActionResult, error)
	ProjectStatus(context.Context, client.Identity, string) (protocol.ProjectStatus, error)
}
type projectCreateProgress struct {
	Phase       string `json:"phase"`
	ProjectID   string `json:"project_id,omitempty"`
	Step        string `json:"step,omitempty"`
	InstanceID  string `json:"instance_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Message     string `json:"message,omitempty"`
}
type projectCreateStep struct{ kind, member string }

func projectCreateSteps(plan projectnetwork.Plan) []projectCreateStep {
	out := []projectCreateStep{{"prepare", ""}, {"start-anchor", ""}}
	for _, m := range plan.Members {
		out = append(out, projectCreateStep{"prepare-storage", m.InstanceID}, projectCreateStep{"launch", m.InstanceID})
	}
	// Readiness requires the whole membership to have launched first.
	for _, m := range plan.Members {
		out = append(out, projectCreateStep{"observe", m.InstanceID})
	}
	return out
}
func projectCreateRequestID(parent string, step projectCreateStep) string {
	h := sha256.Sum256([]byte("omabit-project-create-v1\x00" + parent + "\x00" + step.kind + "\x00" + step.member))
	return fmt.Sprintf("create-%x", h[:20])
}
func createProject(ctx context.Context, c projectCreateClient, pin client.Identity, actor string, plan projectnetwork.Plan, requestID string, resume bool, emit func(projectCreateProgress) error, wait func(context.Context) error) error {
	digest, err := plan.Digest()
	if err != nil {
		return err
	}
	req := protocol.ProjectRegisterRequest{RequestID: requestID, Label: plan.Label, ReviewDigest: digest, ConfirmReservation: true, ConfirmSharedAdministration: true}
	for _, m := range plan.Members {
		req.Members = append(req.Members, protocol.ProjectMemberInput{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	registerCtx := ctx
	if resume {
		registerCtx = client.WithCreationReplayOnly(ctx)
	}
	if err = emit(projectCreateProgress{Phase: "working", Step: "reserve"}); err != nil {
		return err
	}
	reg, err := c.RegisterProject(registerCtx, pin, req)
	if err != nil {
		return fmt.Errorf("reserve project: %w; keep the same inputs and request ID; use --resume to look up this reservation", err)
	}
	project := reg.Operation.ProjectID
	steps := projectCreateSteps(plan)
	allowed := map[string]bool{requestID: true}
	for _, step := range steps {
		allowed[projectCreateRequestID(requestID, step)] = true
	}
	check := func(s protocol.ProjectStatus) error {
		if s.Project.ID != project || s.Project.Revision != reg.Operation.ConfigRevision || s.Project.ReviewDigest != digest || s.HasOlderOperations {
			return errors.New("project history changed or exceeds this guided workflow; inspect project-status before continuing")
		}
		for _, op := range s.Operations {
			if op.Actor != actor || !allowed[op.RequestID] {
				return errors.New("another operation changed this project; use its saved history for an explicit lifecycle action")
			}
		}
		return nil
	}
	for _, step := range steps {
		status, err := c.ProjectStatus(ctx, pin, project)
		if err != nil {
			return err
		}
		if err = check(status); err != nil {
			return err
		}
		// Cache preparation is safe to repeat explicitly, but only before the
		// first project effect and after original-reservation/history checks.
		if step.kind == "prepare" {
			already := false
			for _, op := range status.Operations {
				if op.Kind == "prepare-project" {
					already = true
				}
			}
			if !already {
				if err = emit(projectCreateProgress{Phase: "working", ProjectID: project, Step: "download-runtime"}); err != nil {
					return err
				}
				if err = c.PrepareProjectDependencies(ctx, pin); err != nil {
					return fmt.Errorf("project dependencies: %w; reservation retained; explicitly continue the same attempt after resolving downloads", err)
				}
				if err = emit(projectCreateProgress{Phase: "completed", ProjectID: project, Step: "download-runtime"}); err != nil {
					return err
				}
			}
		}
		request := protocol.ProjectActionRequest{Kind: step.kind, InstanceID: step.member, RequestID: projectCreateRequestID(requestID, step), ExpectedRevision: reg.Operation.ConfigRevision, ReviewDigest: digest, ConfirmAction: true}
		if err = emit(projectCreateProgress{Phase: "working", ProjectID: project, Step: step.kind, InstanceID: step.member}); err != nil {
			return err
		}
		// Same-ID submissions return existing receipts; one call only, never an
		// automatic transport retry or a replacement ID for interrupted work.
		result, err := c.ProjectOperate(ctx, pin, project, request)
		if err != nil {
			return fmt.Errorf("%s %s: %w; inspect project-status %s and resume only with the same request ID", step.kind, step.member, err, project)
		}
		op := result.Operation
		for {
			if op.Status == "succeeded" {
				break
			}
			if op.Status != "queued" && op.Status != "running" {
				return fmt.Errorf("project %s stopped at %s %s (%s, operation %s); no further actions submitted", project, step.kind, step.member, op.Status, op.ID)
			}
			if err = wait(ctx); err != nil {
				return fmt.Errorf("project %s is retained; read its status before continuing: %w", project, err)
			}
			status, err = c.ProjectStatus(ctx, pin, project)
			if err != nil {
				return err
			}
			if err = check(status); err != nil {
				return err
			}
			found := false
			for _, saved := range status.Operations {
				if saved.ID == op.ID && saved.RequestID == op.RequestID {
					op.Status = saved.Status
					found = true
					break
				}
			}
			if !found {
				return errors.New("pending project receipt is absent; no further actions submitted")
			}
		}
		if err = emit(projectCreateProgress{Phase: "completed", ProjectID: project, Step: step.kind, InstanceID: step.member, OperationID: op.ID}); err != nil {
			return err
		}
	}
	return emit(projectCreateProgress{Phase: "ready", ProjectID: project, Message: "All project members passed identity/readiness checks. Their saved receipts are available in project-status."})
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
