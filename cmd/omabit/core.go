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
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/checkplan"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/mcpserver"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
)

type allowedRunners []service.Target

type sourceDirectories [][2]string

type testReportSelections [][3]string

func (r *testReportSelections) String() string {
	return "explicit NAME:SHA256:/absolute/report.json snapshots"
}
func (r *testReportSelections) Set(value string) error {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 || parts[0] == "" || len(parts[1]) != 64 || !filepath.IsAbs(parts[2]) || len(*r) >= checkplan.MaxReports {
		return errors.New("--allow-test-report requires NAME:SHA256:/absolute/report.json, at most32 snapshots")
	}
	*r = append(*r, [3]string{parts[0], parts[1], parts[2]})
	return nil
}

func (s *sourceDirectories) String() string { return "explicit NAME:/absolute/source pairs" }
func (s *sourceDirectories) Set(value string) error {
	name, path, ok := strings.Cut(value, ":")
	if !ok || !filepath.IsAbs(path) || name == "" {
		return errors.New("--allow-source requires NAME:/absolute/source")
	}
	*s = append(*s, [2]string{name, path})
	return nil
}

func (a *allowedRunners) String() string { return "explicit NAME:ID pairs" }
func (a *allowedRunners) Set(value string) error {
	name, id, ok := strings.Cut(value, ":")
	t := service.Target{Name: name, RunnerID: id}
	if !ok || t.Validate() != nil {
		return errors.New("--allow-runner requires NAME:ID")
	}
	*a = append(*a, t)
	return nil
}
func coreCommand(ctx context.Context, command string, args []string, config, skillsRoot, skillsSHA string) int {
	fail := func(err error) int { fmt.Fprintln(os.Stderr, err); return 1 }
	options := service.Options{ConfigDir: config, EnableMutations: command == "call"}
	var id string
	var input []byte
	var sources sourceDirectories
	var reports testReportSelections
	var httpOptions *mcpserver.HTTPOptions
	if command == "mcp" {
		if len(args) > 0 && args[0] == "create-http-token" {
			f := flag.NewFlagSet("omabit mcp create-http-token", flag.ContinueOnError)
			path := f.String("output", "", "new owner-private credential file in an existing physical directory")
			if err := f.Parse(args[1:]); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					return 0
				}
				return 2
			}
			if f.NArg() != 0 || *path == "" {
				return 2
			}
			info, err := mcpserver.CreateHTTPToken(*path)
			if err != nil {
				return fail(err)
			}
			if err = json.NewEncoder(os.Stdout).Encode(info); err != nil {
				return fail(err)
			}
			return 0
		}
		if len(args) == 0 || args[0] != "serve" {
			return 2
		}
		args = args[1:]
		f := flag.NewFlagSet("omabit mcp serve", flag.ContinueOnError)
		transport := f.String("transport", "stdio", "stdio or explicit local authenticated http")
		listen := f.String("listen", "", "HTTP only: explicit numeric loopback IP:port")
		tokenFile := f.String("token-file", "", "HTTP only: existing private credential created by create-http-token")
		mutations := f.Bool("enable-mutations", false, "expose typed mutation tools; runner role/policy still apply")
		var allowed allowedRunners
		f.Var(&allowed, "allow-runner", "explicit NAME:ID target allowlist; repeat for each runner")
		f.Var(&sources, "allow-source", "explicit NAME:/absolute/source root; repeat only for approved source trees")
		f.Var(&reports, "allow-test-report", "explicit NAME:SHA256:/absolute/report.json private historical snapshot; never executes tests")
		if err := f.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return 2
		}
		if f.NArg() != 0 || *transport != "stdio" && *transport != "http" {
			return fail(errors.New("select stdio or explicit local authenticated http"))
		}
		if *transport == "stdio" && (*listen != "" || *tokenFile != "") {
			return fail(errors.New("HTTP options require transport http"))
		}
		if *transport == "http" {
			if *listen == "" || *tokenFile == "" {
				return fail(errors.New("HTTP requires explicit --listen and --token-file"))
			}
			httpOptions = &mcpserver.HTTPOptions{Listen: *listen, TokenFile: *tokenFile, Ready: func(url string) {
				json.NewEncoder(os.Stderr).Encode(map[string]string{"transport": "http", "url": url})
			}}
		}
		options.RequireAllowed = true
		options.Allowed = allowed
		options.EnableMutations = *mutations
	} else if command == "call" {
		if len(args) == 0 {
			return 2
		}
		id, args = args[0], args[1:]
		f := flag.NewFlagSet("omabit call "+id, flag.ContinueOnError)
		file := f.String("input-file", "", "bounded JSON input file; - reads stdin")
		raw := f.String("input-json", "", "typed JSON input (mutually exclusive with input-file)")
		f.Var(&sources, "allow-source", "explicit NAME:/absolute/source root required for local staging")
		f.Var(&reports, "allow-test-report", "explicit NAME:SHA256:/absolute/report.json private historical snapshot; never executes tests")
		if err := f.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return 2
		}
		if f.NArg() != 0 || *raw != "" && *file != "" {
			return 2
		}
		input = []byte(*raw)
		if *file != "" {
			var reader io.Reader = os.Stdin
			if *file != "-" {
				handle, err := os.Open(*file)
				if err != nil {
					return fail(errors.New("cannot read bounded input file"))
				}
				defer handle.Close()
				reader = handle
			}
			var err error
			input, err = io.ReadAll(io.LimitReader(reader, service.MaxInputBytes+1))
			if err != nil {
				return fail(errors.New("cannot read bounded input file"))
			}
		}
	} else if command != "operations" || len(args) != 0 {
		return 2
	}
	for _, selected := range sources {
		root, err := service.OpenSourceRoot(selected[0], selected[1])
		if err != nil {
			return fail(err)
		}
		defer root.Close()
		options.Sources = append(options.Sources, root)
	}
	for _, selected := range reports {
		report, err := checkplan.OpenReport(selected[0], selected[2], selected[1])
		if err != nil {
			return fail(errors.New("selected checkpoint report snapshot refused"))
		}
		options.TestReports = append(options.TestReports, report)
	}
	if command != "operations" && options.ConfigDir == "" {
		var err error
		options.ConfigDir, err = collection.DefaultDir()
		if err != nil {
			return fail(err)
		}
	}
	if skillsRoot != "" || skillsSHA != "" {
		bundle, err := guidance.Open(skillsRoot, skillsSHA)
		if err != nil {
			return fail(errors.New("matching skill bundle verification failed"))
		}
		options.Guidance = bundle
	}
	core, err := service.New(options)
	if err != nil {
		return fail(err)
	}
	if command == "mcp" {
		if httpOptions != nil {
			if err = mcpserver.ServeHTTP(ctx, core, *httpOptions); err != nil && !errors.Is(err, context.Canceled) {
				return fail(errors.New("authenticated local HTTP MCP server failed"))
			}
			return 0
		}
		if err = mcpserver.Serve(ctx, core); err != nil && !errors.Is(err, context.Canceled) {
			return fail(errors.New("MCP stdio session failed"))
		}
		return 0
	}
	if command == "operations" {
		if err = json.NewEncoder(os.Stdout).Encode(core.Discovery()); err != nil {
			return fail(err)
		}
		return 0
	}
	out := core.Call(ctx, id, input)
	if err = json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fail(err)
	}
	if out.Error != nil {
		return 1
	}
	return 0
}
