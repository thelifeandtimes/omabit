package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskInspection struct {
	Observation protocol.DeskObservation `json:"observation"`
	Permissions apppolicy.Effective      `json:"permissions"`
}
type DeskPlan struct {
	Version  int             `json:"version"`
	RunnerID string          `json:"runner_id"`
	Mode     string          `json:"mode"`
	Intent   deskwork.Intent `json:"intent"`
}

func (p DeskPlan) Validate() error {
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || p.Intent.Validate() != nil || p.Intent.Action != "create-desk" {
		return errors.New("invalid desk creation plan")
	}
	return nil
}
func (r *Registry) deskConnection(ctx context.Context, name, runner string) (Connection, error) {
	entry, err := r.Get(ctx, name)
	if err != nil {
		return entry, err
	}
	if entry.RunnerID != runner || !protocol.ValidID(runner) {
		return Connection{}, errors.New("selected runner enrollment changed")
	}
	return entry, nil
}
func (r *Registry) InspectDesk(ctx context.Context, name, runner, instance, desk string) (DeskInspection, error) {
	var out DeskInspection
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
	observation, err := c.DeskObservation(ctx, entry.Identity(), selection)
	if err != nil {
		return out, err
	}
	if observation.PolicyRevision != policy.Policy.Revision {
		return out, errors.New("policy changed during desk inspection; inspect again")
	}
	out.Observation = observation
	out.Permissions = policy.Policy.Effective(selection.Target)
	return out, nil
}
func (r *Registry) PlanDeskCreation(ctx context.Context, name, runner, instance, desk string) (DeskPlan, error) {
	var out DeskPlan
	inspected, err := r.InspectDesk(ctx, name, runner, instance, desk)
	if err != nil {
		return out, err
	}
	if inspected.Observation.Observation.Exists {
		return out, errors.New("desk already exists")
	}
	var nonce [32]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return out, err
	}
	v := inspected.Observation
	out = DeskPlan{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Intent: deskwork.Intent{Version: 1, ID: hex.EncodeToString(nonce[:]), Target: v.Selection.Target, RuntimeID: v.Selection.RuntimeID, ConfigRevision: v.Selection.ConfigRevision, PolicyRevision: v.PolicyRevision, Action: "create-desk", Baseline: v.Observation}}
	return out, out.Validate()
}
func (r *Registry) CreateDesk(ctx context.Context, name, runner string, plan DeskPlan) (protocol.DeskOperation, error) {
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
	return c.CreateDesk(ctx, entry.Identity(), plan.Intent)
}
func (r *Registry) DeskOperation(ctx context.Context, name, runner, id string) (protocol.DeskOperation, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskOperation{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.DeskOperation(ctx, entry.Identity(), id)
}

func (r *Registry) ReconcileDeskOperation(ctx context.Context, name, runner, id string) (protocol.DeskOperation, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskOperation{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ReconcileDeskOperation(ctx, entry.Identity(), id)
}

// Resolve a current process from the same policy target and snapshot. The runner
// checks this selection again under lifecycle admission before any private read.
func selectDeskRuntime(ctx context.Context, c *client.Client, identity client.Identity, policy protocol.AppPolicyStatus, instance, desk string) (protocol.DeskSelection, error) {
	var selection protocol.DeskSelection
	for _, t := range policy.Targets {
		if t.Target.InstanceID == instance && t.Ready {
			selection.Target = t.Target
			selection.Target.Desk = desk
		}
	}
	if selection.Target.InstanceID == "" {
		return selection, errors.New("ship or project generation is unresolved")
	}
	snapshot, err := c.Snapshot(ctx)
	if err != nil {
		return selection, err
	}
	if err = identity.Check(snapshot.RunnerID, snapshot.Mode); err != nil {
		return selection, err
	}
	count := 0
	for _, s := range snapshot.Instances {
		if s.ID == instance {
			count++
			if s.ProjectID != selection.Target.ProjectID || s.ObservedState != "running" || s.ActiveOperationID != "" || s.RecoveryRequired || s.Resources == nil {
				return selection, errors.New("select a ready running ship")
			}
			selection.ConfigRevision = s.ConfigRevision
			selection.RuntimeID = s.Resources.ContainerID
		}
	}
	if count != 1 || selection.Validate() != nil {
		return selection, errors.New("current desk runtime selection unavailable")
	}
	return selection, nil
}
func (r *Registry) DiscoverDesks(ctx context.Context, name, runner, instance string) (protocol.DeskDiscovery, error) {
	var out protocol.DeskDiscovery
	if !protocol.ValidID(instance) {
		return out, errors.New("select a registered ship instance")
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
	selection, err := selectDeskRuntime(ctx, c, entry.Identity(), policy, instance, "base")
	if err != nil {
		return out, err
	}
	return c.DiscoverDesks(ctx, entry.Identity(), protocol.DeskDiscoverySelection{Target: selection.Target.Ship(), RuntimeID: selection.RuntimeID, ConfigRevision: selection.ConfigRevision})
}
