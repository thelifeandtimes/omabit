//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

const privateGatewayMemory = int64(1536 << 20)

// The input program remains a guest-only candidate. This pure contract describes
// its container; it does not grant execution or claim a production runtime.
// A driver must freshly bind real filesystem objects, inspect the sealed inputs,
// verify its engine/image, check port availability and retain dispatch beforehand.
type privateGatewayContainerContract struct {
	network               privateGatewayNetworkContract
	members               []privatechain.Member
	root, inputs, imageID string
}

func newPrivateGatewayContainerContract(p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, stateRoot, inputs, imageID string) (privateGatewayContainerContract, error) {
	n, err := newPrivateGatewayNetworkContract(p, w, g)
	if err != nil {
		return privateGatewayContainerContract{}, err
	}
	bad := errors.New("private gateway container requires distinct canonical workspace and input paths with the pinned image")
	if !privateGatewayMountPath(stateRoot) || !privateGatewayMountPath(inputs) || !acceptedImageID(imageID) {
		return privateGatewayContainerContract{}, bad
	}
	root := filepath.Join(stateRoot, w.Path)
	if root == inputs || strings.HasPrefix(root, inputs+"/") || strings.HasPrefix(inputs, root+"/") {
		return privateGatewayContainerContract{}, bad
	}
	return privateGatewayContainerContract{network: n, members: slices.Clone(p.Members), root: root, inputs: inputs, imageID: imageID}, nil
}

func privateGatewayMountPath(p string) bool {
	// Docker's mount argument is CSV. Refuse separators/quotes rather than
	// reinterpret an accepted pathname as additional mount options.
	return len(p) > 1 && len(p) <= 4096 && filepath.IsAbs(p) && filepath.Clean(p) == p &&
		!strings.ContainsAny(p, ",\"\\") && strings.IndexFunc(p, unicode.IsControl) < 0
}

func (c privateGatewayContainerContract) binds() []mount {
	out := []mount{{Type: "bind", Source: c.root, Destination: "/lab", RW: true, Propagation: "rprivate"}}
	for _, p := range [][2]string{{"node", "/node"}, {"tools", "/tools"}, {"source", "/source"}, {"compiled", "/compiled"}, {"deployment", "/deployment"}, {"vere", "/vere"}, {"gateway/docker_gateway.cjs", "/gateway.cjs"}} {
		out = append(out, mount{Type: "bind", Source: filepath.Join(c.inputs, p[0]), Destination: p[1], RW: false, Propagation: "rprivate"})
	}
	return out
}

func privateGatewayTmpfs() map[string]string {
	return map[string]string{"/tmp": "rw,nosuid,nodev,mode=1777,size=64m", "/urbit": "ro,nosuid,nodev,noexec,mode=0555,size=64k"}
}

func privateGatewayEnvironment() map[string]string {
	return map[string]string{
		"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin",
		"HOME": "/lab/home", "LIBC": "glibc", "PREBUILDS_ONLY": "1",
		"OMABIT_PRIVATE_CHAIN":         "private-chain-docker-gateway-v1",
		"OMABIT_PRIVATE_CHAIN_NETWORK": "published-v1",
	}
}

func (c privateGatewayContainerContract) containerArgs(networkID string) ([]string, error) {
	if !privateGatewayID(networkID) {
		return nil, errUntrustedNetwork
	}
	args := []string{"container", "create", "--name", c.network.gatewayName(), "--network", networkID,
		"--init", "--read-only", "--user", "0:0", "--workdir", "/lab", "--entrypoint", "/node/usr/lib/ld-linux-x86-64.so.2",
		"--stop-signal", "SIGTERM", "--stop-timeout", "10", "--restart", "no", "--ipc", "private", "--cgroupns", "private",
		"--cap-drop", "ALL", "--security-opt", "no-new-privileges=true", "--memory", "1610612736", "--memory-swap", "1610612736",
		"--cpus", "2", "--pids-limit", "128", "--log-driver", "local", "--log-opt", "max-size=1m", "--log-opt", "max-file=2", "--no-healthcheck"}
	args = appendLabels(args, c.network.labels())
	for _, m := range c.members {
		args = append(args, "--publish", fmt.Sprintf("127.0.0.1:%d:%d/tcp", m.HTTPPort, m.InternalHTTPPort))
	}
	for _, key := range []string{"HOME", "LIBC", "PREBUILDS_ONLY", "OMABIT_PRIVATE_CHAIN", "OMABIT_PRIVATE_CHAIN_NETWORK"} {
		args = append(args, "--env", key+"="+privateGatewayEnvironment()[key])
	}
	for _, target := range []string{"/tmp", "/urbit"} {
		args = append(args, "--tmpfs", target+":"+privateGatewayTmpfs()[target])
	}
	for _, m := range c.binds() {
		value := "type=bind,source=" + m.Source + ",target=" + m.Destination + ",bind-propagation=rprivate"
		if !m.RW {
			value += ",readonly"
		}
		args = append(args, "--mount", value)
	}
	return append(args, c.imageID, "--library-path", "/node/usr/lib", "/node/usr/bin/node", "/gateway.cjs"), nil
}

