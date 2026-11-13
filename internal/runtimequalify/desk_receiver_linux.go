//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func (s *DeskRuntime) Subscription(ctx context.Context, desk string) (deskwork.Subscription, error) {
	var out deskwork.Subscription
	q, err := deskwork.SubscriptionInventory(s.ship, desk)
	if err != nil {
		return out, err
	}
	reply, err := s.call(ctx, q)
	if err != nil {
		return out, err
	}
	body, _ := json.Marshal(reply)
	if deskwork.DecodeJSONOutput(body, &out) != nil || out.Validate() != nil {
		return deskwork.Subscription{}, errors.New("subscription observation unavailable or unsupported")
	}
	return out, nil
}
func (s *DeskRuntime) InstallRemote(ctx context.Context, desk string, p deskwork.Publisher) error {
	q, err := deskwork.InstallRemote(s.ship, desk, p)
	return s.effect(ctx, q, err)
}

func (s *DeskRuntime) ReceiverSourceReady(ctx context.Context, desk string) (bool, error) {
	q, err := deskwork.ReceiverSourceReady(s.ship, desk)
	if err != nil {
		return false, err
	}
	reply, err := s.call(ctx, q)
	if err != nil {
		return false, err
	}
	body, _ := json.Marshal(reply)
	var ready bool
	if deskwork.DecodeJSONOutput(body, &ready) != nil {
		return false, errors.New("receiver source observation unavailable")
	}
	return ready, nil
}
