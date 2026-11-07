package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func deskCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	name := flags.String("runner", "", "enrolled runner name")
	runner := flags.String("expected-runner-id", "", "selected runner identity")
	instance := flags.String("instance", "", "registered ship instance")
	desk := flags.String("desk", "", "desk name without %")
	raw := flags.String("plan-json", "", "exact reviewed desk-plan-create output")
	id := flags.String("id", "", "desk operation ID")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *name == "" || *runner == "" {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	var plan collection.DeskPlan
	switch command {
	case "desk-list":
		if *raw != "" || *id != "" || *desk != "" || *instance == "" {
			return fail(errors.New("require only --instance for desk discovery"))
		}
	case "desk-inspect", "desk-plan-create":
		if *raw != "" || *id != "" || *instance == "" || *desk == "" {
			return fail(errors.New("require --instance and --desk for inspection"))
		}
	case "desk-create":
		if *raw == "" || len(*raw) > 16384 || *id != "" || *instance != "" || *desk != "" {
			return fail(errors.New("require only the exact --plan-json creation plan"))
		}
		decoder := json.NewDecoder(bytes.NewBufferString(*raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&plan) != nil || decoder.Decode(new(any)) != io.EOF || plan.Validate() != nil {
			return fail(errors.New("invalid desk creation plan"))
		}
	case "desk-operation", "desk-reconcile":
		if *id == "" || *raw != "" || *instance != "" || *desk != "" {
			return fail(errors.New("require only --id for a desk operation receipt"))
		}
	default:
		return fail(fmt.Errorf("unsupported desk command %s", command))
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
	var output any
	switch command {
	case "desk-list":
		output, err = registry.DiscoverDesks(ctx, *name, *runner, *instance)
	case "desk-inspect":
		output, err = registry.InspectDesk(ctx, *name, *runner, *instance, *desk)
	case "desk-plan-create":
		output, err = registry.PlanDeskCreation(ctx, *name, *runner, *instance, *desk)
	case "desk-create":
		output, err = registry.CreateDesk(ctx, *name, *runner, plan)
	case "desk-reconcile":
		output, err = registry.ReconcileDeskOperation(ctx, *name, *runner, *id)
	case "desk-operation":
		output, err = registry.DeskOperation(ctx, *name, *runner, *id)
	}
	if err != nil {
		return fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(output); err != nil {
		return fail(err)
	}
	return 0
}
