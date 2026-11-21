package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func runRuntimeProfiles(ctx context.Context, command string, args []string, out, errs io.Writer) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(errs)
	var id, archive string
	if command == "runtime-profile-install" {
		f.StringVar(&id, "profile", "", "reviewed immutable runtime ID")
		f.StringVar(&archive, "archive", "", "exact checksummed archive; never a pier or executable path")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 {
		fmt.Fprintln(errs, "unexpected positional arguments")
		return 2
	}
	if command == "runtime-profiles" {
		if json.NewEncoder(out).Encode(runtimeprofile.Catalogue()) != nil {
			return 1
		}
		return 0
	}
	if id == "" || archive == "" {
		fmt.Fprintln(errs, "runtime-profile-install requires --profile and --archive; it stages an image and never selects or starts a ship")
		return 2
	}
	p, err := runtimequalify.InstallRuntimeProfile(ctx, id, archive)
	if err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	if json.NewEncoder(out).Encode(p) != nil {
		return 1
	}
	return 0
}
