//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func privateMemberContractFixture(t *testing.T, instance, mode string) privateMemberContract {
	t.Helper()
	f := newPrivateObserverFixture(t)
	o, err := f.o.ObserveGateway(context.Background(), f.g)
	if err != nil {
		t.Fatal(err)
	}
	g := f.g.Clone()
	g.Phase = "ready"
	g.Observation = &o
	s := privateMemberSelection{Plan: f.x.p, Workspace: f.x.w, Gateway: g, InstanceID: instance, RuntimeID: strings.Repeat("e", 32), OperationID: "private-member-operation", ContainerGeneration: 1, Mode: mode}
	if mode == "resume" {
		s.ContainerGeneration = 2
		s.PierIdentity = pierstore.Identity{Device: 12, Inode: 100, UrbDevice: 12, UrbInode: 101, Birth: "1000000000:1", UrbBirth: "1000000000:2"}
	}
	c, err := newPrivateMemberContract(s, "/synthetic/private-state", o.ImageID)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func privateMemberContainerFixture(t *testing.T, c privateMemberContract) container {
	t.Helper()
	entry, args, err := privateMemberCommand(c.selection.Plan, c.selection.InstanceID, c.selection.Mode)
	if err != nil {
		t.Fatal(err)
	}
	requested := []any{}
	for _, m := range c.mounts() {
		requested = append(requested, map[string]any{"Type": m.Type, "Source": m.Source, "Target": m.Destination, "ReadOnly": !m.RW, "BindOptions": map[string]any{"Propagation": m.Propagation}})
	}
	doc := map[string]any{
		"Id": strings.Repeat("c", 64), "Name": "/" + c.name, "Image": c.imageID,
		"Config":     map[string]any{"Image": c.imageID, "User": "0:0", "WorkingDir": "/urbit", "Env": []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}, "StopSignal": "SIGTERM", "StopTimeout": 300, "Entrypoint": []string{entry}, "Cmd": args, "Labels": c.labels(), "Healthcheck": map[string]any{"Test": []string{"NONE"}}},
		"HostConfig": map[string]any{"Init": true, "ReadonlyRootfs": true, "NetworkMode": "container:" + c.selection.Gateway.ContainerID, "IpcMode": "private", "CgroupnsMode": "private", "CapDrop": []string{"ALL"}, "SecurityOpt": []string{"no-new-privileges=true"}, "Memory": 4294967296, "MemorySwap": 4294967296, "NanoCpus": 2000000000, "PidsLimit": 256, "RestartPolicy": map[string]any{"Name": "no"}, "LogConfig": map[string]any{"Type": "local", "Config": map[string]string{"max-size": "10m", "max-file": "2"}}, "Tmpfs": privateMemberTmpfs(), "Mounts": requested},
		"Mounts":     c.mounts(), "NetworkSettings": map[string]any{},
		"State": map[string]any{"Status": "created", "Running": false, "ExitCode": 0, "OOMKilled": false, "StartedAt": "0001-01-01T00:00:00Z", "FinishedAt": "0001-01-01T00:00:00Z"},
	}
	raw, _ := json.Marshal(doc)
	var v container
	if json.Unmarshal(raw, &v) != nil {
		t.Fatal("fixture decode")
	}
	return v
}
func privateMemberContainerBytes(t *testing.T, v container) []byte {
	t.Helper()
	raw, err := json.Marshal([]container{v})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPrivateMemberContractBootAndResumeBindings(t *testing.T) {
	for _, instance := range []string{"sender", "receiver"} {
		for _, mode := range []string{"boot", "resume"} {
			t.Run(instance+"/"+mode, func(t *testing.T) {
				c := privateMemberContractFixture(t, instance, mode)
				args := c.createArgs()
				wantMount := c.root + "/" + c.selection.Workspace.Path + "/piers/" + instance
				if c.mountSource != wantMount || !slices.Contains(args, "container:"+c.selection.Gateway.ContainerID) || !slices.Contains(args, "--read-only") || !slices.Contains(args, "--init") {
					t.Fatal("project namespace or member mount changed")
				}
				for _, flag := range []string{"--publish", "--publish-all", "--volume", "--privileged"} {
					if slices.Contains(args, flag) {
						t.Fatal("extra authority", flag)
					}
				}
				entry, cmd, err := privateMemberCommand(c.selection.Plan, instance, mode)
				if err != nil || entry != "/bin/urbit" || !reflect.DeepEqual(args[len(args)-len(cmd):], cmd) {
					t.Fatal("keyed command changed")
				}
				if mode == "boot" {
					if len(c.mounts()) != 3 || !strings.HasSuffix(c.keyPath, "/secrets/"+strings.TrimPrefix(cmd[len(cmd)-1], "/urbit/")+".key") {
						t.Fatal("wrong bootstrap binding")
					}
					for _, m := range c.mounts()[1:] {
						if m.RW {
							t.Fatal("bootstrap input writable")
						}
					}
				} else if len(c.mounts()) != 1 || c.keyPath != "" || c.pillPath != "" || slices.Contains(cmd, "-k") || slices.Contains(cmd, "-B") || slices.Contains(cmd, "-c") {
					t.Fatal("resume regained bootstrap input")
				}
				v := privateMemberContainerFixture(t, c)
				if _, err = c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, false); err != nil {
					t.Fatal(err)
				}
				v.Mounts = append(v.Mounts, mount{Type: "tmpfs", Destination: "/tmp", RW: true})
				if _, err = c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, false); err != nil {
					t.Fatal("reported tmpfs rejected", err)
				}
				v.Mounts = append(v.Mounts, v.Mounts[len(v.Mounts)-1])
				if _, err = c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, false); err == nil {
					t.Fatal("duplicate tmpfs accepted")
				}
			})
		}
	}
}
func TestPrivateMemberContractRefusesUnboundSelection(t *testing.T) {
	changes := map[string]func(*privateMemberSelection){
		"runtime": func(s *privateMemberSelection) { s.RuntimeID = "not-a-token" }, "zero-runtime": func(s *privateMemberSelection) { s.RuntimeID = strings.Repeat("0", 32) }, "instance": func(s *privateMemberSelection) { s.InstanceID = "other" },
		"operation": func(s *privateMemberSelection) { s.OperationID = s.Gateway.Selection.OperationID }, "workspace-operation": func(s *privateMemberSelection) { s.OperationID = s.Workspace.OperationID }, "generation": func(s *privateMemberSelection) { s.ContainerGeneration = 0 }, "mode": func(s *privateMemberSelection) { s.Mode = "auto" },
		"gateway-project": func(s *privateMemberSelection) { s.Gateway.Selection.Binding.ProjectID = "private-project-other" }, "gateway-engine": func(s *privateMemberSelection) { s.Gateway.Selection.EngineID = "other-engine" }, "gateway-phase": func(s *privateMemberSelection) { s.Gateway.Phase = "started"; s.Gateway.Observation = nil }, "gateway-namespace": func(s *privateMemberSelection) { s.Gateway.Observation.Namespace = s.Gateway.Observation.HostNamespace }, "gateway-resource": func(s *privateMemberSelection) { s.Gateway.ContainerID = strings.Repeat("d", 64) },
		"mount-path": func(s *privateMemberSelection) { s.Workspace.Mounts[0].Path = "fakeships/other" }, "mount-object": func(s *privateMemberSelection) { s.Workspace.Observation.Mounts[0].Inode++ }, "boot-generation": func(s *privateMemberSelection) { s.ContainerGeneration = 2 }, "boot-pier": func(s *privateMemberSelection) { s.PierIdentity.Inode = 100 },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			c := privateMemberContractFixture(t, "sender", "boot")
			s := c.selection
			change(&s)
			if _, err := newPrivateMemberContract(s, c.root, c.imageID); err == nil {
				t.Fatal("unbound selection accepted")
			}
		})
	}
	for _, root := range []string{"/", "relative", "/a/../b", "/with,option", "/with\"quote", "/with\\slash", "/with\nnewline"} {
		c := privateMemberContractFixture(t, "sender", "boot")
		if _, err := newPrivateMemberContract(c.selection, root, c.imageID); err == nil {
			t.Fatal("mount argument injection accepted")
		}
	}
	for _, bad := range []string{"unpinned:latest", "sha256:" + strings.Repeat("0", 64)} {
		c := privateMemberContractFixture(t, "sender", "boot")
		if _, err := newPrivateMemberContract(c.selection, c.root, bad); err == nil {
			t.Fatal("wrong image accepted")
		}
	}
}
func TestPrivateMemberContractResumeRequiresDistinctRetainedPier(t *testing.T) {
	for name, change := range map[string]func(*privateMemberSelection){"missing": func(s *privateMemberSelection) { s.PierIdentity = pierstore.Identity{} }, "first-generation": func(s *privateMemberSelection) { s.ContainerGeneration = 1 }, "birth": func(s *privateMemberSelection) { s.PierIdentity.Birth = "arbitrary" }, "urb-birth": func(s *privateMemberSelection) { s.PierIdentity.UrbBirth = "1:000" }, "same-urb": func(s *privateMemberSelection) {
		s.PierIdentity.UrbInode = s.PierIdentity.Inode
		s.PierIdentity.UrbBirth = s.PierIdentity.Birth
	}, "parent-alias": func(s *privateMemberSelection) {
		s.PierIdentity.Inode = s.Workspace.Root.Inode
		s.PierIdentity.Birth = s.Workspace.Root.Birth
	}} {
		t.Run(name, func(t *testing.T) {
			c := privateMemberContractFixture(t, "sender", "resume")
			s := c.selection
			change(&s)
			if _, err := newPrivateMemberContract(s, c.root, c.imageID); err == nil {
				t.Fatal("invalid retained pier accepted")
			}
		})
	}
}
func TestPrivateMemberContractOwnsSelectionAndDigest(t *testing.T) {
	c := privateMemberContractFixture(t, "sender", "boot")
	s := c.selection
	other, err := newPrivateMemberContract(s, c.root, c.imageID)
	if err != nil || other.digest != c.digest {
		t.Fatal("contract is not deterministic")
	}
	before := other.createArgs()
	s.Plan.Members[0].HTTPPort++
	s.Workspace.Mounts[0].Path = "changed"
	s.Gateway.Observation.PublicKeys[0].Auth = "changed"
	s.Gateway.Observation.Foundation.Contracts[0].Address = "changed"
	if !reflect.DeepEqual(before, other.createArgs()) || other.selection.Workspace.Mounts[0].Path == "changed" || other.selection.Gateway.Observation.PublicKeys[0].Auth == "changed" {
		t.Fatal("caller mutated retained contract")
	}
	for _, field := range []string{"operation", "runtime", "generation", "pier"} {
		c := privateMemberContractFixture(t, "sender", "resume")
		s := c.selection
		switch field {
		case "operation":
			s.OperationID = "other-operation"
		case "runtime":
			s.RuntimeID = strings.Repeat("d", 32)
		case "generation":
			s.ContainerGeneration++
		case "pier":
			s.PierIdentity.Inode += 2
		}
		next, err := newPrivateMemberContract(s, c.root, c.imageID)
		if err != nil || next.digest == c.digest {
			t.Fatal("changed ownership retained digest", field, err)
		}
	}
	labels := other.labels()
	labels[projectLabel+".operation"] = "changed"
	if other.labels()[projectLabel+".operation"] == "changed" {
		t.Fatal("labels alias caller")
	}
}
func TestPrivateMemberContractRejectsChangedContainer(t *testing.T) {
	changes := map[string]func(*container){
		"id": func(v *container) { v.ID = strings.Repeat("d", 64) }, "name": func(v *container) { v.Name = "/other" }, "image": func(v *container) { v.Config.Image = "latest" }, "labels": func(v *container) { v.Config.Labels[projectLabel+".project"] = "other" }, "command": func(v *container) { v.Config.Cmd = append(v.Config.Cmd, "-F") }, "environment": func(v *container) { v.Config.Env = append(v.Config.Env, "SECRET=untrusted") },
		"gateway": func(v *container) { v.HostConfig.NetworkMode = "host" }, "rootfs": func(v *container) { v.HostConfig.ReadonlyRootfs = false }, "privileged": func(v *container) { v.HostConfig.Privileged = true }, "restart": func(v *container) { v.HostConfig.RestartPolicy.Name = "always" }, "swap": func(v *container) { v.HostConfig.MemorySwap = -1 }, "cpu": func(v *container) { v.HostConfig.NanoCpus = 0 }, "pids": func(v *container) { v.HostConfig.PidsLimit = nil }, "tmpfs": func(v *container) { v.HostConfig.Tmpfs["/tmp"] = "rw" }, "extra-tmpfs": func(v *container) { v.HostConfig.Tmpfs["/root"] = "rw" },
		"missing-bind": func(v *container) { v.Mounts = v.Mounts[1:] }, "writable-key": func(v *container) { v.Mounts[1].RW = true }, "other-pier": func(v *container) { v.Mounts[0].Source += "-other" }, "extra-mount": func(v *container) {
			v.Mounts = append(v.Mounts, mount{Type: "bind", Source: "/etc", Destination: "/extra", RW: true})
		}, "host-bind": func(v *container) { v.HostConfig.Binds = []string{"/etc:/extra"} }, "requested-readonly": func(v *container) {
			v.HostConfig.Mounts[1] = json.RawMessage(`{"Type":"bind","Source":"/wrong","Target":"/run/omabit-boot/key","ReadOnly":false,"BindOptions":{"Propagation":"rprivate"}}`)
		},
		"running": func(v *container) { *v.State.Running = true }, "oom": func(v *container) { *v.State.OOMKilled = true }, "dead": func(v *container) { v.State.Dead = true }, "autoremove": func(v *container) { v.HostConfig.AutoRemove = true }, "publish-all": func(v *container) { v.HostConfig.PublishAllPorts = true },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			c := privateMemberContractFixture(t, "sender", "boot")
			v := privateMemberContainerFixture(t, c)
			id := v.ID
			change(&v)
			if _, err := c.inspectContainer(privateMemberContainerBytes(t, v), id, false); err == nil {
				t.Fatal("changed container accepted")
			}
		})
	}
	c := privateMemberContractFixture(t, "sender", "resume")
	v := privateMemberContainerFixture(t, c)
	v.Mounts = append(v.Mounts, mount{Type: "bind", Source: "/secret", Destination: "/run/omabit-boot/key", Propagation: "rprivate"})
	if _, err := c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, false); err == nil {
		t.Fatal("resume key mount accepted")
	}
}
func TestPrivateMemberContractProcessStatesAndMalformedInspection(t *testing.T) {
	c := privateMemberContractFixture(t, "sender", "boot")
	v := privateMemberContainerFixture(t, c)
	start := time.Now().UTC().Add(-time.Minute)
	*v.State.Running = true
	v.State.Status = "running"
	v.State.Pid = 90
	v.State.StartedAt = start.Format(time.RFC3339Nano)
	if _, err := c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.inspectContainer(privateMemberContainerBytes(t, v), v.ID, false); err == nil {
		t.Fatal("running accepted as created")
	}
	*v.State.Running = false
	v.State.Status = "exited"
	v.State.Pid = 0
	v.State.FinishedAt = start.Add(time.Second).Format(time.RFC3339Nano)
	if err := c.validateExitedContainer(v, v.ID, start); err != nil {
		t.Fatal(err)
	}
	if err := c.validateExitedContainer(v, v.ID, start.Add(time.Second)); err == nil {
		t.Fatal("other process exit accepted")
	}
	*v.State.ExitCode = 1
	if err := c.validateExitedContainer(v, v.ID, start); err != nil {
		t.Fatal("failure evidence must remain observable", err)
	}
	original := privateMemberContainerBytes(t, privateMemberContainerFixture(t, c))
	for _, raw := range [][]byte{nil, []byte(`null`), []byte(`[]`), append(original, original...), bytesRepeatForPrivateMember(), []byte(strings.Replace(string(original), `"Id":`, `"ID":"duplicate","Id":`, 1))} {
		if _, err := c.inspectContainer(raw, v.ID, false); err == nil {
			t.Fatal("malformed inspection accepted")
		}
	}
	if _, err := c.inspectContainer(original, c.selection.Gateway.ContainerID, false); err == nil {
		t.Fatal("gateway treated as member")
	}
}
func bytesRepeatForPrivateMember() []byte { return []byte(strings.Repeat(" ", 1<<20+1)) }
