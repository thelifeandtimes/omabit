//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const vmServiceToken = "offline-project-service-restart-v1"
const vmServiceDirectory = "/home/lab/.local/share/urbit-runner/managed"

func TestProjectServiceRestartDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, true, true, true, true, false, false, true)
}

// Every service helper refuses before subprocess effects, even when called alone.
func vmServiceAdmission() error {
	if os.Getenv("OMABIT_TEST_PROJECT_VM") != vmServiceToken || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/lab" || os.Getenv("XDG_RUNTIME_DIR") != "/run/user/1000" {
		return errors.New("owner service test requires disposable guest")
	}
	for p, want := range map[string]string{"/opt/omabit-lab/guest-marker": "offline-import-guest-v1\n", "/sys/class/dmi/id/sys_vendor": "QEMU\n", "/proc/1/comm": "systemd\n", "/etc/hostname": "omabit-disposable-import\n"} {
		b, e := os.ReadFile(p)
		if e != nil || string(b) != want {
			return errors.New("owner service guest marker refused")
		}
	}
	b, e := os.ReadFile("/proc/cmdline")
	if e != nil || !strings.Contains(string(b), "omabit.disposable-guest=offline-import-v1") {
		return errors.New("owner service boot marker refused")
	}
	devices, e := os.ReadDir("/sys/class/net")
	if e != nil || len(devices) != 1 || devices[0].Name() != "lo" {
		return errors.New("owner service external network refused")
	}
	return nil
}
func vmServiceCommand(ctx context.Context, executable string, args ...string) ([]byte, error) {
	if e := vmServiceAdmission(); e != nil {
		return nil, e
	}
	deadline, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	c := exec.CommandContext(deadline, executable, args...)
	c.Env = []string{"HOME=/home/lab", "PATH=/usr/bin:/bin", "XDG_RUNTIME_DIR=/run/user/1000", "LC_ALL=C"}
	var out, diagnostic vmRecoveryOutput
	c.Stdout = &out
	c.Stderr = &diagnostic
	if e := c.Run(); e != nil {
		return nil, fmt.Errorf("guest service command %s: %w: %.512s", filepath.Base(executable), e, diagnostic.String())
	}
	return out.Bytes(), nil
}
func vmOwnerService(ctx context.Context, action string) (map[string]string, error) {
	if action != "show" && action != "start" && action != "restart" && action != "stop" {
		return nil, errors.New("unsupported guest service action")
	}
	if action != "show" {
		if _, e := vmServiceCommand(ctx, "/usr/bin/systemctl", "--user", action, "urbit-runner.service"); e != nil {
			return nil, e
		}
	}
	b, e := vmServiceCommand(ctx, "/usr/bin/systemctl", "--user", "show", "urbit-runner.service", "--property=ActiveState,SubState,MainPID,InvocationID,ExecMainStatus,FragmentPath")
	if e != nil {
		return nil, e
	}
	m := map[string]string{}
	for _, s := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			return nil, errors.New("invalid guest service status")
		}
		m[k] = v
	}
	return m, nil
}
func vmServiceStopped(m map[string]string) error {
	if m["ActiveState"] != "inactive" || m["SubState"] != "dead" || m["MainPID"] != "0" || m["ExecMainStatus"] != "0" {
		return errors.New("guest owner service not cleanly inactive")
	}
	return nil
}
func vmServiceProcess(m map[string]string) (string, error) {
	if e := vmServiceAdmission(); e != nil {
		return "", e
	}
	pid, e := strconv.Atoi(m["MainPID"])
	if e != nil || pid <= 1 || m["ActiveState"] != "active" || m["SubState"] != "running" || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(m["InvocationID"]) {
		return "", errors.New("guest service process identity missing")
	}
	hash := func(path string) (string, error) {
		b, e := os.ReadFile(path)
		h := sha256.Sum256(b)
		return hex.EncodeToString(h[:]), e
	}
	want, e := hash("/opt/omabit-lab/urbit-runner")
	if e != nil {
		return "", e
	}
	got, e := hash(fmt.Sprintf("/proc/%d/exe", pid))
	if e != nil || got != want {
		return "", errors.New("guest service executable differs from frozen runner")
	}
	b, e := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if e != nil {
		return "", e
	}
	a := strings.Split(strings.TrimSuffix(string(b), "\x00"), "\x00")
	if len(a) != 4 || a[0] != "/usr/local/lib/urbit-runner/current/urbit-runner" || a[1] != "serve-development" || a[2] != "--acknowledge-development" || a[3] != "--resume-running" {
		return "", errors.New("guest service command differs from installed owner service")
	}
	return got, nil
}
func vmPrepareOwnerService(ctx context.Context, proof map[string]any) error {
	proof["ok"] = false
	before, e := vmOwnerService(ctx, "show")
	proof["initial"] = before
	if e != nil {
		return e
	}
	h, e := vmServiceProcess(before)
	if e != nil {
		return e
	}
	proof["runner_sha256"] = h
	after, e := vmOwnerService(ctx, "stop")
	proof["stopped"] = after
	if e != nil {
		return e
	}
	if e = vmServiceStopped(after); e != nil {
		return e
	}
	proof["ok"] = true
	return nil
}
func vmServiceEmptyStore(s *store, fresh bool) error {
	tables := []string{"instances"}
	if fresh {
		tables = append(tables, "projects", "operations")
	}
	for _, table := range tables {
		var n int
		if e := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil {
			return e
		}
		if n != 0 {
			return fmt.Errorf("guest service requires empty %s", table)
		}
	}
	return nil
}

