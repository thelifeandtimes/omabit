//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const privateContainerStateRoot = "/home/fixture/managed"
const privateContainerInputs = "/home/fixture/input-generation"

func privateContainerFixture(t *testing.T, running bool) (privateGatewayContainerContract, map[string]any, privateGatewayAllocation) {
	t.Helper()
	p, w, g, network, n := privateNetworkFixture(t)
	c, err := newPrivateGatewayContainerContract(p, w, g, privateContainerStateRoot, privateContainerInputs, runtimecandidate.ConfigDigest)
	if err != nil {
		t.Fatal(err)
	}
	var allocation privateGatewayAllocation
	if running {
		privateNetworkAttach(n)
		allocation, err = network.inspectNetwork(privateNetworkBytes(t, n), strings.Repeat("a", 64), strings.Repeat("b", 64))
		if err != nil {
			t.Fatal(err)
		}
	}
	bindings := map[string]any{"18081/tcp": []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "18381"}}, "18082/tcp": []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "18382"}}}
	doc := map[string]any{
		"Id": strings.Repeat("b", 64), "Name": "/urbit-private-project-network-g1-gateway", "Image": runtimecandidate.ConfigDigest,
		"Config":          map[string]any{"Image": runtimecandidate.ConfigDigest, "User": "0:0", "WorkingDir": "/lab", "StopSignal": "SIGTERM", "StopTimeout": 10, "Entrypoint": []string{"/node/usr/lib/ld-linux-x86-64.so.2"}, "Cmd": []string{"--library-path", "/node/usr/lib", "/node/usr/bin/node", "/gateway.cjs"}, "Labels": c.network.labels(), "Healthcheck": map[string]any{"Test": []string{"NONE"}}, "Env": []string{"HOME=/lab/home", "LIBC=glibc", "PREBUILDS_ONLY=1", "OMABIT_PRIVATE_CHAIN=private-chain-docker-gateway-v1", "OMABIT_PRIVATE_CHAIN_NETWORK=published-v1", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}},
		"HostConfig":      map[string]any{"Init": true, "ReadonlyRootfs": true, "NetworkMode": strings.Repeat("a", 64), "IpcMode": "private", "CgroupnsMode": "private", "CapDrop": []string{"ALL"}, "SecurityOpt": []string{"no-new-privileges=true"}, "Memory": int64(1610612736), "MemorySwap": int64(1610612736), "NanoCpus": int64(2000000000), "PidsLimit": 128, "RestartPolicy": map[string]any{"Name": "no"}, "LogConfig": map[string]any{"Type": "local", "Config": map[string]string{"max-size": "1m", "max-file": "2"}}, "Tmpfs": map[string]string{"/tmp": "rw,nosuid,nodev,mode=1777,size=64m", "/urbit": "ro,nosuid,nodev,noexec,mode=0555,size=64k"}, "PortBindings": bindings},
		"State":           map[string]any{"Status": "created", "Pid": 0, "Running": false, "OOMKilled": false, "ExitCode": 0, "StartedAt": "0001-01-01T00:00:00Z", "FinishedAt": "0001-01-01T00:00:00Z"},
		"NetworkSettings": map[string]any{"Networks": map[string]any{"urbit-private-project-network-g1-bridge": map[string]any{"NetworkID": strings.Repeat("a", 64)}}, "Ports": map[string]any{"80/tcp": nil, "34343/udp": nil}},
	}
	var configured, actual []any
	for _, v := range [][3]string{
		{"/home/fixture/managed/private-projects/private-project-network", "/lab", "write"},
		{"/home/fixture/input-generation/node", "/node", "read"},
		{"/home/fixture/input-generation/tools", "/tools", "read"},
		{"/home/fixture/input-generation/source", "/source", "read"},
		{"/home/fixture/input-generation/compiled", "/compiled", "read"},
		{"/home/fixture/input-generation/deployment", "/deployment", "read"},
		{"/home/fixture/input-generation/vere", "/vere", "read"},
		{"/home/fixture/input-generation/gateway/docker_gateway.cjs", "/gateway.cjs", "read"},
	} {
		configured = append(configured, map[string]any{"Type": "bind", "Source": v[0], "Target": v[1], "ReadOnly": v[2] == "read", "BindOptions": map[string]string{"Propagation": "rprivate"}})
		actual = append(actual, map[string]any{"Type": "bind", "Source": v[0], "Destination": v[1], "RW": v[2] == "write", "Propagation": "rprivate"})
	}
	doc["HostConfig"].(map[string]any)["Mounts"] = configured
	if running {
		st := doc["State"].(map[string]any)
		st["Status"] = "running"
		st["Running"] = true
		st["Pid"] = 123
		st["StartedAt"] = "2026-10-25T12:00:00Z"
		ns := doc["NetworkSettings"].(map[string]any)
		// Separate inspection objects must not share mutable maps in this fixture.
		active := map[string]any{}
		for port, value := range bindings {
			active[port] = value
		}
		ns["Ports"] = active
		ns["Networks"] = map[string]any{"urbit-private-project-network-g1-bridge": map[string]any{"NetworkID": strings.Repeat("a", 64), "EndpointID": strings.Repeat("c", 64), "Gateway": "172.22.0.1", "IPAddress": "172.22.0.2", "IPPrefixLen": 16, "IPv6Gateway": "", "GlobalIPv6Address": "", "GlobalIPv6PrefixLen": 0}}
		actual = append(actual, map[string]any{"Type": "tmpfs", "Source": "", "Destination": "/tmp", "RW": true, "Propagation": ""}, map[string]any{"Type": "tmpfs", "Source": "", "Destination": "/urbit", "RW": false, "Propagation": ""})
	}
	doc["Mounts"] = actual
	return c, doc, allocation
}

