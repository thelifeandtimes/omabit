package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/onboarding"
)

func remoteSetup(ctx context.Context, command string, args []string, config, format string) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	id := f.String("id", "", "resume an existing remote setup ID")
	var options onboarding.RemoteOptions
	var revoke *bool
	if command == "remote-setup-start" || command == "remote-setup-prepare" || command == "remote-setup" {
		f.StringVar(&options.Host, "host", "", "existing SSH alias or hostname")
		f.StringVar(&options.Name, "name", "", "connection name; defaults to the host")
		f.StringVar(&options.SSHConfig, "ssh-config", "", "optional absolute OpenSSH config file")
		f.StringVar(&options.LoginUser, "user", "", "optional SSH login account")
		f.UintVar(&options.Port, "port", 0, "optional existing SSH port")
		f.StringVar(&options.Owner, "owner", "", "existing non-root pier owner for fresh hosts")
		f.StringVar(&options.Role, "role", "operator", "operator or observer")
		revoke = f.Bool("revoke", false, "revoke this pairing after remote administrator authorization")
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 {
		return 2
	}
	var result any
	var err error
	switch command {
	case "remote-setup", "remote-setup-prepare":
		var status onboarding.RemoteStatus
		status, err = onboarding.PrepareRemote(ctx, options, config, *id, *revoke)
		if err == nil && command == "remote-setup" {
			err = onboarding.RunRemote(ctx, status.ID)
			if err == nil {
				status, err = onboarding.RemoteState(status.ID)
			}
		}
		result = status
	case "remote-setup-start":
		result, err = onboarding.LaunchRemote(ctx, options, config, *id, *revoke)
	case "remote-setup-status":
		result, err = onboarding.RemoteState(*id)
	case "remote-setup-run":
		err = onboarding.RunRemote(ctx, *id)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nRemote setup:", err)
		} else {
			fmt.Fprintln(os.Stdout, "\nRemote setup complete. Return to Omabit.")
		}
		// Keep authorization failures visible in the transient setup terminal.
		fmt.Fprintln(os.Stdout, "Press Enter to close this setup terminal.")
		var line string
		fmt.Scanln(&line)
		if err != nil {
			return 1
		}
		return 0
	default:
		return 2
	}
	if err != nil {
		if format == "text" {
			fmt.Fprintln(os.Stderr, err)
		} else {
			_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": map[string]string{"code": "remote_setup_error", "message": err.Error()}})
		}
		return 1
	}
	if format == "text" {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		_ = json.NewEncoder(os.Stdout).Encode(result)
	}
	return 0
}
