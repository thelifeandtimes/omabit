package runtimequalify

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"reflect"
)

// ImportRestoreReview describes a data-preserving alternative to continuing a
// partial copy: retain the current destination under archive/, restore the same
// held source object to its original name, and retire the failed registration.
// It grants no mutation authority by itself. The runner requires an idle,
// stopped instance, a current configuration revision and explicit digest-bound
// approval before journaling any moves.
type ImportRestoreReview struct {
	Version           int                         `json:"version"`
	RuntimeID         string                      `json:"runtime_id"`
	OriginalPath      string                      `json:"original_path"`
	Source            pierstore.ExternalSource    `json:"source"`
	SourceParent      pierstore.DirectoryIdentity `json:"source_parent"`
	HoldingParent     pierstore.DirectoryIdentity `json:"holding_parent"`
	SourceTree        *pierstore.RemovalTree      `json:"source_tree"`
	Destination       string                      `json:"destination"`
	DestinationParent pierstore.DirectoryIdentity `json:"destination_parent"`
	DestinationTree   *pierstore.RemovalTree      `json:"destination_tree"`
	ArchiveParent     pierstore.DirectoryIdentity `json:"archive_parent"`
	RetainedPath      string                      `json:"retained_path"`
	Digest            string                      `json:"digest"`
}

func importRestoreCandidate(state ManagedState) bool {
	if state.ExternalImport == nil || state.ExternalImport.Stage != "copying" && state.ExternalImport.Stage != "verifying" {
		return false
	}
	return protocol.ValidID(state.ID) && !state.Prepared && !state.Initialized && reflect.DeepEqual(state.Runtime, Result{}) &&
		state.Storage != nil && !state.Storage.Uncertain && state.Storage.MountIdentity.Inode != 0 && state.Storage.Identity == (pierstore.Identity{}) &&
		state.Adoption != nil && state.Adoption.Uncertain && !state.Adoption.Verified && state.Adoption.FinishedAt == nil && state.Adoption.Fingerprint == state.ExternalImport.Source.Fingerprint &&
		!state.Handoff.Fenced() && state.HandoffStaging == nil && state.StartupResume == nil &&
		state.Routine == nil && state.Deletion == nil && !state.UncertainMaintenance() &&
		state.Import == nil && state.Bootstrap == nil && state.NativeKeyDigest == "" && state.MoveTarget == "" &&
		(state.Fake == nil || !state.Fake.Uncertain) && (state.Comet == nil || !state.Comet.Uncertain && state.Comet.StartedAt == nil && state.Comet.FinishedAt == nil)
}

// ManagedImportRestore is an explicitly approved restoration request. Journal
// remains nil until the worker verifies the current filesystem review and
// durably records intent. Once Journal exists, its digest cannot be replaced.
type ManagedImportRestore struct {
	Digest  string                `json:"digest"`
	Journal *ImportRestoreJournal `json:"journal,omitempty"`
}

type ImportRestoreJournal struct {
	Stage  string              `json:"stage"`
	Review ImportRestoreReview `json:"review"`
}

// ImportRestoreEligible is only a state gate. Filesystem checks are performed
// again by preview and the serialized worker before any move.
func (s ManagedState) ImportRestoreEligible() bool { return importRestoreCandidate(s) }
