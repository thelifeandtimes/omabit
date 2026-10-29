package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type projectPlanView struct {
	SchemaVersion     int                 `json:"schema_version"`
	Plan              projectnetwork.Plan `json:"plan"`
	ReviewDigest      string              `json:"review_digest"`
	ReservationCheck  string              `json:"reservation_check"`
	RuntimeAuthorized bool                `json:"runtime_authorized"`
	Notice            string              `json:"notice"`
}

func runProjectPlan(args []string, selected bool, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector [--format json|text] project-plan --label LABEL
       --member INSTANCE:~GALAXY:HTTP_PORT [--member INSTANCE:~GALAXY:HTTP_PORT ...]
       --shared-administration

Build a deterministic offline plan for fresh fake galaxies. No runner connection,
port reservation, filesystem change or runtime operation occurs. Omit --runner,
--socket and --expected-runner-id. Runner capability/admission remains a separate
step; a plan digest is review data, not a project ID or permission to start ships.

Use a unique instance ID, canonical ~galaxy and distinct unprivileged runner-loopback
Eyre port for each member. Internal ports are derived from the complete sorted set.
--shared-administration acknowledges that members share administrative loopback
access. This is a trusted development network, not isolation between hostile apps.

Example:
  urbit-collector project-plan --label demo --member demo-zod:~zod:18381 \
    --member demo-nec:~nec:18382 --shared-administration
`
	flags := flag.NewFlagSet("project-plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var request projectnetwork.Request
	flags.StringVar(&request.Label, "label", "", "project display label")
	flags.BoolVar(&request.ConfirmSharedAdministration, "shared-administration", false, "acknowledge shared administrative loopback access")
	flags.Func("member", "repeat INSTANCE:~GALAXY:HTTP_PORT", func(value string) error {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return errors.New("member must be INSTANCE:~GALAXY:HTTP_PORT")
		}
		port, err := strconv.ParseUint(parts[2], 10, 16)
		if err != nil {
			return errors.New("member HTTP port must be an unsigned 16-bit number")
		}
		request.Members = append(request.Members, projectnetwork.MemberRequest{InstanceID: parts[0], Ship: parts[1], HTTPPort: uint16(port)})
		return nil
	})
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, err = io.WriteString(stdout, usage)
			if err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 {
		return fail(errors.New("project-plan accepts named flags only; use --help"), 2)
	}
	if selected {
		return fail(errors.New("project-plan is offline; omit --runner, --socket and --expected-runner-id"), 2)
	}
	plan, err := projectnetwork.Build(request, projectnetwork.Reservations{})
	if err != nil {
		return fail(err, 2)
	}
	digest, err := plan.Digest()
	if err != nil {
		return fail(err, 1)
	}
	result := projectPlanView{SchemaVersion: 1, Plan: plan, ReviewDigest: digest, ReservationCheck: "not-performed", RuntimeAuthorized: false, Notice: "Offline review only. No resources reserved or started; runner capability and fresh host admission remain required."}
	if err := newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(fmt.Errorf("write project plan: %w", err), 1)
	}
	return 0
}
