package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
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
	var sshHost, sshConfig string
	var newName string
	var revision uint64
	var watch bool
	interval := time.Second
	usage := command
	switch command {
	case "enroll", "runner-edit":
		usage += " NAME (--socket /absolute/path/runner.sock | --ssh HOST_ALIAS [--ssh-config /absolute/config])"
		flags.StringVar(&path, "socket", "", "required absolute clean runner socket path")
		flags.StringVar(&sshHost, "ssh", "", "OpenSSH host alias (exclusive with --socket)")
		flags.StringVar(&sshConfig, "ssh-config", "", "optional absolute SSH config file (default: normal OpenSSH config)")
		if command == "runner-edit" {
			usage += " --runner-id ID --revision N --name NEW_NAME"
			flags.StringVar(&expectedID, "runner-id", "", "required original pinned runner identity")
			flags.Uint64Var(&revision, "revision", 0, "required original connection revision")
			flags.StringVar(&newName, "name", "", "required new connection name (may be unchanged)")
		}
	case "forget":
		usage += " NAME --runner-id ID"
		flags.StringVar(&expectedID, "runner-id", "", "required expected enrolled runner identity")
	case "runner-accept-mode":
		usage += " NAME --runner-id ID --revision N --mode development-managed"
		flags.StringVar(&expectedID, "runner-id", "", "original pinned runner ID")
		flags.Uint64Var(&revision, "revision", 0, "original connection revision")
		flags.StringVar(&newName, "mode", "", "explicit new mode: development-managed")
	case "inventory":
		usage += " [--watch] [--interval 1s]"
		flags.BoolVar(&watch, "watch", false, "emit per-runner NDJSON observations; Ctrl+C exits successfully")
		flags.DurationVar(&interval, "interval", time.Second, "per-runner delay after each poll, at least 100ms")
	}
	if (command == "enroll" || command == "runner-edit" || command == "forget" || command == "runner-accept-mode") && len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
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
	if (command == "enroll" || command == "runner-edit" || command == "forget" || command == "runner-accept-mode") && !validID(name) {
		return fail(errors.New("a valid connection NAME is required"), 2)
	}
	if command == "enroll" || command == "runner-edit" {
		if (path == "") == (sshHost == "") || (path != "" && (!collection.ValidSocket(path) || sshConfig != "")) {
			return fail(errors.New("enroll requires either --socket with an absolute clean path or --ssh with a configured host alias"), 2)
		}
		if sshHost != "" {
			if err := (client.SSHConfig{Host: sshHost, ConfigFile: sshConfig}).Validate(); err != nil {
				return fail(err, 2)
			}
		}
	}
	if command == "runner-edit" && (!validID(expectedID) || !validID(newName) || revision == 0) {
		return fail(errors.New("runner-edit requires --runner-id, --revision and --name from the selected enrollment"), 2)
	}
	if command == "runner-accept-mode" && (!validID(expectedID) || revision == 0 || newName != "development-managed") {
		return fail(errors.New("require the selected runner ID/revision and explicit --mode development-managed"), 2)
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
		if sshHost != "" {
			result, err = registry.EnrollSSH(ctx, name, client.SSHConfig{Host: sshHost, ConfigFile: sshConfig})
		} else {
			result, err = registry.Enroll(ctx, name, path)
		}
	case "runners":
		result, err = registry.List(ctx)
	case "runner-accept-mode":
		result, err = registry.AcceptDevelopmentMode(ctx, name, expectedID, revision)
	case "runner-edit":
		proposed := collection.Connection{Name: newName, Transport: "unix", Socket: path}
		if sshHost != "" {
			proposed.Transport, proposed.Socket, proposed.SSH = "ssh", "", &client.SSHConfig{Host: sshHost, ConfigFile: sshConfig}
		}
		result, err = registry.Update(ctx, name, expectedID, revision, proposed)
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
