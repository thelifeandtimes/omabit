package privatechain

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func workspaceFixture(t *testing.T) (Plan, Workspace) {
	t.Helper()
	p, err := Build(request(t), Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	b := Binding{Mode: Mode, RunnerID: "runner-workspace", ProjectID: "private-project-workspace", Generation: 1, FoundationDigest: p.FoundationDigest}
	w, err := NewWorkspace(p, b, 1, "workspace-operation", workspaceID(9))
	if err != nil {
		t.Fatal(err)
	}
	return p, w
}

var workspaceNow = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func workspaceID(n uint64) pierstore.DirectoryIdentity {
	return pierstore.DirectoryIdentity{Device: 12, Inode: n, Birth: "1000000000:1"}
}
func workspaceObservation() WorkspaceObservation {
	return WorkspaceObservation{ObservedAt: workspaceNow, Parent: workspaceID(9), Root: workspaceID(10), PiersRoot: workspaceID(11), Mounts: []pierstore.DirectoryIdentity{workspaceID(12), workspaceID(13)}, Complete: true, Private: true, Empty: true, Synced: true}
}

type workspaceDriver struct {
	events      *[]string
	root        pierstore.DirectoryIdentity
	observation WorkspaceObservation
	fail        string
	cancel      context.CancelFunc
	mutate      bool
}

func (d *workspaceDriver) call(name string, w Workspace) error {
	*d.events = append(*d.events, name)
	if d.mutate {
		w.Mounts[0].Path = "outside"
	}
	if d.fail == name {
		if d.cancel != nil {
			d.cancel()
		}
		return errors.New("private driver detail must not escape")
	}
	return nil
}
func (d *workspaceDriver) CreateRoot(_ context.Context, w Workspace) (pierstore.DirectoryIdentity, error) {
	return d.root, d.call("create-root", w)
}
func (d *workspaceDriver) InspectRoot(_ context.Context, w Workspace) error {
	return d.call("inspect-root", w)
}
func (d *workspaceDriver) CreateLayout(_ context.Context, w Workspace) error {
	return d.call("create-layout", w)
}
func (d *workspaceDriver) ObserveLayout(_ context.Context, w Workspace) (WorkspaceObservation, error) {
	return d.observation, d.call("observe-layout", w)
}

func TestWorkspaceDerivedLayoutAndBinding(t *testing.T) {
	p, w := workspaceFixture(t)
	if w.Path != "private-projects/private-project-workspace" {
		t.Fatal(w.Path)
	}
	for i, m := range p.Members {
		if w.Mounts[i] != (WorkspaceMount{m.InstanceID, m.Ship, w.Path + "/piers/" + m.InstanceID}) {
			t.Fatal(w.Mounts)
		}
	}
	for name, mutate := range map[string]func(*Workspace){
		"missing-parent":    func(w *Workspace) { w.Parent = pierstore.DirectoryIdentity{} },
		"project-path":      func(w *Workspace) { w.Binding.ProjectID = "private-project-other" },
		"traversal":         func(w *Workspace) { w.Path = "../outside" },
		"unsupported-mode":  func(w *Workspace) { w.Binding.Mode = "fake" },
		"foundation":        func(w *Workspace) { w.Binding.FoundationDigest = strings.Repeat("1", 64) },
		"generation":        func(w *Workspace) { w.Binding.Generation++ },
		"revision":          func(w *Workspace) { w.Revision++ },
		"missing-operation": func(w *Workspace) { w.OperationID = "" },
		"wrong-digest":      func(w *Workspace) { w.PlanDigest = strings.Repeat("1", 64) },
		"extra-mount":       func(w *Workspace) { w.Mounts = append(w.Mounts, w.Mounts[0]) },
		"member":            func(w *Workspace) { w.Mounts[0].InstanceID = "other" },
		"ship":              func(w *Workspace) { w.Mounts[0].Ship = "~bud" },
		"member-path":       func(w *Workspace) { w.Mounts[0].Path = w.Path + "/piers/shared" },
		"member-order":      func(w *Workspace) { slices.Reverse(w.Mounts) },
		"adopt-root":        func(w *Workspace) { w.Root = workspaceID(10) },
		"claim-observation": func(w *Workspace) { o := workspaceObservation(); w.Observation = &o },
		"unknown-phase":     func(w *Workspace) { w.Phase = "ready" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := w.Clone()
			mutate(&bad)
			if bad.Validate(p) == nil {
				t.Fatal("invalid workspace admitted")
			}
		})
	}
	for _, id := range []string{"private-project-../../outside", "private-project-" + strings.Repeat("x", 64), "project-fake"} {
		b := w.Binding
		b.ProjectID = id
		if _, err := NewWorkspace(p, b, 1, w.OperationID, w.Parent); err == nil {
			t.Fatal("invalid project path", id)
		}
	}
}

