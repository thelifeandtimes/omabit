//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

type timings struct{ command, pull, download, ready, poll, stop, cleanup, pack, cleanupCommand time.Duration }
type dependencies struct {
	home, runtimeDir string
	identity         func() (identity, error)
	stat             func(string) (node, error)
	admission        func(string) (uint64, uint64, error)
	execute          func(context.Context, []string, []string) (string, error)
	download         func(context.Context, string) (candidate.Artifact, error)
	randomID         func() (string, error)
	pillSHA          string
	pillBytes        int64
	times            timings
}

type qualifier struct {
	d                           dependencies
	r                           Result
	uid, gid                    int
	socket, base, config, phase string
	progress                    func(string)
}

// Run has no runtime, image, identity, pier, shell or endpoint overrides. All
// injections are package-private and per-run, solely for unprivileged tests.
// Local prevalidation failures return the zero Result without creating a run.
// The dedicated UID, its daemon and its private directories are trusted; these
// metadata checks are not isolation from another process already holding that UID.
func Run(ctx context.Context, progress func(string)) (Result, error) {
	return run(ctx, progress, defaults())
}

func defaults() dependencies {
	return dependencies{
		home: rootless.HomeDir, identity: currentIdentity, stat: statNode,
		admission: available, execute: executeDocker, download: candidate.DownloadPill,
		randomID: func() (string, error) {
			var b [16]byte
			_, err := rand.Read(b[:])
			return hex.EncodeToString(b[:]), err
		},
		pillSHA: candidate.PillSHA256, pillBytes: candidate.PillBytes,
		times: timings{30 * time.Second, 10 * time.Minute, 5 * time.Minute, 10 * time.Minute,
			5 * time.Second, 310 * time.Second, 11 * time.Minute, 10 * time.Minute, 5 * time.Second},
	}
}

func run(ctx context.Context, progress func(string), d dependencies) (result Result, err error) {
	q := &qualifier{d: d, progress: progress}
	if err = ctx.Err(); err != nil {
		return
	}
	mem, disk, err := q.prevalidate()
	if err != nil {
		return Result{}, err
	}
	unlock, err := q.lockStorage()
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	if err = q.priorReports(ctx); err != nil {
		return Result{}, err
	}
	id, err := d.randomID()
	if err != nil || !runIDPattern.MatchString(id) {
		return Result{}, errors.New("cannot allocate run identity")
	}
	dir := filepath.Join(q.base, id)
	if err = os.Mkdir(dir, 0700); err != nil {
		return Result{}, errors.New("fresh run directory unavailable")
	}
	q.r = Result{RunID: id, Directory: dir, ReportPath: filepath.Join(dir, "report.json"),
		Image: candidate.Image, Platform: candidate.Platform,
		EngineVersion: rootless.EngineVersion, VereVersion: candidate.VereVersion,
		PillURL: candidate.PillURL, PillSHA256: d.pillSHA, PillBytes: d.pillBytes,
		PillPath: filepath.Join(dir, candidate.PillName), MemAvailableBytes: mem, DiskAvailableBytes: disk,
		Phases: []Phase{}, Containers: []ContainerRecord{}}
	q.config = filepath.Join(dir, "docker-config")
	defer func() {
		q.phase = "cleanup"
		q.emit()
		cleanupErr := q.cleanup()
		q.r.CleanupComplete = cleanupErr == nil
		cleanupDetail := "owned containers stopped and removed; data retained"
		if cleanupErr != nil {
			cleanupDetail = cleanupErr.Error()
		}
		q.r.Phases = append(q.r.Phases, Phase{"cleanup", cleanupErr == nil, cleanupDetail})
		err = errors.Join(err, cleanupErr)
		q.r.CheckpointPassed = err == nil && q.r.CleanupComplete
		if err != nil {
			q.r.Failure = err.Error()
		}
		if saveErr := q.save(); saveErr != nil {
			q.r.CheckpointPassed = false
			err = errors.Join(err, saveErr)
			q.r.Failure = err.Error()
			// Best effort to replace any older report with the failed outcome.
			_ = q.save()
		}
		result = q.r
	}()
	if err = os.Chmod(dir, 0700); err != nil {
		return q.r, errors.New("cannot secure run directory")
	}
	if err = q.save(); err != nil {
		return q.r, err
	}
	for _, path := range []string{q.config, filepath.Join(dir, "data")} {
		if err = os.Mkdir(path, 0700); err != nil {
			return q.r, errors.New("cannot create private run storage")
		}
		if err = os.Chmod(path, 0700); err != nil {
			return q.r, errors.New("cannot secure run storage")
		}
	}
	steps := []struct {
		name, detail string
		work         func(context.Context) error
	}{
		{"engine", "pinned private rootless engine; no leftover qualification containers", q.engine},
		{"image", "candidate image ID, platform and repository digest verified", q.image},
		{"pill", "local pill size and SHA-256 verified", q.pill},
		{"probe", "live cgroup limits and Vere version verified", q.probe},
		{"boot", "fake zod alive with private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "boot") }},
		{"restart", "local restart and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "restart") }},
		{"pack", "offline pack exited successfully", q.pack},
		{"post-pack", "post-pack restart and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "post-pack") }},
	}
	for _, step := range steps {
		q.phase = step.name
		q.emit()
		err = ctx.Err()
		if err == nil {
			err = step.work(ctx)
		}
		detail := step.detail
		if step.name == "image" && err == nil {
			detail += "; backend ID " + q.r.ImageID
		}
		if err != nil {
			detail = err.Error()
		}
		q.r.Phases = append(q.r.Phases, Phase{step.name, err == nil, detail})
		if saveErr := q.save(); saveErr != nil {
			err = errors.Join(err, saveErr)
		}
		if err != nil {
			return q.r, err
		}
	}
	return q.r, nil
}

func (q *qualifier) emit() {
	if q.progress != nil {
		q.progress(q.phase)
	}
}

func (q *qualifier) save() error {
	data, err := json.MarshalIndent(q.r, "", "  ")
	if err != nil {
		return errors.New("encode private report failed")
	}
	f, err := os.CreateTemp(q.r.Directory, ".report-*")
	if err != nil {
		return errors.New("create private report failed")
	}
	name := f.Name()
	defer os.Remove(name) // only our staging file; never the pier
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(append(data, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err == nil {
		err = os.Rename(name, q.r.ReportPath)
	}
	if err == nil {
		var dir *os.File
		dir, err = os.Open(q.r.Directory)
		if err == nil {
			err = errors.Join(dir.Sync(), dir.Close())
		}
	}
	if err != nil {
		return errors.New("persist private report failed")
	}
	return nil
}
