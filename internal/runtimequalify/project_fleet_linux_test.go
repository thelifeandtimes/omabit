//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
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
