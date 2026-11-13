package shiproute

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

// Runtime comes from the runner's held, verified process and pier selection.
// It is not an external URL's claim about which ship is behind that URL.
type Runtime struct {
	Ship       string
	Continuity string
	Origin     string
}

func (r Runtime) Validate() error {
	local := Route{ID: "managed-http", Origin: r.Origin, Network: "local", Scope: "runner", Source: "managed", Enabled: true}
	if !eyre.ValidShip(r.Ship) || !digest.MatchString(r.Continuity) || local.Validate() != nil {
		return errors.New("invalid live ship route identity")
	}
	return nil
}
