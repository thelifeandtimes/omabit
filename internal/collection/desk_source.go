package collection

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) InspectDeskSource(ctx context.Context, name, runner, instance, desk string) (protocol.DeskSourceStatus, error) {
	var out protocol.DeskSourceStatus
	if instance == "" || desk == "" || (apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil {
		return out, errors.New("select a ship and source desk")
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
	setting, err := c.DeskSetting(ctx, entry.Identity(), selection.Target)
	if err != nil {
		return out, err
	}
	if setting.Setting.Desk.Workflow != "mounted" {
		return out, errors.New("use Urgit's native Git/Clay status for this workflow")
	}
	return c.DeskSource(ctx, entry.Identity(), protocol.DeskSourceRequest{Selection: selection, SettingsRevision: setting.Setting.Revision}, setting.Setting.Desk.SourceName)
}
