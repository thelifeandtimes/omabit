//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

var ErrRecordingSubmissionUncertain = errors.New("recording worker submission has no verified acknowledgement; retain the exact job and inspect its named unit without resubmission")

// recordingWorkerArguments builds only the fixed detached notify-service
// command. Its inputs come from authenticated local configuration/job admission;
// no RPC exposes arbitrary command/property/environment parameters. Submission
// does not establish readiness or permit Start, and normal launch is unwired.
func recordingWorkerArguments(home, development string, job ManagedRecordingJob, image RecordingExecutable) ([]string, error) {
	unit, err := RecordingWorkerUnitName(job)
	if err != nil || !recordingJobPath(home) || home == "/" || strings.ContainsAny(home, "%\"'\\") || job.State.Runtime.Directory != filepath.Join(home, ".local/share/urbit-runner/managed", job.State.ID) || image.SHA256 != job.ExecutableSHA256 || image.Path != filepath.Join(home, ".local/share/urbit-runner/recording-images", job.ExecutableSHA256+".runner") {
		return nil, ErrRecordingWorker
	}
	if development != "" && (!regexp.MustCompile(`^/srv/dev/urbit(?:-[a-z0-9][a-z0-9-]{0,31})?$`).MatchString(development) || home != filepath.Join(development, "state")) {
		return nil, ErrRecordingWorker
	}
	raw, err := job.Bytes()
	if err != nil {
		return nil, ErrRecordingWorker
	}
	args := []string{"--user", "--quiet", "--no-ask-password", "--expand-environment=no", "--unit=" + unit, "--description=Omabit runtime recorder",
		"--property=Type=notify", "--property=NotifyAccess=main", "--property=Restart=no", "--property=RemainAfterExit=no",
		"--property=TimeoutStartSec=60", "--property=TimeoutStopSec=15", "--property=RuntimeMaxSec=infinity",
		"--property=KillMode=control-group", "--property=MemoryMax=256M", "--property=MemorySwapMax=0", "--property=CPUQuota=50%", "--property=TasksMax=64",
		"--property=UMask=0077", "--property=NoNewPrivileges=yes", "--property=StandardInput=null", "--property=StandardOutput=null", "--property=StandardError=null",
		"--working-directory=" + home, "--setenv=HOME=" + home, "--setenv=PATH=/usr/bin:/bin", "--setenv=LC_ALL=C", "--", image.Path}
	if development != "" {
		args = append(args, "--development-root", development)
	}
	return append(args, "recording-worker", "--job", job.ID, "--sha256", recordingJobSHA(raw)), nil
}

// submitRecordingWorkerCommand is a single submission primitive, not the launch
// coordinator. The caller must first commit its exact durable operation/job
// binding and must independently check synced readiness, live unit/process and
// native/journal fences afterward. A lost/canceled reply never triggers retry,
// Stop or replay. The manager owns the service lifetime after submission.
func submitRecordingWorkerCommand(ctx context.Context, job ManagedRecordingJob) error {
	q := &qualifier{d: defaults()}
	if ctx.Err() != nil || q.accountBoundary(false) != nil || q.ancestors("/usr/bin/systemd-run") != nil {
		return ErrRecordingWorker
	}
	n, err := q.d.stat("/usr/bin/systemd-run")
	if err != nil || !n.mode.IsRegular() || n.uid != 0 || n.mode.Perm()&0022 != 0 || n.mode.Perm()&0111 == 0 || n.mode&(os.ModeSetuid|os.ModeSetgid) != 0 || job.Validate() != nil {
		return ErrRecordingWorker
	}
	ctx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	imageRoot := filepath.Join(q.d.home, ".local/share/urbit-runner/recording-images")
	if q.ancestors(imageRoot, true) != nil || q.owned(imageRoot, os.ModeDir, true) != nil {
		return ErrRecordingWorker
	}
	root, err := openRecordingPrivateRoot(imageRoot)
	if err != nil {
		return ErrRecordingWorker
	}
	defer root.Close()
	name := job.ExecutableSHA256 + ".runner"
	info, err := root.Lstat(name)
	if err != nil || !recordingImageInfo(info, info.Size()) || verifyRecordingImage(ctx, root, name, job.ExecutableSHA256, info.Size()) != nil {
		return ErrRecordingWorker
	}
	image := RecordingExecutable{Path: filepath.Join(imageRoot, name), SHA256: job.ExecutableSHA256}
	args, err := recordingWorkerArguments(q.d.home, ownerhost.DevelopmentRoot(), job, image)
	if err != nil {
		return ErrRecordingWorker
	}
	runtime := filepath.Join("/run/user", strconv.Itoa(os.Geteuid()))
	cmd := exec.CommandContext(ctx, "/usr/bin/systemd-run", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C", "XDG_RUNTIME_DIR=" + runtime, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtime + "/bus"}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	cmd.WaitDelay = time.Second
	if cmd.Run() != nil || ctx.Err() != nil {
		return ErrRecordingSubmissionUncertain
	}
	return nil
}
