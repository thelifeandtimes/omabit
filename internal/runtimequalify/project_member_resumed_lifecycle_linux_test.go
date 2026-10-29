//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestLaterMemberGenerationLifecycleAdapters(t *testing.T) {
	for _, generation := range []int{2, 3} {
		for _, mode := range []string{"pass", "stopped-peer", "lost-stop", "lost-remove", "lost-start", "conflicting-source", "missing-history", "changed-namespace"} {
			t.Run(fmt.Sprintf("generation%d/%s", generation, mode), func(t *testing.T) {
				f, input, sentinel := resumedStopAdapterFixture(t, mode == "stopped-peer", generation)
				ctx := context.Background()
				plan := input.Project.Plan
				must := func(err error) {
					t.Helper()
					if err != nil {
						t.Fatal(err)
					}
				}
				originalHistory, _ := json.Marshal(input.Resumes)
				bytes, err := os.ReadFile(sentinel)
				must(err)
				stat, err := os.Stat(sentinel)
				must(err)
				raw, err := openResumedProjectMemberStop(ctx, input, f.f.d)
				must(err)
				stopDriver := raw.(*projectMemberStopEngine)
				stopDriver.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				if mode == "lost-stop" {
					signal := f.signal
					f.signal = func(ctx context.Context, id string) (string, error) {
						_, err := signal(ctx, id)
						if err != nil {
							return "", err
						}
						return "", errors.New("lost Stop reply")
					}
				}
				retained, err := projectnetwork.ExecuteMemberStop(ctx, plan, input.Stop, stopDriver, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
				if (err == nil) != (mode != "lost-stop") {
					t.Fatal("wrong Stop result", err)
				}
				if mode == "lost-stop" && retained.Phase != "stop-dispatched" {
					t.Fatal("old uncertainty rewritten")
				}
				input.Stop = retained
				exitRaw, err := openResumedProjectMemberStopExit(ctx, input, f.f.d)
				must(err)
				exitDriver := exitRaw.(*projectMemberStopExitEngine)
				exitDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				exit, err := projectnetwork.CollectMemberStopExit(ctx, plan, retained, "later-exit", exitDriver, time.Now)
				must(err)
				if !reflect.DeepEqual(exit.Stop, retained) || f.signals != 1 {
					t.Fatal("exit observation rewrote or retried Stop")
				}
				removalInput := ProjectResumedMemberRetirementSelection{Stop: input, Exit: exit, OperationID: "later-retirement"}
				retirement, retireRaw, err := openResumedProjectMemberRetirement(ctx, removalInput, f.f.d)
				must(err)
				retireDriver := retireRaw.(*projectMemberRetirementEngine)
				retireDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				originalExecute := f.f.d.execute
				removes, creates, starts := 0, 0, 0
				var resumeDriver *projectMemberResumeEngine
				var savedResume projectnetwork.MemberResume
				savedRetirement := retirement
				newID := fmt.Sprintf("%064x", generation+7)
				execute := func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					if reflect.DeepEqual(a, []string{"container", "rm", stopDriver.target.id}) {
						if savedRetirement.Phase != "remove-dispatched" {
							t.Fatal("uncommitted removal")
						}
						removes++
						delete(f.containers, stopDriver.target.id)
						if mode == "lost-remove" {
							return "", errors.New("lost removal reply")
						}
						return stopDriver.target.id, nil
					}
					if len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectMountFormat}) {
						inspect := append(slices.Clone(args[:4]), "container", "inspect")
						inspect = append(inspect, a[4:]...)
						raw, err := originalExecute(ctx, inspect, env)
						if err != nil {
							return "", err
						}
						var values []container
						if err = json.Unmarshal([]byte(raw), &values); err != nil {
							t.Fatal(err)
						}
						var out strings.Builder
						for _, v := range values {
							b, _ := json.Marshal(projectContainerMounts{ID: v.ID, Mounts: v.Mounts})
							out.Write(b)
							out.WriteByte('\n')
						}
						return out.String(), nil
					}
					if len(a) > 1 && a[0] == "container" && a[1] == "create" {
						if resumeDriver == nil || savedResume.Phase != "create-dispatched" || !reflect.DeepEqual(a, resumeDriver.contract.createArgs()) || resumeDriver.contract.selection.ContainerGeneration != uint64(generation+1) || resumeDriver.contract.pillPath != "" || len(resumeDriver.contract.mounts()) != 1 {
							t.Fatal("unbound or bootstrap Create")
						}
						creates++
						v := projectMemberContainerFixture(t, resumeDriver.contract)
						v.ID = newID
						f.containers[newID] = &v
						return newID, nil
					}
					if reflect.DeepEqual(a, []string{"container", "start", newID}) {
						if savedResume.Phase != "start-dispatched" {
							t.Fatal("uncommitted Start")
						}
						starts++
						v := f.containers[newID]
						*v.State.Running = true
						v.State.Status = "running"
						v.State.Pid = 111
						v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						if mode == "lost-start" {
							return "", errors.New("lost Start reply")
						}
						return newID, nil
					}
					return originalExecute(ctx, args, env)
				}
				f.f.d.execute = execute
				// Factories freeze dependency values; install the inert command hook into
				// this already-open retirement driver as well as subsequent factories.
				retireDriver.source.source.anchor.base.q.d.execute = execute
				retired, err := projectnetwork.ExecuteMemberRetirement(ctx, plan, retirement, retireDriver, func(phase string, next projectnetwork.MemberRetirement) error {
					if phase != savedRetirement.Phase {
						t.Fatal("stale removal commit")
					}
					savedRetirement = next
					return nil
				}, time.Now)
				if (err == nil) != (mode != "lost-remove") || removes != 1 {
					t.Fatal("retirement result", retired.Phase, removes, err)
				}
				selected := ProjectMemberResumeSelection{ResumedStop: &input, Retirement: retired, OperationID: "later-resume"}
				switch mode {
				case "conflicting-source":
					selected.Stop = f.selection
				case "missing-history":
					selected.ResumedStop.Resumes = nil
				}
				initial, nextRaw, err := openProjectMemberResume(ctx, selected, f.f.d)
				if slices.Contains([]string{"lost-remove", "conflicting-source", "missing-history"}, mode) {
					if err == nil || creates != 0 || starts != 0 {
						t.Fatal("invalid later resume opened", err)
					}
				} else {
					must(err)
					resumeDriver = nextRaw.(*projectMemberResumeEngine)
					resumeDriver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
					savedResume = initial
					if initial.PreviousGeneration != uint64(generation) {
						t.Fatal("wrong predecessor generation")
					}
					result, err := projectnetwork.ExecuteMemberResume(ctx, plan, initial, resumeDriver, func(phase string, next projectnetwork.MemberResume) error {
						if phase != savedResume.Phase {
							t.Fatal("stale resume commit")
						}
						savedResume = next
						if mode == "changed-namespace" && next.Phase == "create-dispatched" {
							resumeDriver.source.source.anchor.readBootID = func() (string, error) { return "22345678-1234-1234-1234-123456789abc", nil }
						}
						return nil
					}, time.Now)
					wantSuccess := mode != "lost-start" && mode != "changed-namespace"
					if (err == nil) != wantSuccess {
						t.Fatal("resume result", result.Phase, err)
					}
					if mode == "changed-namespace" {
						if creates != 0 || starts != 0 || result.Phase != "create-dispatched" {
							t.Fatal("changed namespace reached Create")
						}
					} else if creates != 1 || starts != 1 {
						t.Fatal("wrong Create/Start count", creates, starts)
					}
					readRaw, readErr := openResumedProjectReadiness(ctx, ProjectResumedReadinessSelection{Resume: selected, Completed: result}, f.f.d)
					if !wantSuccess {
						if readErr == nil {
							t.Fatal("uncertain resume gained readiness")
						}
					} else {
						must(readErr)
						reader := readRaw.(*projectResumedReadinessEngine)
						reader.source.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
						proof, err := projectnetwork.CollectResumedMemberReadiness(ctx, plan, result, reader, time.Now)
						must(err)
						if proof.ContainerGeneration != uint64(generation+1) || proof.Storage != input.Stop.Storage || proof.Before.StartedAt != result.After.StartedAt {
							t.Fatal("readiness changed generation/pier/process")
						}
						history := append(slices.Clone(input.Resumes), result)
						current, err := projectnetwork.ResolveMemberGenerations(plan, input.Launches, history)
						must(err)
						if !reflect.DeepEqual(current.Binding, proof.Binding) {
							t.Fatal("new receipt lost causal history")
						}
					}
				}
				if f.signals != 1 {
					t.Fatal("later lifecycle signalled old Stop again")
				}
				after, err := os.Stat(sentinel)
				must(err)
				actual, err := os.ReadFile(sentinel)
				must(err)
				if !os.SameFile(after, stat) || string(actual) != string(bytes) {
					t.Fatal("saved state modified")
				}
				if mode != "missing-history" {
					retainedHistory, _ := json.Marshal(input.Resumes)
					if string(retainedHistory) != string(originalHistory) {
						t.Fatal("earlier resume history changed")
					}
				}
			})
		}
	}
}
