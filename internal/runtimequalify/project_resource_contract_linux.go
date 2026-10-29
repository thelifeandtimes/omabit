//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const projectLabel = "org.urbit-runner.project"
const anchorMemory = int64(128 << 20)
const anchorTmpfs = int64(1 << 20)

type projectSelection struct {
	RunnerID, ProjectID, OperationID string
	Revision, Generation             uint64
	Plan                             projectnetwork.Plan
}
type projectResourceContract struct {
	Selection   projectSelection
	Preparation projectnetwork.Preparation
	ImageID     string
}

func newProjectResourceContract(s projectSelection, engineID, imageID string) (projectResourceContract, error) {
	digest, err := s.Plan.Digest()
	if err != nil || !protocol.ValidID(s.RunnerID) || !protocol.ValidID(s.ProjectID) || !protocol.ValidID(s.OperationID) || s.Revision == 0 || s.Generation == 0 || !hostcompat.ValidEngineIdentity(engineID) || !acceptedImageID(imageID) {
		return projectResourceContract{}, errors.New("invalid project resource selection")
	}
	s.Plan.Members = slices.Clone(s.Plan.Members)
	prefix := "urbit-" + s.ProjectID + "-g" + strconv.FormatUint(s.Generation, 10)
	c := projectResourceContract{Selection: s, ImageID: imageID, Preparation: projectnetwork.Preparation{ProjectID: s.ProjectID, ProjectRevision: s.Revision, Generation: s.Generation, OperationID: s.OperationID, PlanDigest: digest, EngineID: engineID, Bridge: projectnetwork.Creation{Name: prefix + "-bridge", Phase: "planned"}, Anchor: projectnetwork.Creation{Name: prefix + "-anchor", Phase: "planned"}}}
	c.Preparation.Bridge.ContractDigest = c.digest("bridge")
	c.Preparation.Anchor.ContractDigest = c.digest("anchor")
	if err = c.Preparation.Validate(s.Plan); err != nil {
		return projectResourceContract{}, err
	}
	return c, nil
}
func (c projectResourceContract) labels() map[string]string {
	return map[string]string{projectLabel: "true", projectLabel + ".runner": c.Selection.RunnerID, projectLabel + ".project": c.Selection.ProjectID, projectLabel + ".generation": strconv.FormatUint(c.Selection.Generation, 10), projectLabel + ".operation": c.Selection.OperationID}
}
func appendLabels(args []string, labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--label", k+"="+labels[k])
	}
	return args
}
func (c projectResourceContract) bridgeArgs() []string {
	args := appendLabels([]string{"network", "create", "--driver", "bridge", "--internal=false", "--ipv4=true", "--ipv6=false"}, c.labels())
	return append(args, "--opt", "com.docker.network.bridge.host_binding_ipv4=127.0.0.1", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=nat", c.Preparation.Bridge.Name)
}
func (c projectResourceContract) anchorArgs(bridgeID string) []string {
	args := []string{"container", "create", "--name", c.Preparation.Anchor.Name, "--network", bridgeID, "--init", "--read-only", "--user", "0:0", "--workdir", "/", "--entrypoint", "/bin/sleep", "--stop-signal", "SIGTERM", "--stop-timeout", "10", "--restart", "no", "--ipc", "private", "--cgroupns", "private", "--cap-drop", "ALL", "--security-opt", "no-new-privileges=true", "--memory", "134217728", "--memory-swap", "134217728", "--cpus", "0.25", "--pids-limit", "64", "--log-driver", "local", "--log-opt", "max-size=1m", "--log-opt", "max-file=2", "--no-healthcheck", "--mount", "type=tmpfs,destination=/urbit,tmpfs-size=1048576,tmpfs-mode=0700"}
	args = appendLabels(args, c.labels())
	for _, m := range c.Selection.Plan.Members {
		args = append(args, "--publish", fmt.Sprintf("127.0.0.1:%d:%d/tcp", m.HTTPPort, m.InternalHTTPPort))
	}
	return append(args, c.ImageID, "infinity")
}
func (c projectResourceContract) digest(kind string) string {
	args := c.bridgeArgs()
	if kind == "anchor" {
		args = c.anchorArgs("recorded-bridge:" + c.Preparation.Bridge.Name)
	}
	data, _ := json.Marshal(struct {
		Domain, Kind, Engine, Image string
		Selection                   projectSelection
		Arguments                   []string
	}{"omabit-project-resource-v1", kind, c.Preparation.EngineID, c.ImageID, c.Selection, args})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func (c projectResourceContract) validateBridge(n bridge, id, allowedAnchor string) error {
	if !containerIDPattern.MatchString(id) || n.ID != id || n.Name != c.Preparation.Bridge.Name || n.Driver != "bridge" || n.Scope != "local" || n.Internal == nil || *n.Internal || n.EnableIPv4 == nil || !*n.EnableIPv4 || n.EnableIPv6 || n.Attachable || n.Ingress || n.ConfigOnly || !reflect.DeepEqual(n.Options, bridgeOptions()) || !reflect.DeepEqual(n.Labels, c.labels()) {
		return errUntrustedNetwork
	}
	for id := range n.Containers {
		if allowedAnchor == "" || id != allowedAnchor {
			return errUntrustedNetwork
		}
	}
	return nil
}

// The pinned image declares VOLUME /urbit. An explicit bounded tmpfs prevents
// this sleeping anchor from creating an unjournaled persistent volume.
type projectTmpfsMount struct {
	Type, Source, Target, Consistency string
	ReadOnly                          bool
	BindOptions, VolumeOptions        json.RawMessage
	TmpfsOptions                      *struct {
		SizeBytes int64
		Mode      uint32
	}
}

func validAnchorMount(raw []json.RawMessage, actual []mount) bool {
	if len(raw) != 1 || len(actual) > 1 {
		return false
	}
	var m projectTmpfsMount
	decoder := json.NewDecoder(bytes.NewReader(raw[0]))
	decoder.DisallowUnknownFields()
	if !json.Valid(raw[0]) || decoder.Decode(&m) != nil || m.Type != "tmpfs" || m.Source != "" || m.Target != "/urbit" || m.Consistency != "" || m.ReadOnly || len(m.BindOptions) != 0 && string(m.BindOptions) != "null" || len(m.VolumeOptions) != 0 && string(m.VolumeOptions) != "null" || m.TmpfsOptions == nil || m.TmpfsOptions.SizeBytes != anchorTmpfs || m.TmpfsOptions.Mode != 0700 {
		return false
	}
	for _, v := range actual {
		if v.Type != "tmpfs" || v.Source != "" || v.Destination != "/urbit" || !v.RW || v.Propagation != "" {
			return false
		}
	}
	return true
}
func (c projectResourceContract) validateStoppedAnchor(v container, id, bridgeID string) error {
	return c.validateAnchor(v, id, bridgeID, false)
}
func (c projectResourceContract) validateAnchor(v container, id, bridgeID string, running bool) error {
	if v.State == nil || v.State.FinishedAt != "0001-01-01T00:00:00Z" || v.State.Running == nil || *v.State.Running != running || v.State.OOMKilled == nil || *v.State.OOMKilled || v.State.ExitCode == nil || *v.State.ExitCode != 0 || v.State.Error != "" || v.State.Paused || v.State.Restarting || v.State.Dead {
		return errUntrustedContainer
	}
	if running {
		started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		if v.State.Status != "running" || v.State.Pid <= 0 || err != nil || started.IsZero() || len(v.Mounts) != 1 {
			return errUntrustedContainer
		}
	} else if v.State.Status != "created" || v.State.Pid != 0 || v.State.StartedAt != "0001-01-01T00:00:00Z" {
		return errUntrustedContainer
	}
	return c.validateAnchorConfiguration(v, id, bridgeID, running)
}

// Recovery inspection accepts an exited/restarted anchor without changing the initial
// Start contract. It observes failure (including OOM); it grants no restart right.
func (c projectResourceContract) validateRecoveryAnchor(v container, id, bridgeID string) error {
	if v.State == nil || v.State.Running == nil {
		return errUntrustedContainer
	}
	st := v.State
	start, se := time.Parse(time.RFC3339Nano, st.StartedAt)
	finish, fe := time.Parse(time.RFC3339Nano, st.FinishedAt)
	if *st.Running {
		// Docker retains the previous exit timestamp after restarting the same
		// anchor. It must precede this process start; it is not a current exit.
		// Initial Start validation above deliberately remains stricter.
		if st.Status != "running" || st.Pid <= 0 || st.OOMKilled == nil || *st.OOMKilled || st.ExitCode == nil || *st.ExitCode != 0 || st.Error != "" || st.Paused || st.Restarting || st.Dead || se != nil || fe != nil || start.IsZero() || start.After(time.Now().UTC().Add(5*time.Second)) || !finish.IsZero() && !finish.Before(start) || len(v.Mounts) != 1 {
			return errUntrustedContainer
		}
		return c.validateAnchorConfiguration(v, id, bridgeID, true)
	}
	if st.Status != "exited" || st.Pid != 0 || st.OOMKilled == nil || st.ExitCode == nil || *st.ExitCode < 0 || *st.ExitCode > 255 || st.Error != "" || st.Paused || st.Restarting || st.Dead || se != nil || fe != nil || start.IsZero() || finish.IsZero() || finish.Before(start) || finish.After(time.Now().UTC().Add(5*time.Second)) {
		return errUntrustedContainer
	}
	return c.validateAnchorConfiguration(v, id, bridgeID, false)
}

func (c projectResourceContract) validateAnchorConfiguration(v container, id, bridgeID string, running bool) error {
	if !containerIDPattern.MatchString(id) || !containerIDPattern.MatchString(bridgeID) || v.ID != id || v.Name != "/"+c.Preparation.Anchor.Name || v.Image != c.ImageID || v.Config.Image != c.ImageID || !reflect.DeepEqual(v.Config.Labels, c.labels()) {
		return errUntrustedContainer
	}
	cfg, h := v.Config, v.HostConfig
	if cfg.User != "0:0" || cfg.WorkingDir != "/" || cfg.StopSignal != "SIGTERM" || cfg.StopTimeout != 10 || cfg.Tty || cfg.OpenStdin || !reflect.DeepEqual(cfg.Entrypoint, []string{"/bin/sleep"}) || !reflect.DeepEqual(cfg.Cmd, []string{"infinity"}) || !reflect.DeepEqual(cfg.Env, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}) || !reflect.DeepEqual(cfg.Healthcheck.Test, []string{"NONE"}) {
		return errUntrustedContainer
	}
	if h.Init == nil || !*h.Init || !h.ReadonlyRootfs || h.NetworkMode != bridgeID || h.PidMode != "" || h.IpcMode != "private" || h.UTSMode != "" || h.UsernsMode != "" || h.CgroupnsMode != "private" || h.Privileged || h.PublishAllPorts || h.AutoRemove || len(h.CapAdd) != 0 || !reflect.DeepEqual(h.CapDrop, []string{"ALL"}) || !reflect.DeepEqual(h.SecurityOpt, []string{"no-new-privileges=true"}) || len(h.Binds) != 0 || len(h.VolumesFrom) != 0 || len(h.Links) != 0 || len(h.Devices) != 0 || len(h.DeviceRequests) != 0 || len(h.Tmpfs) != 0 || len(h.ExtraHosts) != 0 || len(h.Dns) != 0 || len(h.DnsSearch) != 0 || len(h.Sysctls) != 0 || h.Memory != anchorMemory || h.MemorySwap != anchorMemory || h.NanoCpus != 250_000_000 || h.CpuQuota != 0 || h.CpuPeriod != 0 || h.PidsLimit == nil || *h.PidsLimit != 64 || h.RestartPolicy.Name != "no" || h.RestartPolicy.MaximumRetryCount != 0 || h.LogConfig.Type != "local" || !reflect.DeepEqual(h.LogConfig.Config, map[string]string{"max-size": "1m", "max-file": "2"}) || !validAnchorMount(h.Mounts, v.Mounts) {
		return errUntrustedContainer
	}
	if len(h.PortBindings) != len(c.Selection.Plan.Members) {
		return errUntrustedNetwork
	}
	for _, m := range c.Selection.Plan.Members {
		b, err := decodeLoopbackBinding(h.PortBindings[strconv.Itoa(int(m.InternalHTTPPort))+"/tcp"], false)
		if err != nil || b.HostPort != strconv.Itoa(int(m.HTTPPort)) {
			return errUntrustedNetwork
		}
	}
	active := 0
	for port, raw := range v.NetworkSettings.Ports {
		if string(bytes.TrimSpace(raw)) == "null" {
			continue
		}
		if !running {
			return errUntrustedNetwork
		}
		expected, ok := h.PortBindings[port]
		if !ok {
			return errUntrustedNetwork
		}
		want, err := decodeLoopbackBinding(expected, false)
		if err != nil {
			return errUntrustedNetwork
		}
		got, err := decodeLoopbackBinding(raw, false)
		if err != nil || got != want {
			return errUntrustedNetwork
		}
		active++
	}
	if running && active != len(c.Selection.Plan.Members) {
		return errUntrustedNetwork
	}
	if len(v.NetworkSettings.Networks) > 1 || running && len(v.NetworkSettings.Networks) != 1 {
		return errUntrustedNetwork
	}
	for name, raw := range v.NetworkSettings.Networks {
		var endpoint struct{ NetworkID string }
		if (name != c.Preparation.Bridge.Name && name != bridgeID) || json.Unmarshal(raw, &endpoint) != nil || endpoint.NetworkID != "" && endpoint.NetworkID != bridgeID || running && endpoint.NetworkID != bridgeID {
			return errUntrustedNetwork
		}
	}

	return nil
}
