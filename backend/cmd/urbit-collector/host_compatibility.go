package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func runHostCheck(ctx context.Context, args []string, dir, socket, runner, expected, format string, stdout, stderr io.Writer) int {
	fail := func(err error, code int) int {
		problem := protocol.APIError{Code: "host_report_unavailable", Message: err.Error()}
		if errors.Is(err, client.ErrHostCompatibilityUnsupported) {
			problem.Code = "host_report_unsupported"
		} else if errors.Is(err, client.ErrIdentityMismatch) {
			problem.Code = "runner_identity_mismatch"
		} else if errors.Is(err, context.DeadlineExceeded) {
			problem.Code, problem.Message = "host_report_timeout", "host compatibility request timed out"
		} else if code == 2 {
			problem.Code = "client_error"
		}
		_ = newOutput(stderr, format, false).Encode(protocol.ErrorResponse{Error: problem})
		return code
	}
	f := flag.NewFlagSet("host-check", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&format, "format", format, "output format: json or text")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage: urbit-collector [--format json|text] [--runner NAME | --socket PATH] host-check [--format json|text]\n\n--format is accepted before or after host-check.\nRead cached public native-package and live private-engine evidence from the selected runner.\nRequires operator or observer access. The collector needs no local Docker installation.\nPackage verification and guard repair require explicit local root runner commands:\n  urbit-runner verify-host-packages --acknowledge-local-verification [--fetch-missing]\n  urbit-runner repair-engine-guard --acknowledge-host-changes [--fetch-missing]\nThese local helpers default to offline verification. --fetch-missing permits native\npackage download-only commands, never package installation, updates or key imports.")
			return 0
		}
		return fail(err, 2)
	}
	if f.NArg() != 0 || (format != "json" && format != "text") {
		return fail(errors.New("host-check accepts only --format json or text"), 2)
	}
	if (socket == "") == (runner == "") || (socket != "" && !collection.ValidSocket(socket)) || (runner != "" && !validID(runner)) {
		return fail(errors.New("select either an enrolled --runner NAME or an absolute clean --socket PATH before host-check"), 2)
	}
	if dir != "" && (!filepath.IsAbs(dir) || filepath.Clean(dir) != dir) {
		return fail(errors.New("--config-dir must be absolute and clean"), 2)
	}
	var c *client.Client
	var identity client.Identity
	if runner != "" {
		registry, err := openRegistry(dir)
		if err != nil {
			return fail(err, 1)
		}
		entry, err := registry.Get(ctx, runner)
		if err != nil {
			return fail(err, 1)
		}
		identity = entry.Identity()
		if expected != "" && expected != identity.RunnerID {
			return fail(client.ErrIdentityMismatch, 1)
		}
		c = entry.Client(true)
	} else {
		// An explicit socket has no enrollment pin. Discover once, then pin all
		// subsequent connections and the report body to that observed identity.
		initial := client.New(socket)
		h, err := initial.Hello(ctx)
		initial.Close()
		if err != nil {
			return fail(client.ErrHostCompatibilityUnavailable, 1)
		}
		identity = client.Identity{RunnerID: h.RunnerID, Mode: h.Mode}
		c = client.NewPinned(socket, identity)
	}
	defer c.Close()
	report, err := c.HostCompatibility(ctx, identity)
	if err != nil {
		return fail(err, 1)
	}
	if err := newOutput(stdout, format, false).Encode(report); err != nil {
		return fail(errors.New("could not write host compatibility report"), 1)
	}
	return 0
}

func hostCompatibilityText(w io.Writer, v protocol.HostCompatibilityReport) {
	r := v.Report
	fmt.Fprintf(w, "Runner: %s (%s)\nHost report schema: %d; protocol: %d\nObserved: %s\nCache expires: %s\n", cell(v.RunnerID), cell(v.Mode), v.SchemaVersion, v.ProtocolVersion, r.ObservedAt.UTC().Format("2006-01-02T15:04:05Z"), v.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"))
	if v.MetadataObservedAt != nil {
		fmt.Fprintf(w, "Package evidence observed: %s\n", v.MetadataObservedAt.UTC().Format("2006-01-02T15:04:05Z"))
	}
	fmt.Fprintf(w, "Distribution: %s %s; provider: %s; profile: %s\nCompatibility: %s; qualification: %s; provenance: %s\nInstalled engine: %s; restart pending: %t\n", cell(r.Distribution.ID), cell(r.Distribution.VersionID), cell(r.Distribution.Provider), cell(r.Distribution.Profile), cell(r.Compatibility), cell(r.Qualification), cell(r.Provenance), cell(r.InstalledEngineVersion), r.RestartPending)
	if r.Engine != nil {
		fmt.Fprintf(w, "Running engine: %s; API: %s\n", cell(r.Engine.Version), cell(r.Engine.APIVersion))
	} else {
		fmt.Fprintln(w, "Running engine: unavailable")
	}
	fmt.Fprintf(w, "Engine policy: observe=%t; stop=%t; mutate=%t\n", r.CanObserve, r.CanStop, r.CanMutate)
	for _, p := range r.Packages {
		fmt.Fprintf(w, "Package %s: %s %s; provenance: %s\n", cell(p.Role), cell(p.Name), cell(p.Version), cell(p.Provenance))
	}
	for _, c := range r.Checks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", cell(c.ID), cell(c.Status), cell(c.Detail))
	}
	for _, guidance := range r.Guidance {
		fmt.Fprintln(w, cell(guidance))
	}
}
