package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskFrontendPlan struct {
	Version  int                         `json:"version"`
	RunnerID string                      `json:"runner_id"`
	Mode     string                      `json:"mode"`
	Intent   deskwork.FrontendIntent     `json:"intent"`
	Review   protocol.DeskFrontendReview `json:"review"`
	Checks   []deskwork.FrontendCheck    `json:"checks"`
}

func (p DeskFrontendPlan) Validate() error {
	i, v := p.Intent, p.Review
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || v.Validate() != nil || v.RunnerID != p.RunnerID || v.Mode != p.Mode || i.CheckReview(v.Review, p.Checks) != nil || i.Intent.Target != v.Request.Selection.Target || i.Intent.RuntimeID != v.Request.Selection.RuntimeID || i.Intent.ConfigRevision != v.Request.Selection.ConfigRevision || i.Intent.PolicyRevision != v.PolicyRevision {
		return errors.New("invalid reviewed frontend plan")
	}
	return nil
}
func (r *Registry) ReviewDeskFrontend(ctx context.Context, name, runner, stageID string) (protocol.DeskFrontendReview, error) {
	var out protocol.DeskFrontendReview
	if !protocol.ValidDeskOperationID(stageID) {
		return out, errors.New("invalid source stage ID")
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	stage, err := c.DeskStage(ctx, entry.Identity(), stageID)
	if err != nil {
		return out, err
	}
	policy, err := c.AppPolicy(ctx, entry.Identity())
	if err != nil {
		return out, err
	}
	target := stage.Receipt.Intent.Target
	selection, err := selectDeskRuntime(ctx, c, entry.Identity(), policy, target.InstanceID, target.Desk)
	if err != nil {
		return out, err
	}
	if selection.Target != target {
		return out, errors.New("staged project generation or target changed")
	}
	setting, err := c.DeskSetting(ctx, entry.Identity(), target)
	if err != nil {
		return out, err
	}
	if setting.Setting.Revision != stage.Receipt.Intent.SettingsRevision || setting.Setting.Desk == nil || setting.Setting.Desk.Workflow != "mounted" {
		return out, errors.New("source workflow changed after staging")
	}
	out, err = c.DeskFrontendReview(ctx, entry.Identity(), protocol.DeskFrontendRequest{Selection: selection, StageID: stageID, SettingsRevision: setting.Setting.Revision})
	if err != nil {
		return protocol.DeskFrontendReview{}, err
	}
	if out.Review.Stage != stage.Receipt.Intent || out.PolicyRevision != policy.Policy.Revision {
		return protocol.DeskFrontendReview{}, errors.New("policy changed during deployment review; inspect again")
	}
	return out, nil
}
func (r *Registry) PlanDeskFrontend(ctx context.Context, name, runner, stageID string, checks []deskwork.FrontendCheck) (DeskFrontendPlan, error) {
	var out DeskFrontendPlan
	v, err := r.ReviewDeskFrontend(ctx, name, runner, stageID)
	if err != nil {
		return out, err
	}
	if err = v.Review.CheckConfiguredUpload(); err != nil {
		return out, err
	}
	if err = deskwork.CheckFrontendEvidence(v.Review, checks); err != nil {
		return out, err
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return out, err
	}
	i := deskwork.FrontendIntent{Intent: deskwork.Intent{Version: 1, ID: hex.EncodeToString(id[:]), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "upload-frontend", Baseline: v.Review.Observation, SourceDigest: v.Review.ArtifactDigest}, StageID: v.Request.StageID, SettingsRevision: v.Review.DeskSetting.Revision, ShipSettingsRevision: v.Review.ShipSetting.Revision, Continuity: v.Review.Continuity, ReviewDigest: v.Review.Digest, ChecksDigest: deskwork.FrontendChecksDigest(checks)}
	out = DeskFrontendPlan{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Intent: i, Review: v, Checks: checks}
	return out, out.Validate()
}
func (r *Registry) UploadDeskFrontend(ctx context.Context, name, runner string, plan DeskFrontendPlan) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := plan.Validate(); err != nil {
		return out, err
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	if err = entry.Identity().Check(plan.RunnerID, plan.Mode); err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.UploadDeskFrontend(ctx, entry.Identity(), protocol.DeskFrontendSubmission{Intent: plan.Intent, Checks: plan.Checks})
}
