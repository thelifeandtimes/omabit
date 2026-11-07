package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"time"
)

const ShipRoutesFeature = "ship-routes-v1"
const ShipRoutesPath = "/v1/ship-routes"
const ShipRoutesTimeout = 30 * time.Second

// ShipRouteSelection reuses the generation/configuration-bound process selector;
// it does not select or change any desk.
type ShipRouteSelection = DeskDiscoverySelection

type ShipRoutes struct {
	Version   int                `json:"version"`
	RunnerID  string             `json:"runner_id"`
	Mode      string             `json:"mode"`
	Selection ShipRouteSelection `json:"selection"`
	Record    shiproute.Record   `json:"record"`
	Registry  shiproute.Registry `json:"registry"`
	ReadAt    time.Time          `json:"read_at"`
}

func (s ShipRoutes) Validate() error {
	if s.Version != 1 || !ValidID(s.RunnerID) || !ValidID(s.Mode) || s.Selection.Validate() != nil || s.Record.Validate() != nil || s.Registry.Validate() != nil || s.ReadAt.IsZero() || s.Record.Binding.RunnerID != s.RunnerID || s.Record.Binding.Target != s.Selection.Target || s.Registry.Binding != s.Record.Binding || s.Registry.Revision != s.Record.Revision {
		return errors.New("invalid ship routes response")
	}
	// Check that the effective registry cannot silently override authored choices.
	automatic := []shiproute.Route{}
	for _, route := range s.Registry.Routes {
		if route.Source == "managed" && route.Scope != s.RunnerID {
			return errors.New("automatic local route belongs to another runner host")
		}
		if route.Source == "managed" || route.Source == "tailscale" {
			automatic = append(automatic, route)
		}
	}
	effective, err := s.Record.Settings.Effective(s.Record.Binding, s.Record.Revision, automatic)
	if err != nil || len(effective.Routes) != len(s.Registry.Routes) {
		return errors.New("ship routes disagree with settings")
	}
	for i, route := range effective.Routes {
		if route != s.Registry.Routes[i] {
			return errors.New("ship routes disagree with settings")
		}
	}
	return nil
}

type ShipRoutesUpdate struct {
	Selection ShipRouteSelection `json:"selection"`
	Update    shiproute.Update   `json:"update"`
}

func (u ShipRoutesUpdate) Validate() error {
	if u.Selection.Validate() != nil || u.Update.Validate() != nil || u.Selection.Target != u.Update.Binding.Target {
		return errors.New("invalid ship routes submission")
	}
	return nil
}
