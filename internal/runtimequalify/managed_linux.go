//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// OpenManagedHost locks the fixed development state directory. Docker may be
// unavailable: the control service still needs to serve its durable inventory.
func OpenManagedHost() (*ManagedHost, error) {
	q := &qualifier{d: defaults()}
	if err := q.accountBoundary(false); err != nil {
		return nil, err
	}
	// Keep inventory and Stop available even if an externally edited pier tree
	// needs repair. Discovery reports that problem; runtime path checks stay strict.
	_ = pierstore.Init(q.d.home)
	q.base = filepath.Join(q.d.home, ".local/share/urbit-runner/managed")
	unlock, err := q.lockStorage()
	if err != nil {
		return nil, err
	}
	runtime := filepath.Join(filepath.Dir(q.socket), "urbit-runner")
	if err := os.Mkdir(runtime, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		unlock()
		return nil, err
	}
	if err := q.owned(runtime, os.ModeDir, true); err != nil {
		unlock()
		return nil, err
	}
	return &ManagedHost{Directory: q.base, Socket: filepath.Join(runtime, "runner.sock"), close: unlock}, nil
}

// ApplyManaged controls only a registered instance. It does not run the
// qualifier's automatic cleanup: a running managed ship outlives its client and
// runner process. All resource intents go through persist BEFORE Docker calls.
func ApplyManaged(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error) (ManagedState, ManagedObservation, error) {
	return applyManaged(ctx, action, state, persist, defaults())
}

