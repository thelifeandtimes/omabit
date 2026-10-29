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

type hookedAnchorReview struct {
	projectnetwork.AnchorRecoveryObserver
	afterFirst func()
	reads      int
}

func (d *hookedAnchorReview) InspectAnchorRecovery(ctx context.Context, c projectnetwork.AnchorRecoveryContext) (projectnetwork.AnchorRecoveryObservation, error) {
	o, err := d.AnchorRecoveryObserver.InspectAnchorRecovery(ctx, c)
	d.reads++
	if d.reads == 1 && d.afterFirst != nil {
		d.afterFirst()
	}
	return o, err
}
func TestProjectAnchorRecoveryReviewAdapter(t *testing.T) {
	for _, state := range []string{"stopped", "restarted"} {
		for _, mode := range []string{"pass", "member-restart", "peer-restart", "pier-replaced", "engine-changed", "member-exited", "member-migrated", "anchor-changed", "foreign-config", "missing-publication", "foreign-bridge", "caller-context", "cancelled"} {
			t.Run(state+"/"+mode, func(t *testing.T) {
				f, input, sentinel := resumedStopAdapterFixture(t, false, 3)
				selection := ProjectAnchorRecoveryReviewSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Originals: []projectnetwork.InitialMemberReadiness{f.readiness["receiver"], f.readiness["sender"]}}
				c, err := projectnetwork.SelectAnchorRecoveryContext(selection.Project.Plan, selection.Launches, selection.Resumes, selection.Originals, nil)
				if err != nil {
					t.Fatal(err)
				}
				oldBytes, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				oldStat, err := os.Stat(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				oldSignals := f.signals
				original := f.f.d.execute
				fault := false
				when := time.Now().UTC().Add(-time.Millisecond).Format(time.RFC3339Nano)
				f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					a := args[4:]
					namespace := len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
					if len(a) < 2 || !(namespace || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
						t.Fatal("review attempted mutation", a)
					}
					if namespace {
						inode := f.ns.Inode
						if a[2] == c.Binding.Anchor.ID {
							if state == "stopped" {
								t.Fatal("probed stopped anchor")
							}
							inode++
							if fault && mode == "anchor-changed" {
								inode++
							}
						} else if fault && mode == "member-migrated" {
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
							}
							if fault && mode == "anchor-changed" && state == "stopped" {
								*v.State.ExitCode = 137
							}
							if fault && mode == "foreign-config" {
								v.HostConfig.Privileged = true
							}
							if fault && mode == "missing-publication" {
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
						if fault && mode == "foreign-bridge" {
							values[0].Containers[strings.Repeat("1", 64)] = json.RawMessage(`{}`)
						}
						b, _ := json.Marshal(values)
						return string(b), nil
					}
					return out, nil
				}
				f.f.d.hostIdentity = func(context.Context, string) (string, error) {
					t.Fatal("review attempted HTTP")
					return "", errors.New("unexpected")
				}
				raw, err := openProjectAnchorRecoveryReview(context.Background(), selection, f.f.d)
				if err != nil {
					t.Fatal(err)
				}
				e := raw.(*projectAnchorRecoveryReviewEngine)
				e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				retained := sentinel
				hook := &hookedAnchorReview{AnchorRecoveryObserver: e, afterFirst: func() {
					fault = true
					var target, peer string
					for _, m := range c.Members {
						if m.Context.Member.InstanceID == "receiver" {
							target = m.Context.Member.Container.ID
						} else {
							peer = m.Context.Member.Container.ID
						}
					}
					switch mode {
					case "member-restart":
						f.containers[target].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "peer-restart":
						f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "member-exited":
						f.exit(peer, 0, false)
					case "engine-changed":
						f.f.engine.ID = "changed-engine"
					case "pier-replaced":
						p := filepath.Join(f.paths["receiver"], ".urb")
						if err := os.Rename(p, p+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(p, 0700); err != nil {
							t.Fatal(err)
						}
						if strings.HasPrefix(sentinel, p+string(os.PathSeparator)) {
							retained = p + "-retained" + strings.TrimPrefix(sentinel, p)
						}
					case "cancelled":
						cancel()
					}
				}}
				if mode == "caller-context" {
					c.Members[0].StartedAt = time.Now().Add(-time.Hour)
					if _, err = e.InspectAnchorRecovery(ctx, c); err == nil {
						t.Fatal("changed context accepted")
					}
				} else {
					review, err := projectnetwork.CollectAnchorRecoveryReview(ctx, selection.Project.Plan, selection.Launches, selection.Resumes, selection.Originals, nil, hook, time.Now)
					if mode == "pass" {
						if err != nil || review.Validate(selection.Project.Plan, selection.Launches, selection.Resumes, selection.Originals, nil) != nil || hook.reads != 2 {
							t.Fatal("review failed", err)
						}
						want := "anchor-unavailable"
						if state == "restarted" {
							want = "namespace-changed"
						}
						if review.Assessment.State != want || review.Context.Binding.Namespace != f.ns {
							t.Fatal("adopted namespace or wrong diagnosis")
						}
					} else if err == nil || !reflect.DeepEqual(review, projectnetwork.AnchorRecoveryReview{}) {
						t.Fatal("fault accepted", err)
					}
				}
				if f.signals != oldSignals {
					t.Fatal("review signalled member")
				}
				got, err := os.ReadFile(retained)
				if err != nil || string(got) != string(oldBytes) {
					t.Fatal("pier data changed", err)
				}
				info, err := os.Stat(retained)
				if err != nil || !os.SameFile(oldStat, info) {
					t.Fatal("pier data identity changed", err)
				}
			})
		}
	}
}

func TestProjectRecoveryAnchorDoesNotRelaxInitialStart(t *testing.T) {
	for _, mode := range []string{"exited", "oom", "created", "future-exit", "reversed-time", "privileged", "bad-publication"} {
		t.Run(mode, func(t *testing.T) {
			c, n, v := resourceContractFixture(t)
			runningAnchorFixture(&v, &n)
			v.State.Status = "exited"
			v.State.Pid = 0
			*v.State.Running = false
			*v.State.ExitCode = 143
			v.State.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			v.NetworkSettings.Ports = map[string]json.RawMessage{}
			switch mode {
			case "oom":
				*v.State.OOMKilled = true
				*v.State.ExitCode = 137
			case "created":
				v.State.Status = "created"
				v.State.StartedAt = "0001-01-01T00:00:00Z"
				v.State.FinishedAt = "0001-01-01T00:00:00Z"
				*v.State.ExitCode = 0
			case "future-exit":
				v.State.FinishedAt = time.Now().Add(time.Minute).Format(time.RFC3339Nano)
			case "reversed-time":
				v.State.FinishedAt = "2026-10-23T11:59:00Z"
			case "privileged":
				v.HostConfig.Privileged = true
			case "bad-publication":
				v.NetworkSettings.Ports = v.HostConfig.PortBindings
			}
			err := c.validateRecoveryAnchor(v, v.ID, n.ID)
			if (err == nil) != (mode == "exited" || mode == "oom") {
				t.Fatal("wrong recovery classification", err)
			}
			if mode != "created" && c.validateStoppedAnchor(v, v.ID, n.ID) == nil {
				t.Fatal("exited anchor gained initial Start authority")
			}
		})
	}
}
