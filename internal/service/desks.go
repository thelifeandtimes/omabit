package service

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskInput struct {
	Target
	InstanceID string `json:"instance_id"`
	Desk       string `json:"desk"`
}
type DeskExportInput struct {
	Target
	Observation protocol.DeskObservation `json:"observation"`
	Paths       []string                 `json:"paths"`
}

func (in DeskExportInput) Validate() error {
	if in.Target.Validate() != nil || (protocol.DeskSourceExportRequest{Selection: in.Observation.Selection, Baseline: in.Observation.Observation, Paths: in.Paths}).Validate() != nil || in.Observation.RunnerID != in.RunnerID {
		return invalid()
	}
	return nil
}

type DeskListInput struct {
	Target
	InstanceID string `json:"instance_id"`
}
type DeskSettingsInput struct {
	Target
	InstanceID string `json:"instance_id"`
	Desk       string `json:"desk,omitempty"`
}
type StageInput struct {
	Target
	StageID string `json:"stage_id"`
}
type DeskOperationInput struct {
	Target
	OperationID string `json:"operation_id"`
}
type DeskCreateInput struct {
	Target
	Plan collection.DeskPlan `json:"plan"`
}
type DeskCommitInput struct {
	Target
	Plan collection.DeskCommitPlan `json:"plan"`
}
type DeskAdoptInput struct {
	Target
	Plan collection.DeskAdoptionPlan `json:"plan"`
}
type DeskReleaseInput struct {
	Target
	Plan collection.DeskReleasePlan `json:"plan"`
}
type DeskFrontendInput struct {
	Target
	Plan collection.DeskFrontendPlan `json:"plan"`
}
type DeskReceiverInput struct {
	Target
	Plan collection.DeskReceiverPlan `json:"plan"`
}
type ReleaseReviewInput struct {
	DeskInput
	Action string `json:"action"`
}
type ReleasePlanInput struct {
	ReleaseReviewInput
	Checks []deskwork.ReleaseCheck `json:"checks"`
}
type FrontendPlanInput struct {
	StageInput
	Checks []deskwork.FrontendCheck `json:"checks"`
}
type ReceiverReviewInput struct {
	DeskInput
	PublisherInstance string             `json:"publisher_instance,omitempty"`
	Publisher         deskwork.Publisher `json:"publisher"`
}

