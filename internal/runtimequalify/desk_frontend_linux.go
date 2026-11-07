//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func (s *DeskRuntime) Frontend(ctx context.Context, desk string) (deskwork.FrontendState, error) {
	var out deskwork.FrontendState
	request, err := deskwork.FrontendInventory(s.ship, desk)
	if err != nil {
		return out, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return out, err
	}
	body, _ := json.Marshal(reply)
	if deskwork.DecodeJSONOutput(body, &out) != nil || out.Validate() != nil {
		return deskwork.FrontendState{}, errors.New("frontend observation unavailable or unsupported")
	}
	return out, nil
}

func (s *DeskRuntime) FrontendHash(ctx context.Context, m deskwork.FrontendManifest, files []deskwork.SourceFile) (string, error) {
	request, err := deskwork.FrontendGlobHash(m, files)
	if err != nil {
		return "", err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(reply)
	var hash string
	if deskwork.DecodeJSONOutput(body, &hash) != nil || (deskwork.Observation{Exists: true, Revision: 1, ClayHash: hash, Publication: "private"}).Validate() != nil {
		return "", errors.New("frontend artifact hash unavailable or unsupported")
	}
	return hash, nil
}
