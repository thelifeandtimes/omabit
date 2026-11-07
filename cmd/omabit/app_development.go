package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"io"
	"os"
)

// Setup-only authoring transport. This is not exposed by urbit-collector and is
// not an OS security boundary against an unrestricted Unix-owner process.
type appDevelopmentEdit struct {
	Revision uint64                        `json:"revision"`
	Name     string                        `json:"name"`
	RunnerID string                        `json:"runner_id"`
	Update   apppolicy.Update              `json:"update"`
	Runners  []collection.AppPolicyRollout `json:"runners"`
}

func appDevelopment(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	raw := flags.String("request-json", "", "Setup's reviewed policy edit")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		return fail(errors.New("unexpected arguments"))
	}
	var edit appDevelopmentEdit
	if command != "app-development-status" {
		if len(*raw) == 0 || len(*raw) > 32768 {
			return fail(errors.New("require a bounded Setup request"))
		}
		d := json.NewDecoder(bytes.NewBufferString(*raw))
		d.DisallowUnknownFields()
		if d.Decode(&edit) != nil || d.Decode(new(any)) != io.EOF {
			return fail(errors.New("invalid Setup request"))
		}
	} else if *raw != "" {
		return fail(errors.New("status is read-only"))
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
	switch command {
	case "app-development-save":
		if err = edit.Update.Validate(); err != nil {
			return fail(err)
		}
		if edit.Update.Scope == "global" {
			g, err := r.SaveAppPolicyGlobal(ctx, edit.Revision, edit.Update.Gates, edit.Runners)
			if err != nil {
				return fail(err)
			}
			err = r.ApplyAppPolicyGlobal(ctx, g.Revision)
			if err != nil {
				return fail(err)
			}
		} else {
			if err = r.SaveAppPolicyScope(ctx, edit.Name, edit.RunnerID, edit.Update); err != nil {
				return fail(err)
			}
		}
	case "app-development-retry":
		if err = r.ApplyAppPolicyGlobal(ctx, edit.Revision); err != nil {
			return fail(err)
		}
	case "app-development-status":
	default:
		return fail(fmt.Errorf("unknown app development helper %s", command))
	}
	out, err := r.AppDevelopment(ctx)
	if err != nil {
		return fail(err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
