package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shipaccess"
)

var fleetEnrollmentName = regexp.MustCompile(`^fleet-[0-9a-f]{20}$`)

type shipInfo struct {
	Managed     bool   `json:"managed,omitempty"`
	ID          string `json:"id,omitempty"`
	Revision    uint64 `json:"revision"`
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
	return shipInfo{Managed: fleetEnrollmentName.MatchString(s.Name), ID: s.ID, Revision: s.Revision, Name: s.Name, Origin: s.Origin, Ship: s.Ship, Credentials: s.Credentials, Status: status}
}
func storeFor(r *collection.Registry, s collection.Ship) (credentials.Store, error) {
	return shipaccess.Store(r, s)
}
func secretKey(s collection.Ship) string { return shipaccess.Key(s) }

func sessionClient(ctx context.Context, r *collection.Registry, s collection.Ship) (*eyre.Client, error) {
	return shipaccess.Open(ctx, r, s)
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
	updated, err := r.SaveShipSession(ctx, s, ref, func(save context.Context) error {
		return store.Save(save, key, data)
	})
	if err != nil {
		// Publication errors can follow rename. Re-read before revoking; if state
		// is unavailable, the durable intent remains for a later logout/retry.
		check, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		current, readErr := r.Ship(check, s.Name)
		cancel()
		if errors.Is(err, collection.ErrShipConflict) || readErr == nil && current.ID == s.ID && current.SessionRef != ref {
			remote, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.Logout(remote)
			cancel()
			if readErr == nil && current.ID == s.ID {
				_, _ = cleanupRetired(context.Background(), r, current)
			}
		}
		return shipInfo{}, err
	}
	if _, err := cleanupRetired(ctx, r, updated); err != nil {
		return info(updated, "authenticated"), errors.New("new session saved; retired credential cleanup is pending and can be retried with ship-logout")
	}
	return info(updated, "authenticated"), nil
}

func logoutShip(ctx context.Context, r *collection.Registry, s collection.Ship) (shipInfo, error) {
	// Failed/interrupted saves can fill every cleanup slot while a session is
	// still active. Drain the selected retired references before needing a slot
	// for that session. Cleanup never includes its active reference.
	remoteConfirmed, err := cleanupRetired(ctx, r, s)
	if err != nil {
		return shipInfo{}, errors.New("retired credential cleanup is pending, retry ship-logout")
	}
	current, err := r.Ship(ctx, s.Name)
	if err != nil {
		return shipInfo{}, err
	}
	// Cleanup does not advance the session revision. A rotation during that IO
	// belongs to another command; do not adopt it as this logout's new target.
	if current.ID != s.ID || current.Revision != s.Revision || current.SessionRef != s.SessionRef {
		return shipInfo{}, collection.ErrShipConflict
	}
	if current.SessionRef != "" {
		updated, err := r.SetShipSession(ctx, current, "")
		if err != nil {
			return shipInfo{}, err
		}
		current = updated
	}
	confirmed, err := cleanupRetired(ctx, r, current)
	if err != nil {
		return shipInfo{}, errors.New("session detached; credential cleanup is pending, retry ship-logout")
	}
	result := info(current, "logged_out")
	if !remoteConfirmed || !confirmed {
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
		remote, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if c, e := sessionClient(remote, r, old); e == nil {
			if c.Logout(remote) != nil {
				confirmed = false
			}
			c.Close()
		} else {
			confirmed = false
		}
		cancel()
		// A stalled remote response or canceled caller cannot consume either the
		// local deletion budget or the independent journal-completion budget.
		local, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := store.Delete(local, secretKey(old))
		cancel()
		if err != nil {
			return false, err
		}
		finish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = r.CompleteCredentialCleanup(finish, s.ID, []string{ref})
		cancel()
		if err != nil {
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
	var expectedID string
	if command != "ships" && command != "ship-enroll" {
		flags.StringVar(&expectedID, "expected-ship-id", "", "optional immutable enrollment ID from UI selection")
	}
	var codeStdin, uniqueShip bool
	var events uint
	var duration time.Duration
	usage := command
	switch command {
	case "ship-enroll":
		flags.BoolVar(&uniqueShip, "unique-ship", false, "refuse another enrollment for this hosted identity")
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
		var s collection.Ship
		var e error
		if uniqueShip {
			s, e = r.EnrollHostedShip(ctx, name, origin, ship, adapter)
		} else {
			s, e = r.EnrollShip(ctx, name, origin, ship, adapter)
		}
		err = e
		state := "login_required"
		if s.SessionRef != "" {
			state = "session_stored"
		}
		result = info(s, state)
	} else if command == "ship-forget" {
		err = r.ForgetShipExpected(ctx, name, origin, expectedID)
		result = shipInfo{Name: name, Origin: origin, Status: "forgotten"}
	} else {
		s, e := r.Ship(ctx, name)
		if e != nil {
			return fail(e, 1)
		}
		if expectedID != "" && s.ID != expectedID {
			return fail(collection.ErrShipConflict, 1)
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