func privateContainerData(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal([]any{doc})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func privateContainerCheck(t *testing.T, c privateGatewayContainerContract, doc map[string]any, a privateGatewayAllocation, running bool) error {
	t.Helper()
	_, err := c.inspectContainer(privateContainerData(t, doc), strings.Repeat("a", 64), strings.Repeat("b", 64), a, running)
	return err
}

func TestPrivateGatewayContainerCommandAndPaths(t *testing.T) {
	c, _, _ := privateContainerFixture(t, false)
	args, err := c.containerArgs(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	var ports, mounts, environment []string
	for i, v := range args {
		switch v {
		case "--publish":
			ports = append(ports, args[i+1])
		case "--mount":
			mounts = append(mounts, args[i+1])
		case "--env":
			environment = append(environment, args[i+1])
		}
	}
	if !reflect.DeepEqual(ports, []string{"127.0.0.1:18382:18082/tcp", "127.0.0.1:18381:18081/tcp"}) {
		t.Fatal(ports)
	}
	if len(mounts) != 8 || mounts[0] != "type=bind,source=/home/fixture/managed/private-projects/private-project-network,target=/lab,bind-propagation=rprivate" {
		t.Fatal(mounts)
	}
	for _, v := range mounts[1:] {
		if !strings.HasSuffix(v, ",readonly") {
			t.Fatal("input mount is writable", v)
		}
	}
	if len(environment) != 5 || !slices.Contains(environment, "OMABIT_PRIVATE_CHAIN_NETWORK=published-v1") {
		t.Fatal(environment)
	}
	if !reflect.DeepEqual(args[len(args)-5:], []string{runtimecandidate.ConfigDigest, "--library-path", "/node/usr/lib", "/node/usr/bin/node", "/gateway.cjs"}) {
		t.Fatal("wrong payload")
	}
	for _, v := range []string{"start", "--privileged", "--network=host", "--volume", "--publish-all", "--restart=always"} {
		if slices.Contains(args, v) {
			t.Fatal("unexpected effect/authority", v)
		}
	}
	for _, id := range []string{"host", "--network=host", strings.Repeat("0", 64), "short"} {
		if _, err := c.containerArgs(id); err == nil {
			t.Fatal("unrecorded network accepted")
		}
	}
	p, w, g, _, _ := privateNetworkFixture(t)
	for _, path := range []string{"", "/", "relative", "/tmp/../outside", "/tmp/", "/tmp,readonly", "/tmp/quote\"", "/tmp/back\\slash", "/tmp/new\nline"} {
		for _, inputs := range []bool{false, true} {
			state, input := privateContainerStateRoot, privateContainerInputs
			if inputs {
				input = path
			} else {
				state = path
			}
			if _, err := newPrivateGatewayContainerContract(p, w, g, state, input, runtimecandidate.ConfigDigest); err == nil {
				t.Fatal("invalid path admitted", path)
			}
		}
	}
	for _, input := range []string{privateContainerStateRoot, privateContainerStateRoot + "/" + w.Path, privateContainerStateRoot + "/" + w.Path + "/nested"} {
		if _, err := newPrivateGatewayContainerContract(p, w, g, privateContainerStateRoot, input, runtimecandidate.ConfigDigest); err == nil {
			t.Fatal("overlapping storage admitted")
		}
	}
	frozen, err := newPrivateGatewayContainerContract(p, w, g, "/home/fixture/managed space", "/home/fixture/inputs space", runtimecandidate.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := frozen.containerArgs(strings.Repeat("a", 64))
	if _, err := newPrivateGatewayContainerContract(p, w, g, privateContainerStateRoot, privateContainerInputs, "unqualified-image"); err == nil {
		t.Fatal("unqualified image admitted")
	}
	p.Members[0].HTTPPort++
	after, _ := frozen.containerArgs(strings.Repeat("a", 64))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("contract retained caller slice")
	}
}

func TestPrivateGatewayContainerCreatedAndRunning(t *testing.T) {
	for _, running := range []bool{false, true} {
		c, doc, a := privateContainerFixture(t, running)
		if err := privateContainerCheck(t, c, doc, a, running); err != nil {
			t.Fatal(running, err)
		}
	}
	c, doc, a := privateContainerFixture(t, true)
	env := doc["Config"].(map[string]any)["Env"].([]string)
	slices.Reverse(env)
	if err := privateContainerCheck(t, c, doc, a, true); err != nil {
		t.Fatal("environment order should not change its meaning", err)
	}
	doc["NetworkSettings"].(map[string]any)["Ports"].(map[string]any)["34343/udp"] = nil
	if err := privateContainerCheck(t, c, doc, a, true); err != nil {
		t.Fatal("unpublished image port refused", err)
	}
}

func TestPrivateGatewayContainerRejectsAuthorityAndResourceChanges(t *testing.T) {
	changes := map[string]func(map[string]any){
		"foreign-id":    func(d map[string]any) { d["Id"] = strings.Repeat("d", 64) },
		"foreign-name":  func(d map[string]any) { d["Name"] = "/other" },
		"foreign-image": func(d map[string]any) { d["Image"] = "sha256:" + strings.Repeat("d", 64) },
		"operation": func(d map[string]any) {
			d["Config"].(map[string]any)["Labels"].(map[string]string)[projectLabel+".operation"] = "other"
		},
		"program": func(d map[string]any) { d["Config"].(map[string]any)["Cmd"] = []string{"/other.cjs"} },
		"loader":  func(d map[string]any) { d["Config"].(map[string]any)["Entrypoint"] = []string{"/bin/sh"} },
		"env-injection": func(d map[string]any) {
			cfg := d["Config"].(map[string]any)
			cfg["Env"] = append(cfg["Env"].([]string), "LD_PRELOAD=/other.so")
		},
		"env-duplicate": func(d map[string]any) {
			cfg := d["Config"].(map[string]any)
			cfg["Env"] = append(cfg["Env"].([]string), "HOME=/lab/home")
		},
		"env-override": func(d map[string]any) { d["Config"].(map[string]any)["Env"].([]string)[0] = "HOME=/" },
		"automatic-restart": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["RestartPolicy"] = map[string]any{"Name": "always"}
		},
		"extra-capability":     func(d map[string]any) { d["HostConfig"].(map[string]any)["CapAdd"] = []string{"NET_ADMIN"} },
		"caps-not-dropped":     func(d map[string]any) { delete(d["HostConfig"].(map[string]any), "CapDrop") },
		"privilege-escalation": func(d map[string]any) { d["HostConfig"].(map[string]any)["SecurityOpt"] = nil },
		"device": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Devices"] = []any{map[string]string{"PathOnHost": "/dev/kvm"}}
		},
		"host-dns":   func(d map[string]any) { d["HostConfig"].(map[string]any)["Dns"] = []string{"8.8.8.8"} },
		"host-alias": func(d map[string]any) { d["HostConfig"].(map[string]any)["ExtraHosts"] = []string{"host:host-gateway"} },
		"sysctl": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Sysctls"] = map[string]string{"net.ipv4.ip_forward": "1"}
		},
		"bind-shortcut": func(d map[string]any) { d["HostConfig"].(map[string]any)["Binds"] = []string{"/:/host"} },
		"shared-volume": func(d map[string]any) { d["HostConfig"].(map[string]any)["VolumesFrom"] = []string{"other"} },
		"unbounded-log": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["LogConfig"] = map[string]any{"Type": "json-file"}
		},
	}
	for field, value := range map[string]any{"ReadonlyRootfs": false, "Init": false, "Privileged": true, "PublishAllPorts": true, "AutoRemove": true, "NetworkMode": "host", "PidMode": "host", "IpcMode": "host", "UTSMode": "host", "UsernsMode": "host", "CgroupnsMode": "host", "Memory": 0, "MemorySwap": -1, "NanoCpus": 0, "CpuQuota": 100000, "CpuPeriod": 100000, "PidsLimit": -1} {
		changes["host-"+field] = func(d map[string]any) { d["HostConfig"].(map[string]any)[field] = value }
	}
	for field, value := range map[string]any{"User": "1000:1000", "WorkingDir": "/", "StopSignal": "SIGKILL", "StopTimeout": 0, "Tty": true, "OpenStdin": true} {
		changes["config-"+field] = func(d map[string]any) { d["Config"].(map[string]any)[field] = value }
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			change(d)
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("changed authority/resource admitted")
			}
		})
	}
}

