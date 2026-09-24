//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const (
	label       = "org.omabit.qualification"
	runLabel    = label + ".run"
	outputLimit = 1 << 20
	memoryLimit = int64(4 << 30)
)

var (
	runIDPattern          = regexp.MustCompile(`^[0-9a-f]{32}$`)
	containerIDPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	errUntrustedContainer = errors.New("container ownership, identity or inspection mismatch")
)

func (q *qualifier) resourceLabel() string {
	if q.d.managed {
		return "org.urbit-runner.managed-fake"
	}
	return label
}

func (q *qualifier) resourceRunLabel() string {
	return q.resourceLabel() + ".run"
}

func (q *qualifier) resourceName(kind string) string {
	prefix := "urbit-qualification-"
	if q.d.managed {
		prefix = "urbit-managed-fake-"
	}
	return prefix + q.r.RunID + "-" + kind
}

type limitedOutput struct {
	buffer bytes.Buffer
	cancel context.CancelFunc
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(p) > outputLimit-b.buffer.Len() {
		b.cancel()
		return 0, errors.New("output limit exceeded")
	}
	return b.buffer.Write(p)
}

func executeDocker(ctx context.Context, args, env []string) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", args...)
	cmd.Env = env
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	stdout, stderr := &limitedOutput{cancel: cancel}, &limitedOutput{cancel: cancel}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if cmd.Process != nil && (ctx.Err() != nil || errors.Is(err, exec.ErrWaitDelay)) {
		// WaitDelay can fire after the client exits but a descendant retains a
		// pipe. Reap the entire client process group in that case as well.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return stdout.buffer.String(), err // stderr is deliberately never returned
}

func (q *qualifier) docker(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := q.boundary(); err != nil {
		return "", err
	}
	for _, p := range append(q.storagePaths(), q.r.Directory, q.config) {
		if err := q.owned(p, os.ModeDir, p == q.r.Directory || p == q.config); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args = append([]string{"--host", "unix://" + q.socket, "--config", q.config}, args...)
	out, err := q.d.execute(ctx, args, []string{"HOME=" + q.d.home, "PATH=/usr/bin:/bin", "LANG=C"})
	if err != nil || ctx.Err() != nil || len(out) > outputLimit {
		status := "client failure"
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			status = fmt.Sprintf("exit %d", exit.ExitCode())
		}
		if ctx.Err() != nil {
			status = "deadline or cancellation"
		}
		if len(out) > outputLimit {
			status = "output limit exceeded"
		}
		return "", fmt.Errorf("%s: docker command failed (%s)", q.phase, status)
	}
	return out, nil
}

type mount struct {
	Type, Source, Destination string
	RW                        bool
	Propagation               string
}
type container struct {
	ID          string `json:"Id"`
	Name, Image string
	Config      struct {
		Image, User, StopSignal string
		StopTimeout             int
		Entrypoint, Cmd         []string
		Labels                  map[string]string
		Tty, OpenStdin          bool
	}
	HostConfig struct {
		Init                                                             *bool
		NetworkMode, PidMode, IpcMode, UTSMode, UsernsMode, CgroupnsMode string
		Privileged, PublishAllPorts, AutoRemove                          bool
		CapAdd, CapDrop, SecurityOpt, Binds, VolumesFrom, Links          []string
		Devices, DeviceRequests                                          []json.RawMessage
		PortBindings                                                     map[string]json.RawMessage
		Memory, MemorySwap, NanoCpus, CpuQuota, CpuPeriod                int64
		PidsLimit                                                        *int64
		RestartPolicy                                                    struct {
			Name              string
			MaximumRetryCount int
		}
		LogConfig struct {
			Type   string
			Config map[string]string
		}
	}
	Mounts          []mount
	NetworkSettings struct {
		Networks map[string]json.RawMessage
		Ports    map[string]json.RawMessage
	}
	State *struct {
		Running, OOMKilled       *bool
		ExitCode                 *int
		Paused, Restarting, Dead bool
	}
}

