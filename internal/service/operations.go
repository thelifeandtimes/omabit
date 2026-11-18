package service

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectdiagnostic"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type Empty struct{}

func (t Target) selection() Target { return t }

type ProjectInput struct {
	Target
	ProjectID string `json:"project_id"`
}
type CheckpointInput struct {
	ProjectInput
	OperationID string `json:"operation_id"`
}
type DiagnosisInput struct {
	CheckpointInput
	InstanceID string `json:"instance_id"`
	Probe      bool   `json:"probe,omitempty"`
}
type StopReconcileInput struct {
	ProjectInput
	Request    protocol.ProjectStopReconcileRequest `json:"request"`
	LookupOnly bool                                 `json:"lookup_only,omitempty"`
}
type StopBundleInput struct {
	CheckpointInput
	InstanceID string `json:"instance_id"`
}
type ProjectRegisterInput struct {
	Target
	Request    protocol.ProjectRegisterRequest `json:"request"`
	LookupOnly bool                            `json:"lookup_only,omitempty"`
}
type ProjectActionInput struct {
	ProjectInput
	Request    protocol.ProjectActionRequest `json:"request"`
	LookupOnly bool                          `json:"lookup_only,omitempty"`
}
type ProjectRecoverInput struct {
	ProjectInput
	ParentID   string                                `json:"parent_id,omitempty"`
	Kind       string                                `json:"kind"`
	Request    protocol.ProjectRecoveryActionRequest `json:"request"`
	LookupOnly bool                                  `json:"lookup_only,omitempty"`
}
type PreflightInput struct {
	Target
	Workflow   string   `json:"workflow"`
	InstanceID string   `json:"instance_id,omitempty"`
	Desk       string   `json:"desk,omitempty"`
	Effects    []string `json:"effects"`
}

type SampleInput struct {
	Name string `json:"name" jsonschema:"Compiled sample name: gall-state-v1 (native noun) or gall-state-v2 (direct Eyre JSON text)"`
}

func description(id, about string, cli, effects, caps []string, role, bindings, replay string) Descriptor {
	return Descriptor{ID: id, Description: about, CLI: cli, Effects: effects, Capabilities: caps, Role: role, Bindings: bindings, Replay: replay}
}
func selected[I interface{ selection() Target }, O any](s *Service, d Descriptor, handler func(context.Context, *collection.Registry, collection.Connection, I) (O, error)) error {
	return bind(s, d, func(ctx context.Context, s *Service, in I) (O, error) {
		r, e, err := s.authorize(ctx, in.selection(), d)
		if err != nil {
			var out O
			return out, &beforeEffect{err: err}
		}
		return handler(ctx, r, e, in)
	})
}
func lookup(ctx context.Context, only bool) context.Context {
	if only {
		return client.WithCreationReplayOnly(ctx)
	}
	return ctx
}

