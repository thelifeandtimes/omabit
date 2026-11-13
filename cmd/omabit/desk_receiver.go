package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func deskReceiverCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	name := f.String("runner", "", "enrolled receiver runner name")
	runner := f.String("expected-runner-id", "", "selected receiver runner identity")
	instance := f.String("instance", "", "receiver ship instance")
	desk := f.String("desk", "", "unused receiver desk name without %")
	publisherShip := f.String("publisher-ship", "", "continuing publisher ship, beginning with ~")
	publisherDesk := f.String("publisher-desk", "", "publisher desk name without %")
	expectedHash := f.String("expected-source-hash", "", "exact reviewed Clay hash to verify after source arrives; does not pin future updates")
	publisherInstance := f.String("publisher-instance", "", "required source member instance for project receivers")
	raw := f.String("plan-json", "", "unchanged desk-plan-install output")
	file := f.String("plan-file", "", "saved unchanged desk-plan-install JSON file")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	publisher := deskwork.Publisher{Ship: *publisherShip, Desk: *publisherDesk, ExpectedHash: *expectedHash}
	var plan collection.DeskReceiverPlan
	switch command {
	case "desk-receiver-status":
		if *raw != "" || *file != "" || *publisherShip != "" || *publisherDesk != "" || *expectedHash != "" || *publisherInstance != "" || *instance == "" || *desk == "" || (apppolicy.Target{InstanceID: *instance, Desk: *desk}).Validate() != nil {
			return fail(errors.New("select receiver instance and desk only"))
		}
	case "desk-review-install", "desk-plan-install":
		if *raw != "" || *file != "" || *instance == "" || *desk == "" || (apppolicy.Target{InstanceID: *instance, Desk: *desk}).Validate() != nil || publisher.Validate() != nil || *publisherInstance != "" && !protocol.ValidID(*publisherInstance) {
			return fail(errors.New("require receiver instance/desk, publisher ship/desk and expected source hash"))
		}
	case "desk-install":
		if *instance != "" || *desk != "" || *publisherShip != "" || *publisherDesk != "" || *expectedHash != "" || *publisherInstance != "" || (*raw == "") == (*file == "") {
			return fail(errors.New("submit only one unchanged --plan-file or --plan-json"))
		}
		if err := readReleaseJSON(*raw, *file, 128<<10, &plan); err != nil {
			return fail(err)
		}
		if err := plan.Validate(); err != nil {
			return fail(err)
		}
	default:
		return fail(errors.New("unsupported receiver command"))
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
	case "desk-receiver-status":
		out, err = registry.InspectDeskReceiver(ctx, *name, *runner, *instance, *desk)
	case "desk-review-install":
		out, err = registry.ReviewDeskReceiver(ctx, *name, *runner, *instance, *desk, *publisherInstance, publisher)
	case "desk-plan-install":
		out, err = registry.PlanDeskReceiver(ctx, *name, *runner, *instance, *desk, *publisherInstance, publisher)
	case "desk-install":
		out, err = registry.InstallDeskReceiver(ctx, *name, *runner, plan)
	}
	if err != nil {
		return fail(err)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err := e.Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
