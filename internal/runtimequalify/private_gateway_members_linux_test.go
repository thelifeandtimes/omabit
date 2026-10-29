//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

type privateMembersFixture struct {
	f            *privateObserverFixture
	members      []privateGatewayMember
	containers   map[string]container
	sharing      map[string]privateSharingRow
	sharingCalls int
	change       func()
	projection   string
}

func newPrivateMembersFixture(t *testing.T, mode, state string, count int) *privateMembersFixture {
	t.Helper()
	f := newPrivateObserverFixture(t)
	r := f.o.runtime
	o, err := f.o.ObserveGateway(context.Background(), f.g)
	if err != nil {
		t.Fatal(err)
	}
	g := f.g.Clone()
	g.Phase = "ready"
	g.Observation = &o
	x := &privateMembersFixture{f: f, containers: map[string]container{}, sharing: map[string]privateSharingRow{r.containerID: {ID: r.containerID, Name: "/" + r.contract.network.gatewayName(), Mode: r.networkID}}}
	for n := 0; n < count; n++ {
		s := privateMemberSelection{Plan: r.plan, Workspace: r.workspace, Gateway: g, InstanceID: r.plan.Members[n].InstanceID, RuntimeID: strings.Repeat(string(rune('e'+n)), 32), OperationID: "member-" + r.plan.Members[n].InstanceID, ContainerGeneration: 1, Mode: mode}
		if mode == "resume" {
			s.ContainerGeneration = 2
			s.PierIdentity = pierstore.Identity{Device: 12, Inode: uint64(200 + n*2), UrbDevice: 12, UrbInode: uint64(201 + n*2), Birth: "1000000000:1", UrbBirth: "1000000000:2"}
		}
		c, e := newPrivateMemberContract(s, r.q.base, o.ImageID)
		if e != nil {
			t.Fatal(e)
		}
		for _, m := range c.mounts() {
			if m.Destination == "/urbit" {
				e = os.MkdirAll(m.Source, 0700)
			} else {
				e = os.MkdirAll(filepath.Dir(m.Source), 0700)
				if e == nil {
					e = os.WriteFile(m.Source, []byte("synthetic fixture only"), 0600)
				}
			}
			if e != nil {
				t.Fatal(e)
			}
		}
		v := privateMemberContainerFixture(t, c)
		v.ID = strings.Repeat(string(rune('c'+n)), 64)
		m := privateGatewayMember{Contract: c, ContainerID: v.ID, State: state}
		if state != "created" {
			m.StartedAt = time.Date(2026, 10, 26, 1, 2, n, 0, time.UTC)
			v.State.StartedAt = m.StartedAt.Format(time.RFC3339Nano)
			v.State.Status = state
			if state == "running" {
				*v.State.Running = true
				v.State.Pid = 500 + n
			} else {
				v.State.FinishedAt = m.StartedAt.Add(time.Minute).Format(time.RFC3339Nano)
			}
		}
		x.members = append(x.members, m)
		x.containers[v.ID] = v
		x.sharing[v.ID] = privateSharingRow{ID: v.ID, Name: v.Name, Mode: v.HostConfig.NetworkMode}
		f.x.inventory[v.ID] = slices.Clone(v.Mounts)
	}
	original := r.q.d.execute
	r.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		if len(a) == 3 && a[0] == "container" && a[1] == "inspect" {
			if v, ok := x.containers[a[2]]; ok {
				return string(privateMemberContainerBytes(t, v)), nil
			}
			if a[2] != r.containerID {
				return "", errors.New("synthetic container absent")
			}
		}
		if len(a) >= 5 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", privateMemberInventoryFormat}) {
			x.sharingCalls++
			var lines []string
			for _, id := range a[4:] {
				row := x.sharing[id]
				raw, _ := json.Marshal(map[string]any{"id": row.ID, "name": row.Name, "network_mode": row.Mode, "mounts": x.f.x.inventory[id]})
				lines = append(lines, string(raw))
			}
			out := strings.Join(lines, "\n")
			if x.projection == "duplicate-key" {
				out = strings.Replace(out, `"id":`, `"id":"bad","id":`, 1)
			}
			if x.projection == "extra-row" {
				out += "\n" + lines[0]
			}
			if x.projection == "unknown-field" {
				out = strings.Replace(out, `"id":`, `"unexpected":true,"id":`, 1)
			}
			if x.sharingCalls == 1 && x.change != nil {
				x.change()
			}
			return out, nil
		}
		return original(ctx, args, env)
	}
	return x
}

