package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/mcpserver"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
)

type allowedRunners []service.Target

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
	if command == "mcp" {
		if len(args) == 0 || args[0] != "serve" {
			return 2
		}
		args = args[1:]
		f := flag.NewFlagSet("omabit mcp serve", flag.ContinueOnError)
		transport := f.String("transport", "stdio", "stdio only; HTTP is not implemented")
		mutations := f.Bool("enable-mutations", false, "expose typed mutation tools; runner role/policy still apply")
		var allowed allowedRunners
		f.Var(&allowed, "allow-runner", "explicit NAME:ID target allowlist; repeat for each runner")
		if err := f.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			return 2
		}
		if f.NArg() != 0 || *transport != "stdio" {
			return fail(errors.New("only local stdio is supported"))
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
