package runtimequalify

import (
	"encoding/hex"
	"path/filepath"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// Only an intent without any contrary dispatch/daemon evidence can be retired
// locally. Missing containers or a missing dispatch flag alone are not proof.
func containerCreateUndispatched(r ContainerRecord) bool {
	return !r.CreateDispatched && !r.CreateUncertain && r.ID == "" && r.LastState == nil &&
		!r.AliveObserved && !r.PrivateHTTPReachable && r.HostEndpoint == "" && !r.HostHTTPVerified
}

// UndispatchedInitialBoot recognizes only the current writer's complete first-
// boot intent before its dispatch write. It does not authorize Start, clear a
// prior fence, or infer anything from daemon/filesystem absence. Reconciliation
// may avoid inventing private damage flags; explicit Stop must retire the intent
// and fresh observation must validate the workspace before new admission.
func (s ManagedState) UndispatchedInitialBoot() bool {
	id, err := hex.DecodeString(s.ID)
	if err != nil || len(id) != 16 || strings.ToLower(s.ID) != s.ID || !s.Prepared || s.Initialized ||
		s.UncertainProvisioning() || s.UncertainMaintenance() || s.Storage == nil || !pierstore.ValidPath(s.Storage.Path) ||
		s.Storage.MountIdentity.Inode == 0 || s.Storage.Identity != (pierstore.Identity{}) ||
		s.Adoption != nil || s.ExternalImport != nil || s.Import != nil || s.Handoff != nil || s.HandoffStaging != nil ||
		s.Maintenance != nil || s.Routine != nil || s.StartupResume != nil || s.MoveTarget != "" || s.RollNeedsRuntime ||
		s.PruneIntent != nil || s.PrefixPrune != nil || s.Epochs != nil {
		return false
	}
	profile, prefix, kind, storageKind := ProfileHTTP, "urbit-managed-fake-", "http-boot", "fake"
	if s.Comet != nil {
		profile, prefix, kind = ProfileComet, "urbit-managed-comet-", "comet-boot"
		storageKind = "comet"
		if s.NativeShip != "" {
			storageKind = protocol.ShipClass(s.NativeShip)
		}
		if s.Comet.StartedAt != nil || s.Comet.FinishedAt != nil || s.Fake != nil ||
			s.NativeShip == "" && (s.Comet.Ship != "" || s.NativeKeyDigest != "") ||
			s.NativeShip != "" && (s.Comet.Ship != s.NativeShip || !protocol.ValidFakeShip(s.NativeShip) || !containerIDPattern.MatchString(s.NativeKeyDigest)) {
			return false
		}
	} else if s.NativeShip != "" || s.NativeKeyDigest != "" || s.Boot != nil || s.Fake != nil && !protocol.ValidFakeShip(s.Fake.Ship) {
		return false
	}
	if storageKind == "" || pierstore.Kind(s.Storage.Path) != storageKind {
		return false
	}
	r := s.Runtime
	if r.RunID != s.ID || r.Profile != profile || !filepath.IsAbs(r.Directory) || filepath.Clean(r.Directory) != r.Directory || filepath.Base(r.Directory) != s.ID ||
		r.Stage != "preparing-runtime" && r.Stage != "preparing-pill" ||
		r.Platform != candidate.Platform || r.Image != candidate.Image || r.VereVersion != candidate.VereVersion ||
		(r.ImageID != candidate.ConfigDigest && r.ImageID != candidate.ManifestDigest) ||
		(r.RepoDigest != candidate.Image && r.RepoDigest != strings.TrimPrefix(candidate.Image, "docker.io/")) ||
		!hostcompat.ValidateRecordedEngineVersion(r.EngineVersion) || !r.LimitsObserved || r.PierMetadataSeen || r.CheckpointPassed || r.RuntimeQualified ||
		r.ReportPath != "" || r.CleanupComplete || r.Failure != "" || len(r.Phases) != 0 || r.RetainedLogs != nil ||
		r.Meld != nil || r.Prune != nil || r.Recovery != nil || r.EpochLifecycle != nil || len(r.PrefixPrunes) != 0 || r.PierImport != nil || r.Comet != nil ||
		len(r.CometUtilities) != 0 || r.CometFinalEpochs != nil ||
		!protocol.ValidRecordedLoom(protocol.EffectiveLoom(s.Loom)) || len(r.Containers) != 2 {
		return false
	}
	n := r.Network
	if n == nil || n.Name != prefix+s.ID+"-http" || n.Removed || !n.CreateDispatched || n.CreateUncertain || !containerIDPattern.MatchString(n.ID) {
		return false
	}
	probe, boot := r.Containers[0], r.Containers[1]
	last := probe.LastState
	if probe.Kind != "probe" || probe.Name != prefix+s.ID+"-probe" || !probe.Removed || !probe.CreateDispatched || probe.CreateUncertain ||
		!containerIDPattern.MatchString(probe.ID) || !hostcompat.ValidateRecordedEngineVersion(probe.EngineVersion) ||
		last == nil || last.ObservedAt.IsZero() || last.Running || last.Paused || last.Restarting || last.Dead || last.OOMKilled || last.LaunchError != "" {
		return false
	}
	return boot.Kind == kind && boot.Name == prefix+s.ID+"-"+kind && !boot.Removed &&
		hostcompat.ValidateRecordedEngineVersion(boot.EngineVersion) && containerCreateUndispatched(boot) &&
		boot.SnapshotMinutes == DefaultSnapshotMinutes && boot.Loom == protocol.EffectiveLoom(s.Loom) && boot.NativeControl == s.NativeControl
}
