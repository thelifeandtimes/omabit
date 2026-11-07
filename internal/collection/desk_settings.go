package collection

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) DeskSetting(ctx context.Context, name, runner string, target apppolicy.Target) (protocol.DeskSetting, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskSetting{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.DeskSetting(ctx, entry.Identity(), target)
}
func (r *Registry) InspectDeskSetting(ctx context.Context, name, runner, instance, desk string) (protocol.DeskSetting, error) {
	var out protocol.DeskSetting
	if deskwork.DefaultSetting(apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil {
		return out, errors.New("select a ship and optional desk")
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
	for _, t := range policy.Targets {
		if t.Ready && t.Target.InstanceID == instance {
			target := t.Target
			target.Desk = desk
			return c.DeskSetting(ctx, entry.Identity(), target)
		}
	}
	return out, errors.New("ship or project generation is unresolved")
}
func (r *Registry) SaveDeskSetting(ctx context.Context, name, runner string, update deskwork.SettingUpdate) (protocol.DeskSetting, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskSetting{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.SetDeskSetting(ctx, entry.Identity(), update)
}
