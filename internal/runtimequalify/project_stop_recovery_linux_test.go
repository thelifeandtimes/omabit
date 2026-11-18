//go:build linux && amd64

package runtimequalify

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func TestExpandedStopRecoveryObservesOnlyExactRetainedRuntime(t *testing.T) {
	for _, mode := range []string{"clean", "running", "nonzero", "oom", "missing-pier-evidence", "replaced-pier", "changed-namespace", "changed-process"} {
		t.Run(mode, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			s, err := SeedProjectExpansion(ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]})
			if err != nil {
				t.Fatal(err)
			}
			f.f.d.readBootID = func() (string, error) { return f.ns.BootID, nil }
			index := 0
			m := &s.Members[index]
			m.Phase = "stop-dispatched"
			m.Initialized = m.Creator.Storage
			m.Initialized.Identity, err = pierstore.Identify(f.paths[m.Member.InstanceID])
			if err != nil {
				t.Fatal(err)
			}
			id := m.Creator.Container.ID
			started, err := time.Parse(time.RFC3339Nano, f.containers[id].State.StartedAt)
			if err != nil {
				t.Fatal(err)
			}
			m.StopIntent = &ProjectExpandedStopIntent{OperationID: "op-stop", ContainerID: id, StartedAt: started, RecordedAt: time.Now().UTC()}
			if mode != "running" {
				f.exit(id, 0, false)
			}
			switch mode {
			case "nonzero":
				*f.containers[id].State.ExitCode = 1
			case "oom":
				*f.containers[id].State.ExitCode = 137
				*f.containers[id].State.OOMKilled = true
			case "missing-pier-evidence":
				m.Initialized.Identity = pierstore.Identity{}
			case "replaced-pier":
				m.Initialized.Identity.Inode++
			case "changed-namespace":
				s.Anchor.Namespace.Inode++
			case "changed-process":
				f.containers[id].State.StartedAt = started.Add(time.Second).Format(time.RFC3339Nano)
			}
			before, signals := len(f.f.calls), f.signals
			observation, err := observeExpandedProjectStop(context.Background(), s, m.Member.InstanceID, f.f.d)
			refuse := strings.HasPrefix(mode, "changed-") || mode == "missing-pier-evidence" || mode == "replaced-pier"
			if (err != nil) != refuse {
				t.Fatal(mode, err)
			}
			if !refuse && (observation.ContainerID != id || observation.Running != (mode == "running") || !reflect.DeepEqual(observation.Storage, m.Initialized)) {
				t.Fatal("lost exact runtime evidence", observation)
			}
			if f.signals != signals {
				t.Fatal("recovery observer sent a signal")
			}
			for _, call := range f.f.calls[before:] {
				command := strings.Join(call, " ")
				if strings.Contains(command, "container kill") || strings.Contains(command, "container start") || strings.Contains(command, "container rm") {
					t.Fatal("observer dispatched runtime effect", call)
				}
			}
		})
	}
}
