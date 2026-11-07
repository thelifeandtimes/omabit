package main

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

// One bounded maintenance step. There is no background service or hidden grant:
// the desktop chooses an exact durable attempt and revision, and every command
// rechecks permission/binding/helper before it can create a successor.
func runFleetMaintenance(ctx context.Context, r *collection.Registry, id string, revision uint64, lan []string, dir, format string, stdout io.Writer, fail func(error, int) int, emit func(collection.ApplicationAttempt, error) int) int {
	ctx, stop := context.WithTimeout(ctx, 180*time.Second)
	defer stop()
	a, archived, err := r.Application(ctx, id)
	if err != nil {
		return emit(a, err)
	}
	if archived {
		return emit(a, collection.ErrApplicationArchived)
	}
	if a.Revision != revision || a.Fleet.Version != 1 {
		return emit(a, collection.ErrApplicationConflict)
	}
	if (a.Phase == "connected" || a.Phase == "retiring" && a.Replacement.SuccessorID == "") && !a.Replacement.AutomaticDisabled {
		if a.Phase == "retiring" {
			all, err := r.Applications(ctx)
			if err != nil {
				return emit(a, err)
			}
			failures := 0
			var latest time.Time
			for _, child := range all {
				if child.Replacement.PreviousID == a.ID {
					if child.Phase != "closed" || child.ReceiverReconciliation {
						return emit(a, applicationauth.ErrRecovery)
					}
					failures++
					if child.UpdatedAt.After(latest) {
						latest = child.UpdatedAt
					}
				}
			}
			// Failed children remain visible. Automatic maintenance never archives
			// them to evade the retry budget. A manual retry is a separate action.
			if failures >= 3 {
				return emit(a, applicationauth.ErrRecovery)
			}
			if !latest.IsZero() && time.Since(latest) < 5*time.Minute {
				view := appView(a)
				if err := newOutput(stdout, format, false).Encode(applicationResult{Code: "ok", Health: "retry-backoff", Connection: &view}); err != nil {
					return fail(err, 1)
				}
				return 0
			}
		}
		u := collection.FleetApplicationRequest{Version: 1, Binding: a.Fleet.Request.Binding, Target: a.Target, GrantRevision: a.Fleet.GrantRevision, LANProfiles: lan}
		raw, _ := json.Marshal(u)
		return runFleetApplication(ctx, "app-fleet-reconnect", []string{"--request-json", string(raw), "--request-id", collection.NewSessionRef(), "--previous-request-id", id, "--expected-revision", strconv.FormatUint(revision, 10)}, dir, "", a.Fleet.RunnerName, a.Fleet.Request.Binding.RunnerID, format, stdout, fail)
	}
	if a.Replacement == (collection.ApplicationReplacement{}) {
		return emit(a, collection.ErrApplicationConflict)
	}
	env, err := applicationEnvironment()
	if err != nil {
		return emit(a, err)
	}
	receiver, err := applicationauth.LoadReceiver(r.ApplicationCacheDir(), a.Target.SHA256, env)
	if err != nil || receiver.Target() != a.Target || !receiver.SupportsReplacement() {
		return emit(a, applicationauth.ErrArtifact)
	}
	engine := applicationauth.New(r).AtRevision(a.Revision)
	if a.Phase == "delivering" && a.Replacement.PreviousID != "" {
		a, err = engine.RecoverReplacement(ctx, id, receiver)
	} else if a.Phase == "connected" && a.Replacement.AutomaticDisabled {
		a, err = engine.Disconnect(ctx, id)
	} else if a.Phase != "closed" {
		a, err = engine.Recover(ctx, id)
	}
	if err != nil {
		return emit(a, err)
	}
	if a.Phase == "closed" && a.ReceiverReconciliation {
		a, err = applicationauth.New(r).AtRevision(a.Revision).Reconcile(ctx, id, receiver)
		if err != nil {
			return emit(a, err)
		}
	}
	if a.Phase == "closed" && !a.ReceiverReconciliation && a.Replacement.SuccessorID != "" {
		a, err = r.ArchiveApplication(ctx, id, a.Revision)
		if err == nil {
			view := appView(a)
			view.Archived = true
			view.NextActions = []string{}
			if err := newOutput(stdout, format, false).Encode(applicationResult{Code: "ok", Connection: &view}); err != nil {
				return fail(err, 1)
			}
			return 0
		}
	}
	return emit(a, err)
}
