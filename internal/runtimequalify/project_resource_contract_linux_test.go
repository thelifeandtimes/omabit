//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func resourceContractFixture(t *testing.T) (projectResourceContract, bridge, container) {
	t.Helper()
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "Isolated development", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "sender", Ship: "~zod", HTTPPort: 18281}, {InstanceID: "receiver", Ship: "~nec", HTTPPort: 18282}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := newProjectResourceContract(projectSelection{RunnerID: "runner-example", ProjectID: "project-example", OperationID: "op-example", Revision: 1, Generation: 1, Plan: plan}, "engine-example", candidate.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	truth, falsity := true, false
	n := bridge{ID: strings.Repeat("a", 64), Name: c.Preparation.Bridge.Name, Driver: "bridge", Scope: "local", Internal: &falsity, EnableIPv4: &truth, Labels: c.labels(), Options: bridgeOptions(), Containers: map[string]json.RawMessage{}}
	doc := map[string]any{
		"Id": strings.Repeat("b", 64), "Name": "/" + c.Preparation.Anchor.Name, "Image": candidate.ManifestDigest,
		"Config": map[string]any{"Image": candidate.ManifestDigest, "User": "0:0", "WorkingDir": "/", "Env": []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}, "StopSignal": "SIGTERM", "StopTimeout": 10, "Entrypoint": []string{"/bin/sleep"}, "Cmd": []string{"infinity"}, "Labels": c.labels(), "Healthcheck": map[string]any{"Test": []string{"NONE"}}},
		"HostConfig": map[string]any{
			"Init": true, "ReadonlyRootfs": true, "NetworkMode": n.ID, "IpcMode": "private", "CgroupnsMode": "private", "CapDrop": []string{"ALL"}, "SecurityOpt": []string{"no-new-privileges=true"},
			"Memory": 134217728, "MemorySwap": 134217728, "NanoCpus": 250000000, "PidsLimit": 64, "RestartPolicy": map[string]any{"Name": "no"}, "LogConfig": map[string]any{"Type": "local", "Config": map[string]string{"max-size": "1m", "max-file": "2"}},
			"Mounts":       []any{map[string]any{"Type": "tmpfs", "Target": "/urbit", "TmpfsOptions": map[string]any{"SizeBytes": 1048576, "Mode": 448}}},
			"PortBindings": map[string]any{"8080/tcp": []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "18282"}}, "8081/tcp": []any{map[string]string{"HostIp": "127.0.0.1", "HostPort": "18281"}}},
		},
		"Mounts":          []any{map[string]any{"Type": "tmpfs", "Destination": "/urbit", "RW": true}},
		"NetworkSettings": map[string]any{"Networks": map[string]any{n.Name: map[string]string{"NetworkID": ""}}, "Ports": map[string]any{"34343/udp": nil, "80/tcp": nil}},
		"State":           map[string]any{"Status": "created", "Pid": 0, "StartedAt": "0001-01-01T00:00:00Z", "FinishedAt": "0001-01-01T00:00:00Z", "Running": false, "ExitCode": 0, "OOMKilled": false},
	}
	raw, _ := json.Marshal(doc)
	var v container
	if err = json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return c, n, v
}
func TestProjectResourceContractUsesOnlyReviewedPublicPortsAndEphemeralStorage(t *testing.T) {
	c, n, v := resourceContractFixture(t)
	if err := c.validateBridge(n, n.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := c.validateStoppedAnchor(v, v.ID, n.ID); err != nil {
		t.Fatal(err)
	}
	args := c.anchorArgs(n.ID)
	var publications []string
	for i, arg := range args {
		if arg == "--publish" {
			publications = append(publications, args[i+1])
		}
	}
	if !reflect.DeepEqual(publications, []string{"127.0.0.1:18282:8080/tcp", "127.0.0.1:18281:8081/tcp"}) {
		t.Fatal(publications)
	}
	for _, want := range []string{"--read-only", "--init", "type=tmpfs,destination=/urbit,tmpfs-size=1048576,tmpfs-mode=0700", n.ID, "/bin/sleep", candidate.ManifestDigest} {
		if !slices.Contains(args, want) {
			t.Fatal("missing anchor contract", want)
		}
	}
	if slices.Contains(args, "--volume") || slices.Contains(args, "--publish-all") || slices.Contains(args, "start") {
		t.Fatal("anchor preparation started work or requested implicit storage/publications")
	}
	original := c.Preparation
	for _, mutate := range []func(*projectSelection){func(s *projectSelection) { s.RunnerID = "other-runner" }, func(s *projectSelection) { s.OperationID = "other-operation" }, func(s *projectSelection) { s.Generation++ }, func(s *projectSelection) { s.Revision++ }, func(s *projectSelection) { s.Plan.Label = "Other label" }} {
		selection := c.Selection
		mutate(&selection)
		other, err := newProjectResourceContract(selection, "engine-example", candidate.ManifestDigest)
		if err != nil || other.Preparation.Bridge.ContractDigest == original.Bridge.ContractDigest || other.Preparation.Anchor.ContractDigest == original.Anchor.ContractDigest {
			t.Fatal("changed selection retained resource contracts", err)
		}
	}
}
func TestProjectAnchorRefusesAlteredContainmentAndUnownedStorage(t *testing.T) {
	cases := map[string]func(*container){
		"replacement":       func(v *container) { v.ID = strings.Repeat("e", 64) },
		"foreign-image":     func(v *container) { v.Image = "sha256:" + strings.Repeat("e", 64) },
		"labels":            func(v *container) { v.Config.Labels[projectLabel+".project"] = "foreign" },
		"command":           func(v *container) { v.Config.Cmd = []string{"changed"} },
		"environment":       func(v *container) { v.Config.Env = append(v.Config.Env, "LD_PRELOAD=/changed.so") },
		"healthcheck":       func(v *container) { v.Config.Healthcheck.Test = []string{"CMD", "/bin/urbit"} },
		"workdir":           func(v *container) { v.Config.WorkingDir = "/urbit" },
		"writable-root":     func(v *container) { v.HostConfig.ReadonlyRootfs = false },
		"privileged":        func(v *container) { v.HostConfig.Privileged = true },
		"host-pid":          func(v *container) { v.HostConfig.PidMode = "host" },
		"host-network":      func(v *container) { v.HostConfig.NetworkMode = "host" },
		"automatic-restart": func(v *container) { v.HostConfig.RestartPolicy.Name = "always" },
		"wildcard-publication": func(v *container) {
			v.HostConfig.PortBindings["8080/tcp"] = json.RawMessage(`[{"HostIp":"0.0.0.0","HostPort":"18282"}]`)
		},
		"extra-admin-port": func(v *container) {
			v.HostConfig.PortBindings["12321/tcp"] = json.RawMessage(`[{"HostIp":"127.0.0.1","HostPort":"18321"}]`)
		},
		"implicit-volume":   func(v *container) { v.HostConfig.Mounts = nil },
		"persistent-volume": func(v *container) { v.Mounts[0].Type = "volume"; v.Mounts[0].Source = "/unowned" },
		"extra-mount": func(v *container) {
			v.Mounts = append(v.Mounts, mount{Type: "bind", Source: "/unowned", Destination: "/other", RW: true})
		},
		"unbounded-tmpfs": func(v *container) {
			v.HostConfig.Mounts[0] = json.RawMessage(`{"Type":"tmpfs","Target":"/urbit","TmpfsOptions":{"SizeBytes":0,"Mode":448}}`)
		},
		"wrong-tmpfs-target": func(v *container) {
			v.HostConfig.Mounts[0] = json.RawMessage(`{"Type":"tmpfs","Target":"/other","TmpfsOptions":{"SizeBytes":1048576,"Mode":448}}`)
		},
		"unknown-tmpfs-options": func(v *container) {
			v.HostConfig.Mounts[0] = json.RawMessage(`{"Type":"tmpfs","Target":"/urbit","TmpfsOptions":{"SizeBytes":1048576,"Mode":448,"Options":[["exec"]]}}`)
		},
		"unexpected-prior-start": func(v *container) { v.State.StartedAt = "2026-10-23T12:00:00Z" },
		"unexpected-exit":        func(v *container) { v.State.Status = "exited" },
		"unexpected-pid":         func(v *container) { v.State.Pid = 42 },
		"bind-opts": func(v *container) {
			v.HostConfig.Mounts[0] = json.RawMessage(`{"Type":"tmpfs","Target":"/urbit","BindOptions":{},"TmpfsOptions":{"SizeBytes":1048576,"Mode":448}}`)
		},
		"extra-network":   func(v *container) { v.NetworkSettings.Networks["foreign"] = json.RawMessage(`{"NetworkID":"foreign"}`) },
		"already-running": func(v *container) { *v.State.Running = true },
		"active-port":     func(v *container) { v.NetworkSettings.Ports["8080/tcp"] = v.HostConfig.PortBindings["8080/tcp"] },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c, n, v := resourceContractFixture(t)
			id := v.ID
			mutate(&v)
			if err := c.validateStoppedAnchor(v, id, n.ID); err == nil {
				t.Fatal("altered anchor accepted")
			}
		})
	}
}
func TestProjectBridgeRefusesReplacementForeignAttachmentsAndTopology(t *testing.T) {
	for _, mutate := range []func(*bridge){func(n *bridge) { n.ID = strings.Repeat("e", 64) }, func(n *bridge) { n.Labels[projectLabel+".runner"] = "other" }, func(n *bridge) { n.Options["com.docker.network.bridge.host_binding_ipv4"] = "0.0.0.0" }, func(n *bridge) { *n.Internal = true }, func(n *bridge) { n.EnableIPv6 = true }, func(n *bridge) { n.Containers[strings.Repeat("e", 64)] = json.RawMessage(`{}`) }} {
		c, n, _ := resourceContractFixture(t)
		id := n.ID
		mutate(&n)
		if err := c.validateBridge(n, id, ""); err == nil {
			t.Fatal("changed bridge accepted")
		}
	}
}
