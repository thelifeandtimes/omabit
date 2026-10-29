//go:build linux && amd64

package runtimequalify

import (
	"context"
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

func TestCurrentProjectReadinessAdapter(t *testing.T) {
	for _, generation := range []int{2, 3} {
		for _, instance := range []string{"receiver", "sender"} {
			for _, mode := range []string{"pass", "wrong-ship", "http-error", "target-restarted", "peer-restarted", "urb-replaced", "namespace-after-http", "engine-after-http", "unobserved-http", "arbitrary-endpoint", "stale-http", "caller-selection", "cancelled", "peer-stopped"} {
				t.Run(fmt.Sprintf("generation%d/%s/%s", generation, instance, mode), func(t *testing.T) {
					f, input, sentinel := resumedStopAdapterFixture(t, false, generation)
					selection := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness[instance]}
					p := input.Project.Plan
					c, err := projectnetwork.SelectMemberContext(p, input.Launches, input.Resumes, selection.Original)
					if err != nil {
						t.Fatal(err)
					}
					original := f.f.d.execute
					f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
						a := args[4:]
						namespaceRead := len(a) == 8 && a[0] == "container" && a[1] == "exec" && (a[2] == c.Binding.Anchor.ID || f.containers[a[2]] != nil) && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
						if len(a) < 2 || !(namespaceRead || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
							t.Fatal("readiness attempted mutation", a)
						}
						if namespaceRead {
							return fmt.Sprintf("%d:%d", f.ns.Device, f.ns.Inode), nil
						}
						return original(ctx, args, env)
					}
					oldBytes, err := os.ReadFile(sentinel)
					if err != nil {
						t.Fatal(err)
					}
					oldStat, err := os.Stat(sentinel)
					if err != nil {
						t.Fatal(err)
					}
					signals := f.signals
					raw, err := openCurrentProjectReadiness(context.Background(), selection, f.f.d)
					if err != nil {
						t.Fatal(err)
					}
					e := raw.(*projectCurrentReadinessEngine)
					e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
					peer := ""
					for id := range f.containers {
						if id != c.Member.Container.ID {
							peer = id
						}
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					reads := 0
					e.source.anchor.base.q.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) {
						reads++
						if endpoint != e.endpoint {
							t.Fatal("wrong endpoint")
						}
						switch mode {
						case "wrong-ship":
							return "~bus", nil
						case "http-error":
							return "", errors.New("private response")
						case "target-restarted":
							f.containers[c.Member.Container.ID].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						case "peer-restarted":
							f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						case "namespace-after-http":
							f.ns.Inode++
						case "engine-after-http":
							f.f.engine.ID = "changed-engine"
						case "urb-replaced":
							path := filepath.Join(f.paths[instance], ".urb")
							if err := os.Rename(path, path+"-retained"); err != nil {
								t.Fatal(err)
							}
							if err := os.Mkdir(path, 0700); err != nil {
								t.Fatal(err)
							}
						case "cancelled":
							cancel()
						case "peer-stopped":
							f.exit(peer, 0, false)
						}
						return e.ship, nil
					}
					if mode == "unobserved-http" || mode == "arbitrary-endpoint" || mode == "stale-http" {
						endpoint := e.endpoint
						if mode != "unobserved-http" {
							if _, err = e.Inspect(ctx, c); err != nil {
								t.Fatal(err)
							}
						}
						if mode == "arbitrary-endpoint" {
							endpoint = "http://127.0.0.1:9/~/host"
						}
						if mode == "stale-http" {
							e.inspectedAt = time.Now().Add(-time.Minute)
						}
						if _, err = e.ReadHostIdentity(ctx, endpoint); err == nil || reads != 0 {
							t.Fatal("unbound HTTP allowed")
						}
					} else if mode == "caller-selection" {
						c.Binding.Members[0].Container.ID = strings.Repeat("1", 64)
						if _, err = e.Inspect(ctx, c); err == nil {
							t.Fatal("caller changed selection")
						}
					} else {
						r, err := projectnetwork.CollectCurrentMemberReadiness(ctx, p, selection.Launches, selection.Resumes, selection.Original, e, time.Now)
						if mode == "pass" {
							if err != nil || r.Validate(p, selection.Launches, selection.Resumes, selection.Original) != nil || reads != 1 {
								t.Fatal("current readiness failed", err)
							}
						} else if err == nil || !reflect.DeepEqual(r, projectnetwork.CurrentMemberReadiness{}) || strings.Contains(err.Error(), "private") {
							t.Fatal("changed observation accepted", err)
						}
					}
					if f.signals != signals {
						t.Fatal("readiness signalled container")
					}
					retainedSentinel := sentinel
					if mode == "urb-replaced" {
						urb := filepath.Join(f.paths[instance], ".urb")
						if strings.HasPrefix(sentinel, urb+string(os.PathSeparator)) {
							retainedSentinel = urb + "-retained" + strings.TrimPrefix(sentinel, urb)
						}
					}
					retainedStat, statErr := os.Stat(retainedSentinel)
					if statErr != nil || !os.SameFile(oldStat, retainedStat) {
						t.Fatal("original pier data identity changed", statErr)
					}
					got, err := os.ReadFile(retainedSentinel)
					if err != nil || string(got) != string(oldBytes) {
						t.Fatal("pier contents changed", err)
					}
				})
			}
		}
	}
}
