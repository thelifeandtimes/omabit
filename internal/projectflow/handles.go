package projectflow

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrHistory = errors.New("project creation history or binding changed; inspect saved project status before further actions")
var ErrPhase = errors.New("project creation phase binding changed; read saved status and explicitly select its exact native request ID")

// Handle describes saved creation progress, never current runtime/app health.
// The caller retains the complete original plan and parent request ID. Each
// Advance can submit at most one missing native phase, whose runner-owned work
// outlives this client. No client daemon or optional MCP task feature is required.
type Handle struct {
	Version         int       `json:"version"`
	ProjectID       string    `json:"project_id"`
	ParentRequestID string    `json:"parent_request_id"`
	Revision        uint64    `json:"revision"`
	ReviewDigest    string    `json:"review_digest"`
	Phase           string    `json:"phase"`
	Step            string    `json:"step,omitempty"`
	InstanceID      string    `json:"instance_id,omitempty"`
	OperationID     string    `json:"operation_id,omitempty"`
	NativeRequestID string    `json:"native_request_id,omitempty"`
	OperationStatus string    `json:"operation_status,omitempty"`
	ReadAt          time.Time `json:"read_at"`
	Basis           string    `json:"basis"`
	NextAction      string    `json:"next_action"`
}

func registration(plan projectnetwork.Plan, parent, actor string) (protocol.ProjectRegisterRequest, string, error) {
	digest, err := plan.Digest()
	if err != nil || len(plan.Members) > 16 || !protocol.ValidID(parent) || actor == "" || len(actor) > 512 {
		return protocol.ProjectRegisterRequest{}, "", errors.New("invalid bounded creation plan or actor")
	}
	request := protocol.ProjectRegisterRequest{RequestID: parent, Label: plan.Label, ReviewDigest: digest, ConfirmReservation: true, ConfirmSharedAdministration: true}
	for _, member := range plan.Members {
		request.Members = append(request.Members, protocol.ProjectMemberInput{InstanceID: member.InstanceID, Ship: member.Ship, HTTPPort: member.HTTPPort})
	}
	return request, digest, nil
}

func checkHistory(s protocol.ProjectStatus, reg protocol.ProjectRegistrationResult, actor string, plan projectnetwork.Plan, parent, digest string) error {
	if s.Project.ID != reg.Operation.ProjectID || s.Project.Revision != reg.Operation.ConfigRevision || s.Project.ReviewDigest != digest || s.HasOlderOperations {
		return ErrHistory
	}
	expected := map[string]Step{parent: {Kind: "register", Member: ""}}
	for _, step := range Steps(plan) {
		expected[RequestID(parent, step)] = step
	}
	seen := map[string]bool{}
	states := map[string]string{}
	ids := map[string]bool{}
	for _, op := range s.Operations {
		step, ok := expected[op.RequestID]
		kind := "register-project"
		if step.Kind != "register" {
			_, kind, _ = protocol.ProjectAction(step.Kind)
		}
		if !ok || seen[op.RequestID] || ids[op.ID] || op.ID == "" || op.Actor != actor || op.Kind != kind || op.InstanceID != step.Member || op.ConfigRevision != s.Project.Revision {
			return ErrHistory
		}
		seen[op.RequestID] = true
		ids[op.ID] = true
		states[op.RequestID] = op.Status
	}
	if states[parent] != "succeeded" {
		return ErrHistory
	}
	unfinished := false
	for _, step := range Steps(plan) {
		state, present := states[RequestID(parent, step)]
		if present && unfinished {
			return ErrHistory
		}
		if state != "succeeded" {
			unfinished = true
		}
	}
	return nil
}

func baseHandle(reg protocol.ProjectRegistrationResult, parent, digest string) Handle {
	return Handle{Version: 1, ProjectID: reg.Operation.ProjectID, ParentRequestID: parent, Revision: reg.Operation.ConfigRevision, ReviewDigest: digest, Phase: "reserved", ReadAt: time.Now().UTC(), Basis: "saved-creation-history; not current runtime/app readiness", NextAction: "Save the full original plan and parent request ID, then explicitly advance the same creation."}
}