func (s *Service) register() error {
	saved := []string{"local-metadata-read", "saved-runner-read"}
	probe := []string{"local-metadata-read", "live-probe"}
	mutation := []string{"local-metadata-read", "journal-write", "runtime-mutation"}
	target := "explicit enrolled name and pinned runner ID/mode on every connection"
	var errs []error
	errs = append(errs, bind(s, description("app.sample.v1", "Read one compiled named application sample, its exact staging manifest and frozen inputs; never writes files or runs tests.", []string{}, []string{"offline-read"}, []string{}, "local-user", "compiled named fixture only; source and notices are separate; native qualification status is explicit", "pure read; sample content is not deployment authority"), func(_ context.Context, _ *Service, in SampleInput) (appdev.Sample, error) {
		sample, err := appdev.ReadSample(in.Name)
		if err != nil {
			return appdev.Sample{}, invalid()
		}
		return sample, nil
	}))
	errs = append(errs, bind(s, description("runners.list.v1", "List configured runner identities without connecting; transport paths are omitted.", []string{"urbit-collector runners"}, []string{"local-metadata-read"}, []string{}, "local-user", "server allowlist filters results", "read-only"), func(ctx context.Context, s *Service, _ Empty) ([]RunnerSummary, error) {
		r, err := s.registry()
		if err != nil {
			return nil, err
		}
		entries, err := r.List(ctx)
		if err != nil {
			return nil, err
		}
		out := []RunnerSummary{}
		for _, e := range entries {
			t := Target{e.Name, e.RunnerID}
			if s.options.RequireAllowed {
				found := false
				for _, allow := range s.options.Allowed {
					if allow == t {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			out = append(out, RunnerSummary{Target: t, Mode: e.Mode, Revision: e.Revision, EnrolledAt: e.EnrolledAt})
		}
		return out, nil
	}))
	errs = append(errs, selected(s, description("runner.hello.v1", "Read actual runner capabilities, role and optional executable provenance.", []string{"urbit-collector hello"}, probe, []string{}, "authenticated", target, "read-only"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, _ Target) (protocol.Hello, error) {
		c := e.Client(true)
		defer c.Close()
		h, err := c.Hello(ctx)
		if err == nil {
			err = e.Identity().Check(h.RunnerID, h.Mode)
		}
		if err != nil {
			return protocol.Hello{}, err
		}
		return h, nil
	}))
	errs = append(errs, selected(s, description("projects.list.v1", "Read saved project reservations without probing runtime.", []string{"urbit-collector projects"}, saved, []string{protocol.ProjectInventoryFeature}, "authenticated", target, "read-only"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, _ Target) (protocol.ProjectInventory, error) {
		c := e.Client(true)
		defer c.Close()
		return c.Projects(ctx, e.Identity())
	}))
	errs = append(errs, selected(s, description("project.status.v1", "Read saved bounded project operation history; queued is not complete.", []string{"urbit-collector project-status"}, saved, []string{protocol.ProjectStatusFeature}, "authenticated", target+"; exact project ID", "read-only"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectInput) (protocol.ProjectStatus, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectStatus(ctx, e.Identity(), in.ProjectID)
	}))
	errs = append(errs, selected(s, description("project.recovery-source.v1", "Read saved recovery history and supported receipt references.", []string{"urbit-collector project-recovery-source"}, saved, []string{protocol.ProjectRecoverySourceFeature}, "authenticated", target+"; exact project ID", "read-only"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectInput) (protocol.ProjectRecoverySource, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectRecoverySource(ctx, e.Identity(), in.ProjectID)
	}))
	errs = append(errs, selected(s, description("project.recovery-checkpoint.v1", "Read a saved recovery checkpoint without runtime effects.", []string{"urbit-collector project-recovery-checkpoint"}, saved, []string{protocol.ProjectRecoveryCheckpointFeature}, "authenticated", target+"; exact project and operation IDs", "read-only"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in CheckpointInput) (protocol.ProjectRecoveryCheckpoint, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectRecoveryCheckpoint(ctx, e.Identity(), in.ProjectID, in.OperationID)
	}))
	errs = append(errs, selected(s, description("project.stop-diagnosis.v1", "Read saved Stop diagnosis or prepare a fresh exact clean-stop review when probe=true; no signal or journal write.", []string{"urbit-collector project-diagnose"}, probe, []string{protocol.ProjectStopRecoveryFeature}, "operator", target+"; exact project/member/original Stop operation; fresh review binds generation/container/pier", "read-only; original interrupted receipt preserved"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in DiagnosisInput) (protocol.ProjectMemberDiagnosis, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectMemberDiagnosis(ctx, e.Identity(), in.ProjectID, in.InstanceID, in.OperationID, in.Probe)
	}))
	errs = append(errs, selected(s, description("project.stop-diagnostic-bundle.v1", "Derive a bounded shareable view of saved Stop diagnosis. Omits exact private bindings, request/actor/member names, free-form diagnostics and sealed reviews. No runtime probe or effect.", []string{}, saved, []string{protocol.ProjectStopRecoveryFeature}, "operator", target+"; exact project/member/original operation selected privately; exported view cannot authorize an action", "saved read only; unknown categories use fixed unsupported guidance; explicit publishing/sharing remains a separate owner choice"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in StopBundleInput) (projectdiagnostic.Bundle, error) {
		c := e.Client(true)
		defer c.Close()
		d, err := c.ProjectMemberDiagnosis(ctx, e.Identity(), in.ProjectID, in.InstanceID, in.OperationID, false)
		if err != nil {
			return projectdiagnostic.Bundle{}, err
		}
		return projectdiagnostic.Derive(d), nil
	}))
	errs = append(errs, selected(s, description("project.stop-reconcile.v1", "Apply an exact clean-stop review with a linked audit receipt; never starts or signals a ship.", []string{"urbit-collector project-reconcile-stop"}, []string{"local-metadata-read", "live-probe", "journal-write"}, []string{protocol.ProjectStopRecoveryFeature}, "operator", target+"; unchanged sealed review, original request binding and new request ID", "lookup_only repeats the exact original reconciliation; never reobserves or redispatches a matched receipt"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in StopReconcileInput) (protocol.ProjectStopReconcileResult, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ReconcileProjectStop(lookup(ctx, in.LookupOnly), e.Identity(), in.ProjectID, in.Request)
	}))
	errs = append(errs, bind(s, description("project.plan.v1", "Build a fresh-fake project plan offline; does not reserve or boot.", []string{"urbit-collector project-plan"}, []string{"offline-read"}, []string{}, "local-user", "explicit fresh fake members; no saved reservation inventory", "pure plan; register rechecks authoritative conflicts"), func(_ context.Context, _ *Service, in projectnetwork.Request) (projectnetwork.Plan, error) {
		plan, err := projectnetwork.Build(in, projectnetwork.Reservations{})
		if err != nil {
			return projectnetwork.Plan{}, invalid()
		}
		return plan, nil
	}))
	errs = append(errs, selected(s, description("project.register.v1", "Reserve exact reviewed fake membership; does not boot.", []string{"urbit-collector project-register"}, []string{"local-metadata-read", "journal-write"}, []string{protocol.ProjectRegistrationFeature}, "operator", target+"; exact plan digest/members, explicit confirmation and original request ID", "lookup_only uses the same request; no automatic retries"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectRegisterInput) (protocol.ProjectRegistrationResult, error) {
		c := e.Client(true)
		defer c.Close()
		return c.RegisterProject(lookup(ctx, in.LookupOnly), e.Identity(), in.Request)
	}))
	d := description("project.action.v1", "Submit an existing typed project/member lifecycle action; capabilities depend on request.kind. Returns a receipt, not completion.", []string{"urbit-collector project-operate"}, mutation, []string{}, "operator", target+"; project/member/container, plan digest, revision, confirmation and original request ID", "lookup_only uses the exact request and guarded bridge; no new IDs or mutation retry")
	d.CapabilityRule = "protocol.ProjectAction(request.kind)"
	errs = append(errs, selected(s, d, func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectActionInput) (protocol.ProjectActionResult, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectOperate(lookup(ctx, in.LookupOnly), e.Identity(), in.ProjectID, in.Request)
	}))
	d = description("project.recover.v1", "Submit an existing typed coordinated recovery action; live observation can persist a receipt. Returns an operation handle.", []string{"urbit-collector project-recover"}, mutation, []string{}, "operator", target+"; exact parent/checkpoint/history sequence, revision, confirmation and original request ID", "lookup_only preserves all original recovery fields; never silently resumes")
	d.CapabilityRule = "protocol.ProjectRecoveryAction(kind)"
	errs = append(errs, selected(s, d, func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectRecoverInput) (protocol.ProjectRecoveryActionResult, error) {
		c := e.Client(true)
		defer c.Close()
		return c.ProjectRecover(lookup(ctx, in.LookupOnly), e.Identity(), in.ProjectID, in.ParentID, in.Kind, in.Request)
	}))
	errs = append(errs, selected(s, description("app.preflight.v1", "Check selected runner APIs/adapters/role and saved target policy before app-specific project boot; grants no action authority.", []string{"omabit app-preflight"}, probe, []string{}, "authenticated", target+"; workflow, requested effects and optional exact instance/desk", "read-only; missing capability is a normal typed negative result"), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in PreflightInput) (appdev.Preflight, error) {
		return r.AppDevelopmentPreflight(ctx, appdev.PreflightRequest{Name: in.Name, RunnerID: in.RunnerID, Workflow: in.Workflow, InstanceID: in.InstanceID, Desk: in.Desk, Effects: in.Effects})
	}))
	errs = append(errs, selected(s, description("app.policy.v1", "Read saved application policy and current target inventory; no policy authoring.", []string{"urbit-collector app-policy"}, saved, []string{protocol.AppPolicyFeature}, "authenticated", target, "read-only; gates are not runtime evidence"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, _ Target) (protocol.AppPolicyStatus, error) {
		c := e.Client(true)
		defer c.Close()
		return c.AppPolicy(ctx, e.Identity())
	}))
	errs = append(errs, s.registerDesks())
	errs = append(errs, s.registerProjectCreation())
	errs = append(errs, s.registerSources())
	errs = append(errs, s.registerGuidance())
	return errors.Join(errs...)
}
