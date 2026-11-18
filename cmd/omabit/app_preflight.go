package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func appPreflight(ctx context.Context, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	f := flag.NewFlagSet("app-preflight", flag.ContinueOnError)
	var req appdev.PreflightRequest
	f.StringVar(&req.Name, "runner", "", "selected enrollment name")
	f.StringVar(&req.RunnerID, "expected-runner-id", "", "required enrolled runner identity")
	f.StringVar(&req.Workflow, "workflow", "", "gall or grubbery")
	f.StringVar(&req.InstanceID, "instance", "", "optional selected instance; requires desk")
	f.StringVar(&req.Desk, "desk", "", "optional selected desk; requires instance")
	effects := f.String("effects", "create-desk,commit", "comma-separated create-desk,commit,publish effects")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 {
		return 2
	}
	req.Effects = strings.Split(*effects, ",")
	if err := req.Validate(); err != nil {
		return fail(err)
	}
	if config == "" {
		var err error
		config, err = collection.DefaultDir()
		if err != nil {
			return fail(err)
		}
	}
	r, err := collection.Open(config)
	if err != nil {
		return fail(err)
	}
	out, err := r.AppDevelopmentPreflight(ctx, req)
	if err != nil {
		return fail(err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fail(err)
	}
	if !out.Compatible() {
		return 1
	}
	return 0
}
