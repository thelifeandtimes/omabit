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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
)

func resumeAdapterFixture(t *testing.T, stoppedPeer bool) (*memberStopFixture, ProjectMemberResumeSelection, string) {
	t.Helper()
	f := memberStopAdapterFixture(t)
	ctx := context.Background()
	plan := f.selection.Project.Plan
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	if stoppedPeer {
		first := f.open(t)
		stopped, err := projectnetwork.ExecuteMemberStop(ctx, plan, f.selection.Stop, first, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
		must(err)
		ready := f.readiness["receiver"]
		next, err := projectnetwork.NewInitialMemberStop(plan, f.selection.Launches, ready, "ready-receiver", "stop-receiver")
		must(err)
		f.selection.Readiness, f.selection.Stop, f.selection.StoppedPeers = ready, next, []projectnetwork.MemberStop{stopped}
		f.signals = 0
	}
	d := f.open(t)
	before, err := d.Inspect(ctx, f.selection.Stop)
	must(err)
	stop, err := f.selection.Stop.Advance(plan, "planned", "stop-dispatched", &before)
	must(err)
	f.exit(d.target.id, 0, false)
	after, _, err := d.observe(ctx, true)
	must(err)
	f.selection.Stop = stop
	proof := projectnetwork.MemberStopExitEvidence{OperationID: "observed-exit", Stop: stop, Observation: after}
	r, err := projectnetwork.NewMemberRetirement(plan, proof, "retire-member")
	must(err)
	r, err = r.Advance(plan, "planned", "remove-dispatched", &after, nil)
	must(err)
	r, err = r.Advance(plan, "remove-dispatched", "removed", nil, nil)
	must(err)
	absent := projectnetwork.MemberAbsenceObservation{Runtime: after.Runtime, Storage: after.Storage, CapturedAt: time.Now().UTC()}
	absent.Runtime.Members = slices.Clone(after.Runtime.Members)
	absent.Runtime.Members = slices.DeleteFunc(absent.Runtime.Members, func(m projectnetwork.MemberObservation) bool { return m.InstanceID == stop.InstanceID })
	absent.Runtime.ObservedAt = absent.CapturedAt
	r, err = r.Advance(plan, "removed", "absent", nil, &absent)
	must(err)
	delete(f.containers, d.target.id)
	path := f.paths[stop.InstanceID]
	must(os.Mkdir(filepath.Join(path, ".urb/log"), 0700))
	must(os.Mkdir(filepath.Join(path, ".urb/chk"), 0700))
	sentinel := filepath.Join(path, ".urb/log/retained-data")
	must(os.WriteFile(sentinel, []byte("retained state\n"), 0600))
	return f, ProjectMemberResumeSelection{Stop: f.selection, Retirement: r, OperationID: "resume-member"}, sentinel
}
func TestProjectMemberResumeAdapterRetainsPierAndExactDispatch(t *testing.T) {
	cases := []struct {
		mode, phase     string
		creates, starts int
	}{
		{"modest-existing", "running", 1, 1}, {"low-create", "create-dispatched", 0, 0}, {"low-start", "start-dispatched", 1, 0}, {"pass", "running", 1, 1}, {"stopped-peer", "running", 1, 1}, {"late-create", "created", 1, 0}, {"no-create-id", "create-dispatched", 1, 0}, {"cancel-create", "created", 1, 0}, {"lost-start", "start-dispatched", 1, 1}, {"bad-start-ack", "start-dispatched", 1, 1}, {"cancel-start", "start-dispatched", 1, 1},
		{"peer-before", "create-dispatched", 0, 0}, {"peer-created", "created", 1, 0}, {"peer-started", "started", 1, 1}, {"stopped-peer-changed", "create-dispatched", 0, 0}, {"pier-before", "create-dispatched", 0, 0}, {"lock-before", "create-dispatched", 0, 0}, {"linked-before", "create-dispatched", 0, 0}, {"unknown-before", "create-dispatched", 0, 0}, {"namespace-before", "create-dispatched", 0, 0}, {"namespace-before-start", "start-dispatched", 1, 0}, {"container-before-start", "start-dispatched", 1, 0}, {"lock-before-start", "start-dispatched", 1, 0}, {"stale-create", "create-dispatched", 0, 0}, {"stale-start", "start-dispatched", 1, 0}, {"duplicate", "running", 1, 1}, {"caller-alias", "running", 1, 1},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			f, input, sentinel := resumeAdapterFixture(t, strings.HasPrefix(c.mode, "stopped-peer"))
			originalStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			originalBytes, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			path := f.paths[input.Stop.Stop.InstanceID]
			plan := input.Stop.Project.Plan
			old := f.f.d.execute
			var saved projectnetwork.MemberResume
			var driver *projectMemberResumeEngine
			creates, starts := 0, 0
			if c.mode == "modest-existing" {
				f.f.d.resourcePolicy = func() (resourcepolicy.Policy, error) {
					v := resourcepolicy.Default()
					v.New.Minimum.RAMMiB = 16384
					v.New.Recommended.RAMMiB = 16384
					v.Existing.Recommended.RAMMiB = 16384
					return v, nil
				}
				f.f.d.admission = func(string) (uint64, uint64, error) { return 2 << 30, 8 << 30, nil }
			}
			if c.mode == "low-create" || c.mode == "low-start" {
				f.f.d.admission = func(string) (uint64, uint64, error) {
					if c.mode == "low-create" || creates > 0 {
						return 0, 8 << 30, nil
					}
					return 2 << 30, 8 << 30, nil
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			newID := strings.Repeat("8", 64)
			var peer string
			for id := range f.containers {
				peer = id
			}
			f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectMountFormat}) {
					inspect := append(slices.Clone(args[:4]), "container", "inspect")
					inspect = append(inspect, a[4:]...)
					raw, err := old(ctx, inspect, env)
					if err != nil {
						return "", err
					}
					var values []container
					if json.Unmarshal([]byte(raw), &values) != nil {
						t.Fatal("fixture inspection malformed")
					}
					var out strings.Builder
					for _, v := range values {
						b, err := json.Marshal(projectContainerMounts{ID: v.ID, Mounts: v.Mounts})
						if err != nil {
							t.Fatal(err)
						}
						out.Write(b)
						out.WriteByte('\n')
					}
					return out.String(), nil
				}
				if len(a) > 1 && a[0] == "container" && a[1] == "create" {
					if saved.Phase != "create-dispatched" || !reflect.DeepEqual(a, driver.contract.createArgs()) || driver.contract.selection.ContainerGeneration != 2 || driver.contract.selection.Mode != "resume" || driver.contract.pillPath != "" || len(driver.contract.mounts()) != 1 {
						t.Fatal("uncommitted or bootstrap replacement", a)
					}
					for _, flag := range []string{"-F", "-B", "-c", "--force", "--volumes"} {
						if slices.Contains(a, flag) {
							t.Fatal("forbidden replacement flag", flag)
						}
					}
					creates++
					if c.mode == "duplicate" {
						if _, err := driver.Create(ctx, saved); err == nil {
							t.Fatal("same-phase Create reentered")
						}
					}
					v := projectMemberContainerFixture(t, driver.contract)
					v.ID = newID
					f.containers[newID] = &v
					if c.mode == "peer-created" {
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					}
					if c.mode == "late-create" {
						return newID + "\n", errors.New("private late Create failure")
					}
					if c.mode == "no-create-id" {
						return "", nil
					}
					if c.mode == "cancel-create" {
						cancel()
					}
					return newID + "\n", nil
				}
				if len(a) > 1 && a[0] == "container" && a[1] == "start" {
					if saved.Phase != "start-dispatched" || !reflect.DeepEqual(a, []string{"container", "start", newID}) {
						t.Fatal("uncommitted or foreign Start", a)
					}
					starts++
					if c.mode == "duplicate" {
						if err := driver.Start(ctx, saved); err == nil {
							t.Fatal("same-phase Start reentered")
						}
					}
					v := f.containers[newID]
					*v.State.Running = true
					v.State.Status = "running"
					v.State.Pid = 99
					v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					if c.mode == "peer-started" {
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					}
					if c.mode == "lost-start" {
						return "", errors.New("private lost Start reply")
					}
					if c.mode == "bad-start-ack" {
						return "wrong", nil
					}
					if c.mode == "cancel-start" {
						cancel()
					}
					return newID + "\n", nil
				}
				return old(ctx, args, env)
			}
			initial, raw, err := openProjectMemberResume(ctx, input, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			driver = raw.(*projectMemberResumeEngine)
			driver.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			saved = initial
			if c.mode == "caller-alias" {
				input.Retirement.Exit.Stop.Before.Runtime.Members[0].InstanceID = "mutated"
				if initial.Validate(plan) != nil {
					t.Fatal("input aliases result")
				}
			}
			commit := func(expected string, next projectnetwork.MemberResume) error {
				if saved.Phase != expected {
					t.Fatal("stale phase")
				}
				saved = next
				if next.Phase == "create-dispatched" {
					switch c.mode {
					case "peer-before":
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "stopped-peer-changed":
						*f.containers[peer].State.ExitCode = 1
					case "pier-before":
						if err = os.Rename(path, path+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err = os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
					case "lock-before":
						if err = os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("1"), 0600); err != nil {
							t.Fatal(err)
						}
					case "linked-before":
						if err = os.Link(sentinel, sentinel+"-linked"); err != nil {
							t.Fatal(err)
						}
					case "unknown-before":
						v := *f.containers[peer]
						v.ID = strings.Repeat("a", 64)
						v.Name = "/unknown"
						f.containers[v.ID] = &v
					case "namespace-before":
						driver.source.source.anchor.readBootID = func() (string, error) { return "22345678-1234-1234-1234-123456789abc", nil }
					case "stale-create":
						driver.before.CapturedAt = driver.before.CapturedAt.Add(-time.Minute)
					}
				}
				if next.Phase == "start-dispatched" {
					switch c.mode {
					case "namespace-before-start":
						driver.source.source.anchor.readBootID = func() (string, error) { return "22345678-1234-1234-1234-123456789abc", nil }
					case "container-before-start":
						f.containers[newID].Config.Cmd = append(f.containers[newID].Config.Cmd, "-c")
					case "lock-before-start":
						if err = os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("1"), 0600); err != nil {
							t.Fatal(err)
						}
					case "stale-start":
						driver.created.CapturedAt = driver.created.CapturedAt.Add(-time.Minute)
					}
				}
				return nil
			}
			got, err := projectnetwork.ExecuteMemberResume(ctx, plan, initial, driver, commit, time.Now)
			success := c.phase == "running"
			if (err == nil) != success || got.Phase != c.phase || creates != c.creates || starts != c.starts || f.signals != 0 {
				t.Fatal(c, got.Phase, err, creates, starts, f.signals)
			}
			if !reflect.DeepEqual(got, saved) || got.Validate(plan) != nil {
				t.Fatal("retained result mismatch")
			}
			if c.mode == "duplicate" {
				if _, err = driver.Create(context.Background(), saved); err == nil {
					t.Fatal("Create replayed")
				}
				if err = driver.Start(context.Background(), saved); err == nil {
					t.Fatal("Start replayed")
				}
			}
			retained := sentinel
			if c.mode == "pier-before" {
				retained = strings.Replace(sentinel, path, path+"-retained", 1)
			}
			info, err := os.Stat(retained)
			if err != nil || !os.SameFile(info, originalStat) {
				t.Fatal("pier file identity changed", err)
			}
			data, err := os.ReadFile(retained)
			if err != nil || !slices.Equal(data, originalBytes) {
				t.Fatal("pier bytes changed", err)
			}
		})
	}
}
