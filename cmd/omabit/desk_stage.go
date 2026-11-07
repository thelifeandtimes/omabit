package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func deskStageCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	name := flags.String("runner", "", "enrolled runner name")
	runner := flags.String("expected-runner-id", "", "selected runner identity")
	instance := flags.String("instance", "", "registered ship instance")
	desk := flags.String("desk", "", "desk name without %")
	directory := flags.String("source-dir", "", "local prepared desk source directory")
	raw := flags.String("plan-json", "", "unchanged desk-plan-stage output")
	id := flags.String("id", "", "source staging request ID")
	digest := flags.String("source-digest", "", "expected source digest for discard")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	if command != "desk-stage-discard" && *digest != "" {
		return fail(errors.New("source-digest is only valid for discard"))
	}
	var plan collection.DeskStagePlan
	switch command {
	case "desk-plan-stage":
		if *directory == "" || *raw != "" || *id != "" || *instance == "" || *desk == "" || (apppolicy.Target{InstanceID: *instance, Desk: *desk}).Validate() != nil {
			return fail(errors.New("require --instance, --desk and --source-dir for staging planning"))
		}
	case "desk-stage":
		if *directory == "" || *raw == "" || len(*raw) > 16384 || *id != "" || *instance != "" || *desk != "" {
			return fail(errors.New("require only --plan-json and --source-dir to stage source"))
		}
		d := json.NewDecoder(bytes.NewBufferString(*raw))
		d.DisallowUnknownFields()
		if d.Decode(&plan) != nil || d.Decode(new(any)) != io.EOF || plan.Validate() != nil {
			return fail(errors.New("invalid source staging plan"))
		}
	case "desk-stage-status":
		if !protocol.ValidDeskOperationID(*id) || *directory != "" || *raw != "" || *instance != "" || *desk != "" {
			return fail(errors.New("require only --id to inspect a source stage"))
		}
	case "desk-stage-discard":
		if !protocol.ValidDeskOperationID(*id) || !protocol.ValidDeskOperationID(*digest) || *directory != "" || *raw != "" || *instance != "" || *desk != "" {
			return fail(errors.New("require only --id and --source-digest to discard cached source"))
		}
	default:
		return fail(errors.New("unsupported staging command"))
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
	var out any
	switch command {
	case "desk-plan-stage":
		out, err = r.PlanDeskStage(ctx, *name, *runner, *instance, *desk, *directory)
	case "desk-stage":
		out, err = r.StageDesk(ctx, *name, *runner, plan, *directory)
	case "desk-stage-status":
		out, err = r.DeskStage(ctx, *name, *runner, *id)
	case "desk-stage-discard":
		out, err = r.DiscardDeskStage(ctx, *name, *runner, *id, *digest)
	}
	if err != nil {
		return fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