func TestWorkspaceCommitBeforeEffectsAndIndependentCopies(t *testing.T) {
	p, w := workspaceFixture(t)
	original := w.Clone()
	events := []string{}
	d := &workspaceDriver{events: &events, root: workspaceID(10), observation: workspaceObservation(), mutate: true}
	retained := w.Clone()
	commit := func(expected string, next Workspace) error {
		if retained.Phase != expected || next.Validate(p) != nil {
			t.Fatal("bad checkpoint", expected, next.Phase)
		}
		events = append(events, "commit-"+next.Phase)
		retained = next.Clone()
		next.Mounts[0].Path = "mutated callback copy"
		if next.Observation != nil {
			next.Observation.Mounts[0].Inode = 999
		}
		return nil
	}
	got, err := ExecuteWorkspace(context.Background(), p, w, d, commit, func() time.Time { return workspaceNow })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"commit-root-dispatched", "create-root", "commit-root-created", "inspect-root", "commit-layout-dispatched", "create-layout", "commit-layout-created", "observe-layout", "commit-observed"}
	if !reflect.DeepEqual(events, want) || !reflect.DeepEqual(got, retained) || !reflect.DeepEqual(w, original) {
		t.Fatal(events, got, w)
	}
	d.observation.Mounts[0].Inode = 888
	if got.Observation.Mounts[0].Inode != 12 {
		t.Fatal("driver retained mutable observation")
	}
}

func TestWorkspaceCommitFailureNeverContinues(t *testing.T) {
	for _, phase := range []string{"root-dispatched", "root-created", "layout-dispatched", "layout-created", "observed"} {
		t.Run(phase, func(t *testing.T) {
			p, w := workspaceFixture(t)
			events := []string{}
			d := &workspaceDriver{events: &events, root: workspaceID(10), observation: workspaceObservation()}
			retained := w.Clone()
			sentinel := errors.New("store unavailable")
			commit := func(expected string, next Workspace) error {
				events = append(events, "commit-"+next.Phase)
				if next.Phase == phase {
					return sentinel
				}
				retained = next.Clone()
				return nil
			}
			got, err := ExecuteWorkspace(context.Background(), p, w, d, commit, func() time.Time { return workspaceNow })
			if !errors.Is(err, sentinel) || !reflect.DeepEqual(got, retained) || events[len(events)-1] != "commit-"+phase {
				t.Fatal(got.Phase, err, events)
			}
			if phase != "root-dispatched" {
				before := len(events)
				_, err = ExecuteWorkspace(context.Background(), p, got, d, commit, func() time.Time { return workspaceNow })
				if err == nil || len(events) != before {
					t.Fatal("replayed interrupted mutation")
				}
			}
		})
	}
}

func TestWorkspaceLostRepliesPreserveDispatchAndKnownRoot(t *testing.T) {
	for _, tc := range []struct {
		name  string
		root  pierstore.DirectoryIdentity
		phase string
	}{
		{"create-root", pierstore.DirectoryIdentity{}, "root-dispatched"},
		{"create-root", workspaceID(10), "root-created"},
		{"inspect-root", workspaceID(10), "root-created"},
		{"create-layout", workspaceID(10), "layout-dispatched"},
		{"observe-layout", workspaceID(10), "layout-created"},
	} {
		t.Run(tc.name+"/"+tc.phase, func(t *testing.T) {
			p, w := workspaceFixture(t)
			events := []string{}
			d := &workspaceDriver{events: &events, root: tc.root, observation: workspaceObservation(), fail: tc.name}
			got, err := ExecuteWorkspace(context.Background(), p, w, d, func(string, Workspace) error { return nil }, func() time.Time { return workspaceNow })
			if err == nil || strings.Contains(err.Error(), "private driver") || got.Phase != tc.phase || got.Root != tc.root || events[len(events)-1] != tc.name {
				t.Fatal(got, err, events)
			}
		})
	}
}