func (c privateGatewayContainerContract) validMounts(configured []json.RawMessage, actual []mount, running bool) bool {
	want := map[string]mount{}
	for _, m := range c.binds() {
		want[m.Destination] = m
	}
	if len(configured) != len(want) {
		return false
	}
	seen := map[string]bool{}
	for _, raw := range configured {
		var v struct {
			Type, Source, Target, Consistency string
			ReadOnly                          bool
			BindOptions                       *struct{ Propagation string }
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&v) != nil || !json.Valid(raw) || v.BindOptions == nil || v.Consistency != "" || seen[v.Target] {
			return false
		}
		got := mount{Type: v.Type, Source: v.Source, Destination: v.Target, RW: !v.ReadOnly, Propagation: v.BindOptions.Propagation}
		if expected, ok := want[v.Target]; !ok || got != expected {
			return false
		}
		seen[v.Target] = true
	}
	seen = map[string]bool{}
	binds, tmpfs := 0, 0
	for _, m := range actual {
		if seen[m.Destination] {
			return false
		}
		seen[m.Destination] = true
		if expected, ok := want[m.Destination]; ok {
			if m != expected {
				return false
			}
			binds++
			continue
		}
		if m.Type != "tmpfs" || m.Source != "" || m.Propagation != "" || (m.Destination != "/tmp" && m.Destination != "/urbit") || m.RW != (m.Destination == "/tmp") {
			return false
		}
		tmpfs++
	}
	// Docker keeps --tmpfs mounts in HostConfig.Tmpfs, separately from
	// MountPoints/Mounts even while running. The caller checks that exact
	// configuration. Validate both reported entries if the engine includes them.
	return binds == len(want) && (tmpfs == 0 || tmpfs == 2)
}

func (c privateGatewayContainerContract) inspectContainer(data []byte, networkID, gatewayID string, allocation privateGatewayAllocation, running bool) (container, error) {
	fail := func(err error) (container, error) { return container{}, err }
	if !privateGatewayID(networkID) || !privateGatewayID(gatewayID) || networkID == gatewayID || len(data) > 1<<20 || !privateGatewayUnambiguousJSON(data) {
		return fail(errUntrustedContainer)
	}
	var rows []container
	if json.Unmarshal(data, &rows) != nil || len(rows) != 1 {
		return fail(errUntrustedContainer)
	}
	v := rows[0]
	cfg, h, st := v.Config, v.HostConfig, v.State
	if v.ID != gatewayID || v.Name != "/"+c.network.gatewayName() || v.Image != c.imageID || cfg.Image != c.imageID || !reflect.DeepEqual(cfg.Labels, c.network.labels()) {
		return fail(errUntrustedContainer)
	}
	if st == nil || st.Running == nil || *st.Running != running || st.OOMKilled == nil || *st.OOMKilled || st.ExitCode == nil || *st.ExitCode != 0 || st.Error != "" || st.Paused || st.Restarting || st.Dead || st.FinishedAt != "0001-01-01T00:00:00Z" {
		return fail(errUntrustedContainer)
	}
	if running {
		start, err := time.Parse(time.RFC3339Nano, st.StartedAt)
		if st.Status != "running" || st.Pid <= 0 || err != nil || start.IsZero() || start.After(time.Now().UTC().Add(5*time.Second)) {
			return fail(errUntrustedContainer)
		}
	} else if st.Status != "created" || st.Pid != 0 || st.StartedAt != "0001-01-01T00:00:00Z" {
		return fail(errUntrustedContainer)
	}
	if cfg.User != "0:0" || cfg.WorkingDir != "/lab" || cfg.StopSignal != "SIGTERM" || cfg.StopTimeout != 10 || cfg.Tty || cfg.OpenStdin ||
		!reflect.DeepEqual(cfg.Entrypoint, []string{"/node/usr/lib/ld-linux-x86-64.so.2"}) ||
		!reflect.DeepEqual(cfg.Cmd, []string{"--library-path", "/node/usr/lib", "/node/usr/bin/node", "/gateway.cjs"}) ||
		!reflect.DeepEqual(cfg.Healthcheck.Test, []string{"NONE"}) {
		return fail(errUntrustedContainer)
	}
	env := map[string]string{}
	for _, value := range cfg.Env {
		key, value, ok := strings.Cut(value, "=")
		if _, duplicate := env[key]; !ok || duplicate {
			return fail(errUntrustedContainer)
		}
		env[key] = value
	}
	if !reflect.DeepEqual(env, privateGatewayEnvironment()) {
		return fail(errUntrustedContainer)
	}
	if h.Init == nil || !*h.Init || !h.ReadonlyRootfs || h.NetworkMode != networkID || h.PidMode != "" || h.IpcMode != "private" || h.UTSMode != "" || h.UsernsMode != "" || h.CgroupnsMode != "private" ||
		h.Privileged || h.PublishAllPorts || h.AutoRemove || len(h.CapAdd) != 0 || !reflect.DeepEqual(h.CapDrop, []string{"ALL"}) || !reflect.DeepEqual(h.SecurityOpt, []string{"no-new-privileges=true"}) ||
		len(h.Binds) != 0 || len(h.VolumesFrom) != 0 || len(h.Links) != 0 || len(h.Devices) != 0 || len(h.DeviceRequests) != 0 || len(h.ExtraHosts) != 0 || len(h.Dns) != 0 || len(h.DnsSearch) != 0 || len(h.Sysctls) != 0 ||
		h.Memory != privateGatewayMemory || h.MemorySwap != privateGatewayMemory || h.NanoCpus != 2_000_000_000 || h.CpuQuota != 0 || h.CpuPeriod != 0 || h.PidsLimit == nil || *h.PidsLimit != 128 ||
		h.RestartPolicy.Name != "no" || h.RestartPolicy.MaximumRetryCount != 0 || h.LogConfig.Type != "local" || !reflect.DeepEqual(h.LogConfig.Config, map[string]string{"max-size": "1m", "max-file": "2"}) ||
		!reflect.DeepEqual(h.Tmpfs, privateGatewayTmpfs()) || !c.validMounts(h.Mounts, v.Mounts, running) {
		return fail(errUntrustedContainer)
	}
	if err := c.checkPublications(v, networkID, allocation, running); err != nil {
		return fail(err)
	}
	return v, nil
}

