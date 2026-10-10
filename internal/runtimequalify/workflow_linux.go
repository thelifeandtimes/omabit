//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func (q *qualifier) engine(ctx context.Context) error {
	if err := q.verifyEngine(ctx); err != nil {
		return err
	}
	out, err := q.docker(ctx, q.d.times.command, "container", "ls", "--all", "--filter", "label="+label+"=true", "--format", "{{.Names}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("leftover qualification containers require manual review")
	}
	out, err = q.docker(ctx, q.d.times.command, "network", "ls", "--filter", "label="+label+"=true", "--format", "{{.Name}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("leftover qualification networks require manual review")
	}
	return nil
}

func (q *qualifier) verifyEngine(ctx context.Context) error {
	out, err := q.docker(ctx, q.d.times.command, "info", "--format", "{{json .}}")
	if err != nil {
		return err
	}
	var info struct {
		ServerVersion, CgroupDriver, CgroupVersion, DockerRootDir string
		SecurityOptions                                           []string
	}
	if json.Unmarshal([]byte(out), &info) != nil {
		return errors.New("invalid engine information")
	}
	rootlessOption := false
	for _, s := range info.SecurityOptions {
		if s == "name=rootless" {
			rootlessOption = true
		}
	}
	if !rootlessOption || info.ServerVersion != rootless.EngineVersion || info.CgroupDriver != "systemd" || info.CgroupVersion != "2" || info.DockerRootDir != filepath.Join(q.d.home, ".local/share/urbit-runner/docker") {
		return errors.New("engine does not match the pinned private rootless configuration")
	}
	return nil
}

func (q *qualifier) image(ctx context.Context) error {
	if _, err := q.docker(ctx, q.d.times.pull, "image", "pull", "--platform", candidate.Platform, candidate.Image); err != nil {
		return err
	}
	return q.inspectImage(ctx)
}

var errImageUnavailable = errors.New("candidate image inspection unavailable")

func (q *qualifier) inspectImage(ctx context.Context) error {
	out, err := q.docker(ctx, q.d.times.command, "image", "inspect", candidate.Image)
	if err != nil {
		return errors.Join(errImageUnavailable, err)
	}
	var images []struct {
		ID               string `json:"Id"`
		Os, Architecture string
		RepoDigests      []string
	}
	if json.Unmarshal([]byte(out), &images) != nil || len(images) != 1 {
		return errors.New("invalid candidate image inspection")
	}
	i := images[0]
	knownDigest := ""
	for _, digest := range i.RepoDigests {
		if digest == candidate.Image || digest == strings.TrimPrefix(candidate.Image, "docker.io/") {
			knownDigest = digest
		}
	}
	if i.Os != "linux" || i.Architecture != "amd64" || knownDigest == "" || !acceptedImageID(i.ID) {
		return errors.New("candidate image identity, platform or repository digest mismatch")
	}
	// Docker's legacy backend reports the config digest; containerd reports
	// the manifest target digest. Neither is accepted without the pins above.
	q.r.ImageID, q.r.RepoDigest = i.ID, knownDigest
	return nil
}

func (q *qualifier) managedImage(ctx context.Context) error {
	err := q.inspectImage(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, errImageUnavailable) || ctx.Err() != nil {
		return err
	}
	return q.image(ctx)
}

func acceptedImageID(id string) bool {
	return id == candidate.ConfigDigest || id == candidate.ManifestDigest
}

func (q *qualifier) pill(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, q.d.times.download)
	defer cancel()
	if _, err := os.Lstat(q.r.PillPath); !errors.Is(err, os.ErrNotExist) {
		return errors.New("pill destination must be fresh")
	}
	if q.selectedPill() != nil {
		return q.copySelectedPill(ctx)
	}
	a, err := q.d.download(ctx, q.r.PillPath)
	if err != nil || ctx.Err() != nil {
		return errors.New("verified candidate pill download failed")
	}
	if a.Path != q.r.PillPath || a.SHA256 != q.d.pillSHA || a.Bytes != q.d.pillBytes || a.URL != candidate.PillURL {
		return errors.New("candidate pill provenance mismatch")
	}
	return q.verifyLocalPill(ctx)
}

