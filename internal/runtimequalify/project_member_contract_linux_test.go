//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func projectMemberContractFixture(t *testing.T) projectMemberContract {
	t.Helper()
	resource, network, anchor := resourceContractFixture(t)
	p := resource.Preparation
	p.Sequence = 6
	p.Bridge.Phase, p.Bridge.ID = "observed", network.ID
	p.Anchor.Phase, p.Anchor.ID = "observed", anchor.ID
	started := projectnetwork.AnchorStart{Prepared: p, OperationID: "start-anchor", Phase: "observed", Namespace: &projectnetwork.NamespaceIdentity{BootID: "12345678-1234-1234-1234-123456789abc", Device: 5, Inode: 42}}
	home := t.TempDir()
	if err := os.Chmod(home, 0700); err != nil {
		t.Fatal(err)
	}
	if err := pierstore.Init(home); err != nil {
		t.Fatal(err)
	}
	loc := pierstore.Location{Path: "fakeships/sender/zod"}
	mount, err := pierstore.PrepareMount(home, &loc)
	if err != nil {
		t.Fatal(err)
	}
	s := projectMemberSelection{Project: resource.Selection, Anchor: started, InstanceID: "sender", RuntimeID: strings.Repeat("e", 32), OperationID: "boot-member", ContainerGeneration: 1, Mode: "boot", Storage: loc}
	c, err := newProjectMemberContract(s, home, mount, candidate.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func projectMemberContainerFixture(t *testing.T, c projectMemberContract) container {
	t.Helper()
	entry, args, err := projectMemberCommand(c.selection.Project.Plan, c.selection.InstanceID, c.selection.Mode)
	if err != nil {
		t.Fatal(err)
	}
	requested := []any{map[string]any{"Type": "bind", "Source": c.mountSource, "Target": "/urbit", "BindOptions": map[string]any{"Propagation": "rprivate"}}}
	actual := []mount{{Type: "bind", Source: c.mountSource, Destination: "/urbit", RW: true, Propagation: "rprivate"}}
	if c.selection.Mode == "boot" {
		requested = append(requested, map[string]any{"Type": "bind", "Source": c.pillPath, "Target": "/artifacts/boot.pill", "ReadOnly": true, "BindOptions": map[string]any{"Propagation": "rprivate"}})
		actual = append(actual, mount{Type: "bind", Source: c.pillPath, Destination: "/artifacts/boot.pill", RW: false, Propagation: "rprivate"})
	}
	doc := map[string]any{
		"Id": strings.Repeat("c", 64), "Name": "/" + c.name, "Image": candidate.ManifestDigest,
		"Config":     map[string]any{"Image": candidate.ManifestDigest, "User": "0:0", "WorkingDir": "/urbit", "Env": []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bin"}, "StopSignal": "SIGTERM", "StopTimeout": 300, "Entrypoint": []string{entry}, "Cmd": args, "Labels": c.labels(), "Healthcheck": map[string]any{"Test": []string{"NONE"}}},
		"HostConfig": map[string]any{"Init": true, "NetworkMode": "container:" + strings.Repeat("b", 64), "IpcMode": "private", "CgroupnsMode": "private", "CapDrop": []string{"ALL"}, "SecurityOpt": []string{"no-new-privileges=true"}, "Memory": 4294967296, "MemorySwap": 4294967296, "NanoCpus": 2000000000, "PidsLimit": 256, "RestartPolicy": map[string]any{"Name": "no"}, "LogConfig": map[string]any{"Type": "local", "Config": map[string]string{"max-size": "10m", "max-file": "2"}}, "Mounts": requested},
		"Mounts":     actual, "NetworkSettings": map[string]any{},
		"State": map[string]any{"Status": "created", "Running": false, "ExitCode": 0, "OOMKilled": false, "StartedAt": "0001-01-01T00:00:00Z", "FinishedAt": "0001-01-01T00:00:00Z"},
	}
	raw, _ := json.Marshal(doc)
	var v container
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestProjectMemberContractBindsReservedIdentityStorageAndNamespace(t *testing.T) {
	c := projectMemberContractFixture(t)
	if err := c.checkStorageBeforeCreate(context.Background()); err != nil {
		t.Fatal(err)
	}
	args := c.createArgs()
	if !slices.Contains(args, "container:"+c.selection.Anchor.Prepared.Anchor.ID) || !slices.Contains(args, "type=bind,src="+c.mountSource+",dst=/urbit,bind-propagation=rprivate") {
		t.Fatal(args)
	}
	for _, flag := range []string{"--publish", "--publish-all", "--volume"} {
		if slices.Contains(args, flag) {
			t.Fatal("member acquired independent publication/storage or altered signal handling", flag)
		}
	}
	if !slices.Contains(args, "--init") {
		t.Fatal("member lacks the qualified init/signal forwarding contract")
	}
	if args[len(args)-1] != "/urbit/zod" {
		t.Fatal(args)
	}
	v := projectMemberContainerFixture(t, c)
	if err := c.validateContainer(v, v.ID, false); err != nil {
		t.Fatal(err)
	}
	*v.State.Running = true
	v.State.Status = "running"
	v.State.Pid = 91
	v.State.StartedAt = "2026-10-23T12:00:00Z"
	if err := c.validateContainer(v, v.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := c.validateContainer(v, v.ID, false); err == nil {
		t.Fatal("running member treated as fresh")
	}
	for name, mutate := range map[string]func(*projectMemberSelection){
		"operation":    func(s *projectMemberSelection) { s.OperationID = "other-operation" },
		"runtime":      func(s *projectMemberSelection) { s.RuntimeID = strings.Repeat("d", 32) },
		"namespace":    func(s *projectMemberSelection) { s.Anchor.Namespace.Inode++ },
		"mount-object": func(s *projectMemberSelection) { s.Storage.MountIdentity.Inode++ },
	} {
		t.Run(name, func(t *testing.T) {
			s := c.selection
			ns := *s.Anchor.Namespace
			s.Anchor.Namespace = &ns
			mutate(&s)
			other, err := newProjectMemberContract(s, c.home, c.mountSource, c.imageID)
			if err != nil || other.digest == c.digest {
				t.Fatal("changed ownership retained contract digest", err)
			}
		})
	}
	s := c.selection
	other, err := newProjectMemberContract(s, c.home, c.mountSource, c.imageID)
	if err != nil {
		t.Fatal(err)
	}
	original := other.createArgs()
	s.Project.Plan.Members[0].InternalHTTPPort++
	s.Anchor.Namespace.Inode++
	if !reflect.DeepEqual(original, other.createArgs()) || other.selection.Anchor.Namespace.Inode == s.Anchor.Namespace.Inode {
		t.Fatal("caller mutation rewrote frozen contract")
	}
}
func TestProjectMemberContractRefusesUnboundSelections(t *testing.T) {
	for name, mutate := range map[string]func(*projectMemberSelection){
		"unknown-member":       func(s *projectMemberSelection) { s.InstanceID = "unknown" },
		"runtime-name":         func(s *projectMemberSelection) { s.RuntimeID = "sender" },
		"old-start":            func(s *projectMemberSelection) { s.Anchor.Phase = "started"; s.Anchor.Namespace = nil },
		"foreign-project":      func(s *projectMemberSelection) { s.Project.ProjectID = "other" },
		"foreign-create":       func(s *projectMemberSelection) { s.Project.OperationID = "other" },
		"wrong-generation":     func(s *projectMemberSelection) { s.Project.Generation++ },
		"reused-operation":     func(s *projectMemberSelection) { s.OperationID = s.Anchor.OperationID },
		"namespace-missing":    func(s *projectMemberSelection) { s.Anchor.Namespace = nil },
		"boot-again":           func(s *projectMemberSelection) { s.ContainerGeneration = 2 },
		"uninitialized-resume": func(s *projectMemberSelection) { s.Mode = "resume"; s.ContainerGeneration = 2 },
		"uncertain-storage":    func(s *projectMemberSelection) { s.Storage.Uncertain = true },
		"unrecorded-mount":     func(s *projectMemberSelection) { s.Storage.MountIdentity = pierstore.DirectoryIdentity{} },
		"other-leaf":           func(s *projectMemberSelection) { s.Storage.Path = "fakeships/sender/pier" },
		"other-parent":         func(s *projectMemberSelection) { s.Storage.Path = "fakeships/peer/zod" },
		"bootstrap-existing":   func(s *projectMemberSelection) { s.Storage.Identity.Inode = 42 },
	} {
		t.Run(name, func(t *testing.T) {
			c := projectMemberContractFixture(t)
			s := c.selection
			mutate(&s)
			if _, err := newProjectMemberContract(s, c.home, c.mountSource, c.imageID); err == nil {
				t.Fatal("invalid binding accepted")
			}
		})
	}
	c := projectMemberContractFixture(t)
	for _, path := range []string{"/", "relative", "/a/../b", "/bad,readonly", "/bad\npath"} {
		if _, err := newProjectMemberContract(c.selection, c.home, path, c.imageID); err == nil {
			t.Fatal("unsafe Docker mount syntax accepted", path)
		}
	}
}
func TestProjectMemberStorageChecksRefuseReplacementAndPartialBootWithoutChangingData(t *testing.T) {
	for _, scenario := range []string{"partial", "sibling", "parent-replaced", "symlink", "source-mismatch", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			c := projectMemberContractFixture(t)
			ctx := context.Background()
			switch scenario {
			case "partial":
				if err := os.Mkdir(filepath.Join(c.mountSource, "zod"), 0700); err != nil {
					t.Fatal(err)
				}
			case "sibling":
				if err := os.WriteFile(filepath.Join(c.mountSource, "keep"), []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			case "parent-replaced", "symlink":
				if err := os.Rename(c.mountSource, c.mountSource+"-original"); err != nil {
					t.Fatal(err)
				}
				var err error
				if scenario == "symlink" {
					err = os.Symlink(c.mountSource+"-original", c.mountSource)
				} else {
					err = os.Mkdir(c.mountSource, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
			case "source-mismatch":
				c.mountSource += "-other"
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err := c.checkStorageBeforeCreate(ctx); err == nil {
				t.Fatal("unsafe member source accepted")
			}
			if scenario == "sibling" {
				data, err := os.ReadFile(filepath.Join(c.mountSource, "keep"))
				if err != nil || string(data) != "retained" {
					t.Fatal("data changed", err)
				}
			}
			if scenario == "partial" {
				if _, err := os.Stat(filepath.Join(c.mountSource, "zod")); err != nil {
					t.Fatal("partial pier removed", err)
				}
			}
		})
	}
}
func TestProjectMemberResumeRequiresOriginalStoppedPierAndNoPillMount(t *testing.T) {
	c := projectMemberContractFixture(t)
	pier := filepath.Join(c.mountSource, "zod")
	for _, sub := range []string{".urb/log", ".urb/chk"} {
		if err := os.MkdirAll(filepath.Join(pier, sub), 0700); err != nil {
			t.Fatal(err)
		}
	}
	identity, err := pierstore.Identify(pier)
	if err != nil {
		t.Fatal(err)
	}
	s := c.selection
	s.Mode = "resume"
	s.ContainerGeneration = 2
	s.OperationID = "resume-member"
	s.Storage.Identity = identity
	c, err = newProjectMemberContract(s, c.home, c.mountSource, c.imageID)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.checkStorageBeforeCreate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(c.mounts()) != 1 || c.pillPath != "" || slices.Contains(c.createArgs(), "-F") || slices.Contains(c.createArgs(), "-c") || slices.Contains(c.createArgs(), "-B") {
		t.Fatal("resume can bootstrap or mount a pill")
	}
	v := projectMemberContainerFixture(t, c)
	if err = c.validateContainer(v, v.ID, false); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(pier, ".vere.lock")
	if err = os.WriteFile(lock, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = c.checkStorageBeforeCreate(context.Background()); err == nil {
		t.Fatal("locked pier accepted")
	}
	if err = os.Remove(lock); err != nil {
		t.Fatal(err)
	} // private test fixture only
	if err = os.Rename(pier, filepath.Join(c.home, "saved-pier")); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{".urb/log", ".urb/chk"} {
		if err := os.MkdirAll(filepath.Join(pier, sub), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err = c.checkStorageBeforeCreate(context.Background()); err == nil {
		t.Fatal("replacement pier accepted")
	}
}
func TestProjectMemberContainerRefusesPeerMountsAndIndependentNetworkOwnership(t *testing.T) {
	c := projectMemberContractFixture(t)
	for name, mutate := range map[string]func(*container){
		"missing-init":     func(v *container) { v.HostConfig.Init = nil },
		"disabled-init":    func(v *container) { off := false; v.HostConfig.Init = &off },
		"replacement":      func(v *container) { v.ID = strings.Repeat("d", 64) },
		"peer-runtime":     func(v *container) { v.Config.Labels[projectLabel+".runtime"] = strings.Repeat("d", 32) },
		"peer-mount":       func(v *container) { v.Mounts[0].Source += "-peer" },
		"pier-only":        func(v *container) { v.Mounts[0].Source += "/zod" },
		"anonymous-volume": func(v *container) { v.Mounts[0].Type = "volume" },
		"extra-mount": func(v *container) {
			v.Mounts = append(v.Mounts, mount{Type: "bind", Source: "/home", Destination: "/home", RW: true})
		},
		"duplicate-mount":    func(v *container) { v.Mounts[1] = v.Mounts[0] },
		"mutable-pill":       func(v *container) { v.Mounts[1].RW = true },
		"shared-propagation": func(v *container) { v.Mounts[0].Propagation = "shared" },
		"unknown-bind-options": func(v *container) {
			var raw map[string]any
			json.Unmarshal(v.HostConfig.Mounts[0], &raw)
			raw["BindOptions"] = map[string]any{"Propagation": "rprivate", "CreateMountpoint": true}
			v.HostConfig.Mounts[0], _ = json.Marshal(raw)
		},
		"anchor-by-name": func(v *container) { v.HostConfig.NetworkMode = "container:anchor-name" },
		"peer-anchor":    func(v *container) { v.HostConfig.NetworkMode = "container:" + strings.Repeat("d", 64) },
		"own-bridge":     func(v *container) { v.HostConfig.NetworkMode = "bridge" },
		"published": func(v *container) {
			v.HostConfig.PortBindings = map[string]json.RawMessage{"8080/tcp": json.RawMessage(`[{"HostIp":"127.0.0.1","HostPort":"18881"}]`)}
		},
		"active-network": func(v *container) {
			v.NetworkSettings.Networks = map[string]json.RawMessage{"bridge": json.RawMessage(`{}`)}
		},
		"active-port": func(v *container) {
			v.NetworkSettings.Ports = map[string]json.RawMessage{"80/tcp": json.RawMessage(`null`)}
		},
		"wrong-http": func(v *container) { i := slices.Index(v.Config.Cmd, "--http-port"); v.Config.Cmd[i+1] = "8080" },
		"native-control": func(v *container) {
			i := slices.Index(v.Config.Cmd, "--no-conn")
			v.Config.Cmd = append(v.Config.Cmd[:i], v.Config.Cmd[i+1:]...)
		},
		"privileged":        func(v *container) { v.HostConfig.Privileged = true },
		"automatic-restart": func(v *container) { v.HostConfig.RestartPolicy.Name = "always" },
		"foreign-image":     func(v *container) { v.Config.Image = "foreign" },
		"environment":       func(v *container) { v.Config.Env = append(v.Config.Env, "LD_PRELOAD=foreign") },
		"prior-run":         func(v *container) { v.State.StartedAt = "2026-10-23T11:00:00Z" },
	} {
		t.Run(name, func(t *testing.T) {
			v := projectMemberContainerFixture(t, c)
			id := v.ID
			mutate(&v)
			if err := c.validateContainer(v, id, false); err == nil {
				t.Fatal("unsafe member contract accepted")
			}
		})
	}
}