func TestPrivateGatewayContainerRejectsUnexpectedStorage(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"input-writable-config": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Mounts"].([]any)[1].(map[string]any)["ReadOnly"] = false
		},
		"input-writable-live": func(d map[string]any) { d["Mounts"].([]any)[1].(map[string]any)["RW"] = true },
		"foreign-root-config": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Mounts"].([]any)[0].(map[string]any)["Source"] = "/other/project"
		},
		"foreign-root-live": func(d map[string]any) { d["Mounts"].([]any)[0].(map[string]any)["Source"] = "/other/project" },
		"duplicate-mount-config": func(d map[string]any) {
			h := d["HostConfig"].(map[string]any)
			h["Mounts"] = append(h["Mounts"].([]any), h["Mounts"].([]any)[0])
		},
		"duplicate-mount-live": func(d map[string]any) { d["Mounts"] = append(d["Mounts"].([]any), d["Mounts"].([]any)[0]) },
		"host-socket": func(d map[string]any) {
			d["Mounts"] = append(d["Mounts"].([]any), map[string]any{"Type": "bind", "Source": "/run/docker.sock", "Destination": "/var/run/docker.sock", "RW": true, "Propagation": "rprivate"})
		},
		"propagation": func(d map[string]any) { d["Mounts"].([]any)[0].(map[string]any)["Propagation"] = "rshared" },
		"bind-option": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Mounts"].([]any)[0].(map[string]any)["BindOptions"] = map[string]any{"Propagation": "rprivate", "CreateMountpoint": true}
		},
		"partial-tmpfs-report":  func(d map[string]any) { d["Mounts"] = d["Mounts"].([]any)[:9] },
		"writable-image-volume": func(d map[string]any) { d["Mounts"].([]any)[9].(map[string]any)["RW"] = true },
		"unbounded-temp":        func(d map[string]any) { d["HostConfig"].(map[string]any)["Tmpfs"].(map[string]string)["/tmp"] = "rw" },
		"extra-temp": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["Tmpfs"].(map[string]string)["/extra"] = "rw,size=1m"
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			change(d)
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("unexpected storage admitted")
			}
		})
	}
}