func (q *qualifier) inspect(ctx context.Context, index int) (container, error) {
	r := &q.r.Containers[index]
	out, err := q.docker(ctx, q.d.times.command, "container", "inspect", r.Name)
	if err != nil {
		return container{}, err
	}
	var list []container
	if json.Unmarshal([]byte(out), &list) != nil || len(list) != 1 {
		return container{}, errUntrustedContainer
	}
	c := list[0]
	if c.State == nil || c.State.Running == nil || c.State.OOMKilled == nil || c.State.ExitCode == nil {
		return c, errUntrustedContainer
	}
	if !acceptedImageID(q.r.ImageID) || !containerIDPattern.MatchString(c.ID) || c.Name != "/"+r.Name || c.Image != q.r.ImageID || c.Config.Image != q.r.ImageID || c.Config.Labels[q.resourceLabel()] != "true" || c.Config.Labels[q.resourceRunLabel()] != q.r.RunID || (r.ID != "" && r.ID != c.ID) {
		return c, errUntrustedContainer
	}
	r.ID = c.ID
	r.CreateUncertain = false
	return c, nil
}

func command(kind string) (string, []string) {
	switch kind {
	case "probe":
		return "/bin/sleep", []string{"120"}
	case "pack":
		return "/bin/urbit", []string{"pack", "--loom", "31", "/urbit/fake-zod"}
	default:
		bind := "127.0.0.1"
		if isHTTPKind(kind) {
			bind = "0.0.0.0" // inside the dedicated bridge, never a host binding
		}
		args := []string{"-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "-b", bind, "--http-port", "8080"}
		if kind == "boot" || kind == "http-boot" {
			args = append(args, "-F", "zod", "-B", "/artifacts/boot.pill", "-c")
		}
		return "/bin/urbit", append(args, "/urbit/fake-zod")
	}
}

func (q *qualifier) containment(c container, kind string) error {
	h := c.HostConfig
	entry, args := command(kind)
	ok := c.Config.User == "0:0" && c.Config.StopSignal == "SIGTERM" && c.Config.StopTimeout == 300 && !c.Config.Tty && !c.Config.OpenStdin && reflect.DeepEqual(c.Config.Entrypoint, []string{entry}) && reflect.DeepEqual(c.Config.Cmd, args) &&
		h.PidMode == "" && h.IpcMode == "private" && h.UTSMode == "" && h.UsernsMode == "" && h.CgroupnsMode == "private" && !h.Privileged && !h.PublishAllPorts && !h.AutoRemove &&
		len(h.CapAdd) == 0 && reflect.DeepEqual(h.CapDrop, []string{"ALL"}) && reflect.DeepEqual(h.SecurityOpt, []string{"no-new-privileges=true"}) && len(h.Binds) == 0 && len(h.VolumesFrom) == 0 && len(h.Links) == 0 && len(h.Devices) == 0 && len(h.DeviceRequests) == 0 &&
		h.Memory == memoryLimit && h.MemorySwap == memoryLimit && h.PidsLimit != nil && *h.PidsLimit == 256 && h.NanoCpus == 2_000_000_000 && h.CpuQuota == 0 && h.CpuPeriod == 0 &&
		h.RestartPolicy.Name == "no" && h.RestartPolicy.MaximumRetryCount == 0 && h.LogConfig.Type == "local" && reflect.DeepEqual(h.LogConfig.Config, map[string]string{"max-size": "10m", "max-file": "2"})
	if kind == "probe" {
		// sleep does not install a SIGTERM handler. An init must forward the
		// signal to a non-PID-1 child, or stopping waits for the sleep deadline.
		ok = ok && h.Init != nil && *h.Init
	} else {
		// Vere keeps its already-qualified direct signal-handling topology.
		ok = ok && (h.Init == nil || !*h.Init)
	}
	if err := q.networkContainment(c, kind); err != nil {
		return err
	}
	want := []mount{{"bind", filepath.Join(q.r.Directory, "data"), "/urbit", true, "rprivate"}}
	if kind == "boot" || kind == "http-boot" {
		want = append(want, mount{"bind", q.r.PillPath, "/artifacts/boot.pill", false, "rprivate"})
	}
	if len(c.Mounts) != len(want) {
		ok = false
	}
	for _, w := range want {
		found := false
		for _, m := range c.Mounts {
			if m == w {
				found = true
			}
		}
		if !found {
			ok = false
		}
	}
	if !ok {
		return errors.New("container containment inspection failed")
	}
	return nil
}

func (q *qualifier) create(ctx context.Context, kind string) (int, error) {
	index := len(q.r.Containers)
	allowed := kind == "probe" ||
		(q.d.profile == ProfileLifecycle && (kind == "boot" || kind == "restart" || kind == "pack" || kind == "post-pack")) ||
		(q.d.profile == ProfileHTTP && isHTTPKind(kind))
	if !allowed {
		return index, errors.New("container kind is not allowed for this qualification profile")
	}
	network := "none"
	if isHTTPKind(kind) {
		n, err := q.inspectNetwork(ctx)
		if err != nil {
			return index, err
		}
		network = n.ID
	}
	if !acceptedImageID(q.r.ImageID) || q.r.RepoDigest == "" {
		return index, errors.New("verified backend image identity required before create")
	}
	name := q.resourceName(kind)
	q.r.Containers = append(q.r.Containers, ContainerRecord{Name: name, Kind: kind})
	if err := q.save(); err != nil {
		return index, err
	}
	entry, cmd := command(kind)
	args := []string{"container", "create", "--name", name, "--label", q.resourceLabel() + "=true", "--label", q.resourceRunLabel() + "=" + q.r.RunID,
		"--platform", candidate.Platform, "--pull", "never", "--entrypoint", entry,
		"--network", network, "--user", "0:0", "--cap-drop", "ALL", "--security-opt", "no-new-privileges=true",
		"--cgroupns", "private", "--ipc", "private", "--restart", "no", "--stop-signal", "SIGTERM", "--stop-timeout", "300",
		"--memory", "4294967296", "--memory-swap", "4294967296", "--pids-limit", "256", "--cpus", "2",
		"--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=2",
		"--mount", "type=bind,src=" + filepath.Join(q.r.Directory, "data") + ",dst=/urbit,bind-propagation=rprivate"}
	if kind == "probe" {
		args = append(args, "--init")
	}
	if isHTTPKind(kind) {
		port := ""
		if q.d.hostPort != 0 {
			port = fmt.Sprint(q.d.hostPort)
		}
		args = append(args, "--publish", "127.0.0.1:"+port+":8080/tcp")
	}
	if kind == "boot" || kind == "http-boot" {
		args = append(args, "--mount", "type=bind,src="+q.r.PillPath+",dst=/artifacts/boot.pill,readonly,bind-propagation=rprivate")
	}
	args = append(append(args, q.r.ImageID), cmd...)
	if err := ctx.Err(); err != nil {
		return index, err
	}
	// Persist a conservative dispatch marker BEFORE the client can send a
	// request. Crashes or client errors cannot turn an empty listing into proof
	// that the daemon will never finish a create accepted in the background.
	q.r.Containers[index].CreateDispatched = true
	q.r.Containers[index].CreateUncertain = true
	if err := q.save(); err != nil {
		q.r.Containers[index].CreateDispatched = false
		q.r.Containers[index].CreateUncertain = false
		return index, err
	}
	out, err := q.docker(ctx, q.d.times.command, args...)
	if id := strings.TrimSpace(out); containerIDPattern.MatchString(id) {
		q.r.Containers[index].ID = id
	}
	if saveErr := q.save(); saveErr != nil {
		err = errors.Join(err, saveErr)
	}
	if err != nil {
		return index, err
	}
	c, err := q.inspect(ctx, index)
	if err == nil {
		err = q.containment(c, kind)
	}
	if saveErr := q.save(); saveErr != nil {
		err = errors.Join(err, saveErr)
	}
	return index, err
}

func active(c container) bool { return *c.State.Running || c.State.Paused || c.State.Restarting }

// Every control operation first authenticates the exact intended name, full ID,
// labels and image. Control uses the full ID, never a partial ID or broad selector.
func (q *qualifier) control(ctx context.Context, index int, timeout time.Duration, verb string, args ...string) (string, error) {
	c, err := q.inspect(ctx, index)
	if err != nil {
		return "", err
	}
	if verb == "start" || verb == "exec" || verb == "wait" {
		if isHTTPKind(q.r.Containers[index].Kind) {
			if _, err := q.inspectNetwork(ctx); err != nil {
				return "", err
			}
		}
		if err := q.containment(c, q.r.Containers[index].Kind); err != nil {
			return "", err
		}
	}
	if verb == "exec" && !active(c) {
		return "", errors.New("container exited before execution")
	}
	command := []string{"container", verb}
	if verb == "stop" {
		command = append(command, "--timeout", "300")
	}
	command = append(command, c.ID)
	return q.docker(ctx, timeout, append(command, args...)...)
}

func (q *qualifier) remove(ctx context.Context, index int) error {
	c, err := q.inspect(ctx, index)
	if err != nil {
		return err
	}
	if active(c) || c.State.Dead {
		return errors.New("cannot confirm container stopped")
	}
	if _, err := q.docker(ctx, q.d.times.command, "container", "rm", c.ID); err != nil {
		return err
	}
	q.r.Containers[index].Removed = true
	return q.save()
}

func (q *qualifier) stop(ctx context.Context, index int, requireZero bool) error {
	if _, err := q.control(ctx, index, q.d.times.stop, "stop"); err != nil {
		return err
	}
	c, err := q.inspect(ctx, index)
	if err != nil {
		return err
	}
	if active(c) || c.State.Dead || *c.State.OOMKilled || (requireZero && *c.State.ExitCode != 0) {
		return errors.New("graceful shutdown failed: running, nonzero exit or OOM")
	}
	return q.remove(ctx, index)
}

func (q *qualifier) cleanup() error {
	var errs []error
	for i := range q.r.Containers {
		if q.r.Containers[i].Removed {
			continue
		}
		// Independent bounded context for EACH intended container. Killing a
		// Docker client does not stop its daemon-side job.
		ctx, cancel := context.WithTimeout(context.Background(), q.d.times.cleanup)
		err := q.cleanupOne(ctx, i)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("cleanup incomplete for %s: %w", q.r.Containers[i].Name, err))
		}
	}
	if err := q.cleanupNetwork(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (q *qualifier) cleanupOne(ctx context.Context, index int) error {
	r := &q.r.Containers[index]
	if !r.CreateDispatched {
		r.Removed = true // proven locally: no create could have been sent
		return q.save()
	}
	var lastErr, reportErr error
	deadline, _ := ctx.Deadline()
	for ctx.Err() == nil {
		uncertain, oldID := r.CreateUncertain, r.ID
		// A start accepted by Docker can hold its container lock after the
		// client is canceled. Short inspection attempts allow recovery without
		// spending the entire graceful-stop budget on one blocked inspection.
		attempt, cancel := context.WithTimeout(ctx, q.d.times.cleanupCommand)
		c, err := q.inspect(attempt, index)
		cancel()
		if errors.Is(err, errUntrustedContainer) {
			return errors.Join(reportErr, err) // terminal; never control a mismatch
		}
		if err != nil {
			out, listErr := q.docker(ctx, q.d.times.cleanupCommand, "container", "ls", "--all", "--filter", "label="+q.resourceLabel()+"=true", "--filter", "label="+q.resourceRunLabel()+"="+q.r.RunID, "--filter", "name=^/"+r.Name+"$", "--format", "{{.Names}}")
			lastErr = errors.Join(err, listErr)
			if listErr == nil && strings.TrimSpace(out) == "" {
				// Absence also cannot distinguish a lost removal response from
				// a temporarily invisible or relabelled container. Require owned
				// inspection and an acknowledged removal, or retain the intent.
				lastErr = errors.New("empty inventory cannot resolve the pending container outcome")
			}
		} else {
			if uncertain || oldID != r.ID {
				reportErr = errors.Join(reportErr, q.save())
			}
			if active(c) {
				if time.Until(deadline) < q.d.times.stop {
					err = errors.New("insufficient remaining grace-stop budget")
				} else {
					// c was authenticated immediately above. Every retry gets new
					// identity evidence; the stop is always by that full ID.
					_, err = q.docker(ctx, q.d.times.stop, "container", "stop", "--timeout", "300", c.ID)
				}
			}
			if err == nil {
				attempt, cancel = context.WithTimeout(ctx, q.d.times.cleanupCommand)
				err = q.remove(attempt, index) // reinspects; never removes live jobs
				cancel()
			}
			if err == nil || r.Removed || errors.Is(err, errUntrustedContainer) {
				return errors.Join(reportErr, err)
			}
			lastErr = err
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return errors.Join(reportErr, lastErr, errors.New("cleanup deadline exceeded; manual resolution required"))
}
