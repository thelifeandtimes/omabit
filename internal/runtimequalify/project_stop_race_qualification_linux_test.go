//go:build linux && amd64 && omabit_disposable_qualification

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestProjectStopRaceQualificationGuardRefusesHostForeignOrReplayedState(t *testing.T) {
	source := strings.Repeat("a", 40)
	marker := map[string]any{"kind": "issues16-native-transition-v1", "case": projectRaceCase, "source_commit": source, "predecessor_root_sha256": projectRaceBase, "original_schema": 65, "target_schema": 65, "require_native_transition": true, "preserve_stopped_base_instance": true}
	raw, _ := json.Marshal(marker)
	pass := func(b []byte, revision string, uid int, home, run, vendor, cmd string) bool {
		return projectRaceVMGuard(b, revision, uid, home, run, vendor, cmd)
	}
	if !pass(raw, source, 1000, "/home/review", "/run/user/1000", "QEMU\n", "quiet omabit.disposable-guest=release-upgrade-v1") {
		t.Fatal("fixed inert guard refused")
	}
	for key, value := range map[string]any{"kind": "other", "case": "other", "source_commit": strings.Repeat("b", 40), "predecessor_root_sha256": strings.Repeat("b", 64), "original_schema": 64, "target_schema": 66, "require_native_transition": false, "preserve_stopped_base_instance": false} {
		changed := map[string]any{}
		for k, v := range marker {
			changed[k] = v
		}
		changed[key] = value
		b, _ := json.Marshal(changed)
		if pass(b, source, 1000, "/home/review", "/run/user/1000", "QEMU", "omabit.disposable-guest=release-upgrade-v1") {
			t.Fatal("foreign marker accepted", key)
		}
	}
	for _, change := range []struct {
		uid                    int
		home, run, vendor, cmd string
	}{{0, "/home/review", "/run/user/1000", "QEMU", "omabit.disposable-guest=release-upgrade-v1"}, {1000, "/home/other", "/run/user/1000", "QEMU", "omabit.disposable-guest=release-upgrade-v1"}, {1000, "/home/review", "/run/user/2000", "QEMU", "omabit.disposable-guest=release-upgrade-v1"}, {1000, "/home/review", "/run/user/1000", "native", "omabit.disposable-guest=release-upgrade-v1"}, {1000, "/home/review", "/run/user/1000", "QEMU", "omabit.disposable-guest=other"}} {
		if pass(raw, source, change.uid, change.home, change.run, change.vendor, change.cmd) {
			t.Fatal("host/foreign environment accepted")
		}
	}
	if pass(raw, "not-a-source", 1000, "/home/review", "/run/user/1000", "QEMU", "omabit.disposable-guest=release-upgrade-v1") || pass(make([]byte, 65537), source, 1000, "/home/review", "/run/user/1000", "QEMU", "omabit.disposable-guest=release-upgrade-v1") {
		t.Fatal("bad source/oversized marker accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := RunDisposableProjectStopRace(ctx, nil, "", "", source, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled host entry did not refuse", err)
	}
}

