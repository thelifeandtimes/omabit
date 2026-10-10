//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"strings"
	"testing"
	"time"
)

func TestRoutineStopsMaintainsAndResumesOnlyWithConsent(t *testing.T) {
	for _, resume := range []bool{true, false} {
		t.Run(map[bool]string{true: "resume", false: "leave-stopped"}[resume], func(t *testing.T) {
			f, state, persist := managedFixture(t)
			ctx := context.Background()
			state, _, err := applyManaged(ctx, "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state.Routine = &routines.Run{Action: "pack", StopRunning: true, Resume: resume, WasRunning: true, Deadline: time.Now().Add(time.Minute)}
			state, obs, err := applyManaged(ctx, "routine-pack", state, persist, f.d)
			want := "stopped"
			if resume {
				want = "running"
			}
			if err != nil || obs.State != want || !state.Routine.Succeeded || state.Routine.FinishedAt == nil || state.Maintenance == nil || state.Maintenance.Kind != "pack" || f.downloads != 1 {
				t.Fatalf("routine %+v %v", obs, err)
			}
		})
	}
}

func TestRoutineRefusesBeforeStopAndDoesNotResumeOnFailure(t *testing.T) {
	for _, fault := range []string{"memory", "window", "consent", "utility"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			ctx := context.Background()
			state, _, err := applyManaged(ctx, "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state.Routine = &routines.Run{Action: "meld", StopRunning: true, Resume: true, WasRunning: true, Deadline: time.Now().Add(time.Minute)}
			before := len(f.events)
			switch fault {
			case "memory":
				f.d.admission = func(string) (uint64, uint64, error) { return 0, 8 << 30, nil }
			case "window":
				state.Routine.Deadline = time.Now().Add(-time.Second)
			case "consent":
				state.Routine.StopRunning = false
			case "utility":
				execute := f.d.execute
				f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					if strings.Contains(strings.Join(args, " "), "container wait ") {
						return "", errors.New("utility interrupted")
					}
					return execute(ctx, args, env)
				}
			}
			state, _, err = applyManaged(ctx, "routine-meld", state, persist, f.d)
			if err == nil {
				t.Fatal("invalid routine dispatched")
			}
			for _, event := range f.events[before:] {
				if event == "start-http-restart" || (fault != "utility" && event == "stop-http-boot") {
					t.Fatal("unsafe lifecycle continuation", event)
				}
			}
			if fault == "utility" && !state.MaintenanceUncertain {
				t.Fatal("utility uncertainty lost")
			}
		})
	}
}