func managedSession(state *ManagedState, persist func(ManagedState) error, d dependencies) (*qualifier, error) {
	if !runIDPattern.MatchString(state.ID) || state.Port < 1024 {
		return nil, errors.New("invalid managed instance identity or HTTP port")
	}
	if !protocol.ValidRecordedLoom(protocol.EffectiveLoom(state.Loom)) {
		return nil, errors.New("invalid loom")
	}
	d.loom, d.nativeControl = protocol.EffectiveLoom(state.Loom), state.NativeControl
	d.profile, d.managed, d.hostPort = ProfileHTTP, true, state.Port
	d.fake = state.Fake
	d.storage = state.Storage
	d.nativeShip = state.NativeShip
	d.keyDigest = state.NativeKeyDigest
	d.boot = state.Boot
	d.bootstrap = state.Bootstrap
	d.amesPort = state.AmesPort
	if state.Boot != nil {
		if state.NativeShip == "" || state.AmesPort < 1024 {
			return nil, errors.New("invalid owned boot/port configuration")
		}
		if state.Boot.Pill != nil {
			if !protocol.ValidBootPill(state.Boot.Pill) {
				return nil, errors.New("invalid selected boot pill")
			}
			d.pillSHA = state.Boot.Pill.SHA256
			d.pillBytes = state.Boot.Pill.Bytes
		} else {
			d.pillSHA = ""
			d.pillBytes = 0
		}
	}
	if state.Bootstrap != nil {
		if !protocol.ValidBootstrap(state.Bootstrap) || state.Boot != nil {
			return nil, errors.New("invalid or conflicting bootstrap selection")
		}
		d.pillSHA, d.pillBytes = "", 0
		if state.Bootstrap.Pill != nil {
			d.pillSHA, d.pillBytes = state.Bootstrap.Pill.SHA256, state.Bootstrap.Pill.Bytes
		}
	}
	storageKind := "fake"
	if state.Comet != nil {
		storageKind = "comet"
	}
	if state.NativeShip != "" {
		storageKind = protocol.ShipClass(state.NativeShip)
		if storageKind == "" || state.Comet == nil {
			return nil, errors.New("invalid owned ship identity")
		}
	}
	if state.Storage == nil || !pierstore.ValidPath(state.Storage.Path) || pierstore.Kind(state.Storage.Path) != storageKind || state.Initialized && (state.Storage.Identity.Inode == 0 || state.Storage.MountIdentity.Inode == 0) {
		return nil, errors.New("registered pier location does not match its identity mode")
	}
	if _, err := pierstore.Absolute(d.home, state.Storage.Path); err != nil {
		return nil, err
	}
	if id, _ := pierstore.Split(state.Storage.Path); id != "" {
		storageRoot, _, err := pierstore.Resolve(d.home, state.Storage.Path)
		if err != nil {
			return nil, err
		}
		admission := d.admission
		d.admission = func(path string) (uint64, uint64, error) {
			mem, disk, err := admission(path)
			if err != nil {
				return mem, disk, err
			}
			m, available, err := admission(storageRoot)
			return min(mem, m), min(disk, available), err
		}
	}
	if state.Fake != nil {
		if state.Comet != nil || state.Import != nil || !protocol.ValidFakeShip(state.Fake.Ship) {
			return nil, errors.New("invalid registered fake identity or pier directory")
		}
	}
	if state.Comet != nil {
		if state.Import != nil || state.Comet.Ship != "" && !state.networkIdentityValid() || state.Initialized && !state.networkIdentityValid() {
			return nil, errors.New("invalid managed comet identity or provenance")
		}
		d.profile = ProfileComet
		// Production defaults allow mining/boot readiness up to twenty minutes;
		// fixture timings remain bounded at their injected shorter duration.
		if d.times.ready == 10*time.Minute {
			d.times.ready = 20 * time.Minute
		}
	}
	q := &qualifier{d: d, r: state.Runtime, phase: "managed"}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
	dir := filepath.Join(q.base, state.ID)
	if state.Prepared {
		if q.r.EngineVersion != "" && !hostcompat.ValidateRecordedEngineVersion(q.r.EngineVersion) {
			return nil, errors.New("invalid recorded provisioning engine version")
		}
		if q.r.RunID != state.ID || q.r.Directory != dir || q.r.PillPath != filepath.Join(dir, candidate.PillName) || q.r.Profile != d.profile ||
			q.r.Image != candidate.Image || q.r.VereVersion != candidate.VereVersion || q.r.PillSHA256 != d.pillSHA || q.r.PillBytes != d.pillBytes || q.r.PillURL != q.pillSource() {
			return nil, errors.New("managed runtime state does not match its registered identity and candidate")
		}
		if len(q.r.Containers) > 3 {
			return nil, errors.New("unexpected managed container inventory")
		}
		for _, c := range q.r.Containers {
			if c.EngineVersion != "" && !hostcompat.ValidateRecordedEngineVersion(c.EngineVersion) {
				return nil, errors.New("invalid recorded launch engine version")
			}
			entry, args := baseCommand(c.Kind)
			if !protocol.ValidRecordedLoom(protocol.EffectiveLoom(c.Loom)) || c.NativeControl && (entry != "/bin/urbit" || len(args) == 0 || args[0] != "-t") {
				return nil, errors.New("invalid recorded runtime settings")
			}
			if c.SnapshotMinutes != 0 && c.SnapshotMinutes != DefaultSnapshotMinutes {
				return nil, errors.New("invalid recorded snapshot interval")
			}
			if state.Comet != nil && c.Kind != "probe" && !isCometKind(c.Kind) && !isManagedCometUtility(c.Kind) {
				return nil, errors.New("comet state contains an incompatible runtime intent")
			}
			if state.Comet == nil && strings.HasPrefix(c.Kind, "comet-") {
				return nil, errors.New("fake state contains a comet runtime intent")
			}
			if (c.Kind != "probe" && c.Kind != "pack" && c.Kind != "meld" && c.Kind != "roll" && !isManagedCometUtility(c.Kind) && !isNetworkedKind(c.Kind)) || c.Name != q.resourceName(c.Kind) || (c.ID != "" && !containerIDPattern.MatchString(c.ID)) {
				return nil, errors.New("invalid managed container reference")
			}
		}
		if n := q.r.Network; n != nil && (n.Name != q.resourceName("http") || (n.ID != "" && !containerIDPattern.MatchString(n.ID))) {
			return nil, errors.New("invalid managed network reference")
		}
	} else if state.Runtime.RunID != "" || state.Initialized {
		return nil, errors.New("inconsistent unprepared managed state")
	}
	q.config = filepath.Join(dir, "docker-config")
	if persist != nil {
		q.d.persist = func(r Result) error { state.Runtime = r; return persist(*state) }
	}
	return q, nil
}

func applyManaged(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	state, obs, err := applyManagedAction(ctx, action, state, persist, d)
	if errors.Is(err, ErrHostCompatibility) {
		obs = compatibilityObservation(state)
	}
	return state, obs, err
}

