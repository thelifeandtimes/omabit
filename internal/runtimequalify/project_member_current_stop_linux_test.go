//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestCurrentProjectMemberStopAdapter(t *testing.T) {
	for _, instance := range []string{"receiver", "sender"} {
		for _, mode := range []string{"pass", "target-restarted-after-ready", "changed-intent", "stale-ready", "uncommitted"} {
			t.Run(instance+"/"+mode, func(t *testing.T) {
				f, input, _ := resumedStopAdapterFixture(t, false, 3)
				source := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness[instance]}
				p := input.Project.Plan
				original := f.f.d.execute
				f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					if len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
						return fmt.Sprintf("%d:%d", f.ns.Device, f.ns.Inode), nil
					}
					return original(ctx, args, env)
				}
				raw, err := openCurrentProjectReadiness(context.Background(), source, f.f.d)
				if err != nil {
					t.Fatal(err)
				}
				observer := raw.(*projectCurrentReadinessEngine)
				observer.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				ready, err := projectnetwork.CollectCurrentMemberReadiness(context.Background(), p, source.Launches, source.Resumes, source.Original, observer, time.Now)
				if err != nil {
					t.Fatal(err)
				}
				stop, err := projectnetwork.NewCurrentMemberStop(p, source.Launches, source.Resumes, source.Original, ready, "current-readiness", "current-stop")
				if err != nil {
					t.Fatal(err)
				}
				selected := ProjectCurrentMemberStopSelection{ReadinessSource: source, Readiness: ready, Stop: stop}
				if mode == "changed-intent" {
					selected.Stop.ReadinessOperationID = "foreign-readiness"
					selected.Stop.Storage.Identity.Inode++
				}
				if mode == "stale-ready" {
					selected.Readiness.Binding = source.Original.Binding
				}
				driver, err := openCurrentProjectMemberStop(context.Background(), selected, f.f.d)
				if mode == "changed-intent" || mode == "stale-ready" {
					if err == nil {
						t.Fatal("changed context accepted")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				e := driver.(*projectMemberStopEngine)
				e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				signals := f.signals
				if mode == "target-restarted-after-ready" {
					f.containers[e.target.id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				}
				if mode == "uncommitted" {
					if err = e.SignalStop(context.Background(), stop); err == nil || f.signals != signals {
						t.Fatal("uncommitted signal")
					}
					return
				}
				retained := stop
				result, err := projectnetwork.ExecuteMemberStop(context.Background(), p, stop, e, func(_ string, s projectnetwork.MemberStop) error { retained = s; return nil }, time.Now)
				if mode == "pass" {
					if err != nil || result.Phase != "stopped" || f.signals != signals+1 || projectnetwork.ValidateCurrentMemberStop(p, source.Launches, source.Resumes, source.Original, ready, result) != nil {
						t.Fatal("current Stop failed", err)
					}
					if result.LaunchOperationID != ready.CreatorOperationID || !result.Before.StartedAt.Equal(ready.After.StartedAt) {
						t.Fatal("lost observed creator/process")
					}
				} else if err == nil || f.signals != signals || retained.Phase != "planned" {
					t.Fatal("restarted target signalled", err)
				}
			})
		}
	}
}
