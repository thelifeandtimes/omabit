//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostports"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

type timings struct{ command, pull, download, ready, poll, stop, cleanup, pack, cleanupCommand time.Duration }
type dependencies struct {
	bootstrap        *protocol.Bootstrap
	fake             *ManagedFake
	storage          *pierstore.Location
	nativeShip       string
	keyDigest        string
	boot             *ManagedBoot
	amesPort         uint16
	profile          string
	managed          bool
	hostPort         uint16
	persist          func(Result) error
	home, runtimeDir string
	identity         func() (identity, error)
	stat             func(string) (node, error)
	admission        func(string) (uint64, uint64, error)
	execute          func(context.Context, []string, []string) (string, error)
	stats            func(context.Context, string, string) (string, error)
	logs             func(context.Context, string, string) (string, error)
	portAvailable    func(string, uint16) bool
	remove           func(*os.Root, string) error
	download         func(context.Context, string) (candidate.Artifact, error)
	hostProbe        func(context.Context, string) error
	hostIdentity     func(context.Context, string) (string, error)
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

// RunHTTP is the separate host-loopback publication checkpoint. The accepted
// Run lifecycle continues to use network=none and never publishes a port.
func RunHTTP(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileHTTP
	return run(ctx, progress, d)
}

// RunMeld qualifies the pinned utility on a fresh, isolated fake pier. It does
// not enable managed meld or accept an existing pier as input.
func RunMeld(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileMeld
	return run(ctx, progress, d)
}

// RunEpochs qualifies explicit offline rolls and multi-epoch restart on a fresh
// disposable fake. Its final prefix selection is a preview, never a deletion.
func RunEpochs(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileEpochs
	return run(ctx, progress, d)
}

func RunPrune(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfilePrune
	return run(ctx, progress, d)
}

func RunPrefix(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfilePrefix
	return run(ctx, progress, d)
}

func RunImport(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileImport
	return run(ctx, progress, d)
}

func RunComet(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileComet
	d.times.ready = 20 * time.Minute
	return run(ctx, progress, d)
}

func RunCometMaintenance(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileCometMaintenance
	d.times.ready = 20 * time.Minute
	return run(ctx, progress, d)
}

func RunCometEpochs(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileCometEpochs
	d.times.ready = 20 * time.Minute
	return run(ctx, progress, d)
}

func RunCometPrefix(ctx context.Context, progress func(string)) (Result, error) {
	d := defaults()
	d.profile = ProfileCometPrefix
	d.times.ready = 20 * time.Minute
	return run(ctx, progress, d)
}

func defaults() dependencies {
	return dependencies{
		profile: ProfileLifecycle, hostProbe: probeHostHTTP, hostIdentity: readHostIdentity,
		home: rootless.HomeDir, identity: currentIdentity, stat: statNode,
		admission: available, execute: executeDocker, download: candidate.DownloadPill,
		stats:         readDockerStats,
		logs:          readDockerLogs,
		portAvailable: hostports.Available,
		remove:        func(root *os.Root, name string) error { return root.Remove(name) },
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
	if cometProfile(d.profile) && d.managed {
		return Result{}, errors.New("selected disposable qualification cannot use a managed runtime context")
	}
	if d.profile != ProfileLifecycle && d.profile != ProfileHTTP && d.profile != ProfileMeld && d.profile != ProfileImport && !cometProfile(d.profile) && !isEpochProfile(d.profile) {
		return Result{}, errors.New("unknown qualification profile")
	}
	if err = ctx.Err(); err != nil {
		return
	}
	mem, disk, err := q.prevalidate()
	if err != nil {
		return Result{}, err
	}
	if d.profile == ProfileMeld || d.profile == ProfileImport || cometProfile(d.profile) || isEpochProfile(d.profile) {
		// Hold the same process-lifetime lock as OpenManagedHost. This also
		// excludes the installed 6A runner, which predates this profile. Existing
		// ship containers can continue running, but no managed expensive work
		// can be admitted concurrently with qualification.
		base := q.base
		q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
		release, lockErr := q.lockStorage()
		q.base = base
		if lockErr != nil {
			return Result{}, fmt.Errorf("qualification requires management stopped: %w", lockErr)
		}
		defer release()
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
	q.r = Result{Profile: d.profile, RunID: id, Directory: dir, ReportPath: filepath.Join(dir, "report.json"),
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
	type step struct {
		name, detail string
		work         func(context.Context) error
	}
	steps := []step{
		{"engine", "pinned private rootless engine; no leftover qualification containers", q.engine},
		{"image", "candidate image ID, platform and repository digest verified", q.image},
		{"pill", "local pill size and SHA-256 verified", q.pill},
		{"probe", "live cgroup limits and Vere version verified", q.probe},
	}
	if cometProfile(d.profile) {
		steps = append(steps,
			step{"network", "run-owned outbound NAT bridge; only loopback HTTP published", q.createNetwork},
			step{"comet-boot", "fresh comet created without supplied identity/key; public comet name verified and cleanly stopped", func(c context.Context) error { return q.cometRuntime(c, "comet-boot") }},
		)
		if d.profile == ProfileCometMaintenance {
			steps = append(steps,
				step{"comet-pack", "offline network-isolated pack; fresh admission and before/after logical bytes", func(c context.Context) error { return q.cometUtility(c, "comet-pack") }},
				step{"comet-post-pack", "same public comet identity after pack and clean stop", func(c context.Context) error { return q.cometRuntime(c, "comet-post-pack") }},
				step{"comet-meld", "offline network-isolated meld; fresh admission and before/after logical bytes", func(c context.Context) error { return q.cometUtility(c, "comet-meld") }},
				step{"comet-post-meld", "same public comet identity after meld and clean stop", func(c context.Context) error { return q.cometRuntime(c, "comet-post-meld") }},
			)
		}
		if cometEpochProfile(d.profile) {
			steps = append(steps, step{"comet-epoch-baseline", "single canonical initial comet epoch recorded", q.epochBaseline})
			for i := 1; i <= epochRollCount(d.profile); i++ {
				roll, restart := fmt.Sprintf("roll-%d", i), fmt.Sprintf("epoch-restart-%d", i)
				steps = append(steps,
					step{"comet-" + roll, "isolated offline roll created exactly one newer snapshot-bearing epoch", func(c context.Context) error { return q.roll(c, roll) }},
					step{"comet-" + restart, "same comet identity and epoch set after replacement restart and clean stop", func(c context.Context) error { return q.epochRestart(c, roll, restart) }},
				)
			}
		}
		if d.profile == ProfileCometPrefix {
			steps = append(steps, step{"comet-epoch-preview", "five-epoch preview selects the oldest two and retains three", q.epochPreview})
			for round := 1; round <= 2; round++ {
				steps = append(steps,
					step{fmt.Sprintf("comet-prefix-delete-%d", round), "exact prefix removed with durable per-file progress; retained metadata unchanged", func(c context.Context) error { return q.pruneWiderPrefix(c, round) }},
					step{fmt.Sprintf("comet-prefix-restart-%d", round), "same comet identity and retained epochs after deletion", func(c context.Context) error { return q.restartPrefix(c, round) }},
					step{fmt.Sprintf("comet-prefix-recovery-%d", round), "disposable current checkpoint removed; isolated play --full restores from latest retained epoch", func(c context.Context) error { return q.recoverPrefix(c, round) }},
					step{fmt.Sprintf("comet-prefix-recovered-%d", round), "same comet identity and retained epochs after latest-epoch recovery", func(c context.Context) error { return q.restartRecoveredPrefix(c, round) }},
				)
			}
		}
		steps = append(steps, step{"comet-resume", "same comet identity verified after initialized-pier replacement resume and clean stop", func(c context.Context) error { return q.cometRuntime(c, "comet-resume") }})
	} else if d.profile == ProfileImport {
		steps = append(steps,
			step{"boot", "new isolated fake initialized, identity verified and cleanly stopped", q.importSource},
			step{"pier-copy", "stopped source reserved; bounded private copy and content digests verified", q.importCopy},
			step{"import-resume", "copied pier resumed without bootstrap flags and cleanly stopped", func(c context.Context) error { return q.importResume(c, "import-resume") }},
			step{"import-replacement", "replacement runtime resumed the copied pier and cleanly stopped", func(c context.Context) error { return q.importResume(c, "import-replacement") }},
			step{"source-preserved", "original stopped source content unchanged after both target resumes", q.importFinish},
		)
	} else if d.profile == ProfileHTTP {
		steps = append(steps,
			step{"network", "run-owned IPv4 NAT bridge verified; normal outbound access", q.createNetwork},
			step{"host-http-boot", "fake zod verified via host-loopback HTTP; clean shutdown", func(c context.Context) error { return q.boot(c, "http-boot") }},
			step{"host-http-restart", "replacement container verified via host-loopback HTTP; clean shutdown", func(c context.Context) error { return q.boot(c, "http-restart") }},
		)
	} else if d.profile == ProfileMeld {
		steps = append(steps,
			step{"boot", "fresh isolated fake boot and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "boot") }},
			step{"meld", "offline meld exited cleanly; fresh admission and logical pier sizes recorded", q.meld},
			step{"post-meld", "same-pier replacement restart and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "post-meld") }},
		)
	} else if isEpochProfile(d.profile) {
		steps = append(steps,
			step{"boot", "fresh isolated fake boot and private HTTP reachability; clean shutdown", q.epochBoot},
			step{"epoch-baseline", "single canonical initial epoch recorded", q.epochBaseline},
		)
		for i := 1; i <= epochRollCount(d.profile); i++ {
			roll, restart := fmt.Sprintf("roll-%d", i), fmt.Sprintf("epoch-restart-%d", i)
			steps = append(steps,
				step{roll, "offline roll created exactly one newer epoch; old epoch names retained", func(c context.Context) error { return q.roll(c, roll) }},
				step{restart, "same-pier restart and private HTTP reachability; epoch set retained", func(c context.Context) error { return q.epochRestart(c, roll, restart) }},
			)
		}
		steps = append(steps, step{"epoch-preview", "qualified prefix preview retains three newer epochs; no deletion", q.epochPreview})
		if d.profile == ProfilePrune {
			steps = append(steps,
				step{"prune-prefix", "exact revalidated oldest epoch removed; three retained epoch manifests unchanged", q.prunePrefix},
				step{"post-prune", "same-pier restart after prefix deletion; retained epoch set verified", q.postPrune},
				step{"epoch-recovery", "current checkpoint removed; pinned play --full restored from latest retained epoch", q.recoverEpoch},
				step{"post-recovery", "healthy private HTTP restart after latest-epoch recovery; retained epoch set verified", q.postRecovery},
			)
		}
		if d.profile == ProfilePrefix {
			for round := 1; round <= 2; round++ {
				steps = append(steps,
					step{fmt.Sprintf("prefix-delete-%d", round), "exact selected prefix removed; retained epoch fingerprints unchanged", func(c context.Context) error { return q.pruneWiderPrefix(c, round) }},
					step{fmt.Sprintf("prefix-restart-%d", round), "ordinary post-deletion restart and retained epoch set verified", func(c context.Context) error { return q.restartPrefix(c, round) }},
					step{fmt.Sprintf("prefix-recovery-%d", round), "current checkpoint removed and restored from latest retained epoch", func(c context.Context) error { return q.recoverPrefix(c, round) }},
					step{fmt.Sprintf("prefix-recovered-%d", round), "post-recovery HTTP restart and retained epoch set verified", func(c context.Context) error { return q.restartRecoveredPrefix(c, round) }},
				)
			}
		}
	} else {
		steps = append(steps,
			step{"boot", "fake zod alive with private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "boot") }},
			step{"restart", "local restart and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "restart") }},
			step{"pack", "offline pack exited successfully", q.pack},
			step{"post-pack", "post-pack restart and private HTTP reachability; clean shutdown", func(c context.Context) error { return q.boot(c, "post-pack") }},
		)
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
	if q.d.persist != nil {
		return q.d.persist(q.r)
	}
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
