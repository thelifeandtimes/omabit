package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type hostCheckDeps struct {
	inspect func() hostcompat.Report
	owner   func() (*ownerhost.Profile, error)
	probe   func(context.Context, string, string) (hostcompat.Engine, error)
}

func runHostCheck(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runHostCheckWith(ctx, args, stdout, stderr, hostCheckDeps{hostcompat.InspectCurrent, ownerhost.Current, hostcompat.ProbeEngine})
}

func runHostCheckWith(ctx context.Context, args []string, stdout, stderr io.Writer, deps hostCheckDeps) int {
	format := "json"
	fail := func(message string, code int) int {
		if format == "text" {
			fmt.Fprintln(stderr, "Host check:", message)
		} else {
			_ = json.NewEncoder(stderr).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "host_check_error", Message: message}})
		}
		return code
	}
	f := flag.NewFlagSet("host-check", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", "json", "output format: json or text")
	live := f.Bool("live", false, "query only the installed owner's private rootless engine; run as that owner")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-runner host-check [--format json|text] [--live]\n\nDefault: local metadata only. --live explicitly probes the installed owner's private engine\nusing its trusted profile, never DOCKER_HOST or PATH. Run --live as that owner.\nExit 0 means report generation succeeded, not that hosting is qualified.")
			return 0
		}
		return fail("host-check accepts --format json|text and optional --live", 2)
	}
	if f.NArg() != 0 || (format != "json" && format != "text") {
		return fail("host-check accepts --format json|text and optional --live", 2)
	}
	report := deps.inspect()
	code := 0
	if *live {
		p, err := deps.owner()
		if err != nil || p == nil || p.Validate() != nil {
			return fail("live host check requires the installed non-root pier owner and a trusted owner profile", 1)
		}
		// These paths are derived exclusively from the validated installed profile.
		// ProbeEngine verifies socket ownership and uses bounded direct API reads.
		socket := filepath.Join("/run/user", strconv.FormatUint(uint64(p.UID), 10), rootless.SocketName)
		dataRoot := filepath.Join(p.Home, ".local/share/urbit-runner/docker")
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		engine, err := deps.probe(probeCtx, socket, dataRoot)
		cancel()
		if err != nil {
			// Keep metadata useful; raw socket/HTTP errors are never diagnostics.
			report.Checks = append(report.Checks, hostcompat.Check{ID: "live-engine", Status: "unknown", Detail: "Private owner engine probe failed or timed out; running engine evidence is unavailable."})
			code = 1
		} else {
			report.Engine = &engine
			report = hostcompat.Evaluate(report, dataRoot)
		}
	}
	if err := writeLocalHostReport(stdout, format, report); err != nil {
		return fail("could not write host compatibility report", 1)
	}
	return code
}

func writeLocalHostReport(w io.Writer, format string, report hostcompat.Report) error {
	if format == "json" {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(report)
	}
	clean := func(s string) string {
		return strings.Map(func(c rune) rune {
			if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
				return ' '
			}
			return c
		}, s)
	}
	var text strings.Builder
	fmt.Fprintf(&text, "Host compatibility: %s\nDistribution: %s %s; provider: %s; profile: %s\nQualification: %s; provenance: %s\nObserved: %s\nInstalled engine: %s; restart pending: %t\n", clean(report.Compatibility), clean(report.Distribution.ID), clean(report.Distribution.VersionID), clean(report.Distribution.Provider), clean(report.Distribution.Profile), clean(report.Qualification), clean(report.Provenance), report.ObservedAt.UTC().Format(time.RFC3339), clean(report.InstalledEngineVersion), report.RestartPending)
	if report.Engine != nil {
		fmt.Fprintf(&text, "Running engine: %s; API: %s\n", clean(report.Engine.Version), clean(report.Engine.APIVersion))
	}
	for _, p := range report.Packages {
		fmt.Fprintf(&text, "Package %s: %s %s; provenance: %s\n", clean(p.Role), clean(p.Name), clean(p.Version), clean(p.Provenance))
	}
	for _, check := range report.Checks {
		fmt.Fprintf(&text, "%s: %s — %s\n", clean(check.ID), clean(check.Status), clean(check.Detail))
	}
	for _, guidance := range report.Guidance {
		fmt.Fprintln(&text, clean(guidance))
	}
	_, err := io.WriteString(w, text.String())
	return err
}
