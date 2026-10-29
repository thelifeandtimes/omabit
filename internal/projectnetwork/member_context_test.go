package projectnetwork

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func memberContextReady(t *testing.T, instance string) (Plan, []MemberLaunch, InitialMemberReadiness) {
	t.Helper()
	p, launches, o, location := initialReadinessFixture(t)
	b, err := InitialRuntimeBinding(p, launches)
	if err != nil {
		t.Fatal(err)
	}
	creator := "launch-zod"
	ship := "~zod"
	endpoint := ""
	for _, member := range p.Members {
		if member.InstanceID == instance {
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", member.HTTPPort)
		}
	}
	for _, l := range launches {
		if l.Storage.InstanceID == instance {
			creator = l.OperationID
			if instance == "nec" {
				location = l.Storage.Location
				location.Identity = pierstore.Identity{Device: 8, Inode: 120, UrbDevice: 8, UrbInode: 121, Birth: "1000:8", UrbBirth: "1000:9"}
				ship = "~nec"
			}
		}
	}
	ready := InitialMemberReadiness{InstanceID: instance, LaunchOperationID: creator, Binding: b, Storage: location, Ship: ship, Endpoint: endpoint, Before: o, After: o, CapturedAt: time.Unix(1001, 0)}
	if err := ready.Validate(p, launches); err != nil {
		t.Fatal(err)
	}
	return p, launches, ready
}

func TestMemberContextPreservesHistoryAcrossPeerReplacement(t *testing.T) {
	for _, instance := range []string{"nec", "zod"} {
		t.Run(instance, func(t *testing.T) {
			p, launches, ready := memberContextReady(t, instance)
			_, first := resumedReadinessFixture(t)
			second := nextGenerationFixture(t, p, first)
			original := ready
			original.Binding.Members = append([]RuntimeMember(nil), ready.Binding.Members...)
			for _, history := range [][]MemberResume{nil, {first}, {first, second}, {second, first}} {
				out, err := SelectMemberContext(p, launches, history, ready)
				if err != nil {
					t.Fatal(err)
				}
				if out.Storage != ready.Storage || out.Binding.Namespace != ready.Binding.Namespace {
					t.Fatal("lost identity", out)
				}
				if instance == "nec" || len(history) == 0 {
					if out.Member.Generation != 1 || !out.KnownStartedAt.IsZero() {
						t.Fatal("invented initial process history")
					}
				} else {
					if out.Member.Generation != uint64(1+len(history)) || out.KnownStartedAt.IsZero() {
						t.Fatal("wrong current creator")
					}
				}
				if len(history) > 0 && reflect.DeepEqual(out.Binding, ready.Binding) {
					t.Fatal("old complete binding reused")
				}
				out.Binding.Members[0].InstanceID = "caller-edit"
				if !reflect.DeepEqual(ready, original) {
					t.Fatal("historical readiness mutated")
				}
			}
		})
	}
}

func TestMemberContextRefusesUnaccountedOrChangedIdentity(t *testing.T) {
	for _, fault := range []string{"unfinished", "missing-predecessor", "duplicate", "changed-pier", "wrong-creator", "wrong-namespace", "incomplete-launches"} {
		t.Run(fault, func(t *testing.T) {
			p, launches, ready := memberContextReady(t, "zod")
			_, first := resumedReadinessFixture(t)
			second := nextGenerationFixture(t, p, first)
			history := []MemberResume{first, second}
			switch fault {
			case "unfinished":
				history[1].Phase = "started"
				history[1].After = nil
			case "missing-predecessor":
				history = history[1:]
			case "duplicate":
				history = append(history, first)
			case "changed-pier":
				ready.Storage.Identity.Inode++
			case "wrong-creator":
				ready.LaunchOperationID = "foreign"
			case "wrong-namespace":
				ready.Binding.Namespace.Inode++
			case "incomplete-launches":
				launches = launches[:1]
			}
			if out, err := SelectMemberContext(p, launches, history, ready); err == nil || out.Member.InstanceID != "" {
				t.Fatal("invalid current context accepted", out, err)
			}
		})
	}
}
