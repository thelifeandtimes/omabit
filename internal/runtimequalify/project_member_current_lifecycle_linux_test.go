//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func currentLifecycleFixture(t *testing.T, target string, stoppedPeer bool) (*memberStopFixture, ProjectCurrentMemberStopSelection) {
	t.Helper()
	f, input, _ := resumedStopAdapterFixture(t, false, 3)
	ctx := context.Background()
	old := f.f.d.execute
	f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		if len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
			return fmt.Sprintf("%d:%d", f.ns.Device, f.ns.Inode), nil
		}
		if len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectMountFormat}) {
			inspect := append(slices.Clone(args[:4]), "container", "inspect")
			inspect = append(inspect, a[4:]...)
			raw, err := old(ctx, inspect, env)
			if err != nil {
				return "", err
			}
			var rows []container
			if json.Unmarshal([]byte(raw), &rows) != nil {
				t.Fatal("invalid mount fixture")
			}
			var output strings.Builder
			for _, v := range rows {
				b, _ := json.Marshal(projectContainerMounts{ID: v.ID, Mounts: v.Mounts})
				output.Write(b)
				output.WriteByte('\n')
			}
			return output.String(), nil
		}
		return old(ctx, args, env)
	}
	selected := map[string]ProjectCurrentMemberStopSelection{}
	for _, instance := range []string{"receiver", "sender"} {
		source := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness[instance]}
		raw, err := openCurrentProjectReadiness(ctx, source, f.f.d)
		if err != nil {
			t.Fatal(err)
		}
		observer := raw.(*projectCurrentReadinessEngine)
		observer.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
		ready, err := projectnetwork.CollectCurrentMemberReadiness(ctx, input.Project.Plan, source.Launches, source.Resumes, source.Original, observer, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		stop, err := projectnetwork.NewCurrentMemberStop(input.Project.Plan, source.Launches, source.Resumes, source.Original, ready, "context-ready-"+instance, "context-stop-"+instance)
		if err != nil {
			t.Fatal(err)
		}
		selected[instance] = ProjectCurrentMemberStopSelection{ReadinessSource: source, Readiness: ready, Stop: stop}
	}
	result := selected[target]
	if stoppedPeer {
		peer := "sender"
		if target == peer {
			peer = "receiver"
		}
		s := selected[peer]
		raw, err := openCurrentProjectMemberStop(ctx, s, f.f.d)
		if err != nil {
			t.Fatal(err)
		}
		d := raw.(*projectMemberStopEngine)
		d.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
		stop, err := projectnetwork.ExecuteMemberStop(ctx, input.Project.Plan, s.Stop, d, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		result.StoppedPeers = []projectnetwork.MemberStop{stop}
	}
	return f, result
}

