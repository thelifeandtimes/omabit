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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// OpenManagedHost locks the fixed development state directory. Docker may be
// unavailable: the control service still needs to serve its durable inventory.
func OpenManagedHost() (*ManagedHost, error) {
	q := &qualifier{d: defaults()}
	if err := q.accountBoundary(false); err != nil {
		return nil, err
	}
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

// ApplyManaged controls only a registered fake instance. It does not run the
// qualifier's automatic cleanup: a running managed ship outlives its client and
// runner process. All resource intents go through persist BEFORE Docker calls.
func ApplyManaged(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error) (ManagedState, ManagedObservation, error) {
	return applyManaged(ctx, action, state, persist, defaults())
}

func managedSession(state *ManagedState, persist func(ManagedState) error, d dependencies) (*qualifier, error) {
	if !runIDPattern.MatchString(state.ID) || state.Port < 1024 {
		return nil, errors.New("invalid managed instance identity or HTTP port")
	}
	d.profile, d.managed, d.hostPort = ProfileHTTP, true, state.Port
	q := &qualifier{d: d, r: state.Runtime, phase: "managed"}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
	dir := filepath.Join(q.base, state.ID)
	if state.Prepared {
		if q.r.RunID != state.ID || q.r.Directory != dir || q.r.PillPath != filepath.Join(dir, candidate.PillName) || q.r.Profile != ProfileHTTP ||
			q.r.Image != candidate.Image || q.r.PillSHA256 != d.pillSHA || q.r.PillBytes != d.pillBytes || q.r.PillURL != candidate.PillURL {
			return nil, errors.New("managed runtime state does not match its registered identity and candidate")
		}
		if len(q.r.Containers) > 3 {
			return nil, errors.New("unexpected managed container inventory")
		}
		for _, c := range q.r.Containers {
			if (c.Kind != "probe" && c.Kind != "pack" && !isHTTPKind(c.Kind)) || c.Name != q.resourceName(c.Kind) || (c.ID != "" && !containerIDPattern.MatchString(c.ID)) {
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
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || (action != "start" && action != "stop" && action != "pack") {
		return state, unknown, errors.New("invalid managed action or persistence callback")
	}
	if action != "stop" && state.MaintenanceUncertain {
		return state, unknown, errors.New("an interrupted maintenance outcome requires manual review")
	}
	if action == "pack" && (!state.Prepared || !state.Initialized) {
		return state, unknown, errors.New("offline pack requires an initialized registered pier")
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	finish := func(obs ManagedObservation, err error) (ManagedState, ManagedObservation, error) {
		state.Runtime = q.r
		return state, obs, err
	}
	if err := q.boundary(); err != nil {
		return finish(unknown, err)
	}
	for _, path := range q.storagePaths() {
		if err := q.owned(path, os.ModeDir, false); err != nil {
			return finish(unknown, err)
		}
	}
	if action == "start" || action == "pack" {
		mem, disk, err := d.admission(q.base)
		if err != nil || mem < 5<<30 || disk < 8<<30 {
			return finish(unknown, errors.New("runtime work requires at least 5 GiB available RAM and 8 GiB free disk"))
		}
	}
	if action == "stop" && !state.Prepared {
		return state, ManagedObservation{State: "uninitialized", Health: "unknown"}, nil
	}
	if !state.Prepared {
		dir := filepath.Join(q.base, state.ID)
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			return finish(unknown, errors.New("managed bootstrap destination already exists"))
		}
		// Record ownership before creating storage, without adopting any existing
		// path. An interruption here leaves a conservative recovery requirement.
		q.r = Result{Profile: ProfileHTTP, RunID: state.ID, Directory: dir, Image: candidate.Image,
			Platform: candidate.Platform, EngineVersion: rootless.EngineVersion, VereVersion: candidate.VereVersion,
			PillURL: candidate.PillURL, PillSHA256: d.pillSHA, PillBytes: d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Containers: []ContainerRecord{}}
		state.Prepared = true
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		if err := os.Mkdir(dir, 0700); err != nil {
			return finish(unknown, err)
		}
		for _, path := range []string{q.config, filepath.Join(dir, "data")} {
			if err := os.Mkdir(path, 0700); err != nil {
				return finish(unknown, err)
			}
		}
	}
	for _, path := range append(q.storagePaths(), q.r.Directory, q.config, filepath.Join(q.r.Directory, "data")) {
		if err := q.owned(path, os.ModeDir, path == q.r.Directory || path == q.config); err != nil {
			return finish(unknown, err)
		}
	}
	if err := q.verifyEngine(ctx); err != nil {
		return finish(unknown, err)
	}
	if action == "stop" {
		for i, c := range q.r.Containers {
			if c.Removed {
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
		return finish(ManagedObservation{State: "stopped", Health: "unknown", RecoveryRequired: !state.Initialized || state.MaintenanceUncertain}, nil)
	}
	if action == "pack" {
		for _, c := range q.r.Containers {
			if !c.Removed || c.CreateUncertain {
				return finish(unknown, errors.New("offline pack requires explicit stop and removal of all owned containers"))
			}
		}
		if err := q.pier(); err != nil {
			return finish(unknown, err)
		}
		measure, cancel := context.WithTimeout(ctx, 10*time.Second)
		before, err := pierBytes(measure, filepath.Join(q.r.Directory, "data/fake-zod"))
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
		state.Maintenance = &ManagedMaintenance{Kind: "pack", BeforePierBytes: before, StartedAt: time.Now().UTC()}
		state.MaintenanceUncertain = true
		if err := q.save(); err != nil {
			return finish(unknown, err)
		}
		if err := q.pack(ctx); err != nil {
			return finish(unknown, err)
		}
		if err := q.pier(); err != nil {
			return finish(unknown, err)
		}
		measure, cancel = context.WithTimeout(ctx, 10*time.Second)
		after, err := pierBytes(measure, filepath.Join(q.r.Directory, "data/fake-zod"))
		cancel()
		if err != nil {
			return finish(unknown, err)
		}
		now := time.Now().UTC()
		state.Maintenance.AfterPierBytes = after
		state.Maintenance.FinishedAt = &now
		state.MaintenanceUncertain = false
		if err := q.save(); err != nil {
			state.MaintenanceUncertain = true
			state.Maintenance.FinishedAt = nil
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
	mem, disk, err := d.admission(q.r.Directory)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return finish(unknown, errors.New("start requires at least 5 GiB available RAM and 8 GiB free disk"))
	}
	if !state.Initialized {
		entries, err := os.ReadDir(filepath.Join(q.r.Directory, "data"))
		if err != nil || len(entries) != 0 {
			return finish(unknown, errors.New("partial fake pier requires manual recovery; bootstrap will not be replayed"))
		}
	}
	q.r.Containers = []ContainerRecord{}
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if err := q.managedImage(ctx); err != nil {
		return finish(unknown, err)
	}
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	if !state.Initialized {
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
	i, err := q.create(ctx, kind)
	if err != nil {
		return finish(unknown, err)
	}
	if _, err := q.control(ctx, i, q.d.times.command, "start"); err != nil {
		return finish(unknown, err)
	}
	if err := q.readyHostHTTP(ctx, i); err != nil {
		return finish(unknown, err)
	}
	if err := q.pier(); err != nil {
		return finish(unknown, err)
	}
	state.Initialized = true
	if err := q.save(); err != nil {
		return finish(unknown, err)
	}
	return finish(ManagedObservation{State: "running", Health: "healthy", Endpoint: strings.TrimSuffix(q.r.Containers[i].HostEndpoint, "/~/host")}, nil)
}

// ObserveManaged makes no Docker mutations. It may resolve a known create from
// inspected identity, but never infers successful deletion from an empty listing.
func ObserveManaged(ctx context.Context, state ManagedState) (ManagedState, ManagedObservation, error) {
	return observeManaged(ctx, state, defaults())
}

func observeManaged(ctx context.Context, state ManagedState, d dependencies) (ManagedState, ManagedObservation, error) {
	ctx, cancel := context.WithTimeout(ctx, d.times.command)
	defer cancel()
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if state.MaintenanceUncertain {
		return state, unknown, nil
	}
	if !state.Prepared {
		return state, ManagedObservation{State: "uninitialized", Health: "unknown"}, nil
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return state, unknown, err
	}
	if err := q.verifyEngine(ctx); err != nil {
		return state, unknown, err
	}
	obs := ManagedObservation{State: "stopped", Health: "unknown", RecoveryRequired: !state.Initialized}
	if !state.Initialized {
		entries, err := os.ReadDir(filepath.Join(q.r.Directory, "data"))
		if err == nil && len(entries) == 0 {
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
				if err := d.hostProbe(ctx, endpoint); err == nil {
					if err := q.pier(); err != nil {
						return state, unknown, err
					}
					state.Initialized, obs.RecoveryRequired, obs.Health = true, false, "healthy"
				}
			}
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
