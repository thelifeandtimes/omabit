package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			writeSetupStatus(os.Stdout, s)
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

func writeSetupStatus(w io.Writer, s onboarding.Status) {
	fact := func(value string) string {
		if value == "" {
			return "unknown"
		}
		return value
	}
	fmt.Fprintf(w, "Local runner ready: %t\nHost account present: %t\nPrerequisites ready: %t\n", s.Ready, s.HostPresent, s.CanSetup)
	for _, c := range s.Checks {
		if c.Status != "pass" {
			fmt.Fprintf(w, "  %s: %s — %s\n", c.ID, c.Status, c.Detail)
		}
	}
	if r := s.Compatibility; r != nil {
		fmt.Fprintf(w, "Host compatibility\n  Policy: %s\n  Platform: %s\n  Distribution: %s %s · provider: %s · profile: %s\n", r.PolicyID, r.Platform, fact(r.Distribution.ID), r.Distribution.VersionID, fact(r.Distribution.Provider), fact(r.Distribution.Profile))
		fmt.Fprintf(w, "  Installed engine: %s\n", fact(r.InstalledEngineVersion))
		if r.Engine != nil {
			fmt.Fprintf(w, "  Running engine: %s · API: %s · minimum API: %s\n", r.Engine.Version, r.Engine.APIVersion, r.Engine.MinAPIVersion)
		} else {
			fmt.Fprintln(w, "  Running engine: not observed by this check")
		}
		fmt.Fprintf(w, "  Compatibility: %s\n  Qualification: %s\n  Provenance: %s\n  Restart pending: %t\n", r.Compatibility, r.Qualification, r.Provenance, r.RestartPending)
		if r.Engine != nil {
			fmt.Fprintf(w, "  Observe: %t · Stop: %t · Mutate: %t\n", r.CanObserve, r.CanStop, r.CanMutate)
		} else {
			fmt.Fprintln(w, "  Metadata only: live control permissions have not been evaluated.")
		}
		fmt.Fprintf(w, "  Observed: %s\n", r.ObservedAt.Format("2006-01-02T15:04:05Z07:00"))
		for _, p := range r.Packages {
			fmt.Fprintf(w, "  Package %s: %s %s · upstream: %s · provenance: %s\n", p.Role, p.Name, p.Version, p.UpstreamVersion, p.Provenance)
			if p.Detail != "" {
				fmt.Fprintf(w, "    %s\n", p.Detail)
			}
		}
		for _, c := range r.Checks {
			fmt.Fprintf(w, "  %s: %s — %s\n", c.ID, c.Status, c.Detail)
		}
		for _, guidance := range r.Guidance {
			fmt.Fprintln(w, "  "+guidance)
		}
	} else {
		fmt.Fprintln(w, "Host compatibility report unavailable. Update the local Omabit and runner helpers for structured checks.")
	}
	if s.Hint != "" {
		fmt.Fprintln(w, s.Hint)
	}
}
