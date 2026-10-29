//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectAnchorNamespaceAdapter(t *testing.T) {
	for _, state := range []string{"stopped", "restarted"} {
		for _, mode := range []string{"pass", "caller-alias", "commit-intent", "commit-observed", "member-changed", "anchor-changed", "publication-changed", "engine-changed", "pier-replaced", "start-error", "lost-reply", "bad-ack", "commit-ack"} {
			if state == "restarted" && (mode == "start-error" || mode == "lost-reply" || mode == "bad-ack" || mode == "commit-ack") {
				continue
			}
			t.Run(state+"/"+mode, func(t *testing.T) {
				x := newAnchorDrainFixture(t, state)
				h := anchorDrainHistory(x.selection.Source)
				drain := x.selection.Drain
				for !drain.Clean() {
					e := x.open(t, drain)
					var err error
					drain, err = projectnetwork.ExecuteNextAnchorDrain(context.Background(), h, drain, e, func(_ uint64, next projectnetwork.AnchorDrain) error { return nil }, time.Now)
					if err != nil {
						t.Fatal(err)
					}
				}
				beforeSignals := x.f.signals
				s, err := projectnetwork.NewAnchorNamespace(h, drain, "successor-anchor")
				if err != nil {
					t.Fatal(err)
				}
				input := ProjectAnchorNamespaceSelection{Source: x.selection.Source, Namespace: s}
				anchorID := s.Drain.Review.Context.Binding.Anchor.ID
				original := x.f.f.d.execute
				started := false
				startedAt := ""
				starts := 0
				saved := s
				x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					if len(a) >= 2 && a[0] == "container" && a[1] == "start" {
						if !reflect.DeepEqual(a, []string{"container", "start", anchorID}) || saved.Phase != "start-dispatched" || state != "stopped" || starts != 0 {
							t.Fatal("wrong or repeated recovery Start", a)
						}
						starts++
						if mode == "start-error" {
							return "private output", errors.New("private failure")
						}
						started = true
						startedAt = time.Now().UTC().Format(time.RFC3339Nano)
						if mode == "lost-reply" {
							return "private output", errors.New("private failure")
						}
						if mode == "bad-ack" {
							return "wrong-id", nil
						}
						return anchorID, nil
					}
					if started && len(a) == 8 && a[0] == "container" && a[1] == "exec" && a[2] == anchorID {
						return "5:43\n", nil
					}
					out, err := original(ctx, args, env)
					if err != nil || !started {
						return out, err
					}
					if a[0] == "container" && a[1] == "inspect" && a[2] != "--format" {
						var values []container
						if err = json.Unmarshal([]byte(out), &values); err != nil {
							t.Fatal(err)
						}
						for i := range values {
							v := &values[i]
							if v.ID == anchorID {
								*v.State.Running = true
								v.State.Status = "running"
								v.State.Pid = 85
								v.State.StartedAt = startedAt
								// Keep the prior exit timestamp, as the pinned Docker runtime does.
								*v.State.ExitCode = 0
								v.NetworkSettings.Ports = v.HostConfig.PortBindings
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
						values[0].Containers = map[string]json.RawMessage{anchorID: json.RawMessage(`{}`)}
						b, _ := json.Marshal(values)
						return string(b), nil
					}
					return out, nil
				}
				open := func(s projectnetwork.AnchorNamespace) *projectAnchorNamespaceEngine {
					selected := input
					selected.Namespace = s
					raw, err := openProjectAnchorNamespace(context.Background(), selected, x.f.f.d)
					if err != nil {
						t.Fatal(err)
					}
					e := raw.(*projectAnchorNamespaceEngine)
					e.observation.review.source.anchor.readBootID = func() (string, error) { return x.f.ns.BootID, nil }
					return e
				}
				e := open(s)
				if mode == "caller-alias" {
					input.Source.Launches[0].Storage.RuntimeID = "caller-change"
					input.Namespace.Drain.Members[0].InstanceID = "caller-change"
					h = e.observation.history
					s = e.initial
					saved = s
				}
				oldBytes, err := os.ReadFile(x.sentinel)
				if err != nil {
					t.Fatal(err)
				}
				retained := x.sentinel
				oldStat, err := os.Stat(x.sentinel)
				if err != nil {
					t.Fatal(err)
				}
				commit := func(seq uint64, next projectnetwork.AnchorNamespace) error {
					if seq != saved.Sequence {
						t.Fatal("stale namespace commit")
					}
					if mode == "commit-intent" && next.Sequence == 1 || mode == "commit-ack" && next.Phase == "start-acknowledged" || mode == "commit-observed" && next.Phase == "observed" {
						return errors.New("commit failure")
					}
					saved = next
					if next.Sequence == 1 {
						switch mode {
						case "member-changed":
							for _, v := range x.f.containers {
								v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
								break
							}
						case "anchor-changed":
							x.fault = "anchor-process"
						case "publication-changed":
							x.fault = "missing-publication"
						case "engine-changed":
							x.f.f.engine.ID = "changed-engine"
						case "pier-replaced":
							p := filepath.Join(x.f.paths[s.Drain.Members[0].InstanceID], ".urb")
							if err := os.Rename(p, p+"-retained"); err != nil {
								t.Fatal(err)
							}
							if err := os.Mkdir(p, 0700); err != nil {
								t.Fatal(err)
							}
							if strings.HasPrefix(retained, p+string(os.PathSeparator)) {
								retained = p + "-retained" + strings.TrimPrefix(retained, p)
							}

						}
					}
					return nil
				}
				got, err := projectnetwork.ExecuteAnchorNamespace(context.Background(), h, s, e, commit, time.Now)
				if mode == "pass" || mode == "caller-alias" {
					expected := 0
					if state == "stopped" {
						expected = 1
					}
					if err != nil || got.Phase != "observed" || got.Validate(h) != nil || starts != expected || !reflect.DeepEqual(got.Drain, s.Drain) {
						t.Fatal("namespace succession failed", err, got.Phase, starts)
					}
					if _, err = e.InspectNamespace(context.Background(), s); err == nil {
						t.Fatal("preflight replay after completion")
					}
				} else {
					if err == nil || !reflect.DeepEqual(got, saved) {
						t.Fatal("failure not retained", err)
					}
					expected := 0
					if state == "stopped" && (mode == "start-error" || mode == "lost-reply" || mode == "bad-ack" || mode == "commit-ack" || mode == "commit-observed") {
						expected = 1
					}
					if starts != expected {
						t.Fatal("unexpected Start count", starts, expected)
					}
					if err = e.StartRecoveryAnchor(context.Background(), got); err == nil || starts != expected {
						t.Fatal("Start replayed")
					}
					if mode == "lost-reply" || mode == "bad-ack" || mode == "commit-ack" || mode == "commit-observed" || mode == "start-error" {
						observer := open(got)
						recovered, observeErr := projectnetwork.ObserveAnchorNamespace(context.Background(), h, got, observer, func(seq uint64, next projectnetwork.AnchorNamespace) error {
							if seq != saved.Sequence {
								t.Fatal("stale observation")
							}
							saved = next
							return nil
						}, time.Now)
						if mode == "start-error" {
							if observeErr == nil {
								t.Fatal("stopped anchor treated as started")
							}
						} else if observeErr != nil || recovered.Phase != "observed" || recovered.StartAcknowledged != got.StartAcknowledged {
							t.Fatal("read-only recovery failed", observeErr)
						}
						if starts != expected {
							t.Fatal("read-only recovery issued Start")
						}
					}
				}
				if x.f.signals != beforeSignals {
					t.Fatal("namespace selection signalled member")
				}
				data, err := os.ReadFile(retained)
				if err != nil || string(data) != string(oldBytes) {
					t.Fatal("namespace changed pier bytes", err)
				}
				info, err := os.Stat(retained)
				if err != nil || !os.SameFile(oldStat, info) {
					t.Fatal("namespace changed pier identity", err)
				}
			})
		}
	}
}
