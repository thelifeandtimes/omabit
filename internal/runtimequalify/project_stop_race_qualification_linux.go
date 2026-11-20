//go:build linux && amd64 && omabit_disposable_qualification

package runtimequalify

// This file is excluded from ordinary products. It is a timing fault for a
// fixed disposable VM fixture, never a runner option or general signal API.
import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const projectRaceCase = "preliminary-exit-transition-v1"
const projectRaceBase = "0c28d526285e5b4e055e209e9021e5c683505ddabb5f0ce661678052949d141c"
const projectRaceTarget = "issues-race-bud"
const projectRacePeer = "issues-race-nec"

// DisposableProjectStopRaceEvent contains only fixed test steps and exact
// process/operation bindings. It never includes Docker Env or original output.
type DisposableProjectStopRaceEvent struct {
	Step        string    `json:"step"`
	OperationID string    `json:"operation_id"`
	ContainerID string    `json:"container_id"`
	StartedAt   time.Time `json:"started_at"`
	At          time.Time `json:"at"`
}

func projectRaceVMGuard(raw []byte, source string, uid int, home, runtimeDir, vendor, cmdline string) bool {
	if len(raw) > 64<<10 {
		return false
	}
	var s struct {
		Kind     string `json:"kind"`
		Case     string `json:"case"`
		Source   string `json:"source_commit"`
		Base     string `json:"predecessor_root_sha256"`
		Original int    `json:"original_schema"`
		Target   int    `json:"target_schema"`
		Required bool   `json:"require_native_transition"`
		Preserve bool   `json:"preserve_stopped_base_instance"`
	}
	b, err := hex.DecodeString(source)
	return json.Unmarshal(raw, &s) == nil && err == nil && len(b) == 20 && source == strings.ToLower(source) && s.Source == source && s.Kind == "issues16-native-transition-v1" && s.Case == projectRaceCase && s.Base == projectRaceBase && s.Original == 65 && s.Target == 65 && s.Required && s.Preserve && uid == 1000 && home == "/home/review" && runtimeDir == "/run/user/1000" && strings.TrimSpace(vendor) == "QEMU" && slices.Contains(strings.Fields(cmdline), "omabit.disposable-guest=release-upgrade-v1")
}

func projectRaceSelection(s ProjectExpansion, id, operation string) (int, error) {
	if !protocol.ValidID(s.Project.ProjectID) || !protocol.ValidID(operation) || !strings.HasPrefix(operation, "op-") || s.Project.Plan.Label != "issues native transition" || len(s.Members) != 2 || len(s.History) != 0 || id != projectRaceTarget {
		return -1, errors.New("refused non-fresh native race selection")
	}
	found := -1
	for i, m := range s.Members {
		ship, port, added := "~nec", uint16(19580), false
		if m.Member.InstanceID == projectRaceTarget {
			ship, port, added = "~bud", 19581, true
			found = i
		} else if m.Member.InstanceID != projectRacePeer {
			return -1, errors.New("foreign native race member")
		}
		if m.Member.Ship != ship || m.Member.HTTPPort != port || m.Added != added || m.Phase != "ready" || m.Creator.InstanceID != m.Member.InstanceID || m.Creator.Mode != "boot" || m.Creator.Generation != 1 || m.StopIntent != nil || !containerIDPattern.MatchString(m.Creator.Container.ID) {
			return -1, errors.New("changed native race member")
		}
		if added && (m.Initialized.Identity.Inode == 0 || m.Initialized.Path != m.Creator.Storage.Path) {
			return -1, errors.New("native race target lacks retained pier identity")
		}
	}
	if s.Members[0].Creator.Container.ID == s.Members[1].Creator.Container.ID {
		return -1, errors.New("native race shares a container")
	}
	if found < 0 {
		return -1, errors.New("missing native race target")
	}
	return found, nil
}

