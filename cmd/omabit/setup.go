package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/onboarding"
)

func localSetup(ctx context.Context, command string, args []string, config, format string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println("Usage: omabit [--format text] setup | setup-status\n\nSet up this computer's official-runtime host, services and private desktop access.\nRun as your desktop user; only the host step asks for OS authorization. Existing\npiers, runner/SSH identities and other access grants are preserved. Prerequisites\nare checked first. setup-status does not install or generate keys.")
		return 0
	}
	if len(args) != 0 || os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Run local setup as your desktop user, without additional arguments.")
		return 2
	}
	if config == "" {
		var err error
		config, err = collection.DefaultDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	if command == "setup-status" {
		s, err := onboarding.Inspect(ctx, config)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if format == "text" {
			fmt.Printf("Local runner ready: %t\nHost account present: %t\nPrerequisites ready: %t\n", s.Ready, s.HostPresent, s.CanSetup)
			for _, c := range s.Checks {
				if c.Status != "pass" {
					fmt.Printf("  %s: %s — %s\n", c.ID, c.Status, c.Detail)
				}
			}
			if s.Hint != "" {
				fmt.Println(s.Hint)
			}
		} else {
			_ = encoder.Encode(s)
		}
		return 0
	}
	var outputErr error
	_, err := onboarding.Setup(ctx, config, func(e onboarding.Event) {
		if format == "text" {
			message := e.Message
			if message == "" {
				message = e.Step
			}
			fmt.Printf("%s: %s\n", e.Status, message)
		} else if outputErr == nil {
			outputErr = encoder.Encode(e)
		}
	})
	if err != nil || outputErr != nil {
		if err == nil {
			err = outputErr
		}
		if format == "text" {
			fmt.Fprintln(os.Stderr, "Setup stopped:", err)
		} else {
			_ = encoder.Encode(onboarding.Event{Step: "setup", Status: "failed", Message: err.Error()})
		}
		return 1
	}
	return 0
}
