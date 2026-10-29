package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/onboarding"
	"os"
)

func localServiceUpgrade(ctx context.Context, command string, args []string, config, format string) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	expected := flags.String("expected-runner-id", "", "runner identity from the displayed check")
	target := flags.String("target-sha256", "", "exact target executable digest from the displayed check")
	ack := flags.Bool("acknowledge-host-changes", false, "authorize this local service update")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || os.Geteuid() == 0 || command == "service-upgrade" && (!*ack || *expected == "" || *target == "") || command == "service-upgrade-status" && len(args) != 0 {
		fmt.Fprintln(os.Stderr, desktopCommandHelp[command])
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
	emit := func(e onboarding.UpgradeEvent) {
		if format == "text" {
			fmt.Printf("%s: %s\n", e.Status, e.Message)
		} else {
			_ = encoder.Encode(e)
		}
	}
	var s onboarding.UpgradeStatus
	var err error
	if command == "service-upgrade-status" {
		s, err = onboarding.InspectUpgrade(ctx, config)
	} else {
		s, err = onboarding.UpgradeServices(ctx, config, *expected, *target, emit)
	}
	if err != nil {
		emit(onboarding.UpgradeEvent{Step: "upgrade", Status: "failed", Message: err.Error()})
		return 1
	}
	if command == "service-upgrade-status" {
		if format == "text" {
			fmt.Printf("%s\nRunner: %s\nInstalled build: %s\nTarget build: %s\nTarget SHA256: %s\n", s.Message, s.RunnerID, s.InstalledBuild, s.TargetBuild, s.TargetSHA256)
		} else {
			_ = encoder.Encode(s)
		}
	}
	return 0
}
