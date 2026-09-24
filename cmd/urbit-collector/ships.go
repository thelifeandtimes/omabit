package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

type shipInfo struct {
	Name        string `json:"name"`
	Origin      string `json:"origin"`
	Ship        string `json:"ship"`
	Credentials string `json:"credentials"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
}
type shipData struct {
	Name string          `json:"name"`
	App  string          `json:"app"`
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}
type shipEvent struct {
	Name string `json:"name"`
	eyre.Event
}

func info(s collection.Ship, status string) shipInfo {
	return shipInfo{Name: s.Name, Origin: s.Origin, Ship: s.Ship, Credentials: s.Credentials, Status: status}
}
func storeFor(r *collection.Registry, s collection.Ship) (credentials.Store, error) {
	if s.Credentials == "secret-service" {
		return credentials.NewSecretService(), nil
	}
	return credentials.NewFile(r.CredentialsDir())
}
func secretKey(s collection.Ship) string { return credentials.Scope(s.ID, s.Origin, s.SessionRef) }

func sessionClient(ctx context.Context, r *collection.Registry, s collection.Ship) (*eyre.Client, error) {
	if s.SessionRef == "" {
		return nil, credentials.ErrMissing
	}
	store, err := storeFor(r, s)
	if err != nil {
		return nil, err
	}
	data, err := store.Load(ctx, secretKey(s))
	if err != nil {
		return nil, err
	}
	c, err := eyre.New(s.Origin, s.Ship)
	if err != nil {
		clear(data)
		return nil, err
	}
	if err := c.RestoreSession(data); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func loginShip(ctx context.Context, r *collection.Registry, s collection.Ship, code []byte) (shipInfo, error) {
	defer clear(code)
	store, err := storeFor(r, s)
	if err != nil {
		return shipInfo{}, err
	}
	c, err := eyre.New(s.Origin, s.Ship)
	if err != nil {
		return shipInfo{}, err
	}
	defer c.Close()
	if err := c.Login(ctx, code); err != nil {
		return shipInfo{}, err
	}
	data, err := c.ExportSession()
	if err != nil {
		return shipInfo{}, err
	}
	defer clear(data)
	ref := collection.NewSessionRef()
	key := credentials.Scope(s.ID, s.Origin, ref)
	cleanup := func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.Logout(cleanup)
		_ = store.Delete(cleanup, key)
	}
	if err := store.Save(ctx, key, data); err != nil {
		cleanup()
		return shipInfo{}, err
	}
	updated, err := r.SetShipSession(ctx, s, ref)
	if err != nil {
		// A CAS conflict proves the new reference was not committed. Other IO
		// failures can be post-rename durability failures: preserve that credential
		// for a status check rather than deleting a possibly referenced session.
		if errors.Is(err, collection.ErrShipConflict) {
			cleanup()
		}
		return shipInfo{}, err
	}
	if _, err := cleanupRetired(ctx, r, updated); err != nil {
		return info(updated, "authenticated"), errors.New("new session saved; retired credential cleanup is pending and can be retried with ship-logout")
	}
	return info(updated, "authenticated"), nil
}

func logoutShip(ctx context.Context, r *collection.Registry, s collection.Ship) (shipInfo, error) {
	if s.SessionRef != "" {
		updated, err := r.SetShipSession(ctx, s, "")
		if err != nil {
			return shipInfo{}, err
		}
		s = updated
	}
	remoteConfirmed, err := cleanupRetired(ctx, r, s)
	if err != nil {
		return shipInfo{}, errors.New("session detached; credential cleanup is pending, retry ship-logout")
	}
	result := info(s, "logged_out")
	if !remoteConfirmed {
		result.Message = "local credential cleared; server-side logout was not confirmed"
	}
	return result, nil
}

func cleanupRetired(ctx context.Context, r *collection.Registry, s collection.Ship) (bool, error) {
	if len(s.Retired) == 0 {
		return true, nil
	}
	store, err := storeFor(r, s)
	if err != nil {
		return false, err
	}
	confirmed := true
	for _, ref := range s.Retired {
		old := s
		old.SessionRef = ref
		// Only this enrollment's retired sessions are revoked, never all browser
		// sessions. The metadata keeps retry intent until deletion succeeds.
		if c, e := sessionClient(ctx, r, old); e == nil {
			if c.Logout(ctx) != nil {
				confirmed = false
			}
			c.Close()
		} else {
			confirmed = false
		}
		if err := store.Delete(ctx, secretKey(old)); err != nil {
			return false, err
		}
		if err := r.CompleteCredentialCleanup(ctx, s.ID, []string{ref}); err != nil {
			return false, err
		}
	}
	return confirmed, nil
}

func runShips(ctx context.Context, command string, args []string, dir, socket, runner, format string, input io.Reader, stdout io.Writer, fail func(error, int) int) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var name, origin, ship, adapter, app, p string
	var codeStdin bool
	var events uint
	var duration time.Duration
	usage := command
	switch command {
	case "ship-enroll":
		usage += " NAME --url ORIGIN --ship ~SHIP [--credentials secret-service|file]"
		flags.StringVar(&origin, "url", "", "HTTPS origin or literal loopback HTTP origin")
		flags.StringVar(&ship, "ship", "", "expected public ship identity, including ~")
		flags.StringVar(&adapter, "credentials", "secret-service", "secret-service (desktop default) or file (explicit private plaintext headless store)")
	case "ship-login":
		usage += " NAME --code-stdin"
		flags.BoolVar(&codeStdin, "code-stdin", false, "read web login code from stdin; code is never saved")
	case "ship-status", "ship-logout":
		usage += " NAME"
	case "ship-forget":
		usage += " NAME --url EXPECTED_ORIGIN"
		flags.StringVar(&origin, "url", "", "required expected enrolled origin; log out first")
	case "ship-scry", "ship-watch":
		usage += " NAME --app APP --path /PATH"
		flags.StringVar(&app, "app", "", "Gall agent name")
		flags.StringVar(&p, "path", "", "agent path; omit /x and .json for scries")
		if command == "ship-watch" {
			flags.UintVar(&events, "events", 0, "stop after this many diff events; zero is unlimited")
			flags.DurationVar(&duration, "duration", 0, "optional watch time limit, e.g. 30s")
		}
	case "ships":
	}
	if command != "ships" && len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector [--config-dir DIR] [--format text] %s\n\nShip sessions use Eyre directly and do not require a runner or SSH.\n", usage)
			flags.SetOutput(stdout)
			flags.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || socket != "" || runner != "" {
		return fail(errors.New("ship commands do not accept runner/socket selection or trailing arguments"), 2)
	}
	if command != "ships" && !validID(name) {
		return fail(errors.New("require a valid ship connection name"), 2)
	}
	if (command == "ship-scry" || command == "ship-watch") && (app == "" || !eyre.ValidPath(p)) {
		return fail(errors.New("require --app and an absolute agent --path"), 2)
	}
	if command == "ship-login" && !codeStdin {
		return fail(errors.New("ship-login requires --code-stdin; do not put login codes in arguments"), 2)
	}
	if duration < 0 {
		return fail(errors.New("watch duration must not be negative"), 2)
	}
	if command == "ship-enroll" || command == "ship-forget" {
		var err error
		origin, err = eyre.Origin(origin)
		if err != nil {
			return fail(err, 2)
		}
		if command == "ship-enroll" && (!eyre.ValidShip(ship) || (adapter != "secret-service" && adapter != "file")) {
			return fail(errors.New("require --ship and credentials secret-service or file"), 2)
		}
	}
	r, err := openRegistry(dir)
	if err != nil {
		return fail(err, 1)
	}
	output := newOutput(stdout, format, command == "ship-watch")
	var result any
	code := 0
	if command == "ships" {
		all, e := r.Ships(ctx)
		err = e
		views := []shipInfo{}
		for _, s := range all {
			state := "login_required"
			if s.SessionRef != "" {
				state = "session_stored"
			}
			views = append(views, info(s, state))
		}
		result = views
	} else if command == "ship-enroll" {
		s, e := r.EnrollShip(ctx, name, origin, ship, adapter)
		err = e
		state := "login_required"
		if s.SessionRef != "" {
			state = "session_stored"
		}
		result = info(s, state)
	} else if command == "ship-forget" {
		err = r.ForgetShip(ctx, name, origin)
		result = shipInfo{Name: name, Origin: origin, Status: "forgotten"}
	} else {
		s, e := r.Ship(ctx, name)
		if e != nil {
			return fail(e, 1)
		}
		switch command {
		case "ship-login":
			data, e := io.ReadAll(io.LimitReader(input, 129))
			if e != nil || len(data) > 128 {
				clear(data)
				return fail(errors.New("cannot read bounded login code from stdin"), 2)
			}
			result, err = loginShip(ctx, r, s, data)
		case "ship-logout":
			result, err = logoutShip(ctx, r, s)
		default:
			c, e := sessionClient(ctx, r, s)
			if e != nil {
				if command != "ship-status" {
					return fail(e, 1)
				}
				state := info(s, "login_required")
				if errors.Is(e, credentials.ErrStore) {
					state.Status = "credentials_unavailable"
				}
				state.Message = e.Error()
				result = state
				code = 1
				break
			}
			defer c.Close()
			if command == "ship-status" {
				e = c.Check(ctx)
				status := info(s, "authenticated")
				if e != nil {
					code = 1
					status.Status = "unavailable"
					status.Message = e.Error()
					if errors.Is(e, eyre.ErrAuth) {
						status.Status = "login_required"
					}
					if errors.Is(e, eyre.ErrIdentity) {
						status.Status = "identity_mismatch"
					}
				}
				result = status
			} else if command == "ship-scry" {
				data, e := c.Scry(ctx, app, p)
				err = e
				result = shipData{Name: name, App: app, Path: p, Data: data}
			} else {
				watchCtx := ctx
				cancel := func() {}
				if duration > 0 {
					watchCtx, cancel = context.WithTimeout(ctx, duration)
				}
				defer cancel()
				err = c.Watch(watchCtx, app, p, events, func(v eyre.Event) error { return output.Encode(shipEvent{Name: name, Event: v}) })
				if watchCtx.Err() != nil && errors.Is(err, watchCtx.Err()) {
					return 0
				}
				if err != nil {
					return fail(err, 1)
				}
				return 0
			}
		}
	}
	if err != nil {
		return fail(err, 1)
	}
	if err := output.Encode(result); err != nil {
		return fail(err, 1)
	}
	return code
}
