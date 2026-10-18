package runtimequalify

import (
	"errors"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AmesPublicationState proposes only the next-launch publication contract. The
// caller must obtain fresh ValidateAmesPublication evidence and commit this
// proposal with its revision-bound receipt. Enrollment/source evidence, removed
// container receipts, settings and storage identities are retained verbatim.
func AmesPublicationState(s ManagedState, port uint16) (ManagedState, error) {
	if port < 1024 || !s.Prepared || !s.Initialized || s.Boot != nil || s.Bootstrap != nil ||
		s.Fake != nil || s.Import != nil || s.NativeKeyDigest != "" || s.MoveTarget != "" ||
		s.UncertainProvisioning() || s.UncertainMaintenance() ||
		!s.networkIdentityValid() ||
		s.Adoption == nil || s.Adoption.FinishedAt == nil || s.Adoption.FinishedAt.IsZero() ||
		s.Storage == nil || s.Storage.Identity.Inode == 0 || s.Storage.Identity.UrbInode == 0 || s.Storage.MountIdentity.Inode == 0 ||
		s.Adoption.Fingerprint != pierstore.Fingerprint(s.Storage.Identity) ||
		pierstore.Kind(s.Storage.Path) != protocol.ShipClass(s.Comet.Ship) ||
		s.Comet.StartedAt == nil || s.Comet.StartedAt.IsZero() || s.Comet.FinishedAt == nil || s.Comet.FinishedAt.Before(*s.Comet.StartedAt) {
		return s, errors.New("publication requires an initialized unpublished live enrollment with intact storage and completed enrollment provenance")
	}
	if h := s.Handoff; h != nil && (h.Ship != s.Comet.Ship || h.Mode != protocol.ShipClass(s.Comet.Ship) || h.AmesPublished) {
		return s, errors.New("completed handoff does not match the unpublished live identity")
	}
	if source := s.ExternalImport; source != nil && (!filepath.IsAbs(source.Source.Path) || filepath.Clean(source.Source.Path) != source.Source.Path || source.Source.Identity.Inode == 0 || source.Source.Identity.UrbInode == 0 || !epochs.ValidDigest(source.Source.Fingerprint)) {
		return s, errors.New("completed external import has incomplete source provenance")
	}
	for _, c := range s.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return s, errors.New("stop and remove all retained containers before publishing Ames")
		}
		if c.CreateDispatched && (c.ID == "" || c.LastState == nil) || !c.CreateDispatched && (c.ID != "" || c.LastState != nil || c.AliveObserved || c.HostHTTPVerified) {
			return s, errors.New("removed container has no authenticated terminal receipt")
		}
		if last := c.LastState; last != nil && (last.Running || last.Paused || last.Restarting || last.Dead || last.OOMKilled || last.ExitCode != 0 && !(c.Kind == "probe" && last.ExitCode == 143)) {
			return s, errors.New("retained container outcome is not cleanly stopped")
		}
	}
	if n := s.Runtime.Network; n != nil && (n.CreateUncertain || !n.Removed && (!n.CreateDispatched || n.ID == "")) {
		return s, errors.New("resolve the retained network outcome before publishing Ames")
	}
	// Unpublished comet handoffs historically retained the identity only in
	// Comet.Ship. networkIdentityValid above accepts that shape only for actual
	// comet names. Pin the same identity for the publication-aware contract;
	// initialized state and the existing offline verification step are retained.
	s.NativeShip, s.Boot, s.AmesPort = s.Comet.Ship, &ManagedBoot{}, port
	// Legacy enrollment used the qualifier's fixed-pill metadata even though it
	// never bootstrapped. The initialized next-launch contract uses no boot pill;
	// retain the old metadata in the publication operation's runtime receipt.
	s.Runtime.PillURL, s.Runtime.PillSHA256, s.Runtime.PillBytes = "runtime-canonical", "", 0
	return s, nil
}