// Begin reserves only. Inspect and Advance always look up that exact original
// reservation first with the server-side replay-only guard.
func Begin(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, parent string) (Handle, error) {
	request, digest, err := registration(plan, parent, actor)
	if err != nil {
		return Handle{}, err
	}
	reg, err := c.RegisterProject(ctx, pin, request)
	if err != nil {
		return Handle{}, err
	}
	h := baseHandle(reg, parent, digest)
	first := Steps(plan)[0]
	h.Step, h.NativeRequestID = first.Kind, RequestID(parent, first)
	return h, nil
}

// PhaseStep accepts only an original derived ID from this bounded canonical plan.
func PhaseStep(plan projectnetwork.Plan, parent, native string) (Step, bool) {
	if len(plan.Members) > 16 || !protocol.ValidID(parent) || !protocol.ValidID(native) {
		return Step{}, false
	}
	for _, step := range Steps(plan) {
		if RequestID(parent, step) == native {
			return step, true
		}
	}
	return Step{}, false
}

type creationState struct {
	reg    protocol.ProjectRegistrationResult
	status protocol.ProjectStatus
	handle Handle
	step   *Step
}

func inspect(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, parent string) (creationState, error) {
	var out creationState
	request, digest, err := registration(plan, parent, actor)
	if err != nil {
		return out, err
	}
	out.reg, err = c.RegisterProject(client.WithCreationReplayOnly(ctx), pin, request)
	if err != nil {
		return out, err
	}
	out.status, err = c.ProjectStatus(ctx, pin, out.reg.Operation.ProjectID)
	if err != nil {
		return out, err
	}
	if err = checkHistory(out.status, out.reg, actor, plan, parent, digest); err != nil {
		return out, err
	}
	out.handle = baseHandle(out.reg, parent, digest)
	out.handle.ReadAt = out.status.ReadAt
	for _, step := range Steps(plan) {
		requestID := RequestID(parent, step)
		var found *protocol.ProjectOperationSummary
		for _, operation := range out.status.Operations {
			if operation.RequestID == requestID {
				copy := operation
				found = &copy
				break
			}
		}
		if found != nil && found.Status == "succeeded" {
			continue
		}
		out.step = &step
		out.handle.Step = step.Kind
		out.handle.InstanceID = step.Member
		out.handle.NativeRequestID = requestID
		if found == nil {
			out.handle.Phase = "next-phase"
			return out, nil
		}
		out.handle.OperationID = found.ID
		out.handle.OperationStatus = found.Status
		if found.Status == "queued" || found.Status == "running" {
			out.handle.Phase = "waiting"
			out.handle.NextAction = "Read creation status with the unchanged plan/parent ID. Disconnecting or cancelling a wait does not cancel this runner-owned phase."
		} else {
			out.handle.Phase = "needs-recovery"
			out.handle.NextAction = "Inspect the original saved project operation/checkpoint and supported recovery actions. Do not replace IDs or repeat an interrupted phase."
		}
		return out, nil
	}
	out.handle.Phase = "ready"
	out.handle.NextAction = "Initial creation receipts completed. Probe current fleet and application health separately; do not infer it from saved creation history."
	return out, nil
}

func Inspect(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, parent string) (Handle, error) {
	state, err := inspect(ctx, c, pin, actor, plan, parent)
	return state.handle, err
}

type WaitResult struct {
	Handle      Handle `json:"handle"`
	WaitExpired bool   `json:"wait_expired"`
}

// Wait polls only saved receipts. It never advances a phase. The bound applies
// to waiting as well as reads; user cancellation leaves runner-owned work alone.
func Wait(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, parent string, timeout time.Duration) (WaitResult, error) {
	if timeout < 0 || timeout > 30*time.Second {
		return WaitResult{}, errors.New("invalid creation wait bound")
	}
	bound := timeout
	if bound == 0 {
		bound = 30 * time.Second
	} // zero requests one bounded read, no wait.
	readCtx, stop := context.WithTimeout(ctx, bound)
	defer stop()
	deadline := time.Now().Add(timeout)
	var out WaitResult
	for {
		if err := readCtx.Err(); err != nil {
			if ctx.Err() == nil && err == context.DeadlineExceeded && out.Handle.Version != 0 {
				out.WaitExpired = true
				return out, nil
			}
			return WaitResult{}, err
		}
		handle, err := Inspect(readCtx, c, pin, actor, plan, parent)
		if err != nil {
			if ctx.Err() == nil && readCtx.Err() == context.DeadlineExceeded && out.Handle.Version != 0 {
				out.WaitExpired = true
				return out, nil
			}
			return WaitResult{}, err
		}
		out.Handle = handle
		if handle.Phase != "waiting" {
			return out, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			out.WaitExpired = true
			return out, nil
		}
		timer := time.NewTimer(min(remaining, 500*time.Millisecond))
		select {
		case <-readCtx.Done():
			timer.Stop()
			if ctx.Err() != nil {
				return WaitResult{}, ctx.Err()
			}
			out.WaitExpired = true
			return out, nil
		case <-timer.C:
		}
	}
}

