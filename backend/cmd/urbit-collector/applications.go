package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

type applicationView struct {
	Archived               bool                         `json:"archived"`
	RequestID              string                       `json:"request_id"`
	ConnectionID           string                       `json:"connection_id"`
	ConnectionRevision     uint64                       `json:"connection_revision"`
	Origin                 string                       `json:"origin"`
	Ship                   string                       `json:"ship"`
	Target                 collection.ApplicationTarget `json:"target"`
	Authority              string                       `json:"authority"`
	Phase                  string                       `json:"phase"`
	Revision               uint64                       `json:"revision"`
	ReceiverReconciliation bool                         `json:"receiver_reconciliation"`
	Outcome                string                       `json:"outcome,omitempty"`
	ReceiverOutcome        string                       `json:"receiver_outcome,omitempty"`
	UpdatedAt              time.Time                    `json:"updated_at"`
	NextActions            []string                     `json:"next_actions"`
}

func appView(a collection.ApplicationAttempt) applicationView {
	next := []string{}
	switch a.Phase {
	case "connected":
		next = append(next, "app-disconnect")
	case "closed":
		if a.ReceiverReconciliation {
			next = append(next, "app-reconcile")
		} else {
			next = append(next, "app-archive")
		}
	default:
		next = append(next, "app-recover")
	}
	return applicationView{false, a.ID, a.ConnectionID, a.ConnectionRevision, a.Origin, a.Ship, a.Target, a.Authority, a.Phase, a.Revision, a.ReceiverReconciliation, a.Outcome, a.ReceiverOutcome, a.UpdatedAt, next}
}

type applicationResult struct {
	Connection *applicationView `json:"connection,omitempty"`
	Code       string           `json:"code"`
	Message    string           `json:"message,omitempty"`
}

func applicationFailure(err error) (string, string) {
	switch {
	case errors.Is(err, collection.ErrApplicationArchived):
		return "archived", "This request completed and was archived. Its approval cannot be replayed."
	case errors.Is(err, collection.ErrApplicationLimit):
		return "attempt_limit", "Archive fully closed connections to free an active slot. Active or uncertain connections must finish recovery first."
	case errors.Is(err, collection.ErrApplicationBusy):
		return "busy", "This connection has an operation in progress. Refresh its status."
	case errors.Is(err, collection.ErrApplicationConflict), errors.Is(err, collection.ErrShipConflict):
		return "selection_changed", "Refresh the selected ship, plugin and connection. An earlier attempt may require recovery."
	case errors.Is(err, applicationauth.ErrArtifact):
		return "artifact_unavailable", "The selected plugin helper changed or is unavailable. Refresh before approving; preserve retained helpers for recovery."
	case errors.Is(err, applicationauth.ErrNotIssued):
		return "not_issued", "No application session was issued. Check the saved ship login or use the plugin's standalone sign-in."
	case errors.Is(err, applicationauth.ErrCleanup):
		return "cleanup_pending", "Session cleanup is pending. Refresh and retry recovery; do not approve another session."
	case errors.Is(err, applicationauth.ErrRecovery):
		return "recovery_required", "The previous attempt needs recovery or receiver reconciliation; delivery was not retried."
	case errors.Is(err, applicationauth.ErrDelivery):
		return "delivery_failed", "The plugin did not confirm the handoff. Refresh and follow this attempt's recovery actions."
	default:
		return "application_unavailable", "The application operation could not complete. Its durable state was preserved; refresh before continuing."
	}
}

func applicationEnvironment() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, applicationauth.ErrArtifact
	}
	env := map[string]string{"HOME": home}
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "LANG"} {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}
	return env, nil
}

func applicationRequestID(value string) bool {
	data, err := hex.DecodeString(value)
	return err == nil && len(data) == 16 && value == strings.ToLower(value)
}

