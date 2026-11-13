package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskAdoptionPlan struct {
	Version  int                         `json:"version"`
	RunnerID string                      `json:"runner_id"`
	Mode     string                      `json:"mode"`
	Intent   deskwork.AdoptionIntent     `json:"intent"`
	Review   protocol.DeskAdoptionReview `json:"review"`
}

func (p DeskAdoptionPlan) Validate() error {
	v, i := p.Review, p.Intent
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || i.Validate() != nil || v.Validate() != nil || v.RunnerID != p.RunnerID || v.Mode != p.Mode || v.Request.Selection.Target != i.Intent.Target || v.Request.Selection.RuntimeID != i.Intent.RuntimeID || v.Request.Selection.ConfigRevision != i.Intent.ConfigRevision || v.Request.StageID != i.StageID || v.Request.SettingsRevision != i.SettingsRevision || v.Observation != i.Intent.Baseline || v.PolicyRevision != i.Intent.PolicyRevision || v.SourceDigest != i.Intent.SourceDigest || v.SourceName != i.SourceName || v.Continuity != i.Continuity || v.PlanDigest != i.PlanDigest {
		return errors.New("invalid source adoption plan")
	}
	return nil
}
func (r *Registry) ReviewDeskAdoption(ctx context.Context, name, runner, stageID string) (protocol.DeskAdoptionReview, error) {
	var out protocol.DeskAdoptionReview
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
	out, err = c.DeskAdoptionReview(ctx, entry.Identity(), protocol.DeskDeploymentRequest{Selection: selection, StageID: stageID, SettingsRevision: setting.Setting.Revision}, setting.Setting.Desk.SourceName, stage.Receipt.Intent.SourceDigest)
	if err != nil {
		return protocol.DeskAdoptionReview{}, err
	}
	if out.PolicyRevision != policy.Policy.Revision {
		return protocol.DeskAdoptionReview{}, errors.New("policy changed during deployment review; inspect again")
	}
	return out, nil
}
func (r *Registry) PlanDeskAdoption(ctx context.Context, name, runner, stageID string) (DeskAdoptionPlan, error) {
	var out DeskAdoptionPlan
	v, err := r.ReviewDeskAdoption(ctx, name, runner, stageID)
	if err != nil {
		return out, err
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return out, err
	}
	i := deskwork.AdoptionIntent{Intent: deskwork.Intent{Version: 1, ID: hex.EncodeToString(id[:]), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "adopt-source", Baseline: v.Observation, SourceDigest: v.SourceDigest}, StageID: stageID, SettingsRevision: v.Request.SettingsRevision, SourceName: v.SourceName, Continuity: v.Continuity, PlanDigest: v.PlanDigest}
	out = DeskAdoptionPlan{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Intent: i, Review: v}
	return out, out.Validate()
}
func (r *Registry) AdoptDeskSource(ctx context.Context, name, runner string, plan DeskAdoptionPlan) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := plan.Validate(); err != nil {
		return out, err
	}
	if plan.Review.ConflictCount != 0 {
		return out, errors.New("resolve the reviewed source conflicts before adopting")
	}
	if plan.Review.NewCount+plan.Review.RefreshCount == 0 {
		return out, errors.New("selected source baseline is already owned")
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
	return c.AdoptDeskSource(ctx, entry.Identity(), plan.Intent)
}
