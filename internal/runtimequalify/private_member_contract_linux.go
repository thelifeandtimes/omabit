//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// These inputs must come from an accepted durable operation. This pure contract
// grants no dispatch, storage adoption or freshness: its driver must separately
// verify the complete owned member set, current gateway and exact held files.
// The current initial gateway observer requires empty mounts and cannot yet be
// used as that driver once a member has booted.
type privateMemberSelection struct {
	Plan                               privatechain.Plan
	Workspace                          privatechain.Workspace
	Gateway                            privatechain.Gateway
	InstanceID, RuntimeID, OperationID string
	ContainerGeneration                uint64
	Mode                               string
	PierIdentity                       pierstore.Identity
}
type privateMemberContract struct {
	selection                                                   privateMemberSelection
	root, mountSource, pillPath, keyPath, imageID, name, digest string
}

var privateMemberBirth = regexp.MustCompile(`^[0-9]{1,20}:(0|[1-9][0-9]{0,8})$`)

func newPrivateMemberContract(s privateMemberSelection, stateRoot, imageID string) (privateMemberContract, error) {
	bad := func() (privateMemberContract, error) {
		return privateMemberContract{}, errors.New("private member ownership, retained pier or gateway binding differs")
	}
	if !projectRuntimeID.MatchString(s.RuntimeID) || s.RuntimeID == strings.Repeat("0", 32) || !protocol.ValidID(s.OperationID) || s.ContainerGeneration == 0 || s.Gateway.Validate(s.Plan, s.Workspace) != nil || s.Gateway.Phase != "ready" || s.OperationID == s.Workspace.OperationID || s.OperationID == s.Gateway.Selection.OperationID || !privateGatewayMountPath(stateRoot) || !acceptedImageID(imageID) || s.Gateway.Observation.ImageID != imageID {
		return bad()
	}
	_, args, err := privateMemberCommand(s.Plan, s.InstanceID, s.Mode)
	if err != nil {
		return bad()
	}
	var mountPath string
	for _, m := range s.Workspace.Mounts {
		if m.InstanceID == s.InstanceID {
			mountPath = m.Path
		}
	}
	if mountPath == "" {
		return bad()
	}
	if s.Mode == "boot" {
		if s.ContainerGeneration != 1 || s.PierIdentity != (pierstore.Identity{}) {
			return bad()
		}
	} else {
		i := s.PierIdentity
		if s.ContainerGeneration < 2 || !validProjectPier(i) || !privateMemberBirth.MatchString(i.Birth) || !privateMemberBirth.MatchString(i.UrbBirth) {
			return bad()
		}
		pier := pierstore.DirectoryIdentity{Device: i.Device, Inode: i.Inode, Birth: i.Birth}
		urb := pierstore.DirectoryIdentity{Device: i.UrbDevice, Inode: i.UrbInode, Birth: i.UrbBirth}
		if pier.SameObject(urb) {
			return bad()
		}
		reserved := []pierstore.DirectoryIdentity{s.Workspace.Parent, s.Workspace.Root, s.Workspace.Observation.PiersRoot, s.Gateway.Observation.Chain, s.Gateway.Observation.Secrets}
		reserved = append(reserved, s.Workspace.Observation.Mounts...)
		for _, d := range reserved {
			if pier.SameObject(d) || urb.SameObject(d) {
				return bad()
			}
		}
	}
	s.Plan.Members = slices.Clone(s.Plan.Members)
	s.Plan.Foundation.Contracts = slices.Clone(s.Plan.Foundation.Contracts)
	s.Workspace = s.Workspace.Clone()
	s.Gateway = s.Gateway.Clone()
	c := privateMemberContract{selection: s, root: stateRoot, mountSource: filepath.Join(stateRoot, mountPath), imageID: imageID, name: "urbit-private-member-" + s.RuntimeID + "-g" + strconv.FormatUint(s.ContainerGeneration, 10)}
	if s.Mode == "boot" {
		c.keyPath = filepath.Join(stateRoot, s.Workspace.Path, "secrets", filepath.Base(args[len(args)-1])+".key")
		c.pillPath = filepath.Join(stateRoot, s.RuntimeID, candidate.PillName)
	}
	raw, err := json.Marshal(struct {
		Domain     string
		Selection  privateMemberSelection
		Arguments  []string
		PillSHA256 string
		PillBytes  int64
	}{"omabit-private-member-v1", s, c.createArgs(), candidate.PillSHA256, candidate.PillBytes})
	if err != nil {
		return bad()
	}
	c.digest = fmt.Sprintf("%x", sha256.Sum256(raw))
	return c, nil
}
func (c privateMemberContract) labels() map[string]string {
	s := c.selection
	g := s.Gateway.Selection
	return map[string]string{projectLabel: "true", projectLabel + ".mode": privatechain.Mode, projectLabel + ".runner": g.Binding.RunnerID, projectLabel + ".project": g.Binding.ProjectID, projectLabel + ".generation": strconv.FormatUint(g.Binding.Generation, 10), projectLabel + ".operation": s.OperationID, projectLabel + ".gateway-operation": g.OperationID, projectLabel + ".plan": g.PlanDigest, projectLabel + ".workspace": g.WorkspaceDigest, projectLabel + ".inputs": g.Inputs.ManifestSHA256, projectLabel + ".instance": s.InstanceID, projectLabel + ".runtime": s.RuntimeID, projectLabel + ".container-generation": strconv.FormatUint(s.ContainerGeneration, 10)}
}
func (c privateMemberContract) mounts() []mount {
	m := []mount{{Type: "bind", Source: c.mountSource, Destination: "/urbit", RW: true, Propagation: "rprivate"}}
	if c.selection.Mode == "boot" {
		m = append(m, mount{Type: "bind", Source: c.keyPath, Destination: "/run/omabit-boot/key", Propagation: "rprivate"}, mount{Type: "bind", Source: c.pillPath, Destination: "/run/omabit-boot/boot.pill", Propagation: "rprivate"})
	}
	return m
}
func privateMemberTmpfs() map[string]string {
	return map[string]string{"/tmp": "rw,nosuid,nodev,mode=1777,size=64m"}
}
func (c privateMemberContract) createArgs() []string {
	s := c.selection
	entry, cmd, _ := privateMemberCommand(s.Plan, s.InstanceID, s.Mode)
	args := []string{"container", "create", "--name", c.name, "--platform", candidate.Platform, "--pull", "never", "--entrypoint", entry, "--network", "container:" + s.Gateway.ContainerID, "--init", "--read-only", "--user", "0:0", "--workdir", "/urbit", "--cap-drop", "ALL", "--security-opt", "no-new-privileges=true", "--cgroupns", "private", "--ipc", "private", "--restart", "no", "--stop-signal", "SIGTERM", "--stop-timeout", "300", "--memory", "4294967296", "--memory-swap", "4294967296", "--pids-limit", "256", "--cpus", "2", "--log-driver", "local", "--log-opt", "max-size=10m", "--log-opt", "max-file=2", "--no-healthcheck", "--tmpfs", "/tmp:" + privateMemberTmpfs()["/tmp"]}
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
func (c privateMemberContract) validMounts(raw []json.RawMessage, actual []mount) bool {
	want := c.mounts()
	if len(raw) != len(want) {
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
		if !slices.Contains(want, mount{Type: v.Type, Source: v.Source, Destination: v.Target, RW: !v.ReadOnly, Propagation: v.BindOptions.Propagation}) {
			return false
		}
	}
	seen = map[string]bool{}
	binds := 0
	for _, v := range actual {
		if seen[v.Destination] {
			return false
		}
		seen[v.Destination] = true
		if slices.Contains(want, v) {
			binds++
			continue
		}
		// --tmpfs may be absent from Mounts, but its exact HostConfig is mandatory.
		if v != (mount{Type: "tmpfs", Destination: "/tmp", RW: true}) {
			return false
		}
	}
	return binds == len(want)
}
func (c privateMemberContract) inspectContainer(raw []byte, id string, running bool) (container, error) {
	if !privateGatewayID(id) || id == c.selection.Gateway.ContainerID || len(raw) > 1<<20 || !privateGatewayUnambiguousJSON(raw) {
		return container{}, errUntrustedContainer
	}
	var rows []container
	if json.Unmarshal(raw, &rows) != nil || len(rows) != 1 {
		return container{}, errUntrustedContainer
	}
	v := rows[0]
	if err := c.validateContainer(v, id, running); err != nil {
		return container{}, err
	}
	return v, nil
}

func (c privateMemberContract) validateContainer(v container, id string, running bool) error {
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
func (c privateMemberContract) validateExitedContainer(v container, id string, expectedStartedAt time.Time) error {
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
func (c privateMemberContract) validateConfiguration(v container, id string) error {
	entry, cmd, _ := privateMemberCommand(c.selection.Plan, c.selection.InstanceID, c.selection.Mode)
	if !containerIDPattern.MatchString(id) || v.ID != id || v.Name != "/"+c.name || v.Image != c.imageID || v.Config.Image != c.imageID || !reflect.DeepEqual(v.Config.Labels, c.labels()) {
		return errUntrustedContainer
	}
	cfg, h := v.Config, v.HostConfig
	if cfg.User != "0:0" || cfg.WorkingDir != "/urbit" || cfg.StopSignal != "SIGTERM" || cfg.StopTimeout != 300 || cfg.Tty || cfg.OpenStdin || !reflect.DeepEqual(cfg.Entrypoint, []string{entry}) || !reflect.DeepEqual(cfg.Cmd, cmd) || !reflect.DeepEqual(cfg.Env, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}) || !reflect.DeepEqual(cfg.Healthcheck.Test, []string{"NONE"}) {
		return errUntrustedContainer
	}
	if h.Init == nil || !*h.Init || !h.ReadonlyRootfs || h.NetworkMode != "container:"+c.selection.Gateway.ContainerID || h.PidMode != "" || h.IpcMode != "private" || h.UTSMode != "" || h.UsernsMode != "" || h.CgroupnsMode != "private" || h.Privileged || h.PublishAllPorts || h.AutoRemove || len(h.CapAdd) != 0 || !reflect.DeepEqual(h.CapDrop, []string{"ALL"}) || !reflect.DeepEqual(h.SecurityOpt, []string{"no-new-privileges=true"}) || len(h.Binds) != 0 || len(h.VolumesFrom) != 0 || len(h.Links) != 0 || len(h.Devices) != 0 || len(h.DeviceRequests) != 0 || !reflect.DeepEqual(h.Tmpfs, privateMemberTmpfs()) || len(h.ExtraHosts) != 0 || len(h.Dns) != 0 || len(h.DnsSearch) != 0 || len(h.Sysctls) != 0 || h.Memory != 4<<30 || h.MemorySwap != 4<<30 || h.NanoCpus != 2_000_000_000 || h.CpuQuota != 0 || h.CpuPeriod != 0 || h.PidsLimit == nil || *h.PidsLimit != 256 || h.RestartPolicy.Name != "no" || h.RestartPolicy.MaximumRetryCount != 0 || h.LogConfig.Type != "local" || !reflect.DeepEqual(h.LogConfig.Config, map[string]string{"max-size": "10m", "max-file": "2"}) || !c.validMounts(h.Mounts, v.Mounts) {
		return errUntrustedContainer
	}
	if len(h.PortBindings) != 0 || len(v.NetworkSettings.Networks) != 0 || len(v.NetworkSettings.Ports) != 0 {
		return errUntrustedNetwork
	}
	return nil
}