func runApplications(ctx context.Context, command string, args []string, dir, socket, runner, format string, stdout io.Writer, fail func(error, int) int) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	var selected, shipID, requestID string
	var shipRevision, revision uint64
	var target collection.ApplicationTarget
	var approve bool
	usage := command
	switch command {
	case "app-plugins", "app-connections":
	case "app-connection":
		usage += " REQUEST_ID"
	case "app-connect":
		usage += " SHIP_NAME --expected-ship-id ID --expected-ship-revision N --plugin-id ID --profile PROFILE --artifact-sha256 HASH --request-id ID --approve-owner-session"
		flags.StringVar(&shipID, "expected-ship-id", "", "saved ship enrollment ID from ships")
		flags.Uint64Var(&shipRevision, "expected-ship-revision", 0, "saved ship revision from ships")
		flags.StringVar(&target.PluginID, "plugin-id", "", "installed plugin ID from app-plugins")
		flags.StringVar(&target.Profile, "profile", "", "plugin profile from app-plugins")
		flags.StringVar(&target.SHA256, "artifact-sha256", "", "exact helper generation from app-plugins")
		flags.StringVar(&requestID, "request-id", "", "fresh 32 lowercase hexadecimal characters; retain on uncertain results")
		flags.BoolVar(&approve, "approve-owner-session", false, "approve this helper receiving an ordinary Eyre owner session with access across desks")
	default:
		usage += " REQUEST_ID --expected-revision N"
		flags.Uint64Var(&revision, "expected-revision", 0, "attempt revision from app-connections")
	}
	if command != "app-plugins" && command != "app-connections" && len(args) > 0 && args[0] != "--help" && args[0] != "-h" {
		selected, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: urbit-collector [--config-dir DIR] [--format text] %s\n\n", usage)
			fmt.Fprintln(stdout, "app-plugins lists locally installed connection-capable plugins. app-connections lists durable attempts without contacting ships.")
			fmt.Fprintln(stdout, "Approval gives this plugin an independent ordinary Eyre owner session. Eyre does not isolate desks. Plugins then connect directly to the ship and retain standalone sign-in.")
			fmt.Fprintln(stdout, "Disconnect revokes only this attempt's session. Recover never retries issuance or delivery. Reconcile removes this attempt's receiver state after revocation.")
			fmt.Fprintln(stdout, "app-archive removes a fully closed/reconciled attempt from the active list, retaining a permanent replay receipt. app-connection reads one active or archived request.")
			fmt.Fprintln(stdout, "No command installs plugins or backends. No runner, SSH, login code or cookie argument is accepted.")
			flags.SetOutput(stdout)
			flags.PrintDefaults()
			return 0
		}
		return fail(err, 2)
	}
	if socket != "" || runner != "" || flags.NArg() != 0 {
		return fail(errors.New("application connections do not accept runner/socket selection or trailing arguments"), 2)
	}
	if command == "app-connect" {
		if selected == "" || !applicationRequestID(shipID) || shipRevision == 0 || !applicationRequestID(requestID) || !target.Valid() || !approve {
			return fail(errors.New("app-connect requires the exact ship/plugin selection, request ID and --approve-owner-session; use --help"), 2)
		}
	} else if command != "app-plugins" && command != "app-connections" && (!applicationRequestID(selected) || (command != "app-connection" && revision == 0)) {
		return fail(errors.New("select a request ID and --expected-revision from app-connections"), 2)
	}
	out := newOutput(stdout, format, false)
	emit := func(a collection.ApplicationAttempt, err error) int {
		result := applicationResult{Code: "ok"}
		if a.ID != "" {
			view := appView(a)
			if errors.Is(err, collection.ErrApplicationArchived) {
				view.Archived = true
				view.NextActions = []string{}
			}
			result.Connection = &view
		}
		exit := 0
		if err != nil {
			result.Code, result.Message = applicationFailure(err)
			exit = 1
		}
		if err := out.Encode(result); err != nil {
			return fail(err, 1)
		}
		return exit
	}
	if command == "app-plugins" {
		env, err := applicationEnvironment()
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		plugins, err := applicationauth.DiscoverInstalled(filepath.Join(env["HOME"], ".config/omarchy/plugins"), env)
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		if err := out.Encode(plugins); err != nil {
			return fail(err, 1)
		}
		return 0
	}
	r, err := openRegistry(dir)
	if err != nil {
		return emit(collection.ApplicationAttempt{}, err)
	}
	if command == "app-connections" {
		attempts, err := r.Applications(ctx)
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		views := []applicationView{}
		for _, a := range attempts {
			views = append(views, appView(a))
		}
		if err := out.Encode(views); err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if command == "app-connection" {
		a, archived, err := r.Application(ctx, selected)
		if err != nil {
			return emit(a, err)
		}
		view := appView(a)
		view.Archived = archived
		if archived {
			view.NextActions = []string{}
		}
		if err := out.Encode(view); err != nil {
			return fail(err, 1)
		}
		return 0
	}
	if command == "app-archive" {
		a, err := r.ArchiveApplication(ctx, selected, revision)
		if err != nil {
			return emit(a, err)
		}
		view := appView(a)
		view.Archived = true
		view.NextActions = []string{}
		if err := out.Encode(applicationResult{Connection: &view, Code: "ok", Message: "Completed connection archived; its replay receipt is retained."}); err != nil {
			return fail(err, 1)
		}
		return 0
	}
	engine := applicationauth.New(r)
	if command == "app-connect" {
		previous, archived, lookupErr := r.Application(ctx, requestID)
		if lookupErr == nil && archived {
			return emit(previous, collection.ErrApplicationArchived)
		}
		if lookupErr != nil && !errors.Is(lookupErr, collection.ErrApplicationConflict) {
			return emit(previous, lookupErr)
		}
		s, err := r.Ship(ctx, selected)
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		if s.ID != shipID || s.Revision != shipRevision {
			return emit(collection.ApplicationAttempt{}, collection.ErrApplicationConflict)
		}
		env, err := applicationEnvironment()
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		a, _, err := applicationauth.ResolveInstalled(filepath.Join(env["HOME"], ".config/omarchy/plugins"), target.PluginID, env)
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		if a.Target() != target {
			return emit(collection.ApplicationAttempt{}, collection.ErrApplicationConflict)
		}
		retained, err := a.Retain(ctx, r.ApplicationCacheDir())
		if err != nil {
			return emit(collection.ApplicationAttempt{}, err)
		}
		attempt, err := r.ApproveApplication(ctx, s, requestID, target)
		if err != nil {
			return emit(attempt, err)
		}
		attempt, err = engine.Connect(ctx, attempt.ID, retained)
		return emit(attempt, err)
	}
	engine = engine.AtRevision(revision)
	var attempt collection.ApplicationAttempt
	switch command {
	case "app-disconnect":
		attempt, err = engine.Disconnect(ctx, selected)
	case "app-recover":
		attempt, err = engine.Recover(ctx, selected)
	case "app-reconcile":
		attempts, e := r.Applications(ctx)
		if e != nil {
			return emit(attempt, e)
		}
		for _, a := range attempts {
			if a.ID == selected {
				attempt = a
				break
			}
		}
		if attempt.ID == "" || attempt.Revision != revision {
			return emit(attempt, collection.ErrApplicationConflict)
		}
		env, e := applicationEnvironment()
		if e != nil {
			return emit(attempt, e)
		}
		receiver, e := applicationauth.LoadReceiver(r.ApplicationCacheDir(), attempt.Target.SHA256, env)
		if e != nil {
			return emit(attempt, e)
		}
		attempt, err = engine.Reconcile(ctx, selected, receiver)
	default:
		return fail(errors.New("unknown application command"), 2)
	}
	return emit(attempt, err)
}