func applyManagedAction(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if action == "restore-import" {
		return applyImportRestore(ctx, state, persist, d)
	}
	if state.ImportRestore != nil && action != "stop" {
		return state, unknown, errors.New("source restoration must be completed before other runtime work")
	}
	if action == "delete-pier" {
		return applyPierDeletion(ctx, state, persist, d)
	}
	if action == "import-pier" {
		return applyExternalImport(ctx, state, persist, d)
	}
	if action == "recover-import" {
		return applyImportRecovery(ctx, state, persist, d)
	}
	if action == "prepare-handoff" || action == "receive-handoff" || action == "commit-handoff" {
		return applyHandoff(ctx, action, state, persist, d)
	}
	if state.Handoff.Fenced() {
		return state, unknown, errors.New("ship is fenced for a cross-host handoff")
	}
	if action == "enroll-pier" {
		return applyPierEnrollment(ctx, state, persist, d)
	}
	if action == "move-pier" {
		return applyPierMove(ctx, state, persist, d)
	}
	if action == "startup-resume" {
		return applyStartupResume(ctx, state, persist, d)
	}
	if action == "routine-pack" || action == "routine-meld" {
		return applyRoutine(ctx, action, state, persist, d)
	}
	if action == "import-fake" {
		if state.Comet != nil {
			return state, unknown, errors.New("fake import cannot target a comet registration")
		}
		return applyImportedFake(ctx, state, persist, d)
	}
	if action != "stop" && state.UncertainProvisioning() {
		return state, unknown, errors.New("incomplete provisioning requires manual review")
	}
	maintenance := action == "pack" || action == "meld" || action == "roll"
	inspection := action == "inspect-epochs"
	pruning := action == "prune-epochs"
	if persist == nil || (action != "start" && action != "stop" && !maintenance && !inspection && !pruning) {
		return state, unknown, errors.New("invalid managed action or persistence callback")
	}
	if action != "stop" && state.UncertainMaintenance() {
		return state, unknown, errors.New("an interrupted maintenance outcome requires manual review")
	}
	if action == "roll" && state.RollNeedsRuntime {
		return state, unknown, errors.New("resume and cleanly stop the pier before requesting another new roll")
	}
	if (maintenance || inspection || pruning) && (!state.Prepared || !state.Initialized) {
		return state, unknown, errors.New("offline maintenance requires an initialized registered pier")
	}
	if (maintenance || pruning) && !state.MaintenanceQualified() {
		return state, unknown, errors.New("comet maintenance requires completed identity-pinned provisioning")
	}
	if pruning && (state.PruneIntent == nil || !state.PruneIntent.Confirmed || !epochs.ValidDigest(state.PruneIntent.InventoryDigest)) {
		return state, unknown, errors.New("prefix pruning requires its explicit confirmed inventory-bound intent")
	}
	if pruning {
		if _, err := epochs.Number(state.PruneIntent.Cutoff); err != nil {
			return state, unknown, err
		}
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	finish := func(obs ManagedObservation, err error) (ManagedState, ManagedObservation, error) {
		state.Runtime = q.r
		return state, obs, err
	}
	if err := q.accountBoundary(false); err != nil {
		return finish(unknown, err)
	}
	if action != "stop" && state.Initialized {
		if err := q.pier(); err != nil {
			return finish(unknown, err)
		}
	}
	for _, path := range q.storagePaths() {
		if err := q.owned(path, os.ModeDir, false); err != nil {
			return finish(unknown, err)
		}
	}
	if action == "start" {
		if err := q.admitStart(q.base, state.Initialized, "start admission"); err != nil {
			return finish(unknown, err)
		}
	}
	if maintenance {
		mem, disk, err := q.d.admission(q.base)
		if err != nil || mem < 5<<30 || disk < 8<<30 {
			return finish(unknown, errors.New("runtime work requires at least 5 GiB available RAM and 8 GiB free disk"))
		}
	}
	if action == "start" {
		kind := "http-boot"
		if state.Comet != nil {
			kind = "comet-boot"
		}
		if err := q.admitPorts(kind); err != nil {
			return finish(unknown, err)
		}
	}
	if action == "stop" && !state.Prepared {
		return state, ManagedObservation{State: "uninitialized", Health: "unknown", RecoveryRequired: state.UncertainProvisioning()}, nil
	}
	// Admission precedes workspace creation, intent changes and maintenance.
	// Stop may use an API-compatible unsupported family, but all controls remain
	// bound to the recorded container identity below.
	if err := q.verifyEngineAccess(ctx, action == "stop" || inspection); err != nil {
		return finish(unknown, err)
	}
	state.Epochs = nil
	if state.Prepared && action == "start" && q.r.EngineVersion == "" && recordedEngineEvidenceValid(q.r) {
		// An offline enrollment has no original daemon observation. Its first
		// runtime preparation supplies one; nonempty history remains immutable.
		q.r.EngineVersion = q.engineEvidence.Version
	}
	if !state.Prepared {
		dir := filepath.Join(q.base, state.ID)
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			return finish(unknown, errors.New("managed bootstrap destination already exists"))
		}
		// Record ownership before creating storage, without adopting any existing
		// path. An interruption here leaves a conservative recovery requirement.
		q.r = Result{Profile: q.d.profile, RunID: state.ID, Directory: dir, Image: candidate.Image,
			Platform: candidate.Platform, EngineVersion: q.engineEvidence.Version, VereVersion: candidate.VereVersion,
			PillURL: q.pillSource(), PillSHA256: q.d.pillSHA, PillBytes: q.d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Containers: []ContainerRecord{}}
		state.Prepared = true
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		if err := os.Mkdir(dir, 0700); err != nil {
			return finish(unknown, err)
		}
		for _, path := range []string{q.config} {
			if err := os.Mkdir(path, 0700); err != nil {
				return finish(unknown, err)
			}
		}
	}
	for _, path := range append(q.storagePaths(), q.r.Directory, q.config) {
		if err := q.owned(path, os.ModeDir, path == q.r.Directory || path == q.config); err != nil {
			return finish(unknown, err)
		}
	}
	if pruning {
		originalPersist := q.d.persist
		q.d.persist = func(r Result) error {
			if p := state.PrefixPrune; p != nil {
				state.MaintenanceUncertain = p.Uncertain
				if !p.Uncertain && p.FinishedAt != nil {
					state.Epochs = p.After
					state.PruneIntent = nil
				}
			}
			return originalPersist(r)
		}
		intent := *state.PruneIntent
		err := q.deletePrefix(ctx, intent.Cutoff, intent.InventoryDigest, func(p *PrefixPruneRecord) { state.PrefixPrune = p })
		if err != nil {
			state.Epochs = nil
			if state.PrefixPrune != nil && state.PrefixPrune.Uncertain {
				state.MaintenanceUncertain = true
			}
			return finish(unknown, err)
		}
		return finish(ManagedObservation{State: "stopped", Health: "unknown"}, nil)
	}
	if inspection {
		for _, c := range q.r.Containers {
			if !c.Removed || c.CreateUncertain {
				return finish(unknown, errors.New("epoch inspection requires a clean stopped pier with all containers removed"))
			}
		}
		inventory, err := q.inspectEpochs(ctx)
		if err != nil {
			return finish(unknown, err)
		}
		state.Epochs = inventory
		if err := q.save(); err != nil {
			state.Epochs = nil
			return finish(unknown, err)
		}
		return finish(ManagedObservation{State: "stopped", Health: "unknown"}, nil)
	}
	if action == "stop" {
		// Keep a durable pre-dispatch phase through Stop's own callbacks. A
		// restart during cleanup must not turn a report-only intent ambiguous.
		pendingReport := managedMaintenanceReportPending(state, q.r)
		if !pendingReport {
			q.r.Stage = "stopping"
		}
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		for i, c := range q.r.Containers {
			if c.Removed {
				continue
			}
			if containerCreateUndispatched(c) {
				if err := q.cleanupOne(ctx, i); err != nil {
					return finish(unknown, err)
				}
				continue
			}
			actual, err := q.inspect(ctx, i)
			if err != nil {
				return finish(unknown, err)
			}
			if err := q.save(); err != nil {
				return finish(unknown, err)
			}
			if active(actual) {
				if err := q.stop(ctx, i, c.Kind != "probe"); err != nil {
					return finish(unknown, err)
				}
			} else if err := q.remove(ctx, i); err != nil {
				return finish(unknown, err)
			}
		}
		if !q.restrictedEngine && managedMaintenanceUndispatched(state, q.r) {
			previousStage := q.r.Stage
			q.r.Stage = "stopped"
			state.MaintenanceUncertain = false
			if err := q.save(); err != nil {
				q.r.Stage = previousStage
				state.MaintenanceUncertain = true
				return finish(unknown, err)
			}
		} else if !pendingReport {
			q.r.Stage = "stopped"
		}
		return finish(ManagedObservation{State: "stopped", Health: "unknown", RecoveryRequired: !state.Initialized || state.UncertainMaintenance() || state.UncertainProvisioning()}, nil)
	}
	if maintenance {
		for _, c := range q.r.Containers {
			if !c.Removed || c.CreateUncertain {
				return finish(unknown, errors.New("offline maintenance requires explicit stop and removal of all owned containers"))
			}
		}
		if err := q.pier(); err != nil {
			return finish(unknown, err)
		}
		utility := action
		if state.Comet != nil {
			utility = "comet-" + action
		}
		measure, cancel := context.WithTimeout(ctx, 10*time.Second)
		before, err := pierBytes(measure, q.pierPath())
		cancel()
		if err != nil {
			return finish(unknown, err)
		}
		q.r.Containers = []ContainerRecord{}
		if err := q.managedImage(ctx); err != nil {
			return finish(unknown, err)
		}
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		if err := q.probe(ctx); err != nil {
			return finish(unknown, err)
		}
		var beforeEpochs *epochs.Inventory
		if action == "roll" {
			beforeEpochs, err = q.inspectEpochs(ctx)
			if err != nil {
				return finish(unknown, err)
			}
			if len(beforeEpochs.Entries) >= epochs.MaxEpochs {
				return finish(unknown, errors.New("epoch inventory limit reached; roll not dispatched"))
			}
		}
		// Image preparation and the live probe can outlast the initial capacity
		// observation. Re-admit at the utility boundary while holding the host
		// expensive-work gate; never treat swap as available physical memory.
		mem, disk, err := q.d.admission(q.r.Directory)
		if err != nil || mem < 5<<30 || disk < 8<<30 {
			return finish(unknown, errors.New("maintenance requires at least 5 GiB available RAM and 8 GiB free disk immediately before dispatch"))
		}
		state.Maintenance = &ManagedMaintenance{Kind: action, BeforePierBytes: before, StartedAt: time.Now().UTC(), MemAvailableBytes: mem, DiskAvailableBytes: disk}
		if beforeEpochs != nil {
			state.Maintenance.EpochsBefore = epochNames(beforeEpochs)
		}
		state.MaintenanceUncertain = true
		previousStage := q.r.Stage
		// This phase and the report commit together before offlineUtility can
		// even append its container intent. create replaces it atomically with
		// the dispatch marker before sending any request to the daemon.
		q.r.Stage = managedMaintenancePendingStage(utility)
		if err := q.save(); err != nil {
			// Without an acknowledged publication, the returned in-memory state
			// cannot claim durable phase evidence. A committed callback recovered
			// after a crash still contains the pending phase and can resolve it.
			q.r.Stage = previousStage
			return finish(unknown, err)
		}
		if err := q.offlineUtility(ctx, utility); err != nil {
			return finish(unknown, err)
		}
		if err := q.pier(); err != nil {
			return finish(unknown, err)
		}
		if action == "roll" {
			afterEpochs, err := q.inspectEpochs(ctx)
			if err != nil {
				return finish(unknown, err)
			}
			state.Maintenance.EpochsAfter = epochNames(afterEpochs)
			if !managedRollGrowth(beforeEpochs, afterEpochs) {
				return finish(unknown, errors.New("roll did not produce exactly one complete newer epoch; manual review required"))
			}
			state.Maintenance.CreatedEpoch = afterEpochs.Entries[len(afterEpochs.Entries)-1].Name
			state.Epochs = afterEpochs
		}
		measure, cancel = context.WithTimeout(ctx, 10*time.Second)
		after, err := pierBytes(measure, q.pierPath())
		cancel()
		if err != nil {
			return finish(unknown, err)
		}
		now := time.Now().UTC()
		state.Maintenance.AfterPierBytes = after
		state.Maintenance.FinishedAt = &now
		state.MaintenanceUncertain = false
		q.r.Stage = "stopped"
		if action == "roll" {
			state.RollNeedsRuntime = true
		}
		if err := q.save(); err != nil {
			state.MaintenanceUncertain = true
			state.Maintenance.FinishedAt = nil
			state.Epochs = nil
			return finish(unknown, err)
		}
		return finish(ManagedObservation{State: "stopped", Health: "unknown"}, nil)
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return finish(unknown, errors.New("existing or uncertain managed container requires observation/stop before start"))
		}
	}
	if n := q.r.Network; n != nil && n.CreateUncertain {
		return finish(unknown, errors.New("uncertain managed network requires manual recovery"))
	}
	if err := q.admitStart(q.r.Directory, state.Initialized, "start preparation"); err != nil {
		return finish(unknown, err)
	}
	if !state.Initialized {
		if state.NativeShip != "" {
			if err := q.verifyBootKey(); err != nil {
				return finish(unknown, err)
			}
		}
		root, err := pierstore.PrepareMount(d.home, state.Storage)
		if err != nil {
			return finish(unknown, err)
		}
		if err = q.save(); err != nil {
			return finish(unknown, err)
		}
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			return finish(unknown, errors.New("partial pier requires manual recovery; bootstrap will not be replayed"))
		}
	}
	q.r.Containers = []ContainerRecord{}
	q.r.Stage = "preparing-runtime"
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if err := q.managedImage(ctx); err != nil {
		return finish(unknown, err)
	}
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if !state.Initialized && !q.canonicalPill() {
		q.r.Stage = "preparing-pill"
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		// Pure preparation may be resumed while data is empty and every prior
		// container outcome is resolved. Existing pill bytes must match the pin.
		var err error
		if _, statErr := os.Lstat(q.r.PillPath); errors.Is(statErr, os.ErrNotExist) {
			err = q.pill(ctx)
		} else if statErr != nil {
			err = statErr
		} else {
			err = q.verifyLocalPill(ctx)
		}
		if err != nil {
			return finish(unknown, err)
		}
	}
	if err := q.probe(ctx); err != nil {
		return finish(unknown, err)
	}
	if q.r.Network == nil {
		if err := q.createNetwork(ctx); err != nil {
			return finish(unknown, err)
		}
	} else if _, err := q.inspectNetwork(ctx); err != nil {
		return finish(unknown, err)
	}
	kind := "http-restart"
	if !state.Initialized {
		kind = "http-boot"
	} else if err := q.pier(); err != nil {
		return finish(unknown, err)
	}
	// Image/pill preparation, the probe and network setup can outlast the
	// earlier admission. Check both runtime and selected pier storage again
	// before recording provisioning uncertainty or creating a runtime. A
	// refusal here leaves only resolved preparation resources and is retryable.
	for _, path := range []string{q.r.Directory, filepath.Dir(q.pierPath())} {
		if err := q.admitStart(path, state.Initialized, "start immediately before dispatch"); err != nil {
			return finish(unknown, err)
		}
	}
	var bootDispatchIntent func() func()
	if state.Comet != nil {
		kind = "comet-resume"
		if state.Adoption != nil && !state.Adoption.Verified {
			kind = "comet-verify"
		}
		if !state.Initialized {
			kind = "comet-boot"
			bootDispatchIntent = func() func() {
				// Publish this fence atomically with the boot's dispatch marker,
				// after create's final host admission and local preparation. A
				// refused create must not leave a report-only provisioning fence.
				now := time.Now().UTC()
				previous := state.Comet
				next := *previous
				next.StartedAt, next.Uncertain = &now, true
				state.Comet = &next
				return func() { state.Comet = previous }
			}
		}
	}
	i, err := q.createWithDispatchIntent(ctx, kind, bootDispatchIntent)
	if err != nil {
		return finish(unknown, err)
	}
	q.r.Stage = "starting-runtime"
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if _, err := q.control(ctx, i, q.d.times.command, "start"); err != nil {
		return finish(unknown, err)
	}
	q.r.Stage = "waiting-for-identity"
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if state.Comet != nil {
		ship, err := q.readyCometIdentity(ctx, i, state.Comet.Ship)
		if err != nil {
			if errors.Is(err, errCometIdentityChanged) {
				state.Comet.Uncertain = true
				err = errors.Join(err, q.save())
			}
			return finish(unknown, err)
		}
		state.Comet.Ship = ship
	} else {
		if err := q.readyHostHTTP(ctx, i); err != nil {
			if state.Fake != nil && errors.Is(err, errFakeIdentityChanged) {
				state.Fake.Uncertain = true
				err = errors.Join(err, q.save())
			}
			return finish(unknown, err)
		}
	}
	if err := q.pier(); err != nil {
		return finish(unknown, err)
	}
	state.Initialized = true
	if state.Adoption != nil {
		state.Adoption.Verified = true
	}
	state.RollNeedsRuntime = false
	if state.Comet != nil && state.Comet.Uncertain {
		now := time.Now().UTC()
		state.Comet.FinishedAt = &now
		state.Comet.Uncertain = false
	}
	if err := q.save(); err != nil {
		if state.Comet != nil {
			state.Comet.Uncertain = true
		}
		return finish(unknown, err)
	}
	if kind == "comet-verify" {
		state.Runtime = q.r
		state, obs, err := applyManaged(ctx, "stop", state, persist, d)
		if err != nil {
			return state, obs, err
		}
		return applyManaged(ctx, "start", state, persist, d)
	}
	if state.NativeKeyDigest != "" {
		if err := os.Remove(q.keyPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return finish(unknown, errors.New("ship started but private bootstrap key cleanup needs review"))
		}
		state.NativeKeyDigest = ""
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
	}
	q.r.Stage = "ready"
	return finish(ManagedObservation{State: "running", Health: "healthy", Endpoint: strings.TrimSuffix(q.r.Containers[i].HostEndpoint, "/~/host")}, nil)
}