func (q *qualifier) verifyLocalPill(ctx context.Context) error {
	if err := q.owned(q.r.PillPath, 0, true); err != nil {
		return err
	}
	n, err := q.d.stat(q.r.PillPath)
	if err != nil || n.links != 1 {
		return errors.New("candidate pill must be an exclusive regular file")
	}
	f, err := os.Open(q.r.PillPath)
	if err != nil {
		return errors.New("cannot verify local pill")
	}
	defer f.Close()
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(f, q.d.pillBytes+1))
	if err != nil || ctx.Err() != nil || count != q.d.pillBytes || hex.EncodeToString(hash.Sum(nil)) != q.d.pillSHA {
		return errors.New("local pill size or SHA-256 mismatch")
	}
	return nil
}

func (q *qualifier) probe(ctx context.Context) error {
	i, err := q.create(ctx, "probe")
	if err != nil {
		return err
	}
	if _, err := q.control(ctx, i, q.d.times.command, "start"); err != nil {
		return err
	}
	out, err := q.control(ctx, i, q.d.times.command, "exec", "/bin/cat", "/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.swap.max", "/sys/fs/cgroup/pids.max", "/sys/fs/cgroup/cpu.max")
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 4 || lines[0] != "4294967296" || lines[1] != "0" || lines[2] != "256" {
		return errors.New("live memory, swap or PID limit mismatch")
	}
	cpu := strings.Fields(lines[3])
	if len(cpu) != 2 {
		return errors.New("invalid live CPU limit")
	}
	quota, e1 := strconv.ParseUint(cpu[0], 10, 64)
	period, e2 := strconv.ParseUint(cpu[1], 10, 64)
	if e1 != nil || e2 != nil || period == 0 || period > (^uint64(0))/2 || quota != 2*period {
		return errors.New("live CPU quota must be finite and exactly two CPUs")
	}
	q.r.LimitsObserved = true
	out, err = q.control(ctx, i, q.d.times.command, "exec", "/bin/urbit", "-R")
	if err != nil {
		return err
	}
	if strings.SplitN(out, "\n", 2)[0] != "urbit "+candidate.VereVersion {
		return errors.New("live Vere version mismatch")
	}
	return q.stop(ctx, i, false)
}

func (q *qualifier) pier() error {
	if q.d.managed {
		if _, err := pierstore.CheckMount(q.d.home, *q.d.storage); err != nil {
			return err
		}
		if q.d.storage.Identity.Inode == 0 {
			identity, err := pierstore.Identify(q.pierPath())
			if err != nil {
				return err
			}
			q.d.storage.Identity = identity
		}
		if _, err := pierstore.Check(q.d.home, *q.d.storage); err != nil {
			return err
		}
		q.r.PierMetadataSeen = true
		return nil
	}
	if cometProfile(q.d.profile) {
		return q.cometPier()
	}
	data := filepath.Join(q.r.Directory, "data")
	pierName := q.epochPierName()
	entries, err := os.ReadDir(data)
	if err != nil || len(entries) != 1 || entries[0].Name() != pierName {
		return errors.New("data mount must contain only the registered fake pier directory")
	}
	for _, path := range []string{data, filepath.Join(data, pierName), filepath.Join(data, pierName, ".urb")} {
		if err := q.owned(path, os.ModeDir, false); err != nil {
			return errors.New("missing or unsafe persisted pier metadata")
		}
	}
	q.r.PierMetadataSeen = true
	return nil
}

func (q *qualifier) boot(ctx context.Context, kind string) error {
	if !q.r.LimitsObserved {
		return errors.New("live cgroup verification is required before Vere boot")
	}
	if kind == "boot" || kind == "http-boot" {
		entries, err := os.ReadDir(filepath.Join(q.r.Directory, "data"))
		if err != nil || len(entries) != 0 {
			return errors.New("fake-zod data must be absent before first boot")
		}
	} else if err := q.pier(); err != nil {
		return err
	}
	i, err := q.create(ctx, kind)
	if err != nil {
		return err
	}
	if _, err := q.control(ctx, i, q.d.times.command, "start"); err != nil {
		return err
	}
	ready := q.ready
	if q.d.profile == ProfileImport {
		ready = q.readyImport
	}
	if isHTTPKind(kind) {
		ready = q.readyHostHTTP
	}
	if err := ready(ctx, i); err != nil {
		return err
	}
	if err := q.stop(ctx, i, true); err != nil {
		return err
	}
	return q.pier()
}

