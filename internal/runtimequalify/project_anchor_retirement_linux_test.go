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

type anchorRetirementAdapterFixture struct {
	x         *anchorDrainFixture
	selection ProjectAnchorRetirementSelection
	saved     projectnetwork.AnchorRetirement
	removes   []string
	remove    func(context.Context, string) (string, error)
}

func newAnchorRetirementAdapterFixture(t *testing.T) *anchorRetirementAdapterFixture {
	t.Helper()
	x := newAnchorDrainFixture(t, "restarted")
	for _, path := range x.f.paths {
		for _, sub := range []string{".urb/log", ".urb/chk"} {
			if err := os.MkdirAll(filepath.Join(path, sub), 0700); err != nil {
				t.Fatal(err)
			}
		}
	}

	h := anchorDrainHistory(x.selection.Source)
	drain := x.selection.Drain
	for !drain.Clean() {
		e := x.open(t, drain)
		var err error
		drain, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, drain, e, func(uint64, projectnetwork.AnchorDrain) error { return nil }, time.Now)
		if err != nil {
			t.Fatal(err)
		}
	}
	ns, err := projectnetwork.NewAnchorNamespace(h, drain, "successor-anchor")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := openProjectAnchorNamespace(context.Background(), ProjectAnchorNamespaceSelection{Source: x.selection.Source, Namespace: ns}, x.f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectAnchorNamespaceEngine)
	e.observation.review.source.anchor.readBootID = func() (string, error) { return x.f.ns.BootID, nil }
	ns, err = projectnetwork.ExecuteAnchorNamespace(context.Background(), h, ns, e, func(uint64, projectnetwork.AnchorNamespace) error { return nil }, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	s, err := projectnetwork.NewAnchorRetirement(h, ns, "retire-old-members")
	if err != nil {
		t.Fatal(err)
	}
	f := &anchorRetirementAdapterFixture{x: x, selection: ProjectAnchorRetirementSelection{Source: x.selection.Source, Retirement: s}, saved: s.Clone()}
	f.remove = func(_ context.Context, id string) (string, error) { delete(x.f.containers, id); return id, nil }
	original := x.f.f.d.execute
	x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		if len(a) >= 2 && a[0] == "container" && a[1] == "rm" {
			if len(a) != 3 {
				t.Fatal("forced or multiple removal", a)
			}
			index := 0
			for f.saved.Members[index].Phase == "absent" {
				index++
			}
			target := ""
			for _, m := range s.Namespace.Drain.Review.Context.Members {
				if m.Context.Member.InstanceID == f.saved.Members[index].InstanceID {
					target = m.Context.Member.Container.ID
				}
			}
			if target == "" || a[2] != target || f.saved.Members[index].Phase != "remove-dispatched" || x.f.containers[target] == nil || *x.f.containers[target].State.Running {
				t.Fatal("uncommitted, unowned or running removal", a)
			}
			for _, id := range f.removes {
				if id == target {
					t.Fatal("replayed removal")
				}
			}
			f.removes = append(f.removes, target)
			return f.remove(ctx, target)
		}
		return original(ctx, args, env)
	}
	return f
}
func (f *anchorRetirementAdapterFixture) open(t *testing.T, s projectnetwork.AnchorRetirement) *projectAnchorRetirementEngine {
	t.Helper()
	selected := f.selection
	selected.Retirement = s
	raw, err := openProjectAnchorRetirement(context.Background(), selected, f.x.f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectAnchorRetirementEngine)
	e.review.source.anchor.readBootID = func() (string, error) { return f.x.f.ns.BootID, nil }
	return e
}
func (f *anchorRetirementAdapterFixture) commit(t *testing.T, seq uint64, next projectnetwork.AnchorRetirement) error {
	t.Helper()
	if seq != f.saved.Sequence || next.Validate(anchorDrainHistory(f.selection.Source)) != nil {
		t.Fatal("invalid retirement commit")
	}
	f.saved = next.Clone()
	return nil
}
func TestProjectAnchorRetirementAdapter(t *testing.T) {
	for _, mode := range []string{"pass", "caller-alias", "lost-reply", "bad-ack", "remove-error", "no-removal", "cancel-remove", "commit-dispatch", "commit-ack", "commit-absent", "peer-before", "peer-after", "target-restarted", "anchor-process", "anchor-namespace", "foreign-config", "missing-publication", "foreign-bridge", "engine-changed", "lock-before", "linked-state", "pier-before", "pier-after", "foreign-after", "moved-network-after"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorRetirementAdapterFixture(t)
			x := f.x
			s := f.saved.Clone()
			e := f.open(t, s)
			if _, _, err := e.observe(context.Background(), s, false); err != nil {
				t.Fatal("initial retirement preflight", err)
			}
			h := e.history.Clone()
			beforeHistory := h.Clone()
			signals := x.f.signals
			sentinel := x.sentinel
			oldData, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			oldStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			target := e.target.id
			peer := ""
			for id := range x.f.containers {
				if id != target {
					peer = id
				}
			}
			targetPath := x.f.paths[s.Members[0].InstanceID]
			replacePier := func() {
				if err := os.Rename(targetPath, targetPath+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(targetPath, 0700); err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(sentinel, targetPath+"/") {
					sentinel = targetPath + "-retained" + strings.TrimPrefix(sentinel, targetPath)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "caller-alias" {
				f.selection.Retirement.Namespace.Drain.Members[0].InstanceID = "caller-change"
				f.selection.Source.Launches[0].Storage.RuntimeID = "caller-change"
			}
			f.remove = func(_ context.Context, id string) (string, error) {
				if mode == "remove-error" {
					return "private output", errors.New("private failure")
				}
				if mode != "no-removal" && mode != "moved-network-after" {
					delete(x.f.containers, id)
				}
				switch mode {
				case "lost-reply":
					return "private output", errors.New("private failure")
				case "bad-ack":
					return "wrong-id", nil
				case "cancel-remove":
					cancel()
				case "peer-after":
					x.f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				case "pier-after":
					replacePier()
				case "foreign-after":
					v := *x.f.containers[peer]
					v.ID = strings.Repeat("1", 64)
					v.Name = "/foreign"
					x.f.containers[v.ID] = &v
				case "moved-network-after":
					x.f.containers[id].HostConfig.NetworkMode = "none"
				}
				return id, nil
			}
			commit := func(seq uint64, next projectnetwork.AnchorRetirement) error {
				if seq != f.saved.Sequence || next.Validate(h) != nil {
					t.Fatal("bad commit")
				}
				phase := next.Members[0].Phase
				if mode == "commit-dispatch" && phase == "remove-dispatched" || mode == "commit-ack" && phase == "remove-acknowledged" || mode == "commit-absent" && phase == "absent" {
					return errors.New("commit failed")
				}
				f.saved = next.Clone()
				if phase == "remove-dispatched" {
					switch mode {
					case "peer-before":
						x.f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "target-restarted":
						*x.f.containers[target].State.Running = true
						x.f.containers[target].State.Status = "running"
						x.f.containers[target].State.Pid = 99
					case "anchor-process", "anchor-namespace", "foreign-config", "missing-publication", "foreign-bridge":
						x.fault = mode
					case "engine-changed":
						x.f.f.engine.ID = "changed-engine"
					case "lock-before":
						if err := os.WriteFile(filepath.Join(targetPath, ".vere.lock"), []byte("99"), 0600); err != nil {
							t.Fatal(err)
						}
					case "linked-state":
						if err := os.Link(sentinel, sentinel+"-alias"); err != nil {
							t.Fatal(err)
						}
					case "pier-before":
						replacePier()
					}
				}
				return nil
			}
			got, err := projectnetwork.ExecuteNextAnchorRetirement(ctx, h, s, e, commit, time.Now)
			pass := mode == "pass" || mode == "caller-alias"
			if (err == nil) != pass || !reflect.DeepEqual(got, f.saved) || x.f.signals != signals || !reflect.DeepEqual(h, beforeHistory) {
				t.Fatal("unexpected retirement outcome", mode, err, len(f.removes))
			}
			if pass && (got.Members[0].Phase != "absent" || len(f.removes) != 1) {
				t.Fatal("retirement incomplete")
			}
			if strings.HasSuffix(mode, "-before") || mode == "linked-state" || mode == "commit-dispatch" || mode == "target-restarted" || mode == "anchor-process" || mode == "anchor-namespace" || mode == "foreign-config" || mode == "missing-publication" || mode == "foreign-bridge" || mode == "engine-changed" {
				if len(f.removes) != 0 {
					t.Fatal("uncertain preflight reached removal")
				}
			}
			if mode == "lost-reply" || mode == "bad-ack" || mode == "cancel-remove" || mode == "commit-ack" || mode == "commit-absent" || mode == "remove-error" || mode == "no-removal" || mode == "moved-network-after" {
				reopen := f.open(t, got)
				count := len(f.removes)
				if er := reopen.RemoveRetiredMember(context.Background(), got, 0); er == nil || len(f.removes) != count {
					t.Fatal("reopened removal replayed")
				}
				resolved, er := projectnetwork.ObserveAnchorRetirementAbsence(context.Background(), h, got, reopen, func(seq uint64, next projectnetwork.AnchorRetirement) error { return f.commit(t, seq, next) }, time.Now)
				absent := x.f.containers[target] == nil
				if (er == nil) != absent || len(f.removes) != count {
					t.Fatal("wrong absence reconciliation", er, absent)
				}
				if absent && (resolved.Members[0].Phase != "absent" || resolved.Members[0].RemoveAcknowledged != got.Members[0].RemoveAcknowledged) {
					t.Fatal("reconciliation invented acknowledgement")
				}
			}
			data, er := os.ReadFile(sentinel)
			if er != nil || string(data) != string(oldData) {
				t.Fatal("retirement changed pier data", er)
			}
			info, er := os.Stat(sentinel)
			if er != nil || !os.SameFile(oldStat, info) {
				t.Fatal("retirement replaced pier data", er)
			}
		})
	}
}
func TestProjectAnchorRetirementAllMembersAndAllPiers(t *testing.T) {
	f := newAnchorRetirementAdapterFixture(t)
	s := f.saved.Clone()
	h := anchorDrainHistory(f.selection.Source)
	original := s.Namespace.Clone()
	for !s.Complete() {
		e := f.open(t, s)
		var err error
		s, err = projectnetwork.ExecuteNextAnchorRetirement(context.Background(), h, s, e, func(seq uint64, next projectnetwork.AnchorRetirement) error { return f.commit(t, seq, next) }, time.Now)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(f.removes) != len(s.Members) || len(f.x.f.containers) != 0 || !reflect.DeepEqual(s.Namespace, original) || s.Validate(h) != nil {
		t.Fatal("serial retirement incomplete")
	}
	last := s.Members[len(s.Members)-1].After
	if len(last.Piers) != len(s.Members) || len(last.Processes) != 0 || len(last.Runtime.Members) != 0 {
		t.Fatal("absence discarded pier evidence")
	}
	if _, err := openProjectAnchorRetirement(context.Background(), ProjectAnchorRetirementSelection{Source: f.selection.Source, Retirement: s}, f.x.f.f.d); err == nil {
		t.Fatal("completed retirement reopened")
	}
}
