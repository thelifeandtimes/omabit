//go:build linux && amd64

package runtimequalify

import (
	"strings"
	"testing"
	"time"
)

func projectExitedMemberFixture(t *testing.T, c projectMemberContract) (container, time.Time) {
	t.Helper()
	v := projectMemberContainerFixture(t, c)
	start := time.Date(2026, 10, 23, 12, 0, 0, 123, time.UTC)
	v.State.Status = "exited"
	v.State.StartedAt = start.Format(time.RFC3339Nano)
	v.State.FinishedAt = start.Add(time.Minute).Format(time.RFC3339Nano)
	return v, start
}

func TestProjectMemberExitContractRetainsObservedFailures(t *testing.T) {
	c := projectMemberContractFixture(t)
	for _, exit := range []int{0, 1, 137, 143, 255} {
		for _, oom := range []bool{false, true} {
			v, start := projectExitedMemberFixture(t, c)
			*v.State.ExitCode, *v.State.OOMKilled = exit, oom
			if err := c.validateExitedContainer(v, v.ID, start); err != nil {
				t.Fatal(exit, oom, err)
			}
			for _, running := range []bool{false, true} {
				if c.validateContainer(v, v.ID, running) == nil {
					t.Fatal("exit admitted as original Create/Start", exit, oom, running)
				}
			}
		}
	}
	v, start := projectExitedMemberFixture(t, c)
	if c.validateExitedContainer(v, v.ID, time.Time{}) == nil || c.validateExitedContainer(v, v.ID, start.Add(time.Nanosecond)) == nil {
		t.Fatal("unbound process start accepted")
	}
	// Compare the actual instant, not formatting or time-zone spelling.
	v.State.StartedAt = start.In(time.FixedZone("offset", 3600)).Format(time.RFC3339Nano)
	if err := c.validateExitedContainer(v, v.ID, start); err != nil {
		t.Fatal(err)
	}
}

func TestProjectMemberExitContractRefusesUncertainProcessAndChangedOwnership(t *testing.T) {
	c := projectMemberContractFixture(t)
	for name, mutate := range map[string]func(*container){
		"missing-state":    func(v *container) { v.State = nil },
		"missing-running":  func(v *container) { v.State.Running = nil },
		"missing-exit":     func(v *container) { v.State.ExitCode = nil },
		"missing-oom":      func(v *container) { v.State.OOMKilled = nil },
		"running":          func(v *container) { *v.State.Running = true },
		"created":          func(v *container) { v.State.Status = "created" },
		"dead-status":      func(v *container) { v.State.Status = "dead" },
		"pid":              func(v *container) { v.State.Pid = 1 },
		"paused":           func(v *container) { v.State.Paused = true },
		"restarting":       func(v *container) { v.State.Restarting = true },
		"dead":             func(v *container) { v.State.Dead = true },
		"engine-error":     func(v *container) { v.State.Error = "uncertain" },
		"negative-exit":    func(v *container) { *v.State.ExitCode = -1 },
		"overflow-exit":    func(v *container) { *v.State.ExitCode = 256 },
		"missing-start":    func(v *container) { v.State.StartedAt = "" },
		"changed-start":    func(v *container) { v.State.StartedAt = "2026-10-23T12:00:01Z" },
		"missing-finish":   func(v *container) { v.State.FinishedAt = "" },
		"zero-finish":      func(v *container) { v.State.FinishedAt = "0001-01-01T00:00:00Z" },
		"backwards-finish": func(v *container) { v.State.FinishedAt = "2026-10-23T11:59:59Z" },
		"replacement":      func(v *container) { v.ID = strings.Repeat("d", 64) },
		"renamed":          func(v *container) { v.Name += "-foreign" },
		"image":            func(v *container) { v.Image = "foreign" },
		"labels":           func(v *container) { v.Config.Labels[projectLabel+".runtime"] = strings.Repeat("d", 32) },
		"mount":            func(v *container) { v.Mounts[0].Source += "-peer" },
		"init":             func(v *container) { v.HostConfig.Init = nil },
		"anchor":           func(v *container) { v.HostConfig.NetworkMode = "container:" + strings.Repeat("d", 64) },
		"signal":           func(v *container) { v.Config.StopSignal = "SIGKILL" },
		"restart-policy":   func(v *container) { v.HostConfig.RestartPolicy.Name = "always" },
	} {
		t.Run(name, func(t *testing.T) {
			v, start := projectExitedMemberFixture(t, c)
			id := v.ID
			mutate(&v)
			if c.validateExitedContainer(v, id, start) == nil {
				t.Fatal("unbound exit accepted")
			}
		})
	}
}
