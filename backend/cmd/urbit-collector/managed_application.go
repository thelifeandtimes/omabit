package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type managedApplicationSelection struct {
	Runner, RunnerID, InstanceID, ContainerID, Origin, Ship string
	Revision                                                uint64
}

func (s managedApplicationSelection) validate(snapshot protocol.Snapshot, now time.Time) error {
	if snapshot.RunnerID != s.RunnerID {
		return collection.ErrShipConflict
	}
	for _, i := range snapshot.Instances {
		if i.ID != s.InstanceID {
			continue
		}
		if i.ConfigRevision != s.Revision || i.Ship != s.Ship || i.ObservedState != "running" || i.RecoveryRequired || i.IdentityPending || i.ActiveOperationID != "" || i.Endpoint == "" || i.Resources == nil || i.Resources.ContainerID != s.ContainerID || i.ObservedAt == nil || now.Sub(*i.ObservedAt) > 30*time.Second || i.ObservedAt.After(now.Add(5*time.Second)) {
			return collection.ErrShipConflict
		}
		return nil
	}
	return collection.ErrShipConflict
}

// Managed login remains entirely in the collector process. Neither +code nor
// cookies enter argv, QML, stdout, or the application approval journal.
func prepareManagedApplicationShip(ctx context.Context, r *collection.Registry, s managedApplicationSelection) (collection.Ship, error) {
	entry, err := r.Get(ctx, s.Runner)
	if err != nil {
		return collection.Ship{}, err
	}
	if entry.RunnerID != s.RunnerID {
		return collection.Ship{}, collection.ErrShipConflict
	}
	client := entry.Client(true)
	defer client.Close()
	snapshot, err := client.Snapshot(ctx)
	if err != nil {
		return collection.Ship{}, err
	}
	if err = s.validate(snapshot, time.Now()); err != nil {
		return collection.Ship{}, err
	}
	ships, err := r.Ships(ctx)
	if err != nil {
		return collection.Ship{}, err
	}
	var selected collection.Ship
	for _, ship := range ships {
		if ship.Origin == s.Origin {
			if ship.Ship != s.Ship {
				return selected, collection.ErrShipConflict
			}
			selected = ship
			break
		}
	}
	// Reuse a working parent session without another credential retrieval/login.
	if selected.SessionRef != "" {
		c, e := sessionClient(ctx, r, selected)
		if e == nil {
			e = c.Check(ctx)
			c.Close()
		}
		if e == nil {
			return selected, nil
		}
		if !errors.Is(e, eyre.ErrAuth) && !errors.Is(e, credentials.ErrMissing) {
			return selected, e
		}
	}
	// Probe the explicitly selected HTTP origin before obtaining the runner code.
	probe, err := eyre.New(s.Origin, s.Ship)
	if err != nil {
		return selected, err
	}
	err = probe.Probe(ctx)
	probe.Close()
	if err != nil {
		return selected, err
	}
	code, err := r.LoginCode(ctx, s.Runner, s.RunnerID, s.InstanceID, s.Revision, s.ContainerID)
	if err != nil {
		return selected, err
	}
	defer func() { code.Code = "" }()
	if code.RunnerID != s.RunnerID || code.InstanceID != s.InstanceID || code.ConfigRevision != s.Revision || code.ContainerID != s.ContainerID || !protocol.ValidLoginCode(code.Code) {
		return selected, errors.New("runner login response did not match the selected ship")
	}
	if selected.ID == "" {
		// Stable source-derived name; registry still refuses an origin/identity clash.
		digest := sha256.Sum256([]byte(s.RunnerID + "\x00" + s.InstanceID + "\x00" + s.Origin))
		selected, err = r.EnrollShip(ctx, fmt.Sprintf("fleet-%x", digest[:10]), s.Origin, s.Ship, "secret-service")
		if err != nil {
			return selected, err
		}
	}
	if _, err = loginShip(ctx, r, selected, []byte(code.Code)); err != nil {
		return selected, err
	}
	current, err := r.Ship(ctx, selected.Name)
	if err == nil && (current.ID != selected.ID || current.Origin != s.Origin || current.Ship != s.Ship) {
		err = collection.ErrShipConflict
	}
	return current, err
}

