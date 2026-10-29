//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

var projectRuntimeID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// The private member driver must obtain these bindings from an accepted store
// operation, not RPC input. A contract describes one container generation; it
// does not authorize dispatch, adopt storage, or establish current readiness.
type projectMemberSelection struct {
	// Added members bind their own immutable port assignment; the original
	// network/anchor contract remains unchanged. Omission preserves old digests.
	AddedMember                        *projectnetwork.Member `json:",omitempty"`
	Project                            projectSelection
	Anchor                             projectnetwork.AnchorStart
	InstanceID, RuntimeID, OperationID string
	ContainerGeneration                uint64
	Mode                               string
	Storage                            pierstore.Location
}
type projectMemberContract struct {
	postRecovery      *projectnetwork.RecoveredResumeCreation
	recovery          *projectnetwork.RecoveryMemberCreation
	selection         projectMemberSelection
	home, mountSource string
	imageID, pillPath string
	name, digest      string
}

func newProjectMemberContract(s projectMemberSelection, home, mountSource, imageID string) (projectMemberContract, error) {
	fail := func() (projectMemberContract, error) {
		return projectMemberContract{}, errors.New("invalid project member ownership or storage binding")
	}
	if !projectRuntimeID.MatchString(s.RuntimeID) || !protocol.ValidID(s.OperationID) || s.ContainerGeneration == 0 || s.Anchor.Validate(s.Project.Plan) != nil || s.Anchor.Phase != "observed" || s.OperationID == s.Anchor.OperationID || s.OperationID == s.Project.OperationID || s.Storage.Uncertain {
		return fail()
	}
	resource, err := newProjectResourceContract(s.Project, s.Anchor.Prepared.EngineID, imageID)
	if err != nil {
		return fail()
	}
	expected := resource.Preparation
	expected.Sequence = 6
	expected.Bridge.Phase, expected.Bridge.ID = "observed", s.Anchor.Prepared.Bridge.ID
	expected.Anchor.Phase, expected.Anchor.ID = "observed", s.Anchor.Prepared.Anchor.ID
	if !reflect.DeepEqual(expected, s.Anchor.Prepared) {
		return fail()
	}
	_, args, err := s.command()
	if err != nil {
		return fail()
	}
	leaf := filepath.Base(args[len(args)-1])
	if s.Storage.Path != "fakeships/"+s.InstanceID+"/"+leaf || !pierstore.ValidPath(s.Storage.Path) || !validProjectDirectory(s.Storage.MountIdentity) {
		return fail()
	}
	if s.Mode == "boot" {
		if s.ContainerGeneration != 1 || s.Storage.Identity != (pierstore.Identity{}) {
			return fail()
		}
	} else if s.ContainerGeneration < 2 || !validProjectPier(s.Storage.Identity) {
		return fail()
	}
	for _, path := range []string{home, mountSource} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || strings.ContainsAny(path, ",\x00\r\n") {
			return fail()
		}
	}
	// Copy all mutable selection storage. A later caller edit must not silently
	// rewrite the command/namespace whose digest was persisted before dispatch.
	s.Project.Plan.Members = slices.Clone(s.Project.Plan.Members)
	ns := *s.Anchor.Namespace
	s.Anchor.Namespace = &ns
	c := projectMemberContract{selection: s, home: home, mountSource: mountSource, imageID: imageID,
		name: "urbit-project-member-" + s.RuntimeID + "-g" + strconv.FormatUint(s.ContainerGeneration, 10)}
	if s.Mode == "boot" {
		c.pillPath = filepath.Join(home, ".local/share/urbit-runner/managed", s.RuntimeID, candidate.PillName)
	}
	body, err := json.Marshal(struct {
		Domain     string
		Selection  projectMemberSelection
		Arguments  []string
		PillSHA256 string
		PillBytes  int64
	}{"omabit-project-member-v1", s, c.createArgs(), candidate.PillSHA256, candidate.PillBytes})
	if err != nil {
		return fail()
	}
	c.digest = fmt.Sprintf("%x", sha256.Sum256(body))
	return c, nil
}
func validProjectDirectory(v pierstore.DirectoryIdentity) bool {
	return v.Device != 0 && v.Inode != 0 && v.Birth != ""
}
func validProjectPier(v pierstore.Identity) bool {
	return v.Device != 0 && v.Inode != 0 && v.UrbDevice != 0 && v.UrbInode != 0 && v.Birth != "" && v.UrbBirth != ""
}

