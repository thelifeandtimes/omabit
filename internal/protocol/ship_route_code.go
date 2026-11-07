package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

const ShipRouteCodeFeature = "ship-route-code-v1"
const ShipRouteCodePath = "/v1/ship-route-code"

// ShipRouteCodeRequest is internal authentication input. Its full ship binding
// and enabled route are checked under runner lifecycle admission before +code.
// Standing application permission belongs to the collector, not this endpoint.
type ShipRouteCodeRequest struct {
	Selection        ShipRouteSelection `json:"selection"`
	Binding          shiproute.Binding  `json:"binding"`
	RegistryRevision uint64             `json:"registry_revision"`
	Route            shiproute.Route    `json:"route"`
}

func (u ShipRouteCodeRequest) Validate() error {
	if u.Selection.Validate() != nil || u.Binding.Validate() != nil || u.Selection.Target != u.Binding.Target || u.RegistryRevision > apppolicy.MaxRevision || u.Route.Validate() != nil || !u.Route.Enabled {
		return errors.New("invalid ship route credential selection")
	}
	return nil
}

// Never store this response in a journal, status view or diagnostic.
type ShipRouteCode struct {
	Version  int                  `json:"version"`
	RunnerID string               `json:"runner_id"`
	Mode     string               `json:"mode"`
	Request  ShipRouteCodeRequest `json:"request"`
	Code     string               `json:"code"`
}
