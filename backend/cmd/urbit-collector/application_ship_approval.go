package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func readApplicationApprovalInput(raw, path string) ([]byte, error) {
	if (raw == "") == (path == "") {
		return nil, errors.New("select one --request-json or --request-file")
	}
	if raw != "" {
		if len(raw) > 16<<10 {
			return nil, collection.ErrApplicationConflict
		}
		return []byte(raw), nil
	}
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 16<<10 {
		return nil, errors.New("approval input must be a bounded regular file")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(st, opened) {
		return nil, errors.New("approval input changed")
	}
	return io.ReadAll(io.LimitReader(f, (16<<10)+1))
}

func runApplicationShipApproval(ctx context.Context, command string, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	instance := f.String("instance", "", "running managed ship to review")
	raw := f.String("request-json", "", "reviewed approval choice JSON")
	file := f.String("request-file", "", "reviewed approval choice JSON file")
	ack := f.Bool("allow-owner-session", false, "allow this plugin/profile to establish ordinary owner sessions for this ship using its registered URLs")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector [--config-dir DIR] [--runner NAME --expected-runner-id ID] "+command+" [--instance ID | --request-file FILE | --request-json JSON] [--allow-owner-session]\n\napp-ship-review reads a verified running ship and current standing choices. app-ship-approvals lists choices without contacting runners. app-ship-approve saves a revision-bound choice. An allow requires a reviewed selection, matching installed plugin helper and --allow-owner-session; it applies to any verified registered URL of exactly this ship incarnation. Revoke (allowed:false) uses the original binding/target and current approval revision; no runner, helper or online ship is required. Neither operation obtains credentials or starts a connection. Revocation blocks future attempts; existing sessions retain their separate Disconnect/recovery actions. Existing per-attempt approvals never become standing grants.")
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || socket != "" {
		return fail(errors.New("standing approval does not accept socket or trailing arguments"), 2)
	}
	var u collection.ApplicationShipApprovalRequest
	switch command {
	case "app-ship-review":
		if !protocol.ValidID(runner) || !protocol.ValidID(expected) || !protocol.ValidID(*instance) || *raw != "" || *file != "" || *ack {
			return fail(errors.New("review requires a pinned runner and --instance only"), 2)
		}
	case "app-ship-approvals":
		if runner != "" || expected != "" || *instance != "" || *raw != "" || *file != "" || *ack {
			return fail(errors.New("approval listing accepts no selection"), 2)
		}
	case "app-ship-approve":
		if *instance != "" {
			return fail(errors.New("approval requires the reviewed JSON choice"), 2)
		}
		data, err := readApplicationApprovalInput(*raw, *file)
		if err != nil {
			return fail(err, 2)
		}
		u, err = collection.DecodeApplicationShipApprovalRequest(data)
		if err != nil {
			return fail(err, 2)
		}
		if *u.Allowed {
			if !*ack || !protocol.ValidID(runner) || expected != u.Binding.RunnerID {
				return fail(errors.New("allow requires a pinned runner and --allow-owner-session"), 2)
			}
			env, err := applicationEnvironment()
			if err != nil {
				return fail(err, 1)
			}
			a, _, err := applicationauth.ResolveInstalled(filepath.Join(env["HOME"], ".config/omarchy/plugins"), u.Target.PluginID, env)
			if err != nil {
				return fail(err, 1)
			}
			if a.Target() != u.Target {
				return fail(collection.ErrApplicationConflict, 1)
			}
		} else if *ack || runner != "" || expected != "" {
			return fail(errors.New("revocation is local; omit runner selection and owner-session acknowledgement"), 2)
		}
	default:
		return fail(errors.New("unknown standing approval command"), 2)
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	var value any
	switch command {
	case "app-ship-review":
		value, err = r.ReviewApplicationShip(ctx, runner, expected, *instance)
	case "app-ship-approvals":
		var s collection.ApplicationShipApprovals
		s, err = r.ApplicationShipApprovals(ctx)
		value = s.View()
	case "app-ship-approve":
		value, err = r.ConfigureApplicationShipApproval(ctx, runner, expected, u)
	}
	if err != nil {
		return fail(err, 1)
	}
	if err = newOutput(stdout, format, false).Encode(value); err != nil {
		return fail(err, 1)
	}
	return 0
}
