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

type DeskReceiverPlan struct {
	Version  int                         `json:"version"`
	RunnerID string                      `json:"runner_id"`
	Mode     string                      `json:"mode"`
	Intent   deskwork.ReceiverIntent     `json:"intent"`
	Review   protocol.DeskReceiverReview `json:"review"`
}

func (p DeskReceiverPlan) Validate() error {
	i, v := p.Intent, p.Review
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || v.Validate() != nil || v.RunnerID != p.RunnerID || v.Mode != p.Mode || i.CheckReview(v.Review) != nil || i.Intent.Target != v.Request.Selection.Target || i.Intent.RuntimeID != v.Request.Selection.RuntimeID || i.Intent.ConfigRevision != v.Request.Selection.ConfigRevision || i.Intent.PolicyRevision != v.PolicyRevision {
		return errors.New("invalid reviewed receiver plan")
	}
	return nil
}
func (r *Registry) ReviewDeskReceiver(ctx context.Context, name, runner, instance, desk, publisherInstance string, publisher deskwork.Publisher) (protocol.DeskReceiverReview, error) {
	var out protocol.DeskReceiverReview
	if publisher.Validate() != nil || (apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil || instance == "" || desk == "" {
		return out, errors.New("select an instance, desk and publisher")
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
	var publisherTarget apppolicy.Target
	if selection.Target.ProjectID != "" {
		for _, target := range policy.Targets {
			if target.Target.InstanceID == publisherInstance && target.Target.ProjectID == selection.Target.ProjectID && target.Ready && target.Ship == publisher.Ship {
				publisherTarget = target.Target
			}
		}
		if publisherTarget.InstanceID == "" {
			return out, errors.New("select the current publisher member of this project")
		}
	} else if publisherInstance != "" {
		return out, errors.New("publisher instance is only used within a project")
	}
	out, err = c.DeskReceiverReview(ctx, entry.Identity(), protocol.DeskReceiverRequest{Selection: selection, Publisher: publisher, PublisherTarget: publisherTarget})
	if err != nil {
		return protocol.DeskReceiverReview{}, err
	}
	if out.PolicyRevision != policy.Policy.Revision {
		return protocol.DeskReceiverReview{}, errors.New("policy changed during receiver review; inspect again")
	}
	return out, nil
}
func (r *Registry) PlanDeskReceiver(ctx context.Context, name, runner, instance, desk, publisherInstance string, publisher deskwork.Publisher) (DeskReceiverPlan, error) {
	var out DeskReceiverPlan
	v, err := r.ReviewDeskReceiver(ctx, name, runner, instance, desk, publisherInstance, publisher)
	if err != nil {
		return out, err
	}
	if err = v.Review.CheckConfiguredReceiver(); err != nil {
		return out, err
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return out, err
	}
	i := deskwork.ReceiverIntent{Intent: deskwork.Intent{Version: 1, ID: hex.EncodeToString(id[:]), Target: v.Request.Selection.Target, RuntimeID: v.Request.Selection.RuntimeID, ConfigRevision: v.Request.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "install-remote", Baseline: v.Review.Observation, SourceDigest: publisher.Digest()}, Publisher: publisher, PublisherTarget: v.Request.PublisherTarget, SettingsRevision: v.Review.DeskSetting.Revision, ShipSettingsRevision: v.Review.ShipSetting.Revision, Continuity: v.Review.Continuity, ReviewDigest: v.Review.Digest}
	out = DeskReceiverPlan{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Intent: i, Review: v}
	return out, out.Validate()
}
func (r *Registry) InstallDeskReceiver(ctx context.Context, name, runner string, plan DeskReceiverPlan) (protocol.DeskOperation, error) {
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
	return c.InstallDeskReceiver(ctx, entry.Identity(), plan.Intent)
}

func (r *Registry) InspectDeskReceiver(ctx context.Context, name, runner, instance, desk string) (protocol.DeskReceiverStatus, error) {
	var out protocol.DeskReceiverStatus
	if (apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil || instance == "" || desk == "" {
		return out, errors.New("select an instance and desk")
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
	observed, err := c.DeskReceiverStatus(ctx, entry.Identity(), selection)
	if err != nil {
		return out, err
	}
	return observed, nil
}