// checkStorageBeforeCreate is read-only. Boot requires an empty, previously
// journaled exclusive parent; even a partial child is retained for review.
// Resume requires the exact stopped pier and .urb objects. The caller still
// verifies the engine's complete mount inventory and the live anchor namespace.
func (c projectMemberContract) checkStorageBeforeCreate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := pierstore.CheckMount(c.home, c.selection.Storage)
	if err != nil {
		return err
	}
	if p != c.mountSource {
		return errors.New("project member mount source changed")
	}
	if c.selection.Mode == "boot" {
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return errors.New("fresh member mount contains data; retain it for review")
		}
	} else {
		pier, err := pierstore.Check(c.home, c.selection.Storage)
		if err != nil {
			return err
		}
		if pierstore.Busy(pier) {
			return errors.New("project member pier still has a Vere lock")
		}
		if err = pierstore.CheckStateFiles(ctx, pier); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (c projectMemberContract) labels() map[string]string {
	s := c.selection
	labels := map[string]string{projectLabel: "true", projectLabel + ".runner": s.Project.RunnerID, projectLabel + ".project": s.Project.ProjectID, projectLabel + ".generation": strconv.FormatUint(s.Project.Generation, 10), projectLabel + ".instance": s.InstanceID, projectLabel + ".runtime": s.RuntimeID, projectLabel + ".operation": s.OperationID, projectLabel + ".container-generation": strconv.FormatUint(s.ContainerGeneration, 10)}
	if c.recovery != nil {
		labels[projectLabel+".namespace-operation"] = c.recovery.NamespaceOperationID
		labels[projectLabel+".retirement-operation"] = c.recovery.RetirementOperationID
	}
	if c.postRecovery != nil {
		labels[projectLabel+".namespace-operation"] = c.postRecovery.NamespaceOperationID
		labels[projectLabel+".recovery-operation"] = c.postRecovery.RecoveryOperationID
		labels[projectLabel+".retirement-operation"] = c.postRecovery.RetirementOperationID
	}
	return labels
}
func (c projectMemberContract) mounts() []mount {
	m := []mount{{Type: "bind", Source: c.mountSource, Destination: "/urbit", RW: true, Propagation: "rprivate"}}
	if c.pillPath != "" {
		m = append(m, mount{Type: "bind", Source: c.pillPath, Destination: "/artifacts/boot.pill", RW: false, Propagation: "rprivate"})
	}
	return m
}
func (c projectMemberContract) createArgs() []string {
	s := c.selection
	entry, cmd, _ := s.command()
	args := []string{"container", "create", "--name", c.name, "--platform", candidate.Platform, "--pull", "never", "--entrypoint", entry, "--network", "container:" + s.Anchor.Prepared.Anchor.ID, "--init", "--user", "0:0", "--workdir", "/urbit", "--cap-drop", "ALL", "--security-opt", "no-new-privileges=true", "--cgroupns", "private", "--ipc", "private", "--restart", "no", "--stop-signal", "SIGTERM", "--stop-timeout", "300", "--memory", "4294967296", "--memory-swap", "4294967296", "--pids-limit", "256", "--cpus", "2", "--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=2", "--no-healthcheck"}
	args = appendLabels(args, c.labels())
	for _, m := range c.mounts() {
		value := "type=bind,src=" + m.Source + ",dst=" + m.Destination + ",bind-propagation=rprivate"
		if !m.RW {
			value += ",readonly"
		}
		args = append(args, "--mount", value)
	}
	return append(append(args, c.imageID), cmd...)
}

// Match both Docker's requested mounts and effective mounts. This intentionally
// excludes implicit volumes, extra bind options, recursive propagation and any
// independent network/publication. Namespace equality itself needs a fresh probe.
func (c projectMemberContract) validMounts(raw []json.RawMessage, actual []mount) bool {
	want := c.mounts()
	if len(raw) != len(want) || len(actual) != len(want) {
		return false
	}
	seen := map[string]bool{}
	for _, data := range raw {
		var v struct {
			Type, Source, Target, Consistency string
			ReadOnly                          bool
			BindOptions                       *struct{ Propagation string }
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if d.Decode(&v) != nil || !json.Valid(data) || seen[v.Target] || v.BindOptions == nil || v.Consistency != "" {
			return false
		}
		seen[v.Target] = true
		got := mount{Type: v.Type, Source: v.Source, Destination: v.Target, RW: !v.ReadOnly, Propagation: v.BindOptions.Propagation}
		if !slices.Contains(want, got) {
			return false
		}
	}
	seen = map[string]bool{}
	for _, v := range actual {
		if seen[v.Destination] || !slices.Contains(want, v) {
			return false
		}
		seen[v.Destination] = true
	}
	return true
}
func (c projectMemberContract) validateContainer(v container, id string, running bool) error {
	if err := c.validateConfiguration(v, id); err != nil {
		return err
	}
	if v.State == nil || v.State.Running == nil || *v.State.Running != running || v.State.OOMKilled == nil || *v.State.OOMKilled || v.State.ExitCode == nil || *v.State.ExitCode != 0 || v.State.FinishedAt != "0001-01-01T00:00:00Z" || v.State.Error != "" || v.State.Paused || v.State.Restarting || v.State.Dead {
		return errUntrustedContainer
	}
	if running {
		t, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		if err != nil || t.IsZero() || v.State.Status != "running" || v.State.Pid <= 0 {
			return errUntrustedContainer
		}
	} else if v.State.Status != "created" || v.State.Pid != 0 || v.State.StartedAt != "0001-01-01T00:00:00Z" {
		return errUntrustedContainer
	}
	return nil
}

// validateExitedContainer requires a recorded process start as well as the full
// original launch contract. Nonzero/OOM exits are valid observations of failure,
// never evidence of a clean Stop. Callers still establish freshness and ownership.
func (c projectMemberContract) validateExitedContainer(v container, id string, expectedStartedAt time.Time) error {
	if err := c.validateConfiguration(v, id); err != nil {
		return err
	}
	if expectedStartedAt.IsZero() || v.State == nil || v.State.Running == nil || *v.State.Running || v.State.OOMKilled == nil || v.State.ExitCode == nil || *v.State.ExitCode < 0 || *v.State.ExitCode > 255 || v.State.Status != "exited" || v.State.Pid != 0 || v.State.Error != "" || v.State.Paused || v.State.Restarting || v.State.Dead {
		return errUntrustedContainer
	}
	started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
	if err != nil || !started.Equal(expectedStartedAt) {
		return errUntrustedContainer
	}
	finished, err := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
	if err != nil || finished.IsZero() || finished.Before(started) {
		return errUntrustedContainer
	}
	return nil
}

// validateConfiguration contains only immutable launch-contract checks. Each
// lifecycle reader separately requires its exact expected process state.
func (c projectMemberContract) validateConfiguration(v container, id string) error {
	entry, cmd, _ := c.selection.command()
	if !containerIDPattern.MatchString(id) || v.ID != id || v.Name != "/"+c.name || v.Image != c.imageID || v.Config.Image != c.imageID || !reflect.DeepEqual(v.Config.Labels, c.labels()) {
		return errUntrustedContainer
	}
	cfg, h := v.Config, v.HostConfig
	if cfg.User != "0:0" || cfg.WorkingDir != "/urbit" || cfg.StopSignal != "SIGTERM" || cfg.StopTimeout != 300 || cfg.Tty || cfg.OpenStdin || !reflect.DeepEqual(cfg.Entrypoint, []string{entry}) || !reflect.DeepEqual(cfg.Cmd, cmd) || !reflect.DeepEqual(cfg.Env, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}) || !reflect.DeepEqual(cfg.Healthcheck.Test, []string{"NONE"}) {
		return errUntrustedContainer
	}
	if h.Init == nil || !*h.Init || h.ReadonlyRootfs || h.NetworkMode != "container:"+c.selection.Anchor.Prepared.Anchor.ID || h.PidMode != "" || h.IpcMode != "private" || h.UTSMode != "" || h.UsernsMode != "" || h.CgroupnsMode != "private" || h.Privileged || h.PublishAllPorts || h.AutoRemove || len(h.CapAdd) != 0 || !reflect.DeepEqual(h.CapDrop, []string{"ALL"}) || !reflect.DeepEqual(h.SecurityOpt, []string{"no-new-privileges=true"}) || len(h.Binds) != 0 || len(h.VolumesFrom) != 0 || len(h.Links) != 0 || len(h.Devices) != 0 || len(h.DeviceRequests) != 0 || len(h.Tmpfs) != 0 || len(h.ExtraHosts) != 0 || len(h.Dns) != 0 || len(h.DnsSearch) != 0 || len(h.Sysctls) != 0 || h.Memory != 4<<30 || h.MemorySwap != 4<<30 || h.NanoCpus != 2_000_000_000 || h.CpuQuota != 0 || h.CpuPeriod != 0 || h.PidsLimit == nil || *h.PidsLimit != 256 || h.RestartPolicy.Name != "no" || h.RestartPolicy.MaximumRetryCount != 0 || h.LogConfig.Type != "local" || !reflect.DeepEqual(h.LogConfig.Config, map[string]string{"max-size": "10m", "max-file": "2"}) || !c.validMounts(h.Mounts, v.Mounts) {
		return errUntrustedContainer
	}
	if len(h.PortBindings) != 0 || len(v.NetworkSettings.Networks) != 0 || len(v.NetworkSettings.Ports) != 0 {
		return errUntrustedNetwork
	}
	return nil
}

func (s projectMemberSelection) command() (string, []string, error) {
	if s.AddedMember == nil {
		return projectMemberCommand(s.Project.Plan, s.InstanceID, s.Mode)
	}
	m := *s.AddedMember
	if m.InstanceID != s.InstanceID || !protocol.ValidID(m.InstanceID) || len(m.Ship) != 4 || !protocol.ValidFakeShip(m.Ship) || m.HTTPPort < 1024 || m.InternalHTTPPort < 8080 || m.InternalHTTPPort > 8335 || (s.Mode != "boot" && s.Mode != "resume") {
		return "", nil, errors.New("invalid added project member")
	}
	for _, old := range s.Project.Plan.Members {
		if old.InstanceID == m.InstanceID || old.Ship == m.Ship || old.InternalHTTPPort == m.InternalHTTPPort || old.HTTPPort == m.HTTPPort {
			return "", nil, errors.New("added project member conflicts with initial plan")
		}
	}
	return projectCommandForMember(m, s.Mode)
}
