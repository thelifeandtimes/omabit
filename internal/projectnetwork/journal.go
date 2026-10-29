package projectnetwork

import (
	"fmt"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Preparation records bridge/anchor creation before any member is launched.
// An observed resource requires an exact ID returned by this dispatch plus a
// successful independent inspection. Finding its name/labels is not adoption.
// A returned ID is retained as created before inspection; it is not yet trusted.
// A lost create reply stays dispatched; retry/recovery is a separate operation.
type Preparation struct {
	ProjectID       string   `json:"project_id"`
	ProjectRevision uint64   `json:"project_revision"`
	Generation      uint64   `json:"generation"`
	OperationID     string   `json:"operation_id"`
	PlanDigest      string   `json:"plan_digest"`
	EngineID        string   `json:"engine_id"`
	Sequence        uint64   `json:"sequence"`
	Bridge          Creation `json:"bridge"`
	Anchor          Creation `json:"anchor"`
}

type Creation struct {
	Name           string `json:"name"`
	ContractDigest string `json:"contract_digest"`
	Phase          string `json:"phase"`
	ID             string `json:"id,omitempty"`
}

func (c Creation) valid() bool {
	if !protocol.ValidID(c.Name) || !digestPattern.MatchString(c.ContractDigest) {
		return false
	}
	switch c.Phase {
	case "planned", "dispatched":
		return c.ID == ""
	case "created", "observed":
		return digestPattern.MatchString(c.ID)
	default:
		return false
	}
}

func (p Preparation) Validate(plan Plan) error {
	digest, err := plan.Digest()
	if err != nil || !protocol.ValidID(p.ProjectID) || !protocol.ValidID(p.OperationID) || p.ProjectRevision == 0 || p.Generation == 0 || digest != p.PlanDigest || !validEngine(p.EngineID) || !p.Bridge.valid() || !p.Anchor.valid() || p.Bridge.Name == p.Anchor.Name || p.Bridge.ID != "" && p.Bridge.ID == p.Anchor.ID {
		return fmt.Errorf("%w: invalid preparation journal", ErrRuntimeObservation)
	}
	phase := map[string]uint64{"planned": 0, "dispatched": 1, "created": 2, "observed": 3}
	bridge, anchor := phase[p.Bridge.Phase], phase[p.Anchor.Phase]
	if anchor != 0 && bridge != 3 || p.Sequence != bridge+anchor {
		return fmt.Errorf("%w: skipped preparation phase", ErrRuntimeObservation)
	}
	return nil
}

// Advance produces the next write-ahead record. Its caller must commit it with
// an exact sequence comparison before dispatch. This method never calls Docker.
// The immutable contract and engine cannot be changed by completion callbacks.
func (p Preparation) Advance(plan Plan, expected uint64, kind, phase, id string) (Preparation, error) {
	refuse := func() (Preparation, error) {
		return Preparation{}, fmt.Errorf("%w: stale or invalid preparation transition", ErrRuntimeObservation)
	}
	if err := p.Validate(plan); err != nil {
		return Preparation{}, err
	}
	if expected != p.Sequence {
		return refuse()
	}
	var resource *Creation
	switch kind {
	case "bridge":
		resource = &p.Bridge
	case "anchor":
		if p.Bridge.Phase != "observed" {
			return refuse()
		}
		resource = &p.Anchor
	default:
		return refuse()
	}
	if resource.Phase == "planned" && phase == "dispatched" && id == "" || resource.Phase == "dispatched" && phase == "created" && digestPattern.MatchString(id) || resource.Phase == "created" && phase == "observed" && id == resource.ID {
		resource.Phase, resource.ID = phase, id
		p.Sequence++
	} else {
		return refuse()
	}
	if err := p.Validate(plan); err != nil {
		return Preparation{}, err
	}
	return p, nil
}