// Only project-filtered IDs are inspected. This does not list unrelated containers.
func vmServiceContainers(ctx context.Context, runner, project string) (map[string]json.RawMessage, error) {
	if !protocol.ValidID(runner) || !protocol.ValidID(project) {
		return nil, errors.New("invalid guest service project identity")
	}
	base := []string{"--config", filepath.Join(vmServiceDirectory, "project-resources-"+project, "docker-config"), "--host", "unix:///run/user/1000/urbit-docker.sock"}
	args := append(append([]string{}, base...), "container", "ls", "--all", "--quiet", "--no-trunc", "--filter", "label=org.urbit-runner.project.runner="+runner, "--filter", "label=org.urbit-runner.project.project="+project)
	raw, e := vmServiceCommand(ctx, "/usr/bin/docker", args...)
	if e != nil {
		return nil, e
	}
	ids := strings.Fields(string(raw))
	sort.Strings(ids)
	if len(ids) < 1 || len(ids) > 16 {
		return nil, errors.New("unexpected guest project container count")
	}
	out := map[string]json.RawMessage{}
	for _, id := range ids {
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
			return nil, errors.New("invalid project container ID")
		}
		raw, e = vmServiceCommand(ctx, "/usr/bin/docker", append(append([]string{}, base...), "container", "inspect", id)...)
		if e != nil {
			return nil, e
		}
		var rows []map[string]json.RawMessage
		if json.Unmarshal(raw, &rows) != nil || len(rows) != 1 {
			return nil, errors.New("invalid project inspection")
		}
		var actual string
		var config struct{ Labels map[string]string }
		if json.Unmarshal(rows[0]["Id"], &actual) != nil || actual != id || json.Unmarshal(rows[0]["Config"], &config) != nil || config.Labels["org.urbit-runner.project.runner"] != runner || config.Labels["org.urbit-runner.project.project"] != project {
			return nil, errors.New("project inspection ownership mismatch")
		}
		kept := map[string]json.RawMessage{}
		for _, name := range []string{"Id", "Name", "State", "RestartCount", "Config", "HostConfig", "NetworkSettings", "Mounts"} {
			if len(rows[0][name]) == 0 {
				return nil, errors.New("incomplete project inspection")
			}
			kept[name] = rows[0][name]
		}
		out[id], e = json.Marshal(kept)
		if e != nil {
			return nil, e
		}
	}
	return out, nil
}
func vmEqualServiceCheckpoint(a, b protocol.ProjectRecoveryCheckpoint) bool {
	b.ReadAt = a.ReadAt
	return reflect.DeepEqual(a, b)
}
func vmEqualServiceStatus(a, b protocol.ProjectStatus) bool {
	b.ReadAt = a.ReadAt
	return reflect.DeepEqual(a, b)
}

