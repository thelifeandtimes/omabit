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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func readReleaseJSON(raw, path string, limit int64, out any) error {
	if raw != "" && path != "" {
		return errors.New("select either a JSON value or its file")
	}
	data := []byte(raw)
	if path != "" {
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() || st.Size() > limit {
			return errors.New("release input must be a bounded regular JSON file")
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(st, opened) {
			f.Close()
			return errors.New("release input file changed")
		}
		data, err = io.ReadAll(io.LimitReader(f, limit+1))
		err = errors.Join(err, f.Close())
		if err != nil {
			return err
		}
	}
	if int64(len(data)) > limit {
		return errors.New("release input exceeds bound")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid release JSON")
	}
	return nil
}
func deskReleaseCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	name := f.String("runner", "", "enrolled runner name")
	runner := f.String("expected-runner-id", "", "selected runner identity")
	instance := f.String("instance", "", "registered ship instance")
	desk := f.String("desk", "", "desk name without %")
	action := f.String("action", "", "activate-local, publish-source or publish-treaty")
	checksRaw := f.String("checks-json", "", "caller-supplied configured check evidence for this Clay hash")
	checksFile := f.String("checks-file", "", "JSON file containing the configured release check evidence")
	raw := f.String("plan-json", "", "unchanged desk-plan-release output")
	file := f.String("plan-file", "", "saved unchanged desk-plan-release JSON file")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	var plan collection.DeskReleasePlan
	checks := []deskwork.ReleaseCheck{}
	switch command {
	case "desk-review-release", "desk-plan-release":
		if *raw != "" || *file != "" || *instance == "" || *desk == "" || (apppolicy.Target{InstanceID: *instance, Desk: *desk}).Validate() != nil || !deskwork.IsReleaseAction(*action) {
			return fail(errors.New("require --instance, --desk and a typed --action"))
		}
		if command == "desk-review-release" && (*checksRaw != "" || *checksFile != "") {
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
	case "desk-release":
		if *instance != "" || *desk != "" || *action != "" || *checksRaw != "" || *checksFile != "" || (*raw == "") == (*file == "") {
			return fail(errors.New("submit only one unchanged --plan-file or --plan-json"))
		}
		if err := readReleaseJSON(*raw, *file, 1<<20, &plan); err != nil {
			return fail(err)
		}
		if err := plan.Validate(); err != nil {
			return fail(err)
		}
	default:
		return fail(errors.New("unsupported release command"))
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
	case "desk-review-release":
		out, err = registry.ReviewDeskRelease(ctx, *name, *runner, *instance, *desk, *action)
	case "desk-plan-release":
		out, err = registry.PlanDeskRelease(ctx, *name, *runner, *instance, *desk, *action, checks)
	case "desk-release":
		out, err = registry.ReleaseDesk(ctx, *name, *runner, plan)
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
