//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// Application observes installation, declared agents, Treaty and Docket as
// separate facts. It does not assert that source compiled or the app is healthy.
func (s *DeskRuntime) Application(ctx context.Context, desk string) (deskwork.AppState, error) {
	var out deskwork.AppState
	request, err := deskwork.AppInventory(s.ship, desk)
	if err != nil {
		return out, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return out, err
	}
	body, _ := json.Marshal(reply)
	if deskwork.DecodeJSONOutput(body, &out) != nil || out.Normalize() != nil {
		return deskwork.AppState{}, errors.New("application observation unavailable or unsupported")
	}
	return out, nil
}

// Release performs exactly one typed effect. Its caller must journal dispatch
// and check the current policy, workflow and reviewed baseline under its lock.
// Neither this adapter nor the transport retries a failed or uncertain effect.
func (s *DeskRuntime) Release(ctx context.Context, desk, action string, baseline deskwork.Observation) error {
	var request deskwork.LensRequest
	var err error
	switch action {
	case "activate-local":
		request, err = deskwork.ActivateLocal(s.ship, desk, baseline)
	case "publish-source":
		request, err = deskwork.PublishSourceAt(s.ship, desk, baseline)
	case "publish-treaty":
		request, err = deskwork.PublishTreatyAt(s.ship, desk, baseline)
	default:
		return errors.New("unsupported desk release action")
	}
	return s.effect(ctx, request, err)
}
