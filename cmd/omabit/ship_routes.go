package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"os"
)

func shipRoutesCommand(ctx context.Context, command string, args []string, config string) int {
	fail := func(err error) int {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		return 1
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	name := f.String("runner", "", "enrolled runner name")
	runner := f.String("expected-runner-id", "", "selected runner identity")
	instance := f.String("instance", "", "registered ship instance")
	file := f.String("request-file", "", "reviewed URL update JSON file")
	raw := f.String("request-json", "", "reviewed URL update JSON")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 || !protocol.ValidID(*name) || !protocol.ValidID(*runner) {
		return fail(errors.New("require --runner and --expected-runner-id"))
	}
	var u protocol.ShipRoutesUpdate
	switch command {
	case "ship-routes":
		if !protocol.ValidID(*instance) || *file != "" || *raw != "" {
			return fail(errors.New("require only --instance to inspect ship URLs"))
		}
	case "ship-routes-configure":
		if *instance != "" || (*file == "") == (*raw == "") {
			return fail(errors.New("require one reviewed --request-file or --request-json"))
		}
		if err := readReleaseJSON(*raw, *file, 64<<10, &u); err != nil {
			return fail(err)
		}
		if u.Validate() != nil || u.Update.Binding.RunnerID != *runner {
			return fail(errors.New("invalid or mismatched ship URL update"))
		}
	default:
		return fail(errors.New("unsupported ship URL command"))
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
	var out protocol.ShipRoutes
	if command == "ship-routes" {
		out, err = r.ShipRoutes(ctx, *name, *runner, *instance)
	} else {
		out, err = r.PutShipRoutes(ctx, *name, *runner, u)
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