func TestProjectStopRaceQualificationSelectionIsOnlyFreshAddedMember(t *testing.T) {
	var s ProjectExpansion
	s.Project.ProjectID = "project"
	s.Project.Plan.Label = "issues native transition"
	s.Members = make([]ProjectExpandedMember, 2)
	for i := range s.Members {
		m := &s.Members[i]
		m.Member.InstanceID = projectRacePeer
		m.Member.Ship = "~nec"
		m.Member.HTTPPort = 19580
		if i == 1 {
			m.Member.InstanceID = projectRaceTarget
			m.Member.Ship = "~bud"
			m.Member.HTTPPort = 19581
			m.Added = true
		}
		m.Creator.InstanceID = m.Member.InstanceID
		m.Creator.Mode = "boot"
		m.Creator.Generation = 1
		m.Creator.Container.ID = strings.Repeat(string(rune('a'+i)), 64)
		m.Phase = "ready"
		m.Creator.Storage.Path = "fakeships/" + m.Member.InstanceID + "/" + strings.TrimPrefix(m.Member.Ship, "~")
		m.Initialized = m.Creator.Storage
		m.Initialized.Identity.Inode = uint64(i + 1)
	}
	if index, err := projectRaceSelection(s, projectRaceTarget, "op-stop"); err != nil || index != 1 {
		t.Fatal("fixed selection refused", err)
	}
	for _, mode := range []string{"peer", "label", "history", "phase", "intent", "generation", "mode", "ship", "port", "pier", "duplicate-container", "creator"} {
		raw, _ := json.Marshal(s)
		var changed ProjectExpansion
		json.Unmarshal(raw, &changed)
		id := projectRaceTarget
		m := &changed.Members[1]
		switch mode {
		case "peer":
			id = projectRacePeer
		case "label":
			changed.Project.Plan.Label = "other"
		case "history":
			changed.History = []ProjectExpandedMember{*m}
		case "phase":
			m.Phase = "stop-dispatched"
		case "intent":
			m.StopIntent = &ProjectExpandedStopIntent{}
		case "generation":
			m.Creator.Generation++
		case "mode":
			m.Creator.Mode = "resume"
		case "ship":
			m.Member.Ship = "~zod"
		case "port":
			m.Member.HTTPPort++
		case "pier":
			m.Initialized.Identity.Inode = 0
		case "duplicate-container":
			m.Creator.Container.ID = changed.Members[0].Creator.Container.ID
		case "creator":
			m.Creator.InstanceID = "foreign"
		}
		if _, err := projectRaceSelection(changed, id, "op-stop"); err == nil {
			t.Fatal("modified/replayed selection accepted", mode)
		}
	}
}

