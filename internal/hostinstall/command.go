package hostinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const (
	commandTimeout = 120 * time.Second
	maxOutput      = 1 << 20
)

var helpers = strings.Fields("dockerd docker rootlesskit slirp4netns newuidmap newgidmap useradd getsubids systemctl loginctl runuser id getent env nologin test iptables")
var hostHelpers = strings.Fields("sshd ssh-keygen usermod")
var packageHookHelpers = strings.Fields("stat timeout pacman-conf apt-config")

func (i *installer) helper(ctx context.Context, path string) error {
	_, err := i.resolveHelper(ctx, path)
	return err
}

func (i *installer) resolveHelper(ctx context.Context, path string) (string, error) {
	name := filepath.Base(path)
	aliases := rootless.ExecutablePaths(name)
	if !slices.Contains(aliases, path) || (!slices.Contains(helpers, name) && !slices.Contains(hostHelpers, name) && !slices.Contains(packageHookHelpers, name)) {
		return "", fmt.Errorf("command is not an approved native helper: %s", path)
	}
	for _, candidate := range aliases {
		if err := i.parents(ctx, candidate, false); err != nil {
			return "", err
		}
		_, err := i.lstat(i.path(candidate))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("inspect prerequisite %s: %w", candidate, err)
		}
		current := candidate
		for depth := 0; depth < 6; depth++ {
			if err := i.parents(ctx, current, false); err != nil {
				return "", err
			}
			info, err := i.lstat(i.path(current))
			uid, _, ok := ownership(info)
			if err != nil || !ok || uid != 0 {
				return "", fmt.Errorf("unsafe native helper %s", current)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(i.path(current))
				if err != nil {
					return "", err
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(current), target)
				}
				target = filepath.Clean(target)
				if !rootless.ExecutableLink(name, current, target) {
					return "", fmt.Errorf("unapproved native helper alias %s", current)
				}
				current = target
				continue
			}
			if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode().Perm()&0022 != 0 {
				return "", fmt.Errorf("%s must be a regular root-owned executable without group/world write", current)
			}
			// Invoke the validated alias: xtables dispatch depends on argv[0].
			return candidate, nil
		}
		return "", errors.New("native helper alias loop")
	}
	hint := "install its native repository package"
	if name == "rootlesskit" || name == "slirp4netns" {
		hint = "install native repository packages rootlesskit and slirp4netns"
	}
	return "", fmt.Errorf("missing prerequisite %s (including reviewed native aliases): %s, then rerun urbit-runner preflight", path, hint)
}

func (i *installer) command(ctx context.Context, path string, args ...string) ([]byte, error) {
	resolved, err := i.resolveHelper(ctx, path)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Do not inherit Docker overrides, proxies, loader settings, or a caller's bus.
	env := []string{"PATH=" + rootless.SafePath, "HOME=/root", "LANG=C", "LC_ALL=C", "SYSTEMD_PAGER=", "SYSTEMD_PAGERSECURE=1", "SYSTEMD_COLORS=0"}
	out, err := i.run(ctx, resolved, args, env)
	if ctx.Err() != nil {
		err = errors.Join(ctx.Err(), err)
	}
	if len(out) > maxOutput {
		return nil, fmt.Errorf("%s exceeded the 1 MiB output limit", path)
	}
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", path, strings.Join(args, " "), err)
	}
	return out, nil
}

func (i *installer) asAccount(ctx context.Context, path string, args ...string) ([]byte, error) {
	for _, helper := range []string{"/usr/bin/env", path} {
		if err := i.helper(ctx, helper); err != nil {
			return nil, err
		}
	}
	resolved, err := i.resolveHelper(ctx, path)
	if err != nil {
		return nil, err
	}
	return i.command(ctx, "/usr/bin/runuser", append([]string{
		"-u", rootless.AccountName, "--", "/usr/bin/env", "-i",
		"HOME=" + rootless.HomeDir, "PATH=" + rootless.SafePath, "DOCKER_API_VERSION=" + hostcompat.APIVersion, resolved,
	}, args...)...)
}

// A full buffer fails the command instead of silently accepting truncated JSON.
// stdout and stderr are separately bounded, with short escaped diagnostics only.
type boundedOutput struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > maxOutput-b.buffer.Len() {
		n, _ := b.buffer.Write(p[:maxOutput-b.buffer.Len()])
		b.exceeded = true
		return n, fmt.Errorf("command output exceeded 1 MiB")
	}
	return b.buffer.Write(p)
}

func runCommand(ctx context.Context, path string, args, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env, cmd.Dir = env, "/"
	// Bound descendants too (notably runuser's Docker client), without changing
	// mapping-helper privileges or attempting to stop systemd-managed services.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	var stdout, stderr boundedOutput
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.exceeded || stderr.exceeded {
		err = errors.Join(err, fmt.Errorf("command output exceeded 1 MiB"))
	}
	if err != nil {
		return nil, fmt.Errorf("%w (stdout=%s stderr=%s)", err,
			strconv.Quote(string(stdout.buffer.Bytes()[:min(stdout.buffer.Len(), 1024)])),
			strconv.Quote(string(stderr.buffer.Bytes()[:min(stderr.buffer.Len(), 1024)])))
	}
	return stdout.buffer.Bytes(), nil
}
