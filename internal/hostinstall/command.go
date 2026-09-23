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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const (
	commandTimeout = 120 * time.Second
	maxOutput      = 1 << 20
)

var helpers = strings.Fields("dockerd docker rootlesskit slirp4netns newuidmap newgidmap useradd getsubids systemctl loginctl runuser id getent env nologin test")

func (i *installer) helper(ctx context.Context, path string) error {
	if filepath.Dir(path) != "/usr/bin" || !slices.Contains(helpers, filepath.Base(path)) {
		return fmt.Errorf("command is not an approved /usr/bin helper: %s", path)
	}
	if err := i.parents(ctx, path, false); err != nil {
		return err
	}
	info, err := i.lstat(i.path(path))
	if errors.Is(err, os.ErrNotExist) {
		hint := "install its repository package, then rerun urbit-runner preflight"
		if path == "/usr/bin/rootlesskit" || path == "/usr/bin/slirp4netns" {
			hint = "install repository packages rootlesskit and slirp4netns, then rerun urbit-runner preflight"
		}
		return fmt.Errorf("missing prerequisite %s: %s", path, hint)
	}
	if err != nil {
		return fmt.Errorf("inspect prerequisite %s: %w", path, err)
	}
	uid, _, ok := ownership(info)
	if !ok || !info.Mode().IsRegular() || uid != 0 || info.Mode().Perm()&0111 == 0 || info.Mode().Perm()&0022 != 0 {
		return fmt.Errorf("%s must already be a regular root-owned executable, not a symlink or group/world writable", path)
	}
	return nil
}

func (i *installer) command(ctx context.Context, path string, args ...string) ([]byte, error) {
	if err := i.helper(ctx, path); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Do not inherit Docker overrides, proxies, loader settings, or a caller's bus.
	env := []string{"PATH=/usr/bin:/bin", "HOME=/root", "LANG=C", "LC_ALL=C", "SYSTEMD_PAGER=", "SYSTEMD_PAGERSECURE=1", "SYSTEMD_COLORS=0"}
	out, err := i.run(ctx, path, args, env)
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
	return i.command(ctx, "/usr/bin/runuser", append([]string{
		"-u", rootless.AccountName, "--", "/usr/bin/env", "-i",
		"HOME=" + rootless.HomeDir, "PATH=/usr/bin:/bin", path,
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
