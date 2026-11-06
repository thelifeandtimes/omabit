package runtimequalify

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// BootIdentityRepair is an offline-only journal, never a runtime/API override.
// State retains the OLD launch identity until the exact boot container is gone.
type BootIdentityRepair struct {
	Version            int          `json:"version"`
	ExpectedShip       string       `json:"expected_ship"`
	Ship               string       `json:"ship"`
	ContainerID        string       `json:"container_id"`
	ContainerStartedAt string       `json:"container_started_at"`
	VerifiedAt         time.Time    `json:"verified_at"`
	Stage              string       `json:"stage"`
	State              ManagedState `json:"state"`
}

func BootIdentityRepairEligible(s ManagedState, expected, ship, containerID string) error {
	if !protocol.ValidFakeShip(expected) || !protocol.ValidFakeShip(ship) || expected == ship || protocol.ShipClass(expected) != protocol.ShipClass(ship) ||
		!containerIDPattern.MatchString(containerID) || s.NativeShip != expected || !s.Prepared || s.Initialized || len(s.NativeKeyDigest) != 64 ||
		s.Comet == nil || s.Comet.Ship != expected || !s.Comet.Uncertain || s.Comet.StartedAt == nil || s.Comet.StartedAt.IsZero() || s.Comet.FinishedAt != nil ||
		s.Storage == nil || s.Storage.Uncertain || s.Storage.MountIdentity.Inode == 0 || s.UncertainMaintenance() || s.Maintenance != nil || s.Routine != nil ||
		s.Handoff != nil || s.HandoffStaging != nil || s.Deletion != nil || s.ImportRestore != nil || s.Import != nil || s.ExternalImport != nil || s.Adoption != nil || s.Fake != nil || s.MoveTarget != "" || s.StartupResume != nil || s.PruneIntent != nil || s.PrefixPrune != nil {
		return errors.New("identity repair requires only an unresolved keyed first boot with matching old/new ship classes")
	}
	found := false
	for _, c := range s.Runtime.Containers {
		if c.ID == containerID && c.Kind == "comet-boot" && c.CreateDispatched && !c.CreateUncertain && !found {
			found = true
		} else if c.Kind != "probe" || !c.Removed || c.CreateUncertain {
			return errors.New("identity repair found an unrelated runtime intent")
		}
	}
	if !found {
		return errors.New("identity repair requires the exact retained boot container")
	}
	return nil
}
