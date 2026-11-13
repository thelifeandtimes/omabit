package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"regexp"
	"time"
)

const DeskCreateFeature = "desk-create-v1"
const DeskObservationPath = "/v1/desk-observation"
const DeskOperationsPath = "/v1/desk-operations"
const DeskReadTimeout = 95 * time.Second

var deskID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidDeskOperationID(id string) bool { return deskID.MatchString(id) }

type DeskSelection struct {
	Target         apppolicy.Target `json:"target"`
	RuntimeID      string           `json:"runtime_id"`
	ConfigRevision uint64           `json:"config_revision"`
}

func (s DeskSelection) Validate() error {
	if s.Target.Validate() != nil || s.Target.InstanceID == "" || s.Target.Desk == "" || s.Target.ProjectID != "" && s.Target.Generation == 0 || !deskID.MatchString(s.RuntimeID) || s.ConfigRevision == 0 || s.ConfigRevision > apppolicy.MaxRevision {
		return errors.New("invalid desk runtime selection")
	}
	return nil
}

type DeskObservation struct {
	Version        int                  `json:"version"`
	RunnerID       string               `json:"runner_id"`
	Mode           string               `json:"mode"`
	Selection      DeskSelection        `json:"selection"`
	PolicyRevision uint64               `json:"policy_revision"`
	Observation    deskwork.Observation `json:"observation"`
	ReadAt         time.Time            `json:"read_at"`
}
type DeskOperation struct {
	Version  int              `json:"version"`
	RunnerID string           `json:"runner_id"`
	Mode     string           `json:"mode"`
	Receipt  deskwork.Receipt `json:"receipt"`
}

const DeskDiscoveryFeature = "desk-discovery-v1"
const DeskDiscoveryPath = "/v1/desk-discovery"

type DeskDiscoverySelection struct {
	Target         apppolicy.Target `json:"target"`
	RuntimeID      string           `json:"runtime_id"`
	ConfigRevision uint64           `json:"config_revision"`
}

func (s DeskDiscoverySelection) ForDesk(desk string) DeskSelection {
	target := s.Target
	target.Desk = desk
	return DeskSelection{Target: target, RuntimeID: s.RuntimeID, ConfigRevision: s.ConfigRevision}
}
func (s DeskDiscoverySelection) Validate() error {
	if s.Target.Desk != "" {
		return errors.New("discovery selects a ship, not a desk")
	}
	return s.ForDesk("base").Validate()
}

type DeskDiscovery struct {
	Version   int                    `json:"version"`
	RunnerID  string                 `json:"runner_id"`
	Mode      string                 `json:"mode"`
	Selection DeskDiscoverySelection `json:"selection"`
	Desks     []string               `json:"desks"`
	ReadAt    time.Time              `json:"read_at"`
}
