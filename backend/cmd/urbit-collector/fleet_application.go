package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func runFleetApplication(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet("app-fleet-connect", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	raw := f.String("request-json", "", "exact standing ship/helper grant JSON")
	file := f.String("request-file", "", "exact standing ship/helper grant JSON file")
	id := f.String("request-id", "", "fresh connection attempt ID")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID app-fleet-connect --request-file FILE --request-id ID\n\nConnects an exact ship incarnation and installed helper under existing standing permission. Request: version:1, binding, target, grant_revision; optional lan_profiles explicitly selects current LAN scopes. Omabit chooses a verified registered URL from this desktop's current network. Only unauthenticated probes may fall back; login and handoff are never replayed. This command cannot grant permission or print credentials. It creates an independent ordinary Eyre owner session in Secret Service, with no plaintext fallback or parent-login reuse. Eyre does not isolate desks. A repeated request requires inspection/recovery, not another connection. Use app-connections and app-connection for durable status.")
			return 0
		}
		return fail(err, 2)
	}
	if socket != "" || f.NArg() != 0 || !protocol.ValidID(runner) || !protocol.ValidID(expected) || !applicationRequestID(*id) {
		return fail(errors.New("app-fleet-connect requires a pinned runner, reviewed grant and fresh --request-id; use --help"), 2)
	}
	data, err := readApplicationApprovalInput(*raw, *file)
	if err != nil {
		return fail(err, 2)
	}
	u, err := collection.DecodeFleetApplicationRequest(data)
	if err != nil || u.Binding.RunnerID != expected {
		return fail(collection.ErrApplicationConflict, 2)
	}
	var probes []shiproute.Attempt
	emit := func(a collection.ApplicationAttempt, err error) int {
		result := applicationResult{Code: "ok", RouteAttempts: probes}
		if a.ID != "" {
			view := appView(a)
			if errors.Is(err, collection.ErrApplicationArchived) {
				view.Archived, view.NextActions = true, []string{}
			}
			result.Connection = &view
		}
		code := 0
		if err != nil {
			result.Code, result.Message = applicationFailure(err)
			code = 1
		}
		if err := newOutput(stdout, format, false).Encode(result); err != nil {
			return fail(err, 1)
		}
		return code
	}
	r, err := openRegistry(dir)
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	if previous, archived, err := r.Application(ctx, *id); err == nil {
		if archived {
			return emit(previous, collection.ErrApplicationArchived)
		}
		return emit(previous, applicationauth.ErrRecovery)
	} else if !errors.Is(err, collection.ErrApplicationConflict) {
		return emit(collection.ApplicationAttempt{}, err)
	}
	env, err := applicationEnvironment()
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	artifact, _, err := applicationauth.ResolveInstalled(filepath.Join(env["HOME"], ".config/omarchy/plugins"), u.Target.PluginID, env)
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	if artifact.Target() != u.Target {
		return emit(collection.ApplicationAttempt{}, collection.ErrApplicationConflict)
	}
	fleet, attempts, err := r.SelectFleetApplication(ctx, runner, u)
	probes = attempts
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	retained, err := artifact.Retain(ctx, r.ApplicationCacheDir())
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	attempt, err := r.ApproveFleetApplication(ctx, *id, u.Target, fleet, "secret-service")
	if err != nil {
		return emit(attempt, err)
	}
	attempt, err = applicationauth.New(r).Connect(ctx, attempt.ID, retained)
	return emit(attempt, err)
}
