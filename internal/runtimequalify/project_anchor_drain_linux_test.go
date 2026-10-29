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
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type anchorDrainFixture struct {
	f         *memberStopFixture
	selection ProjectAnchorDrainSelection
	sentinel  string
	fault     string
}

func newAnchorDrainFixture(t *testing.T, state string) *anchorDrainFixture {
	t.Helper()
	f, input, sentinel := resumedStopAdapterFixture(t, false, 3)
	s := ProjectAnchorRecoveryReviewSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Originals: []projectnetwork.InitialMemberReadiness{f.readiness["receiver"], f.readiness["sender"]}}
	c, err := projectnetwork.SelectAnchorRecoveryContext(s.Project.Plan, s.Launches, s.Resumes, s.Originals, nil)
	if err != nil {
		t.Fatal(err)
	}
	x := &anchorDrainFixture{f: f, sentinel: sentinel}
	original := f.f.d.execute
	when := time.Now().UTC().Add(-time.Millisecond).Format(time.RFC3339Nano)
	f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		namespace := len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
		signal := len(a) == 5 && reflect.DeepEqual(a[:4], []string{"container", "kill", "--signal", "SIGTERM"})
		if len(a) < 2 || !(namespace || signal || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
			t.Fatal("unapproved drain command", a)
		}
		if signal && f.containers[a[4]] == nil {
			t.Fatal("drain signalled unowned container")
		}
		if namespace {
			inode := f.ns.Inode
			if a[2] == c.Binding.Anchor.ID {
				if state == "stopped" {
					t.Fatal("probed stopped anchor")
				}
				inode++
				if x.fault == "anchor-namespace" {
					inode++
				}
			} else if x.fault == "member-namespace" {
				inode++
			}
			return fmt.Sprintf("%d:%d", f.ns.Device, inode), nil
		}
		out, err := original(ctx, args, env)
		if err != nil {
			return out, err
		}
		if a[0] == "container" && a[1] == "inspect" && a[2] != "--format" {
			var values []container
			if err = json.Unmarshal([]byte(out), &values); err != nil {
				t.Fatal(err)
			}
			for i := range values {
				v := &values[i]
				if v.ID != c.Binding.Anchor.ID {
					continue
				}
				if state == "stopped" {
					v.State.Status = "exited"
					v.State.Pid = 0
					*v.State.Running = false
					v.State.FinishedAt = when
					*v.State.ExitCode = 143
					v.NetworkSettings.Ports = map[string]json.RawMessage{}
				} else {
					v.State.StartedAt = when
					began, _ := time.Parse(time.RFC3339Nano, when)
					v.State.FinishedAt = began.Add(-time.Second).Format(time.RFC3339Nano)
				}
				switch x.fault {
				case "anchor-process":
					v.State.StartedAt = time.Now().UTC().Add(-time.Microsecond).Format(time.RFC3339Nano)
				case "foreign-config":
					v.HostConfig.Privileged = true
				case "missing-publication":
					v.HostConfig.PortBindings = map[string]json.RawMessage{}
					v.NetworkSettings.Ports = map[string]json.RawMessage{}
				}
			}
			b, _ := json.Marshal(values)
			return string(b), nil
		}
		if a[0] == "network" && a[1] == "inspect" {
			var values []bridge
			if err = json.Unmarshal([]byte(out), &values); err != nil {
				t.Fatal(err)
			}
			if state == "stopped" {
				values[0].Containers = map[string]json.RawMessage{}
			}
			if x.fault == "foreign-bridge" {
				values[0].Containers[strings.Repeat("1", 64)] = json.RawMessage(`{}`)
			}
			b, _ := json.Marshal(values)
			return string(b), nil
		}
		return out, nil
	}
	f.f.d.hostIdentity = func(context.Context, string) (string, error) {
		t.Fatal("drain attempted HTTP")
		return "", errors.New("unexpected")
	}
	raw, err := openProjectAnchorRecoveryReview(context.Background(), s, f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	observer := raw.(*projectAnchorRecoveryReviewEngine)
	observer.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	review, err := projectnetwork.CollectAnchorRecoveryReview(context.Background(), s.Project.Plan, s.Launches, s.Resumes, s.Originals, nil, observer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	drain, err := projectnetwork.NewAnchorDrain(anchorDrainHistory(s), review, "drain-anchor")
	if err != nil {
		t.Fatal(err)
	}
	x.selection = ProjectAnchorDrainSelection{Source: s, Drain: drain}
	return x
}
func (x *anchorDrainFixture) open(t *testing.T, s projectnetwork.AnchorDrain) *projectAnchorDrainEngine {
	t.Helper()
	input := x.selection
	input.Drain = s
	raw, err := openProjectAnchorDrain(context.Background(), input, x.f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectAnchorDrainEngine)
	e.review.source.anchor.readBootID = func() (string, error) { return x.f.ns.BootID, nil }
	return e
}

func TestProjectAnchorDrainAdapter(t *testing.T) {
	for _, state := range []string{"stopped", "restarted"} {
		for _, mode := range []string{"pass", "delayed-exit", "live-lock", "caller-alias", "lost-reply", "bad-ack", "signal-error", "no-exit", "nonzero", "oom", "commit-dispatch", "commit-signalled", "commit-exited", "target-restarted", "peer-restarted", "pier-replaced", "engine-changed", "member-namespace", "anchor-namespace", "anchor-process", "foreign-config", "missing-publication", "foreign-bridge", "cancelled", "peer-changed-after-signal"} {
			if state == "stopped" && mode == "anchor-namespace" {
				continue
			}
			t.Run(state+"/"+mode, func(t *testing.T) {
				x := newAnchorDrainFixture(t, state)
				f := x.f
				initial := x.selection.Drain
				h := anchorDrainHistory(x.selection.Source)
				oldBytes, err := os.ReadFile(x.sentinel)
				if err != nil {
					t.Fatal(err)
				}
				oldStat, err := os.Stat(x.sentinel)
				if err != nil {
					t.Fatal(err)
				}
				retained := x.sentinel
				if mode == "no-exit" {
					f.f.d.times.stop = 20 * time.Millisecond
					f.f.d.times.poll = time.Millisecond
				}
				e := x.open(t, initial)
				saved := initial
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				peer := ""
				for id := range f.containers {
					if id != e.target.id {
						peer = id
					}
				}
				if mode == "live-lock" {
					if err = os.WriteFile(filepath.Join(f.paths[initial.Members[0].InstanceID], ".vere.lock"), []byte("99"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "caller-alias" {
					x.selection.Source.Launches[0].Storage.RuntimeID = "changed"
					x.selection.Drain.Review.Context.Members[0].Context.Member.InstanceID = "changed"
					h = e.history
					initial = e.initial
					saved = initial
				}
				polls := 0
				f.singleRead = func(id string) {
					if mode == "delayed-exit" && f.signals > 0 && id == e.target.id {
						polls++
						if polls == 3 {
							f.exit(id, 0, false)
						}
					}
				}
				signal := f.signal
				f.signal = func(ctx context.Context, id string) (string, error) {
					if id != e.target.id || saved.Members[e.index].Phase != "stop-dispatched" {
						t.Fatal("signal before exact durable dispatch")
					}
					switch mode {
					case "signal-error":
						return "private Docker output", errors.New("private failure")
					case "bad-ack":
						f.exit(id, 0, false)
						return "wrong-id", nil
					case "no-exit", "delayed-exit":
						return id, nil
					case "nonzero":
						f.exit(id, 7, false)
						return id, nil
					case "oom":
						f.exit(id, 137, true)
						return id, nil
					}
					out, err := signal(ctx, id)
					if mode == "lost-reply" {
						return "private Docker output", errors.New("private failure")
					}
					if mode == "peer-changed-after-signal" {
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					}
					return out, err
				}
				commit := func(seq uint64, next projectnetwork.AnchorDrain) error {
					if seq != saved.Sequence {
						t.Fatal("stale commit")
					}
					phase := next.Members[e.index].Phase
					if mode == "commit-dispatch" && phase == "stop-dispatched" || mode == "commit-signalled" && phase == "stop-signalled" || mode == "commit-exited" && phase == "exited" {
						return errors.New("commit failed")
					}
					saved = next
					if phase == "stop-dispatched" {
						switch mode {
						case "target-restarted":
							f.containers[e.target.id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						case "peer-restarted":
							f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						case "engine-changed":
							f.f.engine.ID = "changed"
						case "cancelled":
							cancel()
						case "pier-replaced":
							p := filepath.Join(f.paths[e.initial.Members[e.index].InstanceID], ".urb")
							if err := os.Rename(p, p+"-retained"); err != nil {
								t.Fatal(err)
							}
							if err := os.Mkdir(p, 0700); err != nil {
								t.Fatal(err)
							}
							if strings.HasPrefix(retained, p+string(os.PathSeparator)) {
								retained = p + "-retained" + strings.TrimPrefix(retained, p)
							}
						default:
							x.fault = mode
						}
					}
					return nil
				}
				got, err := projectnetwork.ExecuteNextAnchorDrain(ctx, h, initial, e, commit, time.Now)
				success := mode == "pass" || mode == "delayed-exit" || mode == "live-lock" || mode == "caller-alias"
				if success {
					if err != nil || got.Members[0].Phase != "exited" || f.signals != 1 || !reflect.DeepEqual(saved, got) {
						t.Fatal("drain failed", err, got.Members[0].Phase, f.signals)
					}
					first := got.Members[0]
					x.selection.Source = e.historySourceForTest(x.selection.Source)
					e = x.open(t, got)
					f.signal = func(ctx context.Context, id string) (string, error) {
						if saved.Members[e.index].Phase != "stop-dispatched" || id != e.target.id {
							t.Fatal("wrong second signal")
						}
						return signal(ctx, id)
					}
					x.fault = ""
					f.singleRead = nil
					got, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, got, e, commit, time.Now)
					if err != nil || !got.Clean() || f.signals != 2 || !reflect.DeepEqual(first, got.Members[0]) || got.Validate(h) != nil {
						t.Fatal("serial drain failed", err, f.signals)
					}
					if _, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, got, e, commit, time.Now); err == nil || f.signals != 2 {
						t.Fatal("completed replay")
					}
				} else {
					if err == nil || strings.Contains(err.Error(), "private") || !reflect.DeepEqual(saved, got) {
						t.Fatal("failure not retained/redacted", err)
					}
					expectedSignals := 0
					switch mode {
					case "lost-reply", "bad-ack", "signal-error", "no-exit", "nonzero", "oom", "commit-signalled", "commit-exited", "peer-changed-after-signal":
						expectedSignals = 1
					}
					if f.signals != expectedSignals {
						t.Fatal("wrong signal count", f.signals, expectedSignals)
					}
					if _, again := projectnetwork.ExecuteNextAnchorDrain(ctx, h, got, e, commit, time.Now); again == nil || f.signals != expectedSignals {
						t.Fatal("uncertain work replayed")
					}
					if mode == "lost-reply" || mode == "bad-ack" || mode == "commit-signalled" || mode == "commit-exited" {
						observer := x.open(t, got)
						beforeSignals := f.signals
						recovered, re := projectnetwork.ObserveAnchorDrainExit(context.Background(), h, got, observer, func(seq uint64, next projectnetwork.AnchorDrain) error {
							if seq != saved.Sequence {
								t.Fatal("recovery stale")
							}
							saved = next
							return nil
						}, time.Now)
						if re != nil || recovered.Members[0].Phase != "exited" || f.signals != beforeSignals || recovered.Members[0].SignalAcknowledged != (mode == "commit-exited") {
							t.Fatal("lost reply recovery", re)
						}
					}
				}
				data, err := os.ReadFile(retained)
				if err != nil || string(data) != string(oldBytes) {
					t.Fatal("pier bytes changed", err)
				}
				info, err := os.Stat(retained)
				if err != nil || !os.SameFile(oldStat, info) {
					t.Fatal("pier object changed", err)
				}
			})
		}
	}
}

// Restore source from the adapter's detached copy after the deliberate caller-alias test.
func (e *projectAnchorDrainEngine) historySourceForTest(s ProjectAnchorRecoveryReviewSelection) ProjectAnchorRecoveryReviewSelection {
	s.Project.Plan = e.history.Plan
	s.Launches = e.history.Launches
	s.Resumes = e.history.Resumes
	s.Originals = e.history.Originals
	s.Current = e.history.Current
	return s
}

func TestProjectAnchorDrainAdapterRefusals(t *testing.T) {
	for _, mode := range []string{"changed-intent", "missing-history", "wrong-index", "signal-before-observation", "pending-still-running", "prior-exit-changed", "prior-unclean", "complete"} {
		t.Run(mode, func(t *testing.T) {
			x := newAnchorDrainFixture(t, "restarted")
			h := anchorDrainHistory(x.selection.Source)
			s := x.selection.Drain
			e := x.open(t, s)
			commit := func(_ uint64, next projectnetwork.AnchorDrain) error { s = next; return nil }
			switch mode {
			case "changed-intent", "missing-history":
				input := cloneProjectAnchorDrainSelection(x.selection)
				if mode == "changed-intent" {
					input.Drain.Sequence++
				} else {
					input.Source.Resumes = nil
				}
				if _, err := openProjectAnchorDrain(context.Background(), input, x.f.f.d); err == nil {
					t.Fatal("invalid source admitted")
				}
			case "wrong-index":
				before, err := e.InspectDrain(context.Background(), s)
				if err != nil {
					t.Fatal(err)
				}
				next, err := s.Advance(h, 0, s.Sequence, "stop-dispatched", &before)
				if err != nil {
					t.Fatal(err)
				}
				for _, i := range []int{-1, 1, 1000} {
					if err = e.SignalDrainMember(context.Background(), next, i); err == nil {
						t.Fatal("wrong index signalled")
					}
				}
			case "signal-before-observation":
				if err := e.SignalDrainMember(context.Background(), s, 0); err == nil {
					t.Fatal("signal without observation")
				}
			default:
				if mode == "pending-still-running" {
					x.f.signal = func(context.Context, string) (string, error) { return "", errors.New("lost before effect") }
				}
				if mode == "prior-unclean" {
					x.f.signal = func(_ context.Context, id string) (string, error) { x.f.exit(id, 8, false); return id, nil }
				}
				var err error
				s, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, s, e, commit, time.Now)
				if mode == "pending-still-running" {
					if err == nil {
						t.Fatal("lost reply not retained")
					}
					observer := x.open(t, s)
					if _, err = projectnetwork.ObserveAnchorDrainExit(context.Background(), h, s, observer, commit, time.Now); err == nil {
						t.Fatal("running process accepted as exit")
					}
					if err = observer.SignalDrainMember(context.Background(), s, 0); err == nil {
						t.Fatal("reopened signal replayed")
					}
				} else if mode == "prior-unclean" {
					if err == nil || s.Members[0].Phase != "exited" {
						t.Fatal("unclean exit not retained")
					}
					input := x.selection
					input.Drain = s
					if _, err = openProjectAnchorDrain(context.Background(), input, x.f.f.d); err == nil {
						t.Fatal("unclean predecessor advanced")
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					previous := e.target.id
					e = x.open(t, s)
					if mode == "prior-exit-changed" {
						x.f.containers[previous].State.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
					}
					s, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, s, e, commit, time.Now)
					if mode == "prior-exit-changed" {
						if err == nil || x.f.signals != 1 {
							t.Fatal("changed prior exit accepted")
						}
					} else {
						if err != nil || !s.Clean() {
							t.Fatal("drain incomplete", err)
						}
						input := x.selection
						input.Drain = s
						if _, err = openProjectAnchorDrain(context.Background(), input, x.f.f.d); err == nil {
							t.Fatal("complete operation reopened for signal")
						}
					}
				}
			}
			expected := 0
			switch mode {
			case "pending-still-running", "prior-unclean", "prior-exit-changed":
				expected = 1
			case "complete":
				expected = 2
			}
			if x.f.signals != expected {
				t.Fatal("unexpected signal count", x.f.signals, expected)
			}
		})
	}
}
