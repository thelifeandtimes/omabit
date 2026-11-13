//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectDeskUsesCurrentMemberPortAndGeneration(t *testing.T) {
	for _, variant := range []string{"ok", "wrong-container", "wrong-ship", "changed-peer"} {
		t.Run(variant, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			selection := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
			raw, err := openCurrentProjectReadiness(context.Background(), selection, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectCurrentReadinessEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
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