func (q *qualifier) ready(ctx context.Context, index int) error {
	ctx, cancel := context.WithTimeout(ctx, q.d.times.ready)
	defer cancel()
	for {
		if ctx.Err() != nil {
			return errors.New("private HTTP readiness deadline or cancellation")
		}
		c, err := q.inspect(ctx, index)
		if err != nil {
			return err
		}
		if !*c.State.Running || c.State.Paused || c.State.Restarting || *c.State.OOMKilled || c.State.Dead {
			return errors.New("Vere exited or became unhealthy before HTTP readiness")
		}
		q.r.Containers[index].AliveObserved = true
		// Eyre's public host endpoint is independent of a particular app's root
		// page. Discard the body and headers; this is transport reachability only.
		out, curlErr := q.control(ctx, index, q.d.times.command, "exec", "/usr/bin/curl", "--noproxy", "*", "--silent", "--show-error", "--max-time", "5", "--output", "/dev/null", "--write-out", "%{http_code}", "http://127.0.0.1:8080/~/host")
		if curlErr == nil && len(out) == 3 && out[0] >= '2' && out[0] <= '3' && out[1] >= '0' && out[1] <= '9' && out[2] >= '0' && out[2] <= '9' {
			c, err = q.inspect(ctx, index)
			if err != nil {
				return err
			}
			if !*c.State.Running || *c.State.OOMKilled || c.State.Paused || c.State.Restarting || c.State.Dead {
				return errors.New("Vere exited during private HTTP check")
			}
			q.r.Containers[index].PrivateHTTPReachable = true
			return q.save() // reachability ONLY; no response body, headers or login
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.New("private HTTP readiness deadline or cancellation")
		case <-timer.C:
		}
	}
}

func (q *qualifier) pack(ctx context.Context) error {
	return q.offlineUtility(ctx, "pack")
}

func (q *qualifier) offlineUtility(ctx context.Context, kind string) error {
	cometUtility := isCometUtility(kind) && !q.d.managed && q.d.profile == ProfileCometMaintenance || isManagedCometUtility(kind) && q.d.managed && q.d.profile == ProfileComet
	cometUtility = cometUtility || !q.d.managed && cometEpochContainer(q.d.profile, kind) && isCometRoll(kind) || !q.d.managed && q.d.profile == ProfileCometPrefix && isCometPrefixRecovery(kind)
	if kind != "pack" && kind != "meld" && !(kind == "roll" && q.d.managed) && !(isRollKind(kind) && !q.d.managed && isEpochProfile(q.d.profile)) && !(kind == "epoch-recovery" && !q.d.managed && q.d.profile == ProfilePrune) && !(isPrefixRecovery(kind) && !q.d.managed && q.d.profile == ProfilePrefix) && !cometUtility {
		return errors.New("unsupported offline utility")
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return fmt.Errorf("offline %s requires all preceding containers removed with resolved outcomes", kind)
		}
	}
	if err := q.pier(); err != nil {
		return err
	}
	i, err := q.create(ctx, kind)
	if err != nil {
		return err
	}
	if _, err := q.control(ctx, i, q.d.times.command, "start"); err != nil {
		return err
	}
	out, err := q.control(ctx, i, q.d.times.pack, "wait")
	if err != nil {
		return fmt.Errorf("offline %s interrupted or failed; retain for review without retry", kind)
	}
	c, err := q.inspect(ctx, i)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "0" || active(c) || c.State.Dead || *c.State.ExitCode != 0 || *c.State.OOMKilled {
		return fmt.Errorf("offline %s did not exit cleanly", kind)
	}
	return q.remove(ctx, i)
}
