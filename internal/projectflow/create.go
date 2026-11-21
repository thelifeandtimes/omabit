// Package projectflow coordinates existing durable project operations. It never
// calls an engine, chooses a replacement request ID or owns a runtime lifetime.
package projectflow

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This is a coordinator over the existing durable runner operations. It never
// calls Docker or retries a failed/ambiguous mutation. Reopening the same intent
// uses the original request IDs, including after a collector process exits.
type Client interface {
	PrepareProjectDependencies(context.Context, client.Identity) error
	RegisterProject(context.Context, client.Identity, protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error)
	ProjectOperate(context.Context, client.Identity, string, protocol.ProjectActionRequest) (protocol.ProjectActionResult, error)
	ProjectStatus(context.Context, client.Identity, string) (protocol.ProjectStatus, error)
}
type Progress struct {
	Phase       string `json:"phase"`
	ProjectID   string `json:"project_id,omitempty"`
	Step        string `json:"step,omitempty"`
	InstanceID  string `json:"instance_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Message     string `json:"message,omitempty"`
}
type Step struct{ Kind, Member string }

func Steps(plan projectnetwork.Plan) []Step {
	out := []Step{{"prepare", ""}, {"start-anchor", ""}}
	for _, m := range plan.Members {
		out = append(out, Step{"prepare-storage", m.InstanceID}, Step{"launch", m.InstanceID})
	}
	// Readiness requires the whole membership to have launched first.
	for _, m := range plan.Members {
		out = append(out, Step{"observe", m.InstanceID})
	}
	return out
}
func RequestID(parent string, step Step) string {
	h := sha256.Sum256([]byte("omabit-project-create-v1\x00" + parent + "\x00" + step.Kind + "\x00" + step.Member))
	return fmt.Sprintf("create-%x", h[:20])
}
func Create(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, requestID string, resume bool, emit func(Progress) error, wait func(context.Context) error) error {
	req, digest, err := registration(plan, requestID, actor)
	if err != nil {
		return err
	}
	registerCtx := ctx
	if resume {
		registerCtx = client.WithCreationReplayOnly(ctx)
	}
	if err = emit(Progress{Phase: "working", Step: "reserve"}); err != nil {
		return err
	}
	reg, err := c.RegisterProject(registerCtx, pin, req)
	if err != nil {
		return fmt.Errorf("reserve project: %w; keep the same inputs and request ID; use --resume to look up this reservation", err)
	}
	project := reg.Operation.ProjectID
	steps := Steps(plan)
	check := func(status protocol.ProjectStatus) error {
		return checkHistory(status, reg, actor, plan, requestID, digest)
	}
	for _, step := range steps {
		status, err := c.ProjectStatus(ctx, pin, project)
		if err != nil {
			return err
		}
		if err = check(status); err != nil {
			return err
		}
		// Both the bounded CLI/MCP interface and this legacy progress loop use
		// the same cache/phase submission and original derived request IDs.
		result, err := submit(ctx, c, pin, plan, requestID, reg, status, step, emit)
		if err != nil {
			return fmt.Errorf("%s %s: %w; inspect project-status %s and resume only with the same request ID", step.Kind, step.Member, err, project)
		}
		op := result.Operation
		for {
			if op.Status == "succeeded" {
				break
			}
			if op.Status != "queued" && op.Status != "running" {
				return fmt.Errorf("project %s stopped at %s %s (%s, operation %s); no further actions submitted", project, step.Kind, step.Member, op.Status, op.ID)
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
		if err = emit(Progress{Phase: "completed", ProjectID: project, Step: step.Kind, InstanceID: step.Member, OperationID: op.ID}); err != nil {
			return err
		}
	}
	return emit(Progress{Phase: "ready", ProjectID: project, Message: "All project members passed identity/readiness checks. Their saved receipts are available in project-status."})
}