func (s *Service) registerDesks() error {
	var errs []error
	errs = append(errs, selected(s, description("desk.export-source.v1", "Read explicitly selected source files at an exact reviewed Clay/runtime baseline without mounting or adopting; bytes are private source, not a sanitized export.", nil, []string{"local-metadata-read", "live-probe"}, []string{protocol.DeskSourceExportFeature}, "operator", "complete desk observation plus sorted explicit paths; runner rechecks process/pier/Clay identity and byte bounds", "read-only; ownership adoption remains an explicit separately gated operation"), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskExportInput) (protocol.DeskSourceExport, error) {
		return r.ExportDeskSource(ctx, in.Name, in.RunnerID, in.Observation, in.Paths)
	}))
	probe := []string{"local-metadata-read", "live-probe"}
	saved := []string{"local-metadata-read", "saved-runner-read"}
	mutate := []string{"local-metadata-read", "live-probe", "journal-write", "runtime-mutation"}
	bindings := "explicit pinned runner and instance; service resolves authoritative project creator/generation and current runtime"
	planBindings := "explicit pinned runner plus complete unchanged reviewed plan; target/generation/runtime/configuration/policy/source evidence and operation ID rechecked by existing core and runner"
	read := func(id, about, cli, feature string, effects []string) Descriptor {
		return description(id, about, []string{"omabit " + cli}, effects, []string{feature}, "authenticated", bindings, "read-only; live evidence does not grant future effects")
	}
	write := func(id, about, cli, feature string) Descriptor {
		return description(id, about, []string{"omabit " + cli}, mutate, []string{feature}, "operator", planBindings, "submit unchanged plan once; read desk.operation.v1 after uncertainty, desk.reconcile.v1 observes without repeating the effect")
	}
	errs = append(errs, selected(s, read("desk.list.v1", "Discover current installed desk names.", "desk-list", protocol.DeskDiscoveryFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskListInput) (protocol.DeskDiscovery, error) {
		return r.DiscoverDesks(ctx, in.Name, in.RunnerID, in.InstanceID)
	}))
	errs = append(errs, selected(s, read("desk.settings.v1", "Read saved workflow metadata, not executable checks or permissions.", "desk-settings", protocol.DeskSettingsFeature, saved), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskSettingsInput) (protocol.DeskSetting, error) {
		return r.InspectDeskSetting(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, read("desk.inspect.v1", "Read current desk state and effective permissions.", "desk-inspect", protocol.DeskCreateFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskInput) (collection.DeskInspection, error) {
		return r.InspectDesk(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, read("desk.source.v1", "Inspect bounded source changes and ownership alerts.", "desk-source-status", protocol.DeskSourceFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskInput) (protocol.DeskSourceStatus, error) {
		return r.InspectDeskSource(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, read("desk.application.v1", "Read independent compiled/active application stages.", "desk-app-status", protocol.DeskApplicationFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskInput) (collection.DeskApplicationInspection, error) {
		return r.InspectDeskApplication(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, read("desk.receiver.v1", "Read current Kiln subscription and pending-source state.", "desk-receiver-status", protocol.DeskReceiverFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskInput) (protocol.DeskReceiverStatus, error) {
		return r.InspectDeskReceiver(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, read("desk.operation.v1", "Read an existing durable desk receipt with its original operation ID.", "desk-operation", protocol.DeskCreateFeature, saved), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskOperationInput) (protocol.DeskOperation, error) {
		return r.DeskOperation(ctx, in.Name, in.RunnerID, in.OperationID)
	}))
	appReceipt := description("app.source-operation.v1", "Read original saved Grubbery app source phases without source cache or native access.", nil, saved, []string{protocol.GrubberyAppReceiptsFeature}, "operator", bindings, "read-only; preserve the original request ID, plan and uncertainty")
	errs = append(errs, selected(s, appReceipt, func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskOperationInput) (protocol.GrubberyAppOperation, error) {
		return r.GrubberyAppOperation(ctx, in.Name, in.RunnerID, in.OperationID)
	}))
	d := write("desk.reconcile.v1", "Observe an uncertain desk effect without repeating it; busy returns saved evidence without queuing another observer.", "desk-reconcile", protocol.DeskCreateFeature)
	d.Effects = []string{"local-metadata-read", "live-probe", "journal-write"}
	errs = append(errs, selected(s, d, func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskOperationInput) (protocol.DeskOperation, error) {
		return r.ReconcileDeskOperation(ctx, in.Name, in.RunnerID, in.OperationID)
	}))
	errs = append(errs, selected(s, read("desk.plan-create.v1", "Prepare a generation-bound creation plan; save its exact operation ID.", "desk-plan-create", protocol.DeskCreateFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskInput) (collection.DeskPlan, error) {
		return r.PlanDeskCreation(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk)
	}))
	errs = append(errs, selected(s, write("desk.create.v1", "Submit an unchanged native desk creation plan.", "desk-create", protocol.DeskCreateFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskCreateInput) (protocol.DeskOperation, error) {
		return r.CreateDesk(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("desk.stage-status.v1", "Read an existing bounded staging receipt; no source transfer.", "desk-stage-status", protocol.DeskStageFeature, saved), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (protocol.DeskStage, error) {
		return r.DeskStage(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, read("desk.review-commit.v1", "Review staged source against fresh Clay and saved policy without a commit.", "desk-review-commit", protocol.DeskCommitFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (protocol.DeskDeploymentReview, error) {
		return r.ReviewDeskCommit(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, read("desk.plan-commit.v1", "Prepare an immutable commit plan; save it before submission.", "desk-plan-commit", protocol.DeskCommitFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (collection.DeskCommitPlan, error) {
		return r.PlanDeskCommit(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, write("desk.commit.v1", "Submit unchanged source/policy/runtime-bound commit plan.", "desk-commit", protocol.DeskCommitFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskCommitInput) (protocol.DeskOperation, error) {
		return r.CommitDesk(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("desk.review-adopt.v1", "Review explicit adoption of existing source ownership.", "desk-review-adoption", protocol.DeskAdoptionFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (protocol.DeskAdoptionReview, error) {
		return r.ReviewDeskAdoption(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, read("desk.plan-adopt.v1", "Prepare a source-adoption plan without changing ownership.", "desk-plan-adoption", protocol.DeskAdoptionFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (collection.DeskAdoptionPlan, error) {
		return r.PlanDeskAdoption(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, write("desk.adopt.v1", "Submit an unchanged explicit source-adoption plan.", "desk-adopt", protocol.DeskAdoptionFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskAdoptInput) (protocol.DeskOperation, error) {
		return r.AdoptDeskSource(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("desk.review-release.v1", "Review local activation or source publication stages.", "desk-review-release", protocol.DeskReleaseFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in ReleaseReviewInput) (protocol.DeskReleaseReview, error) {
		return r.ReviewDeskRelease(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk, in.Action)
	}))
	errs = append(errs, selected(s, read("desk.plan-release.v1", "Prepare a release plan. Checks are caller-supplied evidence, not tests run by Omabit.", "desk-plan-release", protocol.DeskReleaseFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in ReleasePlanInput) (collection.DeskReleasePlan, error) {
		return r.PlanDeskRelease(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk, in.Action, in.Checks)
	}))
	errs = append(errs, selected(s, write("desk.release.v1", "Submit unchanged activation/publication plan with independently enforced gates.", "desk-release", protocol.DeskReleaseFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskReleaseInput) (protocol.DeskOperation, error) {
		return r.ReleaseDesk(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("desk.review-frontend.v1", "Review transferred frontend bytes and native artifact hash.", "desk-review-frontend", protocol.DeskFrontendFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in StageInput) (protocol.DeskFrontendReview, error) {
		return r.ReviewDeskFrontend(ctx, in.Name, in.RunnerID, in.StageID)
	}))
	errs = append(errs, selected(s, read("desk.plan-frontend.v1", "Prepare frontend upload plan. Supplied checks are external provenance, not executed jobs.", "desk-plan-frontend", protocol.DeskFrontendFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in FrontendPlanInput) (collection.DeskFrontendPlan, error) {
		return r.PlanDeskFrontend(ctx, in.Name, in.RunnerID, in.StageID, in.Checks)
	}))
	errs = append(errs, selected(s, write("desk.upload-frontend.v1", "Upload the reviewed artifact; may immediately change served bytes and Clay docket.", "desk-upload-frontend", protocol.DeskFrontendFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskFrontendInput) (protocol.DeskOperation, error) {
		return r.UploadDeskFrontend(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("desk.review-install.v1", "Review a continuing named publisher and unused receiver desk.", "desk-review-install", protocol.DeskReceiverFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in ReceiverReviewInput) (protocol.DeskReceiverReview, error) {
		return r.ReviewDeskReceiver(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk, in.PublisherInstance, in.Publisher)
	}))
	errs = append(errs, selected(s, read("desk.plan-install.v1", "Prepare exact publisher/hash-bound receiver installation plan.", "desk-plan-install", protocol.DeskReceiverFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in ReceiverReviewInput) (collection.DeskReceiverPlan, error) {
		return r.PlanDeskReceiver(ctx, in.Name, in.RunnerID, in.InstanceID, in.Desk, in.PublisherInstance, in.Publisher)
	}))
	errs = append(errs, selected(s, write("desk.install.v1", "Install a continuing subscription using the exact reviewed receiver plan.", "desk-install", protocol.DeskReceiverFeature), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskReceiverInput) (protocol.DeskOperation, error) {
		return r.InstallDeskReceiver(ctx, in.Name, in.RunnerID, in.Plan)
	}))
	errs = append(errs, selected(s, read("ship.routes.v1", "Read scoped URL records and automatic discovery without exposing a service.", "ship-routes", protocol.ShipRoutesFeature, probe), func(ctx context.Context, r *collection.Registry, _ collection.Connection, in DeskListInput) (protocol.ShipRoutes, error) {
		return r.ShipRoutes(ctx, in.Name, in.RunnerID, in.InstanceID)
	}))
	return errors.Join(errs...)
}