func TestPrivateGatewayContainerRejectsPublicationAndEndpointChanges(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"wildcard": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["PortBindings"].(map[string]any)["18081/tcp"] = []any{map[string]string{"HostIp": "0.0.0.0", "HostPort": "18381"}}
		},
		"ipv6": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["PortBindings"].(map[string]any)["18081/tcp"] = []any{map[string]string{"HostIp": "::1", "HostPort": "18381"}}
		},
		"other-port": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["PortBindings"].(map[string]any)["18081/tcp"] = []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "19381"}}
		},
		"rpc": func(d map[string]any) {
			d["HostConfig"].(map[string]any)["PortBindings"].(map[string]any)["25432/tcp"] = []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "25432"}}
		},
		"missing-active": func(d map[string]any) { d["NetworkSettings"].(map[string]any)["Ports"] = map[string]any{} },
		"extra-active": func(d map[string]any) {
			d["NetworkSettings"].(map[string]any)["Ports"] = map[string]any{"25432/tcp": []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "25432"}}}
		},
		"another-network": func(d map[string]any) {
			d["NetworkSettings"].(map[string]any)["Networks"].(map[string]any)["foreign"] = map[string]any{"NetworkID": strings.Repeat("d", 64)}
		},
		"no-network": func(d map[string]any) { d["NetworkSettings"].(map[string]any)["Networks"] = nil },
	} {
		t.Run(name, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			change(d)
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("changed publication admitted")
			}
		})
	}
	for field, value := range map[string]any{"NetworkID": strings.Repeat("d", 64), "EndpointID": strings.Repeat("d", 64), "Gateway": "172.22.0.3", "IPAddress": "172.22.0.4", "IPPrefixLen": 24, "IPv6Gateway": "fd00::1", "GlobalIPv6Address": "fd00::2", "GlobalIPv6PrefixLen": 64} {
		t.Run(field, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			d["NetworkSettings"].(map[string]any)["Networks"].(map[string]any)["urbit-private-project-network-g1-bridge"].(map[string]any)[field] = value
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("changed endpoint admitted")
			}
		})
	}
	for name, change := range map[string]func(*privateGatewayAllocation){"foreign-network": func(a *privateGatewayAllocation) { a.NetworkID = strings.Repeat("d", 64) }, "foreign-container": func(a *privateGatewayAllocation) { a.ContainerID = strings.Repeat("d", 64) }, "missing-endpoint": func(a *privateGatewayAllocation) { a.EndpointID = "" }} {
		t.Run(name, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			change(&a)
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("allocation was not bound to this inspection")
			}
		})
	}
	c, created, _ := privateContainerFixture(t, false)
	_, running, a := privateContainerFixture(t, true)
	created["NetworkSettings"] = running["NetworkSettings"]
	if err := privateContainerCheck(t, c, created, a, false); err == nil {
		t.Fatal("live publication on never-started container admitted")
	}
}