func runManagedApplication(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	f := flag.NewFlagSet("app-managed-connect", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var s managedApplicationSelection
	s.Runner, s.RunnerID = runner, expected
	f.StringVar(&s.InstanceID, "instance", "", "selected managed instance")
	f.Uint64Var(&s.Revision, "revision", 0, "selected config revision")
	f.StringVar(&s.ContainerID, "container-id", "", "selected full runtime container ID")
	f.StringVar(&s.Origin, "url", "", "HTTP origin reachable from this desktop")
	f.StringVar(&s.Ship, "ship", "", "expected ship identity")
	var target collection.ApplicationTarget
	var request string
	var approve bool
	f.StringVar(&target.PluginID, "plugin-id", "", "installed plugin")
	f.StringVar(&target.Profile, "profile", "", "plugin profile")
	f.StringVar(&target.SHA256, "artifact-sha256", "", "approved receiver artifact")
	f.StringVar(&request, "request-id", "", "fresh application request ID")
	f.BoolVar(&approve, "approve-owner-session", false, "approve an independent ordinary Eyre session")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector --runner NAME --expected-runner-id ID app-managed-connect --instance ID --revision N --container-id ID --url ORIGIN --ship ~SHIP --plugin-id ID --profile PROFILE --artifact-sha256 HASH --request-id ID --approve-owner-session\n\nVerifies the selected running fleet ship and HTTP identity. Reuses its saved login or retrieves +code privately from the pinned runner, then approves one independent plugin session. No code/cookie is printed or saved in diagnostics; normal plugin traffic goes directly to the ship. A stale selection fails; no application delivery is retried. Use app-connections for recovery.")
			return 0
		}
		return fail(err, 2)
	}
	origin, err := eyre.Origin(s.Origin)
	if err != nil || origin != s.Origin || socket != "" || f.NArg() != 0 || !protocol.ValidID(s.Runner) || !protocol.ValidID(s.RunnerID) || !protocol.ValidID(s.InstanceID) || s.Revision == 0 || len(s.ContainerID) != 64 || !eyre.ValidShip(s.Ship) || !target.Valid() || !applicationRequestID(request) || !approve {
		return fail(errors.New("app-managed-connect requires a current fleet selection and explicit plugin approval; use --help"), 2)
	}
	env, err := applicationEnvironment()
	if err != nil {
		return fail(err, 1)
	}
	artifact, _, err := applicationauth.ResolveInstalled(filepath.Join(env["HOME"], ".config/omarchy/plugins"), target.PluginID, env)
	if err != nil {
		return fail(err, 1)
	}
	if artifact.Target() != target {
		return fail(collection.ErrApplicationConflict, 1)
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	if previous, archived, e := r.Application(ctx, request); e == nil {
		// Even completed duplicate requests must not cause a parent login rotation.
		if archived {
			return fail(collection.ErrApplicationArchived, 1)
		}
		_ = previous
		return fail(applicationauth.ErrRecovery, 1)
	} else if !errors.Is(e, collection.ErrApplicationConflict) {
		return fail(e, 1)
	}
	attempts, e := r.Applications(ctx)
	if e != nil {
		return fail(e, 1)
	}
	for _, a := range attempts {
		if a.Origin == s.Origin && a.Target.PluginID == target.PluginID && a.Target.Profile == target.Profile && (a.Phase != "closed" || a.ReceiverReconciliation) {
			return fail(applicationauth.ErrRecovery, 1)
		}
	}
	ship, err := prepareManagedApplicationShip(ctx, r, s)
	if err != nil {
		return fail(err, 1)
	}
	return runApplications(ctx, "app-connect", []string{ship.Name, "--expected-ship-id", ship.ID, "--expected-ship-revision", strconv.FormatUint(ship.Revision, 10), "--plugin-id", target.PluginID, "--profile", target.Profile, "--artifact-sha256", target.SHA256, "--request-id", request, "--approve-owner-session"}, dir, "", "", format, stdout, fail)
}
