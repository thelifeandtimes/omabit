package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectflow"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type CreationSelection struct {
	Target
	Plan      projectnetwork.Plan `json:"plan"`
	RequestID string              `json:"request_id"`
}

func (input CreationSelection) Validate() error {
	_, err := input.Plan.Digest()
	if input.Target.Validate() != nil || err != nil || len(input.Plan.Members) > 16 || !protocol.ValidID(input.RequestID) {
		return invalid()
	}
	return nil
}

type CreationInput struct {
	CreationSelection
	ConfirmStart bool     `json:"confirm_start"`
	Workflow     string   `json:"workflow,omitempty" jsonschema:"Optional declared app workflow: gall or grubbery; preflight runs before reservation/advance; omitted means generic project only"`
	AppEffects   []string `json:"app_effects,omitempty" jsonschema:"Declared create-desk, commit and/or publish effects for an app workflow; compatibility is advisory, never a future policy grant"`
}
type CreationWaitInput struct {
	CreationSelection
	TimeoutMillis uint32 `json:"timeout_millis" jsonschema:"Saved-receipt wait bound from 0 to 30000 ms; never advances or cancels native work"`
}
type CreationAdvanceInput struct {
	CreationInput
	NativeRequestID string `json:"native_request_id" jsonschema:"Exact native phase ID from begin/status; repeating this ID never advances a later phase after a lost reply"`
}

func (input CreationAdvanceInput) Validate() error {
	_, ok := projectflow.PhaseStep(input.Plan, input.RequestID, input.NativeRequestID)
	if input.CreationInput.Validate() != nil || !ok {
		return invalid()
	}
	return nil
}

func (input CreationWaitInput) Validate() error {
	if input.CreationSelection.Validate() != nil || input.TimeoutMillis > 30000 {
		return invalid()
	}
	return nil
}

func (input CreationInput) Validate() error {
	if input.CreationSelection.Validate() != nil || !input.ConfirmStart {
		return invalid()
	}
	if input.Workflow == "" {
		if len(input.AppEffects) != 0 {
			return invalid()
		}
		return nil
	}
	if (appdev.PreflightRequest{Name: input.Name, RunnerID: input.RunnerID, Workflow: input.Workflow, Effects: input.AppEffects}).Validate() != nil {
		return invalid()
	}
	return nil
}

func creationClient(ctx context.Context, e collection.Connection, plan projectnetwork.Plan, checkSteps bool) (*client.Client, protocol.Hello, error) {
	c := e.Client(true)
	hello, err := c.Hello(ctx)
	if err == nil {
		err = e.Identity().Check(hello.RunnerID, hello.Mode)
	}
	if err == nil && hello.Actor == "" {
		err = invalid()
	}
	if err == nil && checkSteps {
		features := []string{protocol.ProjectDependenciesFeature}
		for _, step := range projectflow.Steps(plan) {
			feature, _, _ := protocol.ProjectAction(step.Kind)
			features = append(features, feature)
		}
		for _, feature := range features {
			if !slices.Contains(hello.Features, feature) {
				err = &client.UnsupportedCapabilityError{Feature: feature, Hello: hello}
				break
			}
		}
	}
	if err != nil {
		c.Close()
		return nil, hello, &beforeEffect{err: err}
	}
	return c, hello, nil
}

func creationPreflight(ctx context.Context, r *collection.Registry, input CreationInput) error {
	if input.Workflow == "" {
		return nil
	}
	p, err := r.AppDevelopmentPreflight(ctx, appdev.PreflightRequest{Name: input.Name, RunnerID: input.RunnerID, Workflow: input.Workflow, Effects: input.AppEffects})
	if err != nil {
		return &beforeEffect{err: err}
	}
	if !p.EnvironmentCompatible {
		return &refusal{"application_preflight_failed", "The selected runner cannot support the declared app workflow; this call reserved or advanced no project.", "Read app.preflight.v1 for the same target/workflow/effects and resolve its missing API, adapter or role before booting the app project."}
	}
	return nil
}