func TestPrivateGatewayMembersKnownStates(t *testing.T) {
	for _, mode := range []string{"boot", "resume"} {
		for _, state := range []string{"created", "running", "exited"} {
			for _, count := range []int{0, 1, 2} {
				t.Run(mode+"/"+state+"/"+string(rune('0'+count)), func(t *testing.T) {
					x := newPrivateMembersFixture(t, mode, state, count)
					if err := x.f.o.runtime.checkMemberMounts(context.Background(), x.members); err != nil {
						t.Fatal(err)
					}
					if x.sharingCalls != 2 || x.f.x.networkCalls+x.f.x.containerCalls+x.f.x.startCalls+x.f.x.boot.calls != 0 {
						t.Fatal("missing bracket or unexpected effect")
					}
					if count > 0 && x.f.o.runtime.checkMountsMode(context.Background(), true) == nil {
						t.Fatal("initial observer accepted a member")
					}
				})
			}
		}
	}
}

func TestPrivateGatewayMembersRefuseUnboundAndChangingEvidence(t *testing.T) {
	kinds := []string{"unknown-id", "gateway-id", "duplicate", "wrong-state", "missing-start", "wrong-start", "selection", "contract", "closed", "missing-inventory", "wrong-mount", "extra-mount", "label", "command", "namespace-config", "process", "changed-process", "changed-sharing", "changed-inventory", "changed-file", "duplicate-key", "extra-row", "unknown-field", "cancel"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateMembersFixture(t, "boot", "running", 1)
			r := x.f.o.runtime
			m := &x.members[0]
			v := x.containers[m.ContainerID]
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "unknown-id":
				m.ContainerID = strings.Repeat("f", 64)
			case "gateway-id":
				m.ContainerID = r.containerID
			case "duplicate":
				x.members = append(x.members, *m)
			case "wrong-state":
				m.State = "stopped"
			case "missing-start":
				m.StartedAt = time.Time{}
			case "wrong-start":
				m.StartedAt = m.StartedAt.Add(time.Second)
			case "selection":
				m.Contract.selection.Gateway.Selection.EngineID = "different"
			case "contract":
				m.Contract.digest = strings.Repeat("0", 64)
			case "closed":
				r.closed = true
			case "missing-inventory":
				delete(x.f.x.inventory, m.ContainerID)
			case "wrong-mount":
				x.f.x.inventory[m.ContainerID][0].RW = false
			case "extra-mount":
				x.f.x.inventory[m.ContainerID] = append(x.f.x.inventory[m.ContainerID], mount{Type: "tmpfs", Destination: "/extra", RW: true})
			case "label":
				v.Config.Labels[projectLabel+".operation"] = "foreign"
			case "command":
				v.Config.Cmd = []string{"-F", "zod"}
			case "namespace-config":
				v.HostConfig.NetworkMode = "host"
			case "process":
				v.State.Pid++
				m.StartedAt = m.StartedAt.Add(time.Second)
			case "changed-process":
				x.change = func() { z := x.containers[m.ContainerID]; z.State.Pid++; x.containers[m.ContainerID] = z }
			case "changed-sharing":
				x.change = func() { z := x.sharing[m.ContainerID]; z.Name = "/different"; x.sharing[m.ContainerID] = z }
			case "changed-inventory":
				x.change = func() { delete(x.f.x.inventory, m.ContainerID) }
			case "changed-file":
				original := r.q.d.execute
				calls := 0
				r.q.d.execute = func(ctx context.Context, a, e []string) (string, error) {
					out, err := original(ctx, a, e)
					if len(a) > 8 && a[7] == privateMemberInventoryFormat {
						calls++
						if calls == 2 {
							p := m.Contract.pillPath
							if os.Rename(p, p+".retained") != nil || os.WriteFile(p, []byte("replacement"), 0600) != nil {
								t.Fatal("fixture replacement")
							}
						}
					}
					return out, err
				}
			case "duplicate-key", "extra-row", "unknown-field":
				x.projection = kind
			case "cancel":
				cancel()
			}
			if kind == "label" || kind == "command" || kind == "namespace-config" || kind == "process" {
				x.containers[m.ContainerID] = v
			}
			if err := r.checkMemberMounts(ctx, x.members); err == nil {
				t.Fatal("invalid member evidence accepted")
			}
			if x.f.x.networkCalls+x.f.x.containerCalls+x.f.x.startCalls+x.f.x.boot.calls != 0 {
				t.Fatal("refusal mutated engine")
			}
		})
	}
}