func managedMaintenancePendingStage(kind string) string { return "maintenance-pending-" + kind }

func managedMaintenanceReportPending(state ManagedState, runtime Result) bool {
	m := state.Maintenance
	if !state.MaintenanceUncertain || m == nil || m.FinishedAt != nil || m.StartedAt.IsZero() ||
		(m.Kind != "pack" && m.Kind != "meld" && m.Kind != "roll") {
		return false
	}
	kind := m.Kind
	if state.Comet != nil {
		kind = "comet-" + kind
	}
	return runtime.Stage == managedMaintenancePendingStage(kind)
}

// A maintenance attempt resets the inventory before recording its report and
// pending phase. Either that explicit phase or a matching undispatched utility
// intent proves non-dispatch, provided no record carries contrary evidence.
// Historical report-only states lack that proof; successful cleanup of
// dispatched work and unrelated pruning fences also remain conservative.
func managedMaintenanceUndispatched(state ManagedState, runtime Result) bool {
	m := state.Maintenance
	if !state.MaintenanceUncertain || m == nil || m.StartedAt.IsZero() || m.FinishedAt != nil ||
		(m.Kind != "pack" && m.Kind != "meld" && m.Kind != "roll") ||
		state.Runtime.Prune != nil && (state.Runtime.Prune.Uncertain || state.Runtime.Prune.FinishedAt == nil) ||
		state.PrefixPrune != nil && (state.PrefixPrune.Uncertain || state.PrefixPrune.FinishedAt == nil) {
		return false
	}
	kind := m.Kind
	if state.Comet != nil {
		kind = "comet-" + kind
	}
	found := false
	for _, c := range runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return false
		}
		if c.Kind == "probe" {
			continue
		}
		if found || c.Kind != kind || !containerCreateUndispatched(c) {
			return false
		}
		found = true
	}
	return found || managedMaintenanceReportPending(state, runtime)
}

