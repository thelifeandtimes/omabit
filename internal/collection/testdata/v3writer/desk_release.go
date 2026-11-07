package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskReleasePlan struct {
	Version  int                        `json:"version"`
	RunnerID string                     `json:"runner_id"`
	Mode     string                     `json:"mode"`
	Intent   deskwork.ReleaseIntent     `json:"intent"`
	Review   protocol.DeskReleaseReview `json:"review"`
	Checks   []deskwork.ReleaseCheck    `json:"checks"`
}

func (p DeskReleasePlan) Validate() error {
	i, v := p.Intent, p.Review
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || v.Validate() != nil || v.RunnerID != p.RunnerID || v.Mode != p.Mode || i.CheckReview(v.Review, p.Checks) != nil || i.Intent.Target != v.Request.Selection.Target || i.Intent.RuntimeID != v.Request.Selection.RuntimeID || i.Intent.ConfigRevision != v.Request.Selection.ConfigRevision || i.Intent.PolicyRevision != v.PolicyRevision {
		return errors.New("invalid reviewed release plan")
	}
	return nil
}
func (r *Registry) ReviewDeskRelease(ctx context.Context, name, runner, instance, desk, action string) (protocol.DeskReleaseReview, error) {
	var out protocol.DeskReleaseReview
	if !deskwork.IsReleaseAction(action) || (apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil || instance == "" || desk == "" {
		return out, errors.New("select an instance, desk and typed release action")
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	policy, err := c.AppPolicy(ctx, entry.Identity())
	if err != nil {
		return out, err
	}
	selection, err := selectDeskRuntime(ctx, c, entry.Identity(), policy, instance, desk)
	if err != nil {
		return out, err
	}
	out, err = c.DeskReleaseReview(ctx, entry.Identity(), protocol.DeskReleaseRequest{Selection: selection, Action: action})
	if err != nil {
		return protocol.DeskReleaseReview{}, err
	}
	if out.PolicyRevision != policy.Policy.Revision {
		return protocol.DeskReleaseReview{}, errors.New("policy changed during release review; inspect again")
	}
	return out, nil
}
func (r *Registry) PlanDeskRelease(ctx context.Context, name, runner, instance, desk, action string, checks []deskwork.ReleaseCheck) (DeskReleasePlan, error) {
	var out DeskReleasePlan
	v, err := r.ReviewDeskRelease(ctx, name, runner, instance, desk, action)
	if err != nil {
		return out, err
	}
	if err = v.Review.CheckConfiguredRelease(); err != nil {
		return out, err
	}
	if err = deskwork.CheckReleaseEvidence(v.Review, checks); err != nil {
		return out, err
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return out, err
	}
	i := deskwork.ReleaseIntent{Intent: deskwork.Intent{Version: 1, ID: hex.EncodeToString(id[:]), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: action, Baseline: v.Review.Observation}, SettingsRevision: v.Review.DeskSetting.Revision, ShipSettingsRevision: v.Review.ShipSetting.Revision, Continuity: v.Review.Continuity, ReviewDigest: v.Review.Digest, ChecksDigest: deskwork.ReleaseChecksDigest(checks)}
	out = DeskReleasePlan{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Intent: i, Review: v, Checks: checks}
	return out, out.Validate()
}
func (r *Registry) ReleaseDesk(ctx context.Context, name, runner string, plan DeskReleasePlan) (protocol.DeskOperation, error) {
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
	return c.ReleaseDesk(ctx, entry.Identity(), protocol.DeskReleaseSubmission{Intent: plan.Intent, Checks: plan.Checks})
}
