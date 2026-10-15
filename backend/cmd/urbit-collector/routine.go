package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func runRoutine(ctx context.Context, command string, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	id := ""
	if len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
		id, args = args[0], args[1:]
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var update routines.Update
	var days string
	if command == "routine-set" {
		f.Uint64Var(&update.ExpectedRevision, "revision", 0, "expected policy revision (0 for absent policy)")
		f.Uint64Var(&update.ExpectedInstanceRevision, "instance-revision", 0, "selected instance configuration revision")
		f.BoolVar(&update.Config.Enabled, "enabled", false, "enable or disable the routine")
		f.StringVar(&update.Config.Action, "action", "", "pack or meld")
		f.StringVar(&days, "days", "", "comma-separated UTC weekdays 0=Sunday..6=Saturday")
		f.IntVar(&update.Config.HourUTC, "hour-utc", -1, "UTC hour 0..23")
		f.IntVar(&update.Config.MinuteUTC, "minute-utc", -1, "UTC minute 0..59")
		f.IntVar(&update.Config.WindowMinutes, "window-minutes", 0, "dispatch window 5..180 minutes")
		f.BoolVar(&update.Config.StopRunning, "stop-running", false, "explicitly allow downtime")
		f.BoolVar(&update.Config.Resume, "resume", false, "resume only if originally running and maintenance succeeds")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID %s INSTANCE\n", command)
			if command == "routine-set" {
				fmt.Fprintln(stdout, "  --revision N --instance-revision N --enabled=true|false --action pack|meld\n  --days 0,1,... --hour-utc N --minute-utc N --window-minutes N\n  --stop-running=true|false --resume=true|false\nAll policy flags are required. Times are UTC; missed windows are skipped, not caught up.")
			}
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || !validID(id) || !validID(runner) || !validID(expected) || socket != "" {
		return fail(errors.New("routine commands require a pinned enrolled runner and instance"), 2)
	}
	if command == "routine-set" {
		seen := map[string]bool{}
		f.Visit(func(f *flag.Flag) { seen[f.Name] = true })
		if len(seen) != 10 || update.ExpectedInstanceRevision == 0 {
			return fail(errors.New("all ten policy flags and a positive instance revision are required"), 2)
		}
		for _, part := range strings.Split(days, ",") {
			n, err := strconv.Atoi(part)
			if err != nil || strconv.Itoa(n) != part {
				return fail(errors.New("invalid UTC weekday list"), 2)
			}
			update.Config.Weekdays = append(update.Config.Weekdays, n)
		}
		if err := routines.Validate(&update.Config); err != nil {
			return fail(err, 2)
		}
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := r.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	if entry.RunnerID != expected {
		return fail(errors.New("runner selection changed"), 1)
	}
	c := entry.Client(true)
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err != nil {
		return fail(err, 1)
	}
	if hello.AccessRole != "operator" || !slices.Contains(hello.Features, "maintenance-schedules") {
		return fail(errors.New("routine policy requires operator access and maintenance-schedules capability"), 1)
	}
	var result any
	if command == "routine-set" {
		result, err = c.PutRoutine(ctx, id, update)
	} else {
		result, err = c.Routine(ctx, expected, id)
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