// CheckDisposableProjectStopRaceVM validates the qualification-only VM before a
// test manager opens any runner store or socket. It accepts no path override.
func CheckDisposableProjectStopRaceVM(ctx context.Context, source string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/review" || os.Getenv("OMABIT_PROJECT_RACE_VM") != projectRaceCase {
		return errors.New("native race fixture only")
	}
	f, err := os.Open("/opt/omabit-release-input/scenario.json")
	if err != nil {
		return errors.New("native race scenario unavailable")
	}
	raw, e1 := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	closeErr := f.Close()
	vendor, e2 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e3 := os.ReadFile("/proc/cmdline")
	if e1 != nil || closeErr != nil || e2 != nil || e3 != nil || !projectRaceVMGuard(raw, source, os.Geteuid(), os.Getenv("HOME"), os.Getenv("XDG_RUNTIME_DIR"), string(vendor), string(cmdline)) {
		return errors.New("native race VM/source guard refused")
	}
	return nil
}

// RunDisposableProjectStopRace is compiled only with the explicit qualification
// tag. The caller is the separately pinned test manager, with the real manager
// save callback. Normal RunProjectShipAction always uses unmodified defaults.
func RunDisposableProjectStopRace(ctx context.Context, s *ProjectExpansion, id, operation, source string, save func(ProjectExpansion) error, trace func(DisposableProjectStopRaceEvent) error) (err error) {
	if err = CheckDisposableProjectStopRaceVM(ctx, source); err != nil {
		return err
	}
	if s == nil || save == nil || trace == nil {
		return errors.New("native race selection/callback unavailable")
	}

	index, err := projectRaceSelection(*s, id, operation)
	if err != nil {
		return err
	}
	d := defaults()
	if d.home != "/home/review" || d.runtimeDir != "" {
		return errors.New("native race owner profile changed")
	}
	members := []string{s.Members[0].Creator.Container.ID, s.Members[1].Creator.Container.ID}
	slices.Sort(members)
	r := &projectStopRace{target: s.Members[index].Creator.Container.ID, members: members, operation: operation, raw: d.execute, trace: trace, poll: 100 * time.Millisecond}
	r.readIDs = append(slices.Clone(members), s.Anchor.Prepared.Anchor.ID)
	r.prefix = []string{"--host", "unix:///run/user/1000/" + rootless.SocketName, "--config", filepath.Join(d.home, ".local/share/urbit-runner/managed", "project-resources-"+s.Project.ProjectID, "docker-config")}
	r.intent = func() *ProjectExpandedStopIntent { return s.Members[index].StopIntent }
	d.execute = r.execute
	defer func() { err = errors.Join(err, r.release()) }()
	err = runProjectShipActionWithDependencies(ctx, s, id, "stop", operation, save, d)
	if err == nil && (!r.transitioned || !r.termAcknowledged) {
		return errors.New("native race boundary was not demonstrated")
	}
	return err
}

type projectStopRace struct {
	target, operation                                                        string
	members, readIDs, prefix, env                                            []string
	raw                                                                      func(context.Context, []string, []string) (string, error)
	intent                                                                   func() *ProjectExpandedStopIntent
	trace                                                                    func(DisposableProjectStopRaceEvent) error
	started                                                                  time.Time
	pauseAttempted, termAttempted, termAcknowledged, continued, transitioned bool
	poll                                                                     time.Duration
}

