//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectCreatedRecoveryStartRevalidatesExactReturnedID(t *testing.T) {
	for _, fault := range []string{"", "start-lost-reply", "commit-start-dispatched", "missing-container", "already-running", "changed-label", "changed-command", "foreign-mount", "pier-lock", "anchor-namespace", "changed-after-inspection"} {
		t.Run(fault, func(t *testing.T) {
			f := newAnchorReplacementAdapterFixture(t)
			f.mode = "create-returned-id-error"
			h := anchorDrainHistory(f.selection.Source)
			commit := func(seq uint64, next projectnetwork.AnchorReplacement) error {
				if seq != f.saved.Sequence || next.Validate(h) != nil {
					t.Fatal("invalid continuation commit")
				}
				f.saved = next.Clone()
				return nil
			}
			created, err := projectnetwork.ExecuteNextAnchorReplacement(context.Background(), h, f.saved, f.open(t, f.saved), commit, time.Now)
			if err == nil || created.Members[0].Phase != "created" || f.creates != 1 || f.starts != 0 {
				t.Fatal("missing exact returned ID", err)
			}
			selected := f.selection
			selected.Replacement = created
			if _, err = openProjectAnchorReplacement(context.Background(), selected, f.f.x.f.f.d); err == nil {
				t.Fatal("ordinary reopening acquired continuation authority")
			}

			f.mode = ""
			id := created.Members[0].Container.ID
			target := f.f.x.f.containers[id]
			sentinel := f.f.x.sentinel
			before, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "start-lost-reply":
				f.mode = fault
			case "missing-container":
				delete(f.f.x.f.containers, id)
				execute := f.f.x.f.f.d.execute
				f.f.x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					if len(a) > 2 && a[0] == "container" && a[1] == "inspect" {
						for _, value := range a[2:] {
							if value == id {
								return "", errors.New("synthetic exact container missing")
							}
						}
					}
					return execute(ctx, args, env)
				}
			case "already-running":
				*target.State.Running = true
				target.State.Status = "running"
				target.State.Pid = 500
				target.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			case "changed-label":
				target.Config.Labels["org.urbit-runner.project.operation"] = "foreign"
			case "changed-command":
				target.Config.Cmd = []string{"foreign"}
			case "foreign-mount":
				v := projectMemberContainerFixture(t, f.contracts[0])
				v.ID = fmt.Sprintf("%064x", 999)
				v.Name = "/foreign"
				v.HostConfig.NetworkMode = "none"
				f.f.x.f.containers[v.ID] = &v
			case "pier-lock":
				if err = os.WriteFile(filepath.Join(f.f.x.f.paths[created.Members[0].Creation.InstanceID], ".vere.lock"), []byte("99"), 0600); err != nil {
					t.Fatal(err)
				}
			case "anchor-namespace":
				f.f.x.fault = fault
			}
			e, err := openProjectAnchorReplacementSelected(context.Background(), selected, f.f.x.f.f.d, true)
			if err != nil {
				t.Fatal(err)
			}
			e.review.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
			signals, removes := f.f.x.f.signals, len(f.f.removes)
			result, err := projectnetwork.StartCreatedAnchorReplacement(context.Background(), h, created, struct {
				projectnetwork.AnchorReplacementStartDriver
			}{e}, func(seq uint64, next projectnetwork.AnchorReplacement) error {
				if fault == "commit-start-dispatched" && next.Members[0].Phase == "start-dispatched" {
					return errors.New("synthetic commit failure")
				}
				if er := commit(seq, next); er != nil {
					return er
				}
				if fault == "changed-after-inspection" && next.Members[0].Phase == "start-dispatched" {
					target.HostConfig.Privileged = true
				}
				return nil
			}, time.Now)
			if f.creates != 1 || f.f.x.f.signals != signals || len(f.f.removes) != removes || result.Members[0].Container.ID != id {
				t.Fatal("continuation repeated creation/drain/retirement or changed ID")
			}
			if fault == "" {
				if err != nil || result.Members[0].Phase != "running" || f.starts != 1 {
					t.Fatal("known-ID Start failed", err, result.Members[0].Phase)
				}
			} else if err == nil {
				t.Fatal("unsafe continuation succeeded", fault)
			}
			if fault != "" && fault != "start-lost-reply" && f.starts != 0 {
				t.Fatal("failed recheck reached Start")
			}
			if fault == "start-lost-reply" {
				if result.Members[0].Phase != "start-dispatched" || f.starts != 1 {
					t.Fatal("uncertain Start not retained")
				}
				selected.Replacement = result
				if _, err = openProjectAnchorReplacementSelected(context.Background(), selected, f.f.x.f.f.d, true); err == nil {
					t.Fatal("uncertain Start gained new authority")
				}
				observer := f.open(t, result)
				observed, err := projectnetwork.ObserveAnchorReplacementRunning(context.Background(), h, result, observer, commit, time.Now)
				if err != nil || observed.Members[0].Phase != "running" || observed.Members[0].StartAcknowledged || f.starts != 1 {
					t.Fatal("lost reply was not resolved only by observation", err)
				}
			}
			if _, err = e.CreateReplacement(context.Background(), created, 0); err == nil {
				t.Fatal("created continuation permitted Create")
			}
			after, err := os.ReadFile(sentinel)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("pier content changed", err)
			}
		})
	}
}