func (c privateGatewayContainerContract) checkPublications(v container, networkID string, a privateGatewayAllocation, running bool) error {
	h := v.HostConfig
	if len(h.PortBindings) != len(c.members) {
		return errUntrustedNetwork
	}
	expected := map[string]portBinding{}
	for _, m := range c.members {
		key := strconv.Itoa(int(m.InternalHTTPPort)) + "/tcp"
		b, err := decodeLoopbackBinding(h.PortBindings[key], false)
		if err != nil || b.HostPort != strconv.Itoa(int(m.HTTPPort)) {
			return errUntrustedNetwork
		}
		expected[key] = b
	}
	active := 0
	for port, raw := range v.NetworkSettings.Ports {
		if string(bytes.TrimSpace(raw)) == "null" {
			continue
		}
		b, err := decodeLoopbackBinding(raw, false)
		if !running || err != nil || b != expected[port] {
			return errUntrustedNetwork
		}
		active++
	}
	if running && active != len(expected) || len(v.NetworkSettings.Networks) > 1 || running && len(v.NetworkSettings.Networks) != 1 {
		return errUntrustedNetwork
	}
	for name, raw := range v.NetworkSettings.Networks {
		var n struct {
			NetworkID, EndpointID, Gateway, IPAddress, IPv6Gateway, GlobalIPv6Address string
			IPPrefixLen, GlobalIPv6PrefixLen                                          int
		}
		if json.Unmarshal(raw, &n) != nil || n.IPv6Gateway != "" || n.GlobalIPv6Address != "" || n.GlobalIPv6PrefixLen != 0 {
			return errUntrustedNetwork
		}
		if !running {
			if (name != c.network.networkName() && name != networkID) || n.NetworkID != "" && n.NetworkID != networkID ||
				n.EndpointID != "" || n.Gateway != "" || n.IPAddress != "" || n.IPPrefixLen != 0 {
				return errUntrustedNetwork
			}
			continue
		}
		gateway, ge := netip.ParseAddr(n.Gateway)
		address, ae := netip.ParseAddr(n.IPAddress)
		if name != c.network.networkName() || n.NetworkID != networkID || a.NetworkID != networkID || a.ContainerID != v.ID || !privateGatewayID(a.EndpointID) || n.EndpointID != a.EndpointID ||
			!a.Subnet.IsValid() || !a.Subnet.Addr().Is4() || a.Subnet != a.Subnet.Masked() || !privateGatewayHost(a.Subnet, a.Gateway) || !privateGatewayHost(a.Subnet, a.Address) || a.Gateway == a.Address ||
			ge != nil || ae != nil || gateway.String() != n.Gateway || address.String() != n.IPAddress || gateway != a.Gateway || address != a.Address || n.IPPrefixLen != a.Subnet.Bits() {
			return errUntrustedNetwork
		}
	}
	return nil
}
