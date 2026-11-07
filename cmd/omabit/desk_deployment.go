package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"io"
	"os"
)

func deskDeploymentCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	name := flags.String("runner", "", "enrolled runner name")
	runner := flags.String("expected-runner-id", "", "selected runner identity")
	stage := flags.String("stage-id", "", "original source stage ID")
	raw := flags.String("plan-json", "", "unchanged full desk-plan-commit output")
	file := flags.String("plan-file", "", "saved unchanged desk-plan-commit JSON file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	var plan collection.DeskCommitPlan
	switch command {
	case "desk-plan-commit", "desk-review-commit":
		if !protocol.ValidDeskOperationID(*stage) || *raw != "" || *file != "" {
			return fail(errors.New("require only --stage-id for deployment review"))
		}
	case "desk-commit":
		if *stage != "" || (*raw == "") == (*file == "") {
			return fail(errors.New("require one unchanged --plan-file or --plan-json"))
		}
		data := []byte(*raw)
		if *file != "" {
			st, err := os.Lstat(*file)
			if err != nil || !st.Mode().IsRegular() || st.Size() > 1<<20 {
				return fail(errors.New("commit plan must be a regular JSON file of at most 1 MiB"))
			}
			f, err := os.Open(*file)
			if err != nil {
				return fail(err)
			}
			opened, err := f.Stat()
			if err != nil || !os.SameFile(st, opened) {
				f.Close()
				return fail(errors.New("commit plan file changed"))
			}
			data, err = io.ReadAll(io.LimitReader(f, (1<<20)+1))
			closeErr := f.Close()
			if err = errors.Join(err, closeErr); err != nil {
				return fail(err)
			}
		}
		if len(data) > 1<<20 {
			return fail(errors.New("commit plan exceeds 1 MiB"))
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&plan) != nil || decoder.Decode(new(any)) != io.EOF || plan.Validate() != nil {
			return fail(errors.New("invalid staged commit plan"))
		}
	default:
		return fail(errors.New("unsupported deployment command"))
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
	case "desk-plan-commit":
		out, err = registry.PlanDeskCommit(ctx, *name, *runner, *stage)
	case "desk-review-commit":
		out, err = registry.ReviewDeskCommit(ctx, *name, *runner, *stage)
	case "desk-commit":
		out, err = registry.CommitDesk(ctx, *name, *runner, plan)
	}
	if err != nil {
		return fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
