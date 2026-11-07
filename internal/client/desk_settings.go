package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"reflect"
	"time"
)

func validateDeskSetting(expected Identity, target apppolicy.Target, out protocol.DeskSetting) error {
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return err
	}
	if out.Version != 1 || out.Setting.Validate() != nil || out.Setting.Target != target || out.ReadAt.IsZero() || time.Since(out.ReadAt) > 2*time.Minute || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return errors.New("invalid workflow settings response")
	}
	return nil
}
func (c *Client) DeskSetting(ctx context.Context, expected Identity, target apppolicy.Target) (protocol.DeskSetting, error) {
	var out protocol.DeskSetting
	if deskwork.DefaultSetting(target).Validate() != nil {
		return out, errors.New("invalid workflow target")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskSettingsFeature); err != nil {
		return out, err
	}
	if err := c.do(ctx, http.MethodPost, protocol.DeskSettingsPath, target, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskSetting(expected, target, out); err != nil {
		return protocol.DeskSetting{}, err
	}
	return out, nil
}

// SetDeskSetting is for user-facing Setup. It cannot grant agent permissions.
func (c *Client) SetDeskSetting(ctx context.Context, expected Identity, update deskwork.SettingUpdate) (protocol.DeskSetting, error) {
	var out protocol.DeskSetting
	if err := update.Validate(); err != nil {
		return out, err
	}
	before, err := c.DeskSetting(ctx, expected, update.Target)
	if err != nil {
		return out, err
	}
	if before.Setting.Revision != update.ExpectedRevision {
		return out, errors.New("workflow settings changed; read them again")
	}
	if err = c.do(ctx, http.MethodPut, protocol.DeskSettingsPath, update, &out, true); err != nil {
		return out, err
	}
	if err = validateDeskSetting(expected, update.Target, out); err != nil {
		return protocol.DeskSetting{}, err
	}
	if out.Setting.Revision != update.ExpectedRevision+1 || !reflect.DeepEqual(out.Setting.Ship, update.Ship) || !reflect.DeepEqual(out.Setting.Desk, update.Desk) {
		return protocol.DeskSetting{}, errors.New("workflow save changed the reviewed settings; refresh to reconcile")
	}
	return out, nil
}