func (s *Service) registerProjectCreation() error {
	bindings := "explicit pinned runner, complete canonical fresh-fake plan (0–16 members), original parent request ID and actor; deterministic native phase IDs; no replacement intent"
	caps := []string{protocol.ProjectRegistrationFeature, protocol.ProjectStatusFeature, protocol.CreationReplayFeature}
	begin := description("project.create-begin.v1", "Reserve a guided fresh-fake project only after checking all creation adapters and optional declared app preflight. Does not boot. Save the complete original input.", []string{"urbit-collector project-create"}, []string{"local-metadata-read", "live-probe", "journal-write"}, caps, "operator", bindings, "registration uses the original parent ID; after uncertainty call project.create-status.v1 with the same selection, never a new ID")
	advance := description("project.create-advance.v1", "Advance at most one explicitly bound guided native phase. Repeated IDs return that original phase even after completion; pending/failed/interrupted work is never resubmitted. Downloads only compiled immutable runtime inputs when required.", []string{"urbit-collector project-create"}, []string{"local-metadata-read", "live-probe", "immutable-cache-write", "network-download", "journal-write", "runtime-mutation"}, caps, "operator", bindings+"; exact native_request_id from begin/status", "after a lost reply repeat the same native_request_id or read status; another phase needs its explicitly selected ID; failed/interrupted work requires supported recovery; disconnect does not cancel runner-owned work")
	begin.CapabilityRule = "project-dependencies-v1 and protocol.ProjectAction(step.kind) for every planned phase, checked before reservation"
	advance.CapabilityRule = begin.CapabilityRule
	status := description("project.create-status.v1", "Read exact original reservation and saved guided creation progress. No native phase submission, download or live runtime observation; saved ready is not app/current fleet health.", []string{}, []string{"local-metadata-read", "saved-runner-read"}, caps, "operator", bindings, "guarded original reservation lookup; absent reservation refuses without creation")
	wait := description("project.create-wait.v1", "Wait at most 30 seconds on saved creation receipts. Returns next-phase, waiting/expired, needs-recovery or historical ready; never advances/cancels a runtime phase.", []string{}, status.Effects, caps, "operator", bindings, "repeated guarded reads only; disconnect/cancellation affects the wait, not runner-owned work")
	return errors.Join(
		selected(s, begin, func(ctx context.Context, r *collection.Registry, e collection.Connection, input CreationInput) (projectflow.Handle, error) {
			c, hello, err := creationClient(ctx, e, input.Plan, true)
			if err != nil {
				return projectflow.Handle{}, err
			}
			defer c.Close()
			if err := creationPreflight(ctx, r, input); err != nil {
				return projectflow.Handle{}, err
			}
			return projectflow.Begin(ctx, c, e.Identity(), hello.Actor, input.Plan, input.RequestID)
		}),
		selected(s, advance, func(ctx context.Context, r *collection.Registry, e collection.Connection, input CreationAdvanceInput) (projectflow.Handle, error) {
			ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			c, hello, err := creationClient(ctx, e, input.Plan, true)
			if err != nil {
				return projectflow.Handle{}, err
			}
			defer c.Close()
			if err := creationPreflight(ctx, r, input.CreationInput); err != nil {
				return projectflow.Handle{}, err
			}
			return projectflow.Advance(ctx, c, e.Identity(), hello.Actor, input.Plan, input.RequestID, input.NativeRequestID)
		}),
		selected(s, status, func(ctx context.Context, _ *collection.Registry, e collection.Connection, input CreationSelection) (projectflow.Handle, error) {
			c, hello, err := creationClient(ctx, e, input.Plan, false)
			if err != nil {
				return projectflow.Handle{}, err
			}
			defer c.Close()
			return projectflow.Inspect(ctx, c, e.Identity(), hello.Actor, input.Plan, input.RequestID)
		}),
		selected(s, wait, func(ctx context.Context, _ *collection.Registry, e collection.Connection, input CreationWaitInput) (projectflow.WaitResult, error) {
			c, hello, err := creationClient(ctx, e, input.Plan, false)
			if err != nil {
				return projectflow.WaitResult{}, err
			}
			defer c.Close()
			return projectflow.Wait(ctx, c, e.Identity(), hello.Actor, input.Plan, input.RequestID, time.Duration(input.TimeoutMillis)*time.Millisecond)
		}),
	)
}
