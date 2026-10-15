package runtimequalify

import (
	"regexp"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

// ManagedState is private driver state, persisted transactionally by the runner.
// Only validated qualified-fake provenance can seed an imported initialized pier;
// it is not an API for arbitrary filesystem paths or Docker resources.
type ManagedState struct {
	StartupResume        *StartupResumeIntent   `json:"startup_resume,omitempty"`
	Deletion             *ManagedDeletion       `json:"deletion,omitempty"`
	ExternalImport       *ManagedExternalImport `json:"external_import,omitempty"`
	Bootstrap            *protocol.Bootstrap    `json:"bootstrap,omitempty"`
	Handoff              *protocol.Handoff      `json:"handoff,omitempty"`
	HandoffSigningKey    []byte                 `json:"handoff_signing_key,omitempty"`
	ID                   string                 `json:"id"`
	Port                 uint16                 `json:"port"`
	Prepared             bool                   `json:"prepared"`
	Initialized          bool                   `json:"initialized"`
	Runtime              Result                 `json:"runtime"`
	MaintenanceUncertain bool                   `json:"maintenance_uncertain,omitempty"`
	Maintenance          *ManagedMaintenance    `json:"maintenance,omitempty"`
	Epochs               *epochs.Inventory      `json:"epochs,omitempty"`
	RollNeedsRuntime     bool                   `json:"roll_requires_run,omitempty"`
	PruneIntent          *ManagedPruneIntent    `json:"prune_intent,omitempty"`
	PrefixPrune          *PrefixPruneRecord     `json:"prefix_prune,omitempty"`
	Import               *ManagedImport         `json:"import,omitempty"`
	Comet                *ManagedComet          `json:"comet,omitempty"`
	Routine              *routines.Run          `json:"routine,omitempty"`
	Fake                 *ManagedFake           `json:"fake,omitempty"`
	Storage              *pierstore.Location    `json:"storage,omitempty"`
	MoveTarget           string                 `json:"move_target,omitempty"`
	Adoption             *ManagedAdoption       `json:"adoption,omitempty"`
	NativeShip           string                 `json:"native_ship,omitempty"`
	NativeKeyDigest      string                 `json:"native_key_digest,omitempty"`
	Boot                 *ManagedBoot           `json:"boot,omitempty"`
	AmesPort             uint16                 `json:"ames_port,omitempty"`
}

// StartupResumeIntent is internal authorization from the durable startup job,
// bound to the old container. It is never accepted from a user RPC.
type StartupResumeIntent struct {
	ContainerID  string `json:"container_id"`
	AllowUnclean bool   `json:"allow_unclean"`
}

type ManagedDeletion struct {
	Stage string                      `json:"stage"`
	Mount pierstore.DirectoryIdentity `json:"mount"`
}

type ManagedExternalImport struct {
	CopiedFiles int                      `json:"copied_files"`
	CopiedBytes int64                    `json:"copied_bytes"`
	Source      pierstore.ExternalSource `json:"source"`
	Stage       string                   `json:"stage"`
	HeldPath    string                   `json:"held_path,omitempty"`
}

func (s ManagedState) ExternalImportInFlight() bool {
	if s.ExternalImport == nil || s.ExternalImport.Stage == "complete" {
		return false
	}
	if s.ExternalImport.Stage == "reviewed" && s.ExternalImport.HeldPath == "" {
		return false
	}
	if s.ExternalImport.Stage == "enrolling" && s.Storage != nil && s.Storage.Identity.Inode != 0 {
		return false
	}
	return true
}

// Presence opts into canonical/custom pill selection and host Ames publication.
// Older fixed-preset registrations retain their original runtime contract.
type ManagedBoot struct {
	Pill *protocol.BootPill `json:"pill,omitempty"`
}

type ManagedAdoption struct {
	Fingerprint string     `json:"fingerprint"`
	Uncertain   bool       `json:"uncertain"`
	Verified    bool       `json:"verified"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type ManagedFake struct {
	Ship      string `json:"ship"`
	Uncertain bool   `json:"uncertain"`
}

// The pinned runtime generates the identity; this is a public name-shape check,
// not a general @p decoder or proof of Ames peer connectivity.
var cometName = regexp.MustCompile(`^~[a-z]{6}(-[a-z]{6}){3}--[a-z]{6}(-[a-z]{6}){3}$`)
var containerIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ManagedComet struct {
	Ship       string     `json:"ship,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Uncertain  bool       `json:"uncertain"`
}

// Networked runtime machinery is shared by generated comets and explicitly
// identity-pinned owned ships. NativeShip opts into the latter's name contract.
func (s ManagedState) networkIdentityValid() bool {
	return s.Comet != nil && (s.NativeShip != "" && s.Comet.Ship == s.NativeShip && protocol.ValidFakeShip(s.NativeShip) || s.NativeShip == "" && cometName.MatchString(s.Comet.Ship))
}

// MaintenanceQualified requires completed local provisioning evidence for a comet;
// a registered name or a public HTTP observation alone cannot authorize utilities.
func (s ManagedState) MaintenanceQualified() bool {
	if !s.Initialized || s.Adoption != nil && !s.Adoption.Verified {
		return false
	}
	c := s.Comet
	return c == nil || !c.Uncertain && s.networkIdentityValid() && c.StartedAt != nil &&
		!c.StartedAt.IsZero() && c.FinishedAt != nil && !c.FinishedAt.Before(*c.StartedAt)
}

type ManagedImport struct {
	Source      protocol.ImportSource `json:"source"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	FinishedAt  *time.Time            `json:"finished_at,omitempty"`
	CopiedFiles int                   `json:"copied_files"`
	CopiedBytes int64                 `json:"copied_bytes"`
	Uncertain   bool                  `json:"uncertain"`
}

func (s ManagedState) UncertainProvisioning() bool {
	return s.Deletion != nil || s.ExternalImport != nil && s.ExternalImport.Stage != "complete" || s.Handoff.Fenced() || s.Adoption != nil && (s.Adoption.Uncertain || s.Adoption.FinishedAt == nil) || s.Storage != nil && s.Storage.Uncertain || s.Fake != nil && s.Fake.Uncertain || s.Import != nil && (s.Import.Uncertain || s.Import.FinishedAt == nil || !s.Initialized) ||
		s.Comet != nil && (s.Comet.Uncertain || s.Comet.StartedAt != nil && (s.Comet.FinishedAt == nil || !s.Initialized || !s.networkIdentityValid()))
}

// A partial deletion's own durable marker remains authoritative even if a
// generic maintenance observation is missing or older.
func (s ManagedState) UncertainMaintenance() bool {
	return s.MaintenanceUncertain || (s.Runtime.Prune != nil && (s.Runtime.Prune.Uncertain || s.Runtime.Prune.FinishedAt == nil)) || (s.PrefixPrune != nil && (s.PrefixPrune.Uncertain || s.PrefixPrune.FinishedAt == nil))
}

type ManagedPruneIntent struct {
	Cutoff          string `json:"cutoff"`
	InventoryDigest string `json:"inventory_digest"`
	Confirmed       bool   `json:"confirmed"`
}

type ManagedMaintenance struct {
	Kind               string     `json:"kind"`
	BeforePierBytes    int64      `json:"before_pier_bytes"`
	AfterPierBytes     int64      `json:"after_pier_bytes"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	MemAvailableBytes  uint64     `json:"mem_available_bytes,omitempty"`
	DiskAvailableBytes uint64     `json:"disk_available_bytes,omitempty"`
	EpochsBefore       []string   `json:"epochs_before,omitempty"`
	EpochsAfter        []string   `json:"epochs_after,omitempty"`
	CreatedEpoch       string     `json:"created_epoch,omitempty"`
}

type ManagedObservation struct {
	State            string
	Health           string
	Endpoint         string
	RecoveryRequired bool
	Resources        *ManagedResources
}

// ManagedResources is an all-or-nothing observation from Docker's stats API.
// Raw memory usage includes charged cache/kernel memory; it is not loom usage.
type ManagedResources struct {
	ContainerID        string
	ObservedAt         time.Time
	MemoryCurrentBytes uint64
	MemoryLimitBytes   uint64
	PIDsCurrent        uint64
	CPU                *ManagedCPU
}

// Docker's cumulative nanosecond counters, kept exact until the runner derives
// a rate between two authenticated observations of the same container.
type ManagedCPU struct {
	TotalNS    uint64  `json:"total_ns"`
	SystemNS   uint64  `json:"system_ns"`
	OnlineCPUs uint32  `json:"online_cpus"`
	LimitCores float64 `json:"limit_cores"`
}

type ManagedHost struct {
	Directory string
	Socket    string
	close     func()
}

func (h *ManagedHost) Close() { h.close() }