func (r *projectStopRace) emit(step string) error {
	return r.trace(DisposableProjectStopRaceEvent{Step: step, OperationID: r.operation, ContainerID: r.target, StartedAt: r.started, At: time.Now().UTC()})
}
func (r *projectStopRace) command(ctx context.Context, args ...string) (string, error) {
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return r.raw(bounded, append(slices.Clone(r.prefix), args...), slices.Clone(r.env))
}
func (r *projectStopRace) inspected(raw string, running bool) (container, error) {
	var values []container
	if json.Unmarshal([]byte(raw), &values) != nil {
		return container{}, errors.New("invalid native race inspection")
	}
	for _, c := range values {
		if c.ID == r.target {
			if c.State == nil || c.State.Running == nil || *c.State.Running != running || c.State.Paused || c.State.Restarting || c.State.Dead {
				return container{}, errors.New("native race process state changed")
			}
			started, err := time.Parse(time.RFC3339Nano, c.State.StartedAt)
			if err != nil || !started.Equal(r.started) {
				return container{}, errors.New("native race process was replaced")
			}
			if !running && (c.State.ExitCode == nil || *c.State.ExitCode != 0 || c.State.OOMKilled == nil || *c.State.OOMKilled || c.State.Error != "") {
				return container{}, errors.New("native race did not exit cleanly")
			}
			return c, nil
		}
	}
	return container{}, errors.New("native race selected container missing")
}
func (r *projectStopRace) execute(ctx context.Context, args, env []string) (string, error) {
	if len(args) < 6 || !slices.Equal(args[:4], r.prefix) {
		return "", errors.New("native race engine/config changed")
	}
	if len(args) == 9 && slices.Equal(args[4:8], []string{"container", "kill", "--signal", "SIGTERM"}) {
		if args[8] != r.target || r.termAttempted {
			return "", errors.New("native race foreign or repeated signal")
		}
		intent := r.intent()
		if intent == nil || intent.OperationID != r.operation || intent.ContainerID != r.target || intent.StartedAt.IsZero() {
			return "", errors.New("native race lacks durable exact stop intent")
		}
		r.started = intent.StartedAt
		r.env = slices.Clone(env)
		before, err := r.command(ctx, "container", "inspect", r.target)
		if err != nil {
			return "", err
		}
		if _, err = r.inspected(before, true); err != nil {
			return "", err
		}
		if err = r.emit("pause-intent"); err != nil {
			return "", err
		}
		r.pauseAttempted = true
		if _, err = r.command(ctx, "container", "kill", "--signal", "SIGSTOP", r.target); err != nil {
			return "", err
		}
		if err = r.emit("paused"); err != nil {
			return "", err
		}
		r.termAttempted = true
		out, err := r.raw(ctx, args, env)
		if err != nil {
			return out, err
		}
		r.termAcknowledged = true
		return out, r.emit("term-dispatched")
	}
	a := args[4:]
	read := (a[0] == "image" || a[0] == "network") && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")
	if len(a) == 8 && a[0] == "container" && a[1] == "exec" && slices.Contains(r.readIDs, a[2]) && slices.Equal(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
		read = true
	}
	if !read {
		return "", errors.New("native race refuses unexpected mutation/exec")
	}
	out, err := r.raw(ctx, args, env)
	if err != nil || !r.termAcknowledged || r.transitioned || len(args) != 6+len(r.members) || !slices.Equal(args[4:6], []string{"container", "inspect"}) || !slices.Equal(args[6:], r.members) {
		return out, err
	}
	if _, err = r.inspected(out, true); err != nil {
		return "", err
	}
	if err = r.emit("preliminary-running"); err != nil {
		return "", err
	}
	current, inspectErr := r.command(ctx, "container", "inspect", r.target)
	if inspectErr != nil {
		return "", inspectErr
	}
	if _, inspectErr = r.inspected(current, true); inspectErr != nil {
		return "", inspectErr
	}
	if _, err = r.command(ctx, "container", "kill", "--signal", "SIGCONT", r.target); err != nil {
		return "", err
	}
	r.continued = true
	if err = r.emit("continued"); err != nil {
		return "", err
	}
	for {
		after, readErr := r.command(ctx, "container", "inspect", r.target)
		if readErr != nil {
			return "", readErr
		}
		if _, readErr = r.inspected(after, false); readErr == nil {
			break
		}
		// A running same process is allowed while flushing; other changes refuse.
		if _, err = r.inspected(after, true); err != nil {
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(r.poll):
		}
	}
	r.transitioned = true
	if err = r.emit("exited-before-bracket"); err != nil {
		return "", err
	}
	// Return exactly the earlier real running observation, never fabricated JSON.
	return out, nil
}
func (r *projectStopRace) release() error {
	if !r.pauseAttempted || r.continued {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, err := r.command(ctx, "container", "inspect", r.target)
	if err != nil {
		return fmt.Errorf("native race release inspection: %w", err)
	}
	if _, err = r.inspected(raw, false); err == nil {
		r.continued = true
		return nil
	}
	if _, err = r.inspected(raw, true); err != nil {
		return fmt.Errorf("native race refuses release of changed process: %w", err)
	}
	if _, err = r.command(ctx, "container", "kill", "--signal", "SIGCONT", r.target); err != nil {
		return fmt.Errorf("native race release: %w", err)
	}
	r.continued = true
	return r.emit("cleanup-continued")
}