func TestPrivateGatewayContainerRejectsRestartedAndUncertainState(t *testing.T) {
	for field, value := range map[string]any{"Status": "restarting", "Pid": 0, "Running": false, "OOMKilled": true, "ExitCode": 1, "Error": "private diagnostic", "Paused": true, "Restarting": true, "Dead": true, "StartedAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano), "FinishedAt": "2026-10-25T11:00:00Z"} {
		t.Run(field, func(t *testing.T) {
			c, d, a := privateContainerFixture(t, true)
			d["State"].(map[string]any)[field] = value
			if err := privateContainerCheck(t, c, d, a, true); err == nil {
				t.Fatal("changed process state admitted")
			}
		})
	}
	c, d, a := privateContainerFixture(t, false)
	d["State"].(map[string]any)["Status"] = "exited"
	if err := privateContainerCheck(t, c, d, a, false); err == nil {
		t.Fatal("exited container adopted as never started")
	}
	c, d, a = privateContainerFixture(t, true)
	raw := string(privateContainerData(t, d))
	ambiguous := strings.Replace(raw, "\"ReadonlyRootfs\":true", "\"Privileged\":true,\"privileged\":false,\"ReadonlyRootfs\":true", 1)
	if ambiguous == raw {
		t.Fatal("failed ambiguity fixture")
	}
	if _, err := c.inspectContainer([]byte(ambiguous), strings.Repeat("a", 64), strings.Repeat("b", 64), a, true); err == nil {
		t.Fatal("ambiguous container observation admitted")
	}
}

// --tmpfs is represented in HostConfig.Tmpfs, not Container.MountPoints.
// Match Docker's real eight-bind inspection shape in either lifecycle phase.
func TestPrivateGatewayTmpfsHostConfigRepresentation(t *testing.T) {
	for _, running := range []bool{false, true} {
		c, d, a := privateContainerFixture(t, running)
		d["Mounts"] = d["Mounts"].([]any)[:8]
		if err := privateContainerCheck(t, c, d, a, running); err != nil {
			t.Fatalf("running=%v: --tmpfs configuration rejected: %v", running, err)
		}
		for _, target := range []string{"/tmp", "/urbit"} {
			configured := d["HostConfig"].(map[string]any)["Tmpfs"].(map[string]string)
			saved := configured[target]
			delete(configured, target)
			if err := privateContainerCheck(t, c, d, a, running); err == nil {
				t.Fatalf("running=%v: missing %s configuration accepted", running, target)
			}
			configured[target] = saved
		}
		d["Mounts"] = d["Mounts"].([]any)[:7]
		if err := privateContainerCheck(t, c, d, a, running); err == nil {
			t.Fatal("missing bind accepted")
		}
	}
}
