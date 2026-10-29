//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectMemberRetirementAdapterPreservesPierAndRefusesUncertainty(t *testing.T) {
	for _, mode := range []string{"pass", "lost-reply", "bad-ack", "cancel-remove", "remove-error", "no-removal", "peer-before", "peer-after", "unknown-after", "restarted-before", "pier-before", "pier-after", "lock-before", "linked-state", "commit-dispatch", "commit-removed", "commit-absent", "caller-alias", "duplicate", "stopped-peer", "stopped-peer-changed"} {
		t.Run(mode, func(t *testing.T) {
			f := memberStopAdapterFixture(t)
			if strings.HasPrefix(mode, "stopped-peer") {
				first := f.open(t)
				stopped, err := projectnetwork.ExecuteMemberStop(context.Background(), f.selection.Project.Plan, f.selection.Stop, first, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
				if err != nil {
					t.Fatal(err)
				}
				ready := f.readiness["receiver"]
				next, err := projectnetwork.NewInitialMemberStop(f.selection.Project.Plan, f.selection.Launches, ready, "ready-receiver", "stop-receiver")
				if err != nil {
					t.Fatal(err)
				}
				f.selection.Readiness, f.selection.Stop, f.selection.StoppedPeers = ready, next, []projectnetwork.MemberStop{stopped}
				f.signals = 0
			}

			stop := f.open(t)
			plan := f.selection.Project.Plan
			before, err := stop.Inspect(context.Background(), f.selection.Stop)
			if err != nil {
				t.Fatal(err)
			}
			retained, err := f.selection.Stop.Advance(plan, "planned", "stop-dispatched", &before)
			if err != nil {
				t.Fatal(err)
			}
			target := stop.target.id
			f.exit(target, 0, false)
			exit, _, err := stop.observe(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			f.selection.Stop = retained
			proof := projectnetwork.MemberStopExitEvidence{OperationID: "observed-exit", Stop: retained, Observation: exit}
			input := ProjectMemberRetirementSelection{Stop: f.selection, Exit: proof, OperationID: "retire-member"}
			path := f.paths[retained.InstanceID]
			for _, name := range []string{".urb/log", ".urb/chk"} {
				if err = os.Mkdir(filepath.Join(path, name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			sentinel := filepath.Join(path, ".urb/log/retained-data")
			data := []byte("retained disposable state\n")
			if err = os.WriteFile(sentinel, data, 0600); err != nil {
				t.Fatal(err)
			}
			originalStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			originalExecute := f.f.d.execute
			removes := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var saved projectnetwork.MemberRetirement
			var peer string
			for id := range f.containers {
				if id != target {
					peer = id
				}
			}
			f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if len(a) >= 2 && a[0] == "container" && a[1] == "rm" {
					if !reflect.DeepEqual(a, []string{"container", "rm", target}) || saved.Phase != "remove-dispatched" {
						t.Fatal("unbound, forced or uncommitted removal", a)
					}
					removes++
					if mode == "remove-error" {
						return "", errors.New("private engine removal failure")
					}
					if mode != "no-removal" {
						delete(f.containers, target)
					}
					if mode == "lost-reply" {
						return "", errors.New("private lost reply")
					}
					if mode == "bad-ack" {
						return "wrong-id", nil
					}
					if mode == "cancel-remove" {
						cancel()
					}
					if mode == "peer-after" {
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					}
					if mode == "unknown-after" {
						v := *f.containers[peer]
						v.ID = strings.Repeat("a", 64)
						v.Name = "/foreign"
						f.containers[v.ID] = &v
					}
					if mode == "pier-after" {
						if err = os.Rename(path, path+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err = os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
					}
					return target + "\n", nil
				}
				return originalExecute(ctx, args, env)
			}
			initial, raw, err := openProjectMemberRetirement(ctx, input, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			d := raw.(*projectMemberRetirementEngine)
			d.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			saved = initial
			if mode == "caller-alias" {
				input.Exit.Stop.Before.Runtime.Members[0].InstanceID = "mutated"
				if initial.Validate(plan) != nil {
					t.Fatal("input alias changed initial state")
				}
			}
			commit := func(expected string, next projectnetwork.MemberRetirement) error {
				if saved.Phase != expected {
					t.Fatal("commit phase changed")
				}
				if mode == "commit-dispatch" && next.Phase == "remove-dispatched" || mode == "commit-removed" && next.Phase == "removed" || mode == "commit-absent" && next.Phase == "absent" {
					return errors.New("commit failed")
				}
				saved = next
				if next.Phase == "remove-dispatched" {
					switch mode {
					case "stopped-peer-changed":
						*f.containers[peer].State.ExitCode = 1
					case "peer-before":
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "restarted-before":
						*f.containers[target].State.Running = true
						f.containers[target].State.Status = "running"
						f.containers[target].State.Pid = 99
					case "pier-before":
						if err = os.Rename(path, path+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err = os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
					case "lock-before":
						if err = os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("fixture\n"), 0600); err != nil {
							t.Fatal(err)
						}
					case "linked-state":
						if err = os.Link(sentinel, sentinel+"-alias"); err != nil {
							t.Fatal(err)
						}
					}
				}
				return nil
			}
			got, err := projectnetwork.ExecuteMemberRetirement(ctx, plan, initial, d, commit, time.Now)
			pass := mode == "pass" || mode == "caller-alias" || mode == "duplicate" || mode == "stopped-peer"
			if (err == nil) != pass || !reflect.DeepEqual(got, saved) || f.signals != 0 {
				t.Fatal(mode, got.Phase, removes, err)
			}
			if pass && (got.Phase != "absent" || removes != 1) {
				t.Fatal("incomplete retirement", got.Phase, removes)
			}
			if strings.HasSuffix(mode, "-before") || mode == "linked-state" || mode == "commit-dispatch" || mode == "stopped-peer-changed" {
				if removes != 0 {
					t.Fatal("changed ownership reached removal")
				}
			}
			if mode == "lost-reply" || mode == "bad-ack" || mode == "remove-error" || mode == "commit-removed" {
				if got.Phase != "remove-dispatched" {
					t.Fatal("uncertain removal lost intent")
				}
			}
			if mode == "cancel-remove" && got.Phase != "remove-dispatched" {
				t.Fatal("transport-cancelled reply became acknowledged removal")
			}
			if mode == "duplicate" {
				if err = d.RemoveStopped(context.Background(), projectnetwork.MemberRetirement{OperationID: got.OperationID, Phase: "remove-dispatched", Exit: got.Exit, Before: got.Before}); err == nil || removes != 1 {
					t.Fatal("repeated removal reached engine")
				}
			}
			if mode == "pier-before" || mode == "pier-after" {
				sentinel = filepath.Join(path+"-retained", ".urb/log/retained-data")
			}
			finalStat, statErr := os.Stat(sentinel)
			finalData, readErr := os.ReadFile(sentinel)
			if statErr != nil || readErr != nil || !os.SameFile(originalStat, finalStat) || string(finalData) != string(data) {
				t.Fatal("retirement changed pier state", statErr, readErr)
			}
		})
	}
}