func TestWorkspaceCancellationAtCheckpointsStopsFurtherEffects(t *testing.T) {
	for _, phase := range []string{"planned", "root-dispatched", "root-created", "layout-dispatched", "layout-created", "observed"} {
		t.Run(phase, func(t *testing.T) {
			p, w := workspaceFixture(t)
			events := []string{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &workspaceDriver{events: &events, root: workspaceID(10), observation: workspaceObservation()}
			if phase == "planned" {
				cancel()
			}
			got, err := ExecuteWorkspace(ctx, p, w, d, func(_ string, next Workspace) error {
				events = append(events, "commit-"+next.Phase)
				if next.Phase == phase {
					cancel()
				}
				return nil
			}, func() time.Time { return workspaceNow })
			if phase != "observed" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if got.Phase != phase {
				t.Fatal(got.Phase, phase)
			}
			if len(events) > 0 && events[len(events)-1] != "commit-"+phase {
				t.Fatal(events)
			}
		})
	}
}

type workspaceReadOnly func(context.Context, Workspace) (WorkspaceObservation, error)

func (f workspaceReadOnly) ObserveLayout(ctx context.Context, w Workspace) (WorkspaceObservation, error) {
	return f(ctx, w)
}

func TestWorkspaceRecoveryIsObservationOnlyAndRetainsIdentity(t *testing.T) {
	for _, phase := range []string{"layout-dispatched", "layout-created"} {
		t.Run(phase, func(t *testing.T) {
			p, w := workspaceFixture(t)
			w.Phase = phase
			w.Root = workspaceID(10)
			calls := 0
			observer := workspaceReadOnly(func(context.Context, Workspace) (WorkspaceObservation, error) {
				calls++
				o := workspaceObservation()
				o.Parent.Device = 44
				o.Root.Device = 44
				o.PiersRoot.Device = 44
				for i := range o.Mounts {
					o.Mounts[i].Device = 44
				}
				return o, nil
			})
			got, err := ObserveWorkspace(context.Background(), p, w, observer, func(expected string, next Workspace) error {
				if expected != phase || next.Root != w.Root || next.Phase != "observed" {
					t.Fatal(next)
				}
				return nil
			}, func() time.Time { return workspaceNow })
			if err != nil || calls != 1 || got.Root != w.Root {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, phase := range []string{"planned", "root-dispatched", "root-created", "observed"} {
		p, w := workspaceFixture(t)
		w.Phase = phase
		if phase == "root-created" || phase == "observed" {
			w.Root = workspaceID(10)
		}
		if phase == "observed" {
			o := workspaceObservation()
			w.Observation = &o
		}
		observer := workspaceReadOnly(func(context.Context, Workspace) (WorkspaceObservation, error) {
			t.Fatal("recovery observed unowned root or replayed completion")
			return WorkspaceObservation{}, nil
		})
		if _, err := ObserveWorkspace(context.Background(), p, w, observer, func(string, Workspace) error { t.Fatal("unexpected commit"); return nil }, func() time.Time { return workspaceNow }); err == nil {
			t.Fatal("invalid recovery", phase)
		}
	}
}

func TestWorkspaceRecoveryRefusesIncompleteChangedAndStaleLayout(t *testing.T) {
	for name, mutate := range map[string]func(*WorkspaceObservation){
		"old":              func(o *WorkspaceObservation) { o.ObservedAt = workspaceNow.Add(-31 * time.Second) },
		"future":           func(o *WorkspaceObservation) { o.ObservedAt = workspaceNow.Add(6 * time.Second) },
		"no-time":          func(o *WorkspaceObservation) { o.ObservedAt = time.Time{} },
		"partial":          func(o *WorkspaceObservation) { o.Complete = false },
		"shared":           func(o *WorkspaceObservation) { o.Private = false },
		"unsynced":         func(o *WorkspaceObservation) { o.Synced = false },
		"populated":        func(o *WorkspaceObservation) { o.Empty = false },
		"parent-replaced":  func(o *WorkspaceObservation) { o.Parent.Inode++ },
		"root-replaced":    func(o *WorkspaceObservation) { o.Root.Inode++ },
		"root-reused":      func(o *WorkspaceObservation) { o.Root.Birth = "1000000000:2" },
		"missing-parent":   func(o *WorkspaceObservation) { o.Mounts = o.Mounts[:1] },
		"extra-parent":     func(o *WorkspaceObservation) { o.Mounts = append(o.Mounts, workspaceID(14)) },
		"duplicate-object": func(o *WorkspaceObservation) { o.Mounts[0] = o.PiersRoot },
		"other-device":     func(o *WorkspaceObservation) { o.Mounts[0].Device++ },
		"unknown-birth":    func(o *WorkspaceObservation) { o.PiersRoot.Birth = "" },
		"malformed-birth":  func(o *WorkspaceObservation) { o.Mounts[0].Birth = "not-a-timestamp" },
	} {
		t.Run(name, func(t *testing.T) {
			p, w := workspaceFixture(t)
			w.Phase = "layout-dispatched"
			w.Root = workspaceID(10)
			o := workspaceObservation()
			mutate(&o)
			observer := workspaceReadOnly(func(context.Context, Workspace) (WorkspaceObservation, error) { return o, nil })
			got, err := ObserveWorkspace(context.Background(), p, w, observer, func(string, Workspace) error { t.Fatal("committed invalid observation"); return nil }, func() time.Time { return workspaceNow })
			if err == nil || !reflect.DeepEqual(got, w) {
				t.Fatal(got, err)
			}
		})
	}
}

func TestWorkspaceAdvanceRefusesSkippingAndIdentityChanges(t *testing.T) {
	p, w := workspaceFixture(t)
	if _, err := w.Advance(p, "planned", "root-created", workspaceID(10), nil); err == nil {
		t.Fatal("skipped dispatch")
	}
	if _, err := w.Advance(p, "stale", "root-dispatched", pierstore.DirectoryIdentity{}, nil); err == nil {
		t.Fatal("stale checkpoint")
	}
	w.Phase = "root-created"
	w.Root = workspaceID(10)
	if _, err := w.Advance(p, w.Phase, "layout-dispatched", workspaceID(11), nil); err == nil {
		t.Fatal("changed root")
	}
}