func submit(ctx context.Context, c Client, pin client.Identity, plan projectnetwork.Plan, parent string, reg protocol.ProjectRegistrationResult, status protocol.ProjectStatus, step Step, emit func(Progress) error) (protocol.ProjectActionResult, error) {
	if step.Kind == "prepare" {
		already := false
		for _, op := range status.Operations {
			if op.Kind == "prepare-project" {
				already = true
			}
		}
		if !already {
			if err := emit(Progress{Phase: "working", ProjectID: reg.Operation.ProjectID, Step: "download-runtime"}); err != nil {
				return protocol.ProjectActionResult{}, err
			}
			if err := c.PrepareProjectDependencies(ctx, pin); err != nil {
				return protocol.ProjectActionResult{}, err
			}
			if err := emit(Progress{Phase: "completed", ProjectID: reg.Operation.ProjectID, Step: "download-runtime"}); err != nil {
				return protocol.ProjectActionResult{}, err
			}
		}
	}
	digest, err := plan.Digest()
	if err != nil {
		return protocol.ProjectActionResult{}, err
	}
	request := protocol.ProjectActionRequest{Kind: step.Kind, InstanceID: step.Member, RequestID: RequestID(parent, step), ExpectedRevision: reg.Operation.ConfigRevision, ReviewDigest: digest, ConfirmAction: true}
	if err := emit(Progress{Phase: "working", ProjectID: reg.Operation.ProjectID, Step: step.Kind, InstanceID: step.Member}); err != nil {
		return protocol.ProjectActionResult{}, err
	}
	return c.ProjectOperate(ctx, pin, reg.Operation.ProjectID, request)
}

// Advance binds one original native phase, not the next phase at retry time.
// Repeating after an uncertain reply returns that original saved receipt even
// if it completed before retry. Another phase requires a newly selected ID.
func Advance(ctx context.Context, c Client, pin client.Identity, actor string, plan projectnetwork.Plan, parent, native string) (Handle, error) {
	expected, ok := PhaseStep(plan, parent, native)
	if !ok {
		return Handle{}, ErrPhase
	}
	state, err := inspect(ctx, c, pin, actor, plan, parent)
	if err != nil {
		return Handle{}, err
	}
	for _, op := range state.status.Operations {
		if op.RequestID != native {
			continue
		}
		h := state.handle
		h.Step, h.InstanceID, h.NativeRequestID = expected.Kind, expected.Member, native
		h.OperationID, h.OperationStatus = op.ID, op.Status
		switch op.Status {
		case "succeeded":
			h.Phase = "completed"
			h.NextAction = "This exact original phase completed historically. Read creation status and explicitly select its native_request_id before advancing another phase."
		case "queued", "running":
			h.Phase = "waiting"
			h.NextAction = "Read saved status; this exact runner-owned phase is pending. Do not replace its ID."
		default:
			h.Phase = "needs-recovery"
			h.NextAction = "Inspect the original saved receipt and supported recovery; do not replace IDs or repeat an interrupted phase."
		}
		return h, nil
	}
	if state.handle.Phase != "next-phase" || state.step == nil || RequestID(parent, *state.step) != native {
		return Handle{}, ErrPhase
	}
	result, err := submit(ctx, c, pin, plan, parent, state.reg, state.status, *state.step, func(Progress) error { return nil })
	if err != nil {
		return Handle{}, err
	}
	state.handle.Phase = "submitted"
	state.handle.OperationID = result.Operation.ID
	state.handle.OperationStatus = result.Operation.Status
	state.handle.ReadAt = time.Now().UTC()
	state.handle.NextAction = "The native receipt is submission, not completion. Read creation status with the same plan/parent ID, then explicitly advance only when its saved phase succeeds."
	return state.handle, nil
}
