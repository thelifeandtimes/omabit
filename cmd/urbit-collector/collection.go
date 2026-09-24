package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func openRegistry(dir string) (*collection.Registry, error) {
	if dir == "" {
		var err error
		dir, err = collection.DefaultDir()
		if err != nil {
			return nil, err
		}
	}
	return collection.Open(dir)
}

func runCollection(ctx context.Context, command string, args []string, dir, socket, runner, format string, stdout io.Writer, fail func(error, int) int) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var name, path, expectedID string
	var watch bool
	interval := time.Second
	usage := command
	switch command {
	case "enroll":
		usage += " NAME --socket /absolute/path/runner.sock"
		flags.StringVar(&path, "socket", "", "required absolute clean runner socket path")
	case "forget":
		usage += " NAME --runner-id ID"
		flags.StringVar(&expectedID, "runner-id", "", "required expected enrolled runner identity")
	case "inventory":
		usage += " [--watch] [--interval 1s]"
		flags.BoolVar(&watch, "watch", false, "emit per-runner NDJSON observations; Ctrl+C exits successfully")
		flags.DurationVar(&interval, "interval", time.Second, "per-runner delay after each poll, at least 100ms")
	}
	if (command == "enroll" || command == "forget") && len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector [--config-dir /private/directory] %s\n\n%s\n", usage, scope)
			flags.SetOutput(stdout)
			flags.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || socket != "" || runner != "" {
		return fail(errors.New("collection commands do not accept a global runner/socket selection or trailing arguments; enroll's --socket follows NAME"), 2)
	}
	if dir != "" && (!filepath.IsAbs(dir) || filepath.Clean(dir) != dir) {
		return fail(errors.New("--config-dir must be absolute and clean"), 2)
	}
	if (command == "enroll" || command == "forget") && !validID(name) {
		return fail(errors.New("a valid connection NAME is required"), 2)
	}
	if command == "enroll" && !collection.ValidSocket(path) {
		return fail(errors.New("enroll requires --socket with an absolute clean path after NAME"), 2)
	}
	if command == "forget" && !validID(expectedID) {
		return fail(errors.New("forget requires --runner-id with the expected enrolled identity"), 2)
	}
	if command == "inventory" {
		intervalSet := false
		flags.Visit(func(f *flag.Flag) { intervalSet = intervalSet || f.Name == "interval" })
		if interval < 100*time.Millisecond || (intervalSet && !watch) {
			return fail(errors.New("--interval requires --watch and must be at least 100ms"), 2)
		}
	}
	registry, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	encoder := newOutput(stdout, format, watch)
	var result any
	code := 0
	switch command {
	case "enroll":
		result, err = registry.Enroll(ctx, name, path)
	case "runners":
		result, err = registry.List(ctx)
	case "forget":
		err = registry.Forget(ctx, name, expectedID)
		result = struct {
			Name     string `json:"name"`
			RunnerID string `json:"runner_id"`
			Status   string `json:"status"`
		}{name, expectedID, "forgotten"}
	case "inventory":
		entries, listErr := registry.List(ctx)
		if listErr != nil {
			if watch && ctx.Err() != nil && errors.Is(listErr, ctx.Err()) {
				return 0
			}
			return fail(listErr, 1)
		}
		if watch {
			err := collection.Watch(ctx, entries, interval, func(v collection.Observation) error { return encoder.Encode(v) })
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return 0
			}
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		inventory := collection.Collect(ctx, entries)
		result = inventory
		if !inventory.Complete {
			code = 1 // the full partial result is still emitted on stdout
		}
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := encoder.Encode(result); err != nil {
		return fail(err, 1)
	}
	return code
}
