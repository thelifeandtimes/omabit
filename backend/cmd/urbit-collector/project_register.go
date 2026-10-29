package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runProjectRegister(ctx context.Context, args []string, config, socket, runner, expected, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector --runner NAME --expected-runner-id ID project-register
       --plan /absolute/offline-plan.json --review-digest HASH --request-id ID
       --confirm-reservation [--replay-only]

Reserve the complete fresh-fake membership and runner-loopback ports reviewed with
project-plan. No pier, container or ship is created or started. Requires a pinned
enrolled operator runner and explicit shared-administration consent in the plan.
Keep the same plan, digest and request ID if the reply is lost. --replay-only looks
up that actor's original receipt and never creates a missing reservation. It is
not an automatic retry or a request to start ships. Use project-status on the
returned project ID to inspect saved history.
`
	flags := flag.NewFlagSet("project-register", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var file, digest, requestID string
	var confirm, replay bool
	flags.StringVar(&file, "plan", "", "absolute offline project-plan JSON file")
	flags.StringVar(&digest, "review-digest", "", "reviewed canonical plan digest")
	flags.StringVar(&requestID, "request-id", "", "caller-chosen durable request ID")
	flags.BoolVar(&confirm, "confirm-reservation", false, "reserve complete membership and ports only")
	flags.BoolVar(&replay, "replay-only", false, "lookup original receipt without creating")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(stdout, usage)
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 || runner == "" || socket != "" || !protocol.ValidID(expected) || !protocol.ValidID(requestID) || !filepath.IsAbs(file) || !confirm {
		return fail(errors.New("project-register requires a pinned --runner/--expected-runner-id, absolute --plan, --request-id and --confirm-reservation; use --help"), 2)
	}
	f, err := os.Open(file)
	if err != nil {
		return fail(err, 2)
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		f.Close()
		return fail(errors.New("project plan must be a readable regular file"), 2)
	}
	if info.Size() > 1024*1024 {
		f.Close()
		return fail(errors.New("project plan exceeds 1 MiB"), 2)
	}
	data, readErr := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	closeErr := f.Close()
	if err = errors.Join(readErr, closeErr); err != nil {
		return fail(err, 2)
	}
	if len(data) > 1024*1024 {
		return fail(errors.New("project plan exceeds 1 MiB"), 2)
	}
	var view projectPlanView
	if json.Unmarshal(data, &view) != nil || view.SchemaVersion != 1 || view.RuntimeAuthorized || view.ReservationCheck != "not-performed" {
		return fail(errors.New("expected offline project-plan JSON; no live runtime authority is accepted"), 2)
	}
	canonical, err := view.Plan.Digest()
	if err != nil || digest != canonical || view.ReviewDigest != canonical {
		return fail(errors.New("--review-digest and saved plan must match the canonical reviewed plan"), 2)
	}
	req := protocol.ProjectRegisterRequest{RequestID: requestID, Label: view.Plan.Label, ConfirmSharedAdministration: view.Plan.SharedAdministration, ConfirmReservation: true, ReviewDigest: digest}
	for _, m := range view.Plan.Members {
		req.Members = append(req.Members, protocol.ProjectMemberInput{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	registry, err := openRegistry(config)
	if err != nil {
		return fail(err, 1)
	}
	entry, err := registry.Get(ctx, runner)
	if err != nil {
		return fail(err, 1)
	}
	pin := entry.Identity()
	if pin.RunnerID != expected {
		return fail(client.ErrIdentityMismatch, 1)
	}
	c := entry.Client(true)
	defer c.Close()
	if replay {
		ctx = client.WithCreationReplayOnly(ctx)
	}
	result, err := c.RegisterProject(ctx, pin, req)
	if err != nil {
		return fail(err, 1)
	}
	if format == "text" {
		_, err = fmt.Fprintf(stdout, "Project %s reserved at revision %d. No ships started or piers created.\nOperation: %s\nRequest: %s\nUse project-status %s to read saved history.\n", result.Operation.ProjectID, result.Operation.ConfigRevision, result.Operation.ID, result.Operation.RequestID, result.Operation.ProjectID)
	} else {
		err = newOutput(stdout, format, false).Encode(result)
	}
	if err != nil {
		return fail(err, 1)
	}
	return 0
}
