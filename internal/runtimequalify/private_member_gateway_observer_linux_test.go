//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateMemberObserverFixture(t *testing.T, mode, state string, count int) (*privateMembersFixture, privatechain.Gateway, []privatechain.GatewayMemberPier, *int, *string) {
	t.Helper()
	x := newPrivateMembersFixture(t, mode, state, count)
	o := x.f.o
	g := x.members[0].Contract.selection.Gateway.Clone()
	var piers []privatechain.GatewayMemberPier
	for n, m := range x.members {
		if m.State == "created" && mode == "boot" {
			continue
		}
		id := m.Contract.selection.PierIdentity
		if mode == "boot" {
			id = pierstore.Identity{Device: 12, Inode: uint64(300 + n*2), UrbDevice: 12, UrbInode: uint64(301 + n*2), Birth: "1000000000:1", UrbBirth: "1000000000:2"}
		}
		piers = append(piers, privatechain.GatewayMemberPier{InstanceID: m.Contract.selection.InstanceID, Identity: id})
	}
	bound, err := o.runtime.bindGatewayMemberPiers(x.members, piers)
	if err != nil {
		t.Fatal(err)
	}
	held := g.Clone()
	o.memberGateway = &held
	o.members = bound
	calls := new(int)
	failure := new(string)
	original := o.runtime.q.d.execute
	o.runtime.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		if len(a) >= 4 && a[0] == "container" && a[1] == "exec" {
			if _, ok := x.containers[a[2]]; ok {
				if !reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
					t.Fatal("unexpected member command")
				}
				*calls++
				switch *failure {
				case "member-namespace":
					return "4:102", nil
				case "member-host-namespace":
					return "4:100", nil
				case "member-error":
					return "", errors.New("synthetic namespace refusal")
				case "member-namespace-change":
					if *calls > count {
						return "4:102", nil
					}
				}
				return "4:101", nil
			}
		}
		out, err := original(ctx, args, env)
		if len(a) >= 4 && a[0] == "container" && a[1] == "exec" && a[2] == o.runtime.containerID && a[3] != "/bin/stat" {
			switch *failure {
			case "member-process-change":
				v := x.containers[x.members[0].ContainerID]
				v.State.Pid++
				x.containers[v.ID] = v
			case "state-change":
				x.f.state.value.Chain.Inode++
			case "new-foreign-member":
				id := strings.Repeat("f", 64)
				x.f.x.inventory[id] = nil
				x.sharing[id] = privateSharingRow{ID: id, Name: "/unexpected", Mode: "container:" + o.runtime.containerID}
			}
		}
		return out, err
	}
	return x, g, piers, calls, failure
}

func TestPrivateMemberGatewayObserverComposesRetainedMembers(t *testing.T) {
	for _, mode := range []string{"boot", "resume"} {
		for _, state := range []string{"created", "running", "exited"} {
			for _, count := range []int{1, 2} {
				t.Run(mode+"/"+state+"/"+string(rune('0'+count)), func(t *testing.T) {
					x, g, _, calls, _ := privateMemberObserverFixture(t, mode, state, count)
					before := g.Clone()
					got, err := x.f.o.ObserveGateway(context.Background(), g)
					if err != nil || !got.CompleteContract || got.Namespace != g.Observation.Namespace {
						t.Fatal(err)
					}
					want := 0
					if state == "running" {
						want = 2 * count
					}
					if *calls != want {
						t.Fatal("missing running-member namespace bracket or executed stopped member", *calls, want)
					}
					if !reflect.DeepEqual(g, before) {
						t.Fatal("saved gateway changed")
					}
					if _, ok := any(x.f.o).(privatechain.GatewayDriver); ok {
						t.Fatal("observer exposes effects")
					}
				})
			}
		}
	}
}

func TestPrivateMemberGatewayObserverRefusesChanges(t *testing.T) {
	for _, kind := range []string{"member-namespace", "member-host-namespace", "member-error", "member-namespace-change", "member-process-change", "state-change", "new-foreign-member", "selected-gateway", "selected-namespace", "closed", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			x, g, _, _, failure := privateMemberObserverFixture(t, "boot", "running", 2)
			*failure = kind
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "selected-gateway":
				g.Observation.ObservedAt = g.Observation.ObservedAt.Add(time.Second)
			case "selected-namespace":
				v := g.Clone()
				v.Observation.Namespace.Inode++
				x.f.o.memberGateway = &v
				g = v.Clone()
			case "closed":
				x.f.o.Close()
			case "cancel":
				cancel()
			}
			got, err := x.f.o.ObserveGateway(ctx, g)
			if err == nil || !reflect.DeepEqual(got, privatechain.GatewayObservation{}) {
				t.Fatal("changed evidence produced authority", err)
			}
		})
	}
}

func TestPrivateMemberGatewayObserverPierSelection(t *testing.T) {
	for _, kind := range []string{"missing", "duplicate", "unknown", "invalid", "resume-mismatch", "created-boot-pier", "retired-peer", "cloned"} {
		t.Run(kind, func(t *testing.T) {
			mode, state := "resume", "running"
			if kind == "created-boot-pier" {
				mode, state = "boot", "created"
			}
			x, _, piers, _, _ := privateMemberObserverFixture(t, mode, state, 2)
			switch kind {
			case "missing":
				piers = piers[:1]
			case "duplicate":
				piers[1] = piers[0]
			case "unknown":
				piers[0].InstanceID = "unknown"
			case "invalid":
				piers[0].Identity = pierstore.Identity{}
			case "resume-mismatch":
				piers[0].Identity.Inode++
			case "created-boot-pier":
				piers = []privatechain.GatewayMemberPier{{InstanceID: x.members[0].Contract.selection.InstanceID, Identity: pierstore.Identity{Device: 12, Inode: 300, UrbDevice: 12, UrbInode: 301, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}}
			case "retired-peer":
				x.members = x.members[:1]
			}
			bound, err := x.f.o.runtime.bindGatewayMemberPiers(x.members, piers)
			if (err == nil) != (kind == "retired-peer" || kind == "cloned") {
				t.Fatal("pier selection", err)
			}
			if kind == "cloned" {
				x.members[0].Contract.selection.Gateway.Observation.HeadHash = "changed"
				if bound[0].Contract.selection.Gateway.Observation.HeadHash == "changed" {
					t.Fatal("caller still owns observation")
				}
			}
		})
	}
}

func TestPrivateMemberGatewayObserverConstructorRefusesUnready(t *testing.T) {
	x, g, piers, _, _ := privateMemberObserverFixture(t, "resume", "running", 1)
	g.Phase = "started"
	g.Observation = nil
	if o, err := openPrivateMemberGatewayObserver(context.Background(), x.f.o.runtime.q, "/absent-fixture", "/absent-inputs", x.f.x.p, x.f.x.w, g, x.members, piers); err == nil || o != nil {
		t.Fatal("unready constructor admitted")
	}
}
