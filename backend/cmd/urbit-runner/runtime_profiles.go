package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func configureRuntimeRegistry() {
	if p, err := ownerhost.Current(); err == nil && p != nil {
		runtimeprofile.ConfigureStore(filepath.Join(p.Home, ".local/share/urbit-runner/runtime-profiles"))
	}
}

func runRuntimeProfiles(ctx context.Context, command string, args []string, out, errs io.Writer) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(errs)
	var id, archive, manifest string
	if command == "runtime-profile-install" {
		f.StringVar(&id, "profile", "", "reviewed immutable runtime ID")
		f.StringVar(&archive, "archive", "", "exact checksummed archive; never a pier or executable path")
		f.StringVar(&manifest, "manifest", "", "owner-approved local runtime profile.json; generated from a channel or local Vere build")
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
	if manifest != "" {
		if id != "" || archive == "" {
			fmt.Fprintln(errs, "use --manifest with --archive, without --profile")
			return 2
		}
		input, err := os.OpenFile(manifest, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			fmt.Fprintln(errs, err)
			return 1
		}
		info, err := input.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
			input.Close()
			fmt.Fprintln(errs, "runtime manifest must be a regular file of at most 16 KiB")
			return 1
		}
		var p runtimeprofile.Profile
		dec := json.NewDecoder(io.LimitReader(input, 16385))
		dec.DisallowUnknownFields()
		err = dec.Decode(&p)
		var extra any
		if err == nil && dec.Decode(&extra) != io.EOF {
			err = errors.New("invalid runtime manifest framing")
		}
		input.Close()
		if err == nil {
			err = runtimeprofile.InstallMetadata(p)
		}
		if err != nil {
			fmt.Fprintln(errs, err)
			return 1
		}
		id = p.ID
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
	if err = runtimeprofile.SetChannel(p); err != nil {
		fmt.Fprintln(errs, err)
		return 1
	}
	if json.NewEncoder(out).Encode(p) != nil {
		return 1
	}
	return 0
}