func TestProjectStopRaceQualificationNativeReadAndSignalOrder(t *testing.T) {
	for _, mode := range []string{"clean", "term-lost-reply", "cont-lost-reply", "inspect-failed", "replaced-before-continue", "cancelled-before-continue", "trace-failed-after-pause", "wrong-start"} {
		t.Run(mode, func(t *testing.T) {
			target, peer, anchor := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
			start := time.Date(2026, 11, 19, 1, 2, 3, 456, time.UTC)
			currentStart := start
			prefix := []string{"--host", "unix:///fixture/engine", "--config", "/fixture/project"}
			var calls, steps []string
			held, queued, exited := false, false, false
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			encoded := func(ids []string) string {
				var values []any
				for _, id := range ids {
					running, at := true, start
					if id == target {
						running = !exited
						at = currentStart
					}
					state := "running"
					pid := 123
					if !running {
						state = "exited"
						pid = 0
					}
					values = append(values, map[string]any{"Id": id, "State": map[string]any{"Status": state, "StartedAt": at.Format(time.RFC3339Nano), "Running": running, "Pid": pid, "Paused": false, "Restarting": false, "Dead": false, "ExitCode": 0, "OOMKilled": false, "Error": ""}})
				}
				b, _ := json.Marshal(values)
				return string(b)
			}
			var preliminary string
			raw := func(c context.Context, args, env []string) (string, error) {
				if err := c.Err(); err != nil {
					return "", err
				}
				if _, ok := c.Deadline(); !ok {
					t.Fatal("unbounded command")
				}
				if !slices.Equal(args[:4], prefix) {
					t.Fatal("unbound engine/config")
				}
				a := args[4:]
				calls = append(calls, strings.Join(a, " "))
				if slices.Equal(a, []string{"container", "inspect", target, peer}) {
					if !held || !queued {
						t.Fatal("preliminary observation not held after real one-term intent")
					}
					preliminary = encoded([]string{target, peer})
					if mode == "inspect-failed" {
						return "", errors.New("synthetic read failure")
					}
					if mode == "replaced-before-continue" {
						currentStart = start.Add(time.Second)
						held = false
						queued = false
					}
					if mode == "cancelled-before-continue" {
						cancel()
					}
					return preliminary, nil
				}
				if slices.Equal(a, []string{"container", "inspect", target}) {
					return encoded([]string{target}), nil
				}
				if len(a) == 5 && slices.Equal(a[:3], []string{"container", "kill", "--signal"}) {
					if a[4] != target {
						t.Fatal("unselected member signalled")
					}
					switch a[3] {
					case "SIGSTOP":
						held = true
					case "SIGTERM":
						if !held {
							t.Fatal("TERM not held for deterministic boundary")
						}
						queued = true
						if mode == "term-lost-reply" {
							return "", errors.New("synthetic lost TERM reply")
						}
					case "SIGCONT":
						held = false
						if queued {
							exited = true
						}
						if mode == "cont-lost-reply" {
							return "", errors.New("synthetic lost CONT reply")
						}
					default:
						t.Fatal("force/unexpected signal", a)
					}
					return target + "\n", nil
				}
				t.Fatal("unexpected command", a)
				return "", errors.New("unexpected")
			}
			intent := &ProjectExpandedStopIntent{OperationID: "op-stop", ContainerID: target, StartedAt: start, RecordedAt: start.Add(time.Second)}
			if mode == "wrong-start" {
				intent.StartedAt = start.Add(time.Second)
			}
			r := &projectStopRace{target: target, operation: "op-stop", members: []string{target, peer}, readIDs: []string{target, peer, anchor}, prefix: prefix, raw: raw, intent: func() *ProjectExpandedStopIntent { return intent }, poll: time.Millisecond, trace: func(event DisposableProjectStopRaceEvent) error {
				steps = append(steps, event.Step)
				if event.ContainerID != target || event.OperationID != "op-stop" {
					t.Fatal("trace lost exact target")
				}
				if mode == "trace-failed-after-pause" && event.Step == "paused" {
					return errors.New("synthetic trace failure")
				}
				return nil
			}}
			_, err := r.execute(ctx, append(slices.Clone(prefix), "container", "kill", "--signal", "SIGTERM", target), nil)
			var got string
			if err == nil {
				got, err = r.execute(ctx, append(slices.Clone(prefix), "container", "inspect", target, peer), nil)
			}
			cleanupErr := r.release()
			if mode == "clean" {
				if err != nil || cleanupErr != nil || got != preliminary || !r.transitioned || !r.continued || held || !exited || !slices.Equal(steps, []string{"pause-intent", "paused", "term-dispatched", "preliminary-running", "continued", "exited-before-bracket"}) {
					t.Fatal("native transition proof lost", err, cleanupErr, steps)
				}
				if _, err = r.inspected(got, true); err != nil {
					t.Fatal("returned preliminary no longer the real running observation", err)
				}
			} else if err == nil {
				t.Fatal("injected failure became qualification")
			}
			if held {
				t.Fatal("fixture left selected process held", err, cleanupErr)
			}
			term := 0
			for _, call := range calls {
				if strings.Contains(call, "SIGKILL") {
					t.Fatal("force kill")
				}
				if strings.Contains(call, "SIGTERM") {
					term++
				}
			}
			expected := 1
			if mode == "trace-failed-after-pause" || mode == "wrong-start" {
				expected = 0
			}
			if term != expected {
				t.Fatal("repeated/missing original TERM", calls)
			}
			if mode == "replaced-before-continue" {
				for _, call := range calls {
					if strings.Contains(call, "SIGCONT") {
						t.Fatal("signalled replacement process")
					}
				}
				if cleanupErr == nil {
					t.Fatal("changed-process cleanup did not refuse")
				}
			}
		})
	}
}

func TestProjectStopRaceQualificationRefusesUnexpectedEffects(t *testing.T) {
	target := strings.Repeat("a", 64)
	prefix := []string{"--host", "unix:///fixture/engine", "--config", "/fixture/project"}
	r := &projectStopRace{target: target, prefix: prefix, raw: func(context.Context, []string, []string) (string, error) {
		t.Fatal("refused request reached engine")
		return "", nil
	}}
	for _, a := range [][]string{{"container", "kill", "--signal", "SIGKILL", target}, {"container", "kill", "--signal", "SIGTERM", strings.Repeat("b", 64)}, {"container", "restart", target}, {"container", "rm", target}, {"container", "exec", target, "/bin/sh", "-c", "false"}} {
		if _, err := r.execute(t.Context(), append(slices.Clone(prefix), a...), nil); err == nil {
			t.Fatal("unexpected effect accepted", a)
		}
	}
	bad := slices.Clone(prefix)
	bad[1] = "unix:///other/engine"
	if _, err := r.execute(t.Context(), append(bad, "container", "inspect", target), nil); err == nil {
		t.Fatal("foreign engine accepted")
	}
}
