//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"slices"
	"strings"
	"testing"
)

func TestProjectFleetCurrentContainersAndHTTPIdentity(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		t.Run(fmt.Sprint(wrong), func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
			raw, err := openCurrentProjectReadiness(context.Background(), s, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectCurrentReadinessEngine)
			e.source.anchor.base.q.d.stats = nil
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			e.source.anchor.base.q.d.hostIdentity = func(_ context.Context, url string) (string, error) {
				if wrong {
					return "~bus", nil
				}
				for _, m := range s.Project.Plan.Members {
					if url == fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) {
						return m.Ship, nil
					}
				}
				t.Fatal("unselected HTTP origin")
				return "", nil
			}
			before, err := projectFleetInspect(context.Background(), e)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := projectFleetRows(context.Background(), s, e, before)
			if err != nil || len(rows) != 2 {
				t.Fatal("fleet failed", err)
			}
			current, err := projectnetwork.ResolveMemberGenerations(s.Project.Plan, s.Launches, s.Resumes)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				if row.ProjectID != s.Project.ProjectID || row.ProjectLabel != s.Project.Plan.Label || row.ObservedState != "running" || row.ObservedAt == nil {
					t.Fatal("missing live metadata", row)
				}
				if (row.Health == "healthy") == wrong {
					t.Fatal("wrong HTTP identity accepted as healthy")
				}
				for _, m := range current.Members {
					if m.InstanceID == row.ID && m.Container.ID != row.Resources.ContainerID {
						t.Fatal("stale runtime selected")
					}
				}
			}
		})
	}
}

// Exit at each read boundary in the real bracketed observer. These fixtures
// execute the production inspection/contract code, without a Docker daemon.
func TestProjectFleetStopObservationTransitions(t *testing.T) {
	for _, mode := range []string{"first-bracket", "second-bracket", "namespace-probe", "nonzero", "oom", "changed-start", "changed-image", "changed-namespace"} {
		t.Run(mode, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
			raw, err := openCurrentProjectReadiness(context.Background(), s, f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectCurrentReadinessEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			id := ""
			for _, m := range e.source.members {
				if m.contract.selection.InstanceID == input.Stop.InstanceID {
					id = m.id
				}
			}
			if id == "" {
				t.Fatal("missing selected member")
			}
			reads, transitions, namespaceReads := 0, 0, 0
			original := f.f.d.execute
			e.source.anchor.base.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if len(a) > 2 && a[0] == "container" && a[1] == "inspect" && a[2] != "--format" && slices.Contains(a[2:], id) {
					reads++
					boundary := 2
					if mode == "second-bracket" {
						boundary = 3
					}
					if mode != "namespace-probe" && reads == boundary {
						transitions++
						f.exit(id, 0, false)
						switch mode {
						case "nonzero":
							*f.containers[id].State.ExitCode = 1
						case "oom":
							*f.containers[id].State.ExitCode, *f.containers[id].State.OOMKilled = 137, true
						case "changed-start":
							f.containers[id].State.StartedAt = "2026-10-23T12:00:00Z"
						case "changed-image":
							f.containers[id].Image = "sha256:" + strings.Repeat("f", 64)
						case "changed-namespace":
							f.ns.Inode++
						}
					}
				}
				if mode == "namespace-probe" && transitions == 0 && len(a) == 8 && a[0] == "container" && a[1] == "exec" && a[2] == id {
					transitions++
					f.exit(id, 0, false)
					return "", errors.New("selected process exited during namespace read")
				}
				if mode == "changed-namespace" && transitions != 0 && len(a) == 8 && a[0] == "container" && a[1] == "exec" {
					namespaceReads++
					if namespaceReads > 1 {
						return "5:43\n", nil
					}
				}
				return original(ctx, args, env)
			}
			snapshot, err := projectFleetInspect(context.Background(), e)
			refuse := strings.HasPrefix(mode, "changed-")
			if transitions != 1 || (err != nil) != refuse {
				t.Fatalf("transition=%d, error=%v", transitions, err)
			}
			if !refuse && (!snapshot.Observation.Complete || *snapshot.Containers[id].State.Running) {
				t.Fatal("exit was not freshly verified")
			}
		})
	}
}

func TestProjectFleetEmptyNamespaceBeforeFirstMember(t *testing.T) {
	f, input, _ := resumedStopAdapterFixture(t, false, 2)
	s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
	raw, err := openCurrentProjectReadiness(context.Background(), s, f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectCurrentReadinessEngine)
	e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	for _, m := range e.source.members {
		delete(f.containers, m.id)
	}
	e.source.members = nil
	if _, err := projectFleetInspect(context.Background(), e); err != nil {
		t.Fatal("empty namespace cannot accept first member", err)
	}
}