func TestPrivateGatewayMembersForeignMounts(t *testing.T) {
	for _, kind := range []string{"unrelated", "input-readonly", "input-write", "root", "pier", "key", "pill", "pill-parent", "pill-alias", "volume"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateMembersFixture(t, "boot", "running", 1)
			r := x.f.o.runtime
			c := x.members[0].Contract
			p := filepath.Join(x.f.x.f.d.home, "unrelated")
			if err := os.Mkdir(p, 0700); err != nil {
				t.Fatal(err)
			}
			m := mount{Type: "bind", Source: p, Destination: "/foreign", RW: true, Propagation: "rprivate"}
			switch kind {
			case "input-readonly", "input-write":
				m.Source = r.contract.inputs
				m.RW = kind == "input-write"
			case "root":
				m.Source = r.contract.root
			case "pier":
				m.Source = c.mountSource
			case "key":
				m.Source = c.keyPath
				m.RW = false
			case "pill":
				m.Source = c.pillPath
				m.RW = false
			case "pill-parent":
				m.Source = filepath.Dir(c.pillPath)
			case "pill-alias":
				m.Source = filepath.Join(p, "alias")
				if err := os.Symlink(c.pillPath, m.Source); err != nil {
					t.Fatal(err)
				}
			case "volume":
				m.Type = "volume"
			}
			id := strings.Repeat("f", 64)
			x.f.x.inventory[id] = []mount{m}
			x.sharing[id] = privateSharingRow{ID: id, Name: "/foreign", Mode: "bridge"}
			err := r.checkMemberMounts(context.Background(), x.members)
			pass := kind == "unrelated" || kind == "input-readonly"
			if (err == nil) != pass {
				t.Fatal("foreign mount policy", err)
			}
		})
	}
}

func TestPrivateGatewayMembersSharingReferences(t *testing.T) {
	for _, kind := range []string{"gateway-id", "gateway-name", "gateway-slash-name", "gateway-prefix", "member-id", "member-name", "unresolved", "cycle", "unrelated", "unrelated-sharing", "missing-member", "member-alias"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateMembersFixture(t, "resume", "running", 1)
			r := x.f.o.runtime
			mid := x.members[0].ContainerID
			id := strings.Repeat("f", 64)
			ref := r.containerID
			switch kind {
			case "gateway-name":
				ref = strings.TrimPrefix(x.sharing[r.containerID].Name, "/")
			case "gateway-slash-name":
				ref = x.sharing[r.containerID].Name
			case "gateway-prefix":
				ref = r.containerID[:12]
			case "member-id":
				ref = mid
			case "member-name":
				ref = x.sharing[mid].Name
			case "unresolved":
				ref = "missing"
			case "cycle":
				ref = id
			case "unrelated-sharing":
				ref = strings.Repeat("9", 64)
				x.f.x.inventory[ref] = nil
				x.sharing[ref] = privateSharingRow{ID: ref, Name: "/unrelated-root", Mode: "bridge"}
			}
			x.f.x.inventory[id] = nil
			x.sharing[id] = privateSharingRow{ID: id, Name: "/foreign", Mode: "container:" + ref}
			if kind == "unrelated" || kind == "missing-member" || kind == "member-alias" {
				z := x.sharing[id]
				z.Mode = "bridge"
				x.sharing[id] = z
			}
			if kind == "missing-member" {
				delete(x.sharing, mid)
			}
			if kind == "member-alias" {
				z := x.sharing[mid]
				z.Mode = "container:" + r.containerID[:12]
				x.sharing[mid] = z
			}
			err := r.checkMemberMounts(context.Background(), x.members)
			pass := kind == "unrelated" || kind == "unrelated-sharing"
			if (err == nil) != pass {
				t.Fatal("namespace-sharing policy", err)
			}
		})
	}
}

func TestPrivateGatewayMembersProjectionMustMatchKnownContracts(t *testing.T) {
	for _, kind := range []string{"member-name", "gateway-name", "gateway-mode"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateMembersFixture(t, "boot", "running", 1)
			r := x.f.o.runtime
			id := r.containerID
			if kind == "member-name" {
				id = x.members[0].ContainerID
			}
			v := x.sharing[id]
			if kind == "gateway-mode" {
				v.Mode = "bridge"
			} else {
				v.Name = "/different"
			}
			x.sharing[id] = v
			if r.checkMemberMounts(context.Background(), x.members) == nil {
				t.Fatal("contradictory projection accepted")
			}
		})
	}
}
