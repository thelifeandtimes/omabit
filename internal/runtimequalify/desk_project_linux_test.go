//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProjectDeskUsesCurrentMemberPortAndGeneration(t *testing.T) {
	for _, variant := range []string{"ok", "wrong-container", "wrong-ship", "changed-peer", "process-ok", "process-peer", "process-namespace", "process-ports-during-read"} {
		t.Run(variant, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			selection := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
			raw, err := openCurrentProjectReadiness(context.Background(), selection, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectCurrentReadinessEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			if variant == "process-namespace" {
				q := e.source.anchor.base.q
				execute := q.d.execute
				q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					out, err := execute(ctx, args, env)
					if err == nil && len(args) == 12 && reflect.DeepEqual(args[7:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
						return fmt.Sprintf("%d:%d", f.ns.Device, f.ns.Inode), nil
					}
					return out, err
				}
			}
			before, err := projectFleetInspect(context.Background(), e)
			if err != nil {
				t.Fatal(err)
			}
			selected := e.source.members[0]
			id, instance := selected.id, selected.contract.selection.InstanceID
			var public uint16
			var ship string
			for _, m := range selection.Project.Plan.Members {
				if m.InstanceID == instance {
					public = m.InternalHTTPPort
					ship = m.Ship
				}
			}
			path, err := pierstore.Absolute(e.source.anchor.base.q.d.home, selected.contract.selection.Storage.Path)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(path, ".http.ports"), []byte(fmt.Sprintf("%d insecure public\n12329 insecure loopback\n", public)), 0600); err != nil {
				t.Fatal(err)
			}
			calls, effects := 0, 0
			e.source.anchor.base.q.d.executeInput = func(ctx context.Context, args, env []string, body []byte) (string, error) {
				calls++
				joined := strings.Join(args, " ")
				if !strings.Contains(joined, "container exec --interactive "+id) || !strings.HasSuffix(joined, "http://127.0.0.1:12329") {
					t.Fatal("wrong member or port", joined)
				}
				if strings.Contains(string(body), `"dojo":"our"`) {
					if variant == "wrong-ship" {
						return `"~bus"`, nil
					}
					return `"` + ship + `"`, nil
				}
				effects++
				if variant == "changed-peer" {
					for peer, c := range f.containers {
						if peer != id {
							c.State.Pid++
						}
					}
				}
				return `">="`, nil
			}
			selectedID := id
			if variant == "wrong-container" {
				selectedID = strings.Repeat("f", 64)
			}
			runtime, err := deskFromProject(context.Background(), e, before, instance, selectedID)
			if strings.HasPrefix(variant, "process-") {
				if err != nil {
					t.Fatal(err)
				}
				proof, err := runtime.GrubberyProcess(t.Context())
				if err != nil || proof.Validate() != nil {
					t.Fatal("project process evidence missing", proof, err)
				}
				if variant == "process-peer" {
					for peer, c := range f.containers {
						if peer != id {
							c.State.Pid++
						}
					}
				}
				if variant == "process-namespace" {
					f.ns.Inode++
				}
				if variant == "process-ports-during-read" {
					q := e.source.anchor.base.q
					execute := q.d.execute
					reads := 0
					q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
						out, err := execute(ctx, args, env)
						if err == nil && len(args) == 12 && args[6] == id && reflect.DeepEqual(args[7:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
							reads++
							if reads == 2 {
								if err := os.WriteFile(filepath.Join(path, ".http.ports"), []byte(fmt.Sprintf("%d insecure public\n12330 insecure loopback\n", public)), 0600); err != nil {
									t.Fatal(err)
								}
							}
						}
						return out, err
					}
				}
				fresh, err := runtime.GrubberyProcess(t.Context())
				if variant == "process-ok" {
					if err != nil || fresh != proof {
						t.Fatal("unchanged fleet/process changed", fresh, err)
					}
				} else if err == nil {
					t.Fatal("changed peer/namespace accepted", fresh)
				}
				return
			}
			if err == nil {
				err = runtime.Commit(context.Background(), "omabit-test")
			}
			if variant == "ok" {
				if err != nil || calls != 3 || effects != 1 {
					t.Fatal(err, calls, effects)
				}
			} else if err == nil || effects > 1 {
				t.Fatal("changed project accepted", err, effects)
			}
		})
	}
}