// Poll only read-only hello while the real service initializes its socket/store.
func vmWaitOwnerService(ctx context.Context) error {
	ready, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	c := &vmPublicRecoveryCLI{ctx: ready, registry: "/home/lab/project-service-readiness-registry"}
	for {
		if _, e := c.run("--socket", "/run/user/1000/urbit-runner/runner.sock", "hello"); e == nil {
			return nil
		}
		select {
		case <-ready.Done():
			return errors.New("guest owner service readiness deadline")
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// The caller closes SQLite before this function and reopens it only on success.
// Production service startup, collector reads and a real systemd restart operate
// on the same database. A deferred exact-unit stop precedes every return.
func vmServiceRecoveryCycle(ctx context.Context, directory, runner, project, parent, stage string, checkpoint protocol.ProjectRecoveryCheckpoint, status protocol.ProjectStatus) (proof map[string]any, err error) {
	proof = map[string]any{"ok": false, "stage": stage, "directory": directory, "runner_id": runner}
	if err = vmServiceAdmission(); err != nil {
		return
	}
	if directory != vmServiceDirectory || !regexp.MustCompile(`^[a-z-]+$`).MatchString(stage) {
		err = errors.New("guest service store/stage refused")
		return
	}
	current, e := vmOwnerService(ctx, "show")
	if e != nil {
		err = e
		return
	}
	if err = vmServiceStopped(current); err != nil {
		return
	}
	before, e := vmServiceContainers(ctx, runner, project)
	proof["containers_before"] = before
	if e != nil {
		err = e
		return
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 80*time.Second)
		defer cancel()
		stopped, e := vmOwnerService(cleanup, "stop")
		proof["stopped"] = stopped
		if e == nil {
			e = vmServiceStopped(stopped)
		}
		if e != nil {
			err = errors.Join(err, e)
			proof["ok"] = false
		}
	}()
	first, e := vmOwnerService(ctx, "start")
	proof["first_process"] = first
	if e != nil {
		err = e
		return
	}
	if err = vmWaitOwnerService(ctx); err != nil {
		return
	}
	hash, e := vmServiceProcess(first)
	proof["runner_sha256"] = hash
	if e != nil {
		err = e
		return
	}
	c := &vmPublicRecoveryCLI{ctx: ctx, runner: runner, project: project, registry: "/home/lab/project-service-" + stage + "-registry"}
	if _, e = c.run("enroll", "lab", "--socket", "/run/user/1000/urbit-runner/runner.sock"); e != nil {
		err = e
		return
	}
	read := func(label string) error {
		b, e := c.pinned("project-recovery-checkpoint", project, parent)
		if e != nil {
			return e
		}
		var got protocol.ProjectRecoveryCheckpoint
		if json.Unmarshal(b, &got) != nil || !vmEqualServiceCheckpoint(checkpoint, got) {
			return errors.New("service restart changed saved checkpoint")
		}
		proof[label+"_checkpoint"] = got
		b, e = c.pinned("project-status", project)
		if e != nil {
			return e
		}
		var gotStatus protocol.ProjectStatus
		if json.Unmarshal(b, &gotStatus) != nil || !vmEqualServiceStatus(status, gotStatus) {
			return errors.New("service restart changed saved operation history")
		}
		proof[label+"_status"] = gotStatus
		return nil
	}
	if err = read("first"); err != nil {
		return
	}
	second, e := vmOwnerService(ctx, "restart")
	proof["second_process"] = second
	if e != nil {
		err = e
		return
	}
	if err = vmWaitOwnerService(ctx); err != nil {
		return
	}
	h, e := vmServiceProcess(second)
	if e != nil {
		err = e
		return
	}
	if hash != h || first["MainPID"] == second["MainPID"] || first["InvocationID"] == second["InvocationID"] {
		err = errors.New("owner service did not start a new process generation")
		return
	}
	if err = read("second"); err != nil {
		return
	}
	after, e := vmServiceContainers(ctx, runner, project)
	proof["containers_after"] = after
	if e != nil {
		err = e
		return
	}
	if !reflect.DeepEqual(before, after) {
		err = errors.New("owner service restart changed project containers")
		return
	}
	proof["ok"] = true
	return
}
func TestVMServiceComparison(t *testing.T) {
	a := protocol.ProjectRecoveryCheckpoint{RunnerID: "runner", CheckpointDigest: "checkpoint", Sequence: 3, ReadAt: time.Now()}
	b := a
	b.ReadAt = b.ReadAt.Add(time.Second)
	if !vmEqualServiceCheckpoint(a, b) {
		t.Fatal("read timestamp is observational")
	}
	b.Sequence++
	if vmEqualServiceCheckpoint(a, b) {
		t.Fatal("checkpoint transition ignored")
	}
	b = a
	b.RunnerID = "other"
	if vmEqualServiceCheckpoint(a, b) {
		t.Fatal("runner identity ignored")
	}
	if os.Getenv("OMABIT_TEST_PROJECT_VM") == "" {
		if vmServiceAdmission() == nil {
			t.Fatal("unmarked host admitted")
		}
	}
}