// ObserveManaged inspects runtime state and reads bounded one-shot Docker stats.
// It does not start/stop/remove containers or infer deletion from absence.
func ObserveManaged(ctx context.Context, state ManagedState) (ManagedState, ManagedObservation, error) {
	return observeManaged(ctx, state, defaults())
}

func observeManaged(ctx context.Context, state ManagedState, d dependencies) (ManagedState, ManagedObservation, error) {
	state, obs, err := observeManagedRuntime(ctx, state, d)
	if errors.Is(err, ErrHostCompatibility) {
		obs = compatibilityObservation(state)
	}
	return state, obs, err
}

func observeManagedRuntime(ctx context.Context, state ManagedState, d dependencies) (ManagedState, ManagedObservation, error) {
	if state.Handoff.Fenced() {
		status := "moving"
		if state.Handoff.State == "moved" {
			status = "moved"
		}
		return state, ManagedObservation{State: status, Health: "unknown"}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, d.times.command)
	defer cancel()
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if state.UncertainMaintenance() || state.UncertainProvisioning() {
		return state, unknown, nil
	}
	if !state.Prepared {
		return state, ManagedObservation{State: "uninitialized", Health: "unknown"}, nil
	}
	if state.Initialized {
		if _, err := pierstore.Check(d.home, *state.Storage); err != nil {
			return state, ManagedObservation{State: "missing", Health: "unknown", RecoveryRequired: true}, nil
		}
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return state, unknown, err
	}
	if err := q.verifyEngineAccess(ctx, true); err != nil {
		return state, unknown, err
	}
	obs := ManagedObservation{State: "stopped", Health: "unknown", RecoveryRequired: !state.Initialized}
	resourceIndex := -1
	if !state.Initialized {
		entries, err := os.ReadDir(filepath.Dir(q.pierPath()))
		if err == nil && len(entries) == 0 || errors.Is(err, os.ErrNotExist) && state.Storage.MountIdentity.Inode == 0 {
			obs.State, obs.RecoveryRequired = "uninitialized", false
		}
	}
	for i, c := range q.r.Containers {
		if c.Removed {
			continue
		}
		actual, err := q.inspect(ctx, i)
		if err != nil {
			state.Runtime = q.r
			return state, unknown, err
		}
		if err := q.containment(actual, c.Kind); err != nil {
			return state, unknown, err
		}
		if c.Kind == "probe" {
			return state, unknown, errors.New("interrupted probe requires explicit stop")
		}
		if actual.State.Paused || actual.State.Restarting || actual.State.Dead {
			state.Runtime = q.r
			return state, ManagedObservation{State: "unknown", Health: "unhealthy", RecoveryRequired: true}, nil
		}
		if active(actual) {
			obs.State, obs.Health = "running", "starting"
			endpoint, err := q.publishedEndpoint(actual, c.Kind)
			if err == nil {
				obs.Endpoint = strings.TrimSuffix(endpoint, "/~/host")
				healthErr := error(nil)
				if state.Comet != nil {
					ship, err := d.hostIdentity(ctx, endpoint)
					healthErr = err
					if err == nil && ship != state.Comet.Ship {
						state.Comet.Uncertain = true
						return state, unknown, errors.New("managed comet identity changed")
					}
				} else {
					healthErr = q.probeFakeHTTP(ctx, endpoint)
					if state.Fake != nil && errors.Is(healthErr, errFakeIdentityChanged) {
						state.Fake.Uncertain = true
						return state, unknown, nil
					}
				}
				if healthErr == nil {
					if err := q.pier(); err != nil {
						return state, unknown, err
					}
					state.Initialized, obs.RecoveryRequired, obs.Health = true, false, "healthy"
					state.RollNeedsRuntime = false
				}
			}
			resourceIndex = i
		} else if *actual.State.ExitCode != 0 || *actual.State.OOMKilled || actual.State.Dead {
			obs.State, obs.Health = "failed", "unhealthy"
			obs.RecoveryRequired = true
		} else {
			// A stopped but retained container needs an explicit stop action to
			// finish removal before a replacement may be created.
			obs.RecoveryRequired = true
		}
	}
	if q.r.Network != nil && !q.r.Network.Removed {
		if _, err := q.inspectNetwork(ctx); err != nil {
			state.Runtime = q.r
			return state, unknown, err
		}
	}
	// Finish health/ownership checks before optional telemetry. A timed-out
	// resource read cannot consume a later health check's deadline and turn a
	// healthy HTTP observation into a claimed runtime failure.
	if resourceIndex >= 0 {
		obs.Resources, _ = q.resources(ctx, resourceIndex)
	}
	state.Runtime = q.r
	return state, obs, nil
}

// Logical regular-file bytes, not allocated disk blocks or loom utilization.
// WalkDir never follows symlinks and never reads pier contents.
func pierBytes(ctx context.Context, root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Size() < 0 || size > int64(^uint64(0)>>1)-info.Size() {
				return errors.New("pier measurement overflow")
			}
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, errors.New("could not measure the stopped pier")
	}
	return size, nil
}
