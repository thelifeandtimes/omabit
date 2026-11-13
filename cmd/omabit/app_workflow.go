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

// Setup authoring stays separate from the agent's read-only desk-settings command.
func appWorkflow(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	raw := flags.String("request-json", "", "selected workflow settings")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || len(*raw) == 0 || len(*raw) > 32768 {
		return fail(errors.New("require a bounded workflow request"))
	}
	var request struct {
		Name     string                  `json:"name"`
		RunnerID string                  `json:"runner_id"`
		Target   apppolicy.Target        `json:"target"`
		Update   *deskwork.SettingUpdate `json:"update,omitempty"`
	}
	d := json.NewDecoder(bytes.NewBufferString(*raw))
	d.DisallowUnknownFields()
	if d.Decode(&request) != nil || d.Decode(new(any)) != io.EOF || !protocol.ValidID(request.Name) || !protocol.ValidID(request.RunnerID) || deskwork.DefaultSetting(request.Target).Validate() != nil {
		return fail(errors.New("invalid workflow request"))
	}
	switch command {
	case "app-development-workflow":
		if request.Update != nil {
			return fail(errors.New("workflow inspection is read-only"))
		}
	case "app-development-configure":
		if request.Update == nil || request.Update.Target != request.Target || request.Update.Validate() != nil {
			return fail(errors.New("invalid workflow settings edit"))
		}
	default:
		return fail(errors.New("unsupported workflow command"))
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
	var out protocol.DeskSetting
	if request.Update == nil {
		out, err = r.DeskSetting(ctx, request.Name, request.RunnerID, request.Target)
	} else {
		out, err = r.SaveDeskSetting(ctx, request.Name, request.RunnerID, *request.Update)
	}
	if err != nil {
		return fail(err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fail(err)
	}
	return 0
}