func TestCurrentContextLifecycleRetainsPierAndPeer(t *testing.T) {
	testCurrentContextLifecycle(t, currentLifecycleFixture)
}
func testCurrentContextLifecycle(t *testing.T, fixture func(*testing.T, string, bool) (*memberStopFixture, ProjectCurrentMemberStopSelection)) {
	for _, target := range []string{"receiver", "sender"} {
		for _, mode := range []string{"live-peer", "stopped-peer", "lost-stop-ack", "changed-exit", "stopped-peer-changed", "unrecorded-peer"} {
			t.Run(target+"/"+mode, func(t *testing.T) {
				f, input := fixture(t, target, strings.HasPrefix(mode, "stopped-peer") || mode == "unrecorded-peer")
				ctx := context.Background()
				plan := input.ReadinessSource.Project.Plan
				must := func(err error) {
					t.Helper()
					if err != nil {
						t.Fatal(err)
					}
				}
				path := f.paths[target]
				must(os.MkdirAll(filepath.Join(path, ".urb/log"), 0700))
				must(os.MkdirAll(filepath.Join(path, ".urb/chk"), 0700))
				sentinel := filepath.Join(path, ".urb/log/context-retained")
				bytes := []byte("same initialized pier\n")
				must(os.WriteFile(sentinel, bytes, 0600))
				before, err := os.Stat(sentinel)
				must(err)
				raw, err := openCurrentProjectMemberStop(ctx, input, f.f.d)
				must(err)
				stopDriver := raw.(*projectMemberStopEngine)
				stopDriver.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				id := stopDriver.target.id
				var peer string
				for key := range f.containers {
					if key != id {
						peer = key
					}
				}
				peerBefore, err := json.Marshal(f.containers[peer])
				must(err)
				if mode == "stopped-peer-changed" {
					*f.containers[peer].State.ExitCode = 1
				}
				if mode == "unrecorded-peer" {
					input.StoppedPeers = nil
					raw, err = openCurrentProjectMemberStop(ctx, input, f.f.d)
					must(err)
					stopDriver = raw.(*projectMemberStopEngine)
					stopDriver.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				}
				savedStop := input.Stop
				signals := f.signals
				signal := f.signal
				f.signal = func(ctx context.Context, got string) (string, error) {
					if savedStop.Phase != "stop-dispatched" || got != id {
						t.Fatal("signal preceded exact committed intent")
					}
					out, err := signal(ctx, got)
					if mode == "lost-stop-ack" {
						return "", errors.New("lost stop reply")
					}
					return out, err
				}
				stopped, err := projectnetwork.ExecuteMemberStop(ctx, plan, input.Stop, stopDriver, func(_ string, next projectnetwork.MemberStop) error { savedStop = next; return nil }, time.Now)
				if mode == "stopped-peer-changed" || mode == "unrecorded-peer" {
					if err == nil || f.signals != signals || savedStop.Phase != "planned" {
						t.Fatal("changed or unrecorded peer permitted a signal")
					}
					return
				}
				if mode == "lost-stop-ack" {
					if err == nil || savedStop.Phase != "stop-dispatched" {
						t.Fatal("lost reply not preserved", err)
					}
				} else {
					must(err)
					if !stopped.Clean() {
						t.Fatal("unclean Stop")
					}
				}
				if f.signals != signals+1 {
					t.Fatal("Stop was not single-dispatch")
				}
				input.Stop = savedStop
				exitRaw, err := openCurrentProjectMemberStopExit(ctx, input, f.f.d)
				must(err)
				exitDriver := exitRaw.(*projectMemberStopExitEngine)
				exitDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				if mode == "changed-exit" {
					f.containers[id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				}
				proof, err := projectnetwork.CollectMemberStopExit(ctx, plan, savedStop, "context-exit", exitDriver, time.Now)
				if mode == "changed-exit" {
					if err == nil || f.signals != signals+1 {
						t.Fatal("changed process accepted or signalled")
					}
					return
				}
				must(err)
				if !proof.Clean() || !reflect.DeepEqual(proof.Stop, savedStop) || f.signals != signals+1 {
					t.Fatal("exit altered or replayed Stop")
				}
				var savedRetirement projectnetwork.MemberRetirement
				var savedResume projectnetwork.MemberResume
				var resumeDriver *projectMemberResumeEngine
				removes, creates, starts := 0, 0, 0
				newID := strings.Repeat("7", 64)
				old := f.f.d.execute
				f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					if len(a) > 1 && a[0] == "container" {
						switch a[1] {
						case "rm":
							if savedRetirement.Phase != "remove-dispatched" || !reflect.DeepEqual(a, []string{"container", "rm", id}) {
								t.Fatal("uncommitted or nonexact removal")
							}
							removes++
							delete(f.containers, id)
							return id + "\n", nil
						case "create":
							if savedResume.Phase != "create-dispatched" || !reflect.DeepEqual(a, resumeDriver.contract.createArgs()) {
								t.Fatal("uncommitted or changed create")
							}
							for _, flag := range []string{"-F", "-B", "-c", "--force", "--volumes"} {
								if slices.Contains(a, flag) {
									t.Fatal("bootstrap or destructive flag", flag)
								}
							}
							if len(resumeDriver.contract.mounts()) != 1 || resumeDriver.contract.pillPath != "" {
								t.Fatal("replacement is not retained-pier-only")
							}
							creates++
							v := projectMemberContainerFixture(t, resumeDriver.contract)
							v.ID = newID
							f.containers[newID] = &v
							return newID + "\n", nil
						case "start":
							if savedResume.Phase != "start-dispatched" || !reflect.DeepEqual(a, []string{"container", "start", newID}) {
								t.Fatal("uncommitted or foreign start")
							}
							starts++
							v := f.containers[newID]
							*v.State.Running = true
							v.State.Status = "running"
							v.State.Pid = 150
							v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
							return newID + "\n", nil
						}
					}
					return old(ctx, args, env)
				}
				retirement, retireRaw, err := openCurrentProjectMemberRetirement(ctx, ProjectCurrentMemberRetirementSelection{Stop: input, Exit: proof, OperationID: "context-retire"}, f.f.d)
				must(err)
				retireDriver := retireRaw.(*projectMemberRetirementEngine)
				retireDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				savedRetirement = retirement
				retired, err := projectnetwork.ExecuteMemberRetirement(ctx, plan, retirement, retireDriver, func(_ string, next projectnetwork.MemberRetirement) error { savedRetirement = next; return nil }, time.Now)
				must(err)
				selection := ProjectMemberResumeSelection{CurrentStop: &input, Retirement: retired, OperationID: "context-resume"}
				conflicting := selection
				conflicting.ResumedStop = &ProjectResumedMemberStopSelection{}
				if _, _, err := openProjectMemberResume(ctx, conflicting, f.f.d); err == nil {
					t.Fatal("conflicting sources accepted")
				}
				initial, resumeRaw, err := openProjectMemberResume(ctx, selection, f.f.d)
				must(err)
				resumeDriver = resumeRaw.(*projectMemberResumeEngine)
				resumeDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				savedResume = initial
				resumed, err := projectnetwork.ExecuteMemberResume(ctx, plan, initial, resumeDriver, func(_ string, next projectnetwork.MemberResume) error { savedResume = next; return nil }, time.Now)
				must(err)
				readyRaw, err := openResumedProjectReadiness(ctx, ProjectResumedReadinessSelection{Resume: selection, Completed: resumed}, f.f.d)
				must(err)
				readyDriver := readyRaw.(*projectResumedReadinessEngine)
				readyDriver.source.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				ready, err := projectnetwork.CollectResumedMemberReadiness(ctx, plan, resumed, readyDriver, time.Now)
				must(err)
				if ready.ResumeOperationID != resumed.OperationID || ready.Storage != input.Readiness.Storage {
					t.Fatal("new readiness lost current-context creation source")
				}
				want := input.Readiness.ContainerGeneration
				if resumed.Phase != "running" || resumed.PreviousGeneration != want || resumed.After.Storage != input.Readiness.Storage || removes != 1 || creates != 1 || starts != 1 || f.signals != signals+1 {
					t.Fatal("lifecycle changed ownership or repeated an effect")
				}
				after, err := os.Stat(sentinel)
				must(err)
				data, err := os.ReadFile(sentinel)
				must(err)
				if !os.SameFile(before, after) || string(data) != string(bytes) {
					t.Fatal("pier state was changed")
				}
				peerAfter, err := json.Marshal(f.containers[peer])
				must(err)
				if string(peerBefore) != string(peerAfter) {
					t.Fatal("unrelated peer changed")
				}
			})
		}
	}
}
