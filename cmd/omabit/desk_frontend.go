package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func deskFrontendCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	name := f.String("runner", "", "enrolled runner name")
	runner := f.String("expected-runner-id", "", "selected runner identity")
	stage := f.String("stage-id", "", "verified stage containing the built frontend")
	checksRaw := f.String("checks-json", "", "configured checks bound to both Clay hash and artifact digest")
	checksFile := f.String("checks-file", "", "JSON file containing the configured frontend check evidence")
	raw := f.String("plan-json", "", "unchanged desk-plan-frontend output")
	file := f.String("plan-file", "", "saved unchanged desk-plan-frontend JSON file")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	var plan collection.DeskFrontendPlan
	checks := []deskwork.FrontendCheck{}
	switch command {
	case "desk-review-frontend", "desk-plan-frontend":
		if *raw != "" || *file != "" || !protocol.ValidDeskOperationID(*stage) {
			return fail(errors.New("require --stage-id for the built frontend"))
		}
		if command == "desk-review-frontend" && (*checksRaw != "" || *checksFile != "") {
			return fail(errors.New("review reads the required checks; supply evidence when planning"))
		}
		if *checksRaw != "" || *checksFile != "" {
			if err := readReleaseJSON(*checksRaw, *checksFile, 64<<10, &checks); err != nil {
				return fail(err)
			}
		}
		if checks == nil || len(checks) > 16 {
			return fail(errors.New("require a bounded check array, empty when none are configured"))
		}
	case "desk-upload-frontend":
		if *stage != "" || *checksRaw != "" || *checksFile != "" || (*raw == "") == (*file == "") {
			return fail(errors.New("submit only one unchanged --plan-file or --plan-json"))
		}
		if err := readReleaseJSON(*raw, *file, 1<<20, &plan); err != nil {
			return fail(err)
		}
		if err := plan.Validate(); err != nil {
			return fail(err)
		}
	default:
		return fail(errors.New("unsupported frontend command"))
	}
	if config == "" {
		var err error
		config, err = collection.DefaultDir()
		if err != nil {
			return fail(err)
		}
	}
	registry, err := collection.Open(config)
	if err != nil {
		return fail(err)
	}
	var out any
	switch command {
	case "desk-review-frontend":
		out, err = registry.ReviewDeskFrontend(ctx, *name, *runner, *stage)
	case "desk-plan-frontend":
		out, err = registry.PlanDeskFrontend(ctx, *name, *runner, *stage, checks)
	case "desk-upload-frontend":
		out, err = registry.UploadDeskFrontend(ctx, *name, *runner, plan)
	}
	if err != nil {
		return fail(err)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err = e.Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
