package main

import (
	"errors"
	"flag"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type privateChainPlanView = privatechain.Review

func readPrivateFoundation(path string) (privatechain.Foundation, error) {
	bad := errors.New("foundation must be an absolute regular file containing bounded public review metadata")
	if !filepath.IsAbs(path) {
		return privatechain.Foundation{}, bad
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return privatechain.Foundation{}, bad
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > privatechain.MaxFoundationBytes {
		return privatechain.Foundation{}, bad
	}
	data, err := io.ReadAll(io.LimitReader(f, privatechain.MaxFoundationBytes+1))
	if err != nil {
		return privatechain.Foundation{}, bad
	}
	return privatechain.DecodeFoundation(data)
}
func runPrivateChainPlan(args []string, selected bool, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector [--format json|text] private-chain-plan --label LABEL
       --foundation ABSOLUTE_PUBLIC_METADATA_FILE
       --member INSTANCE:~zod:HTTP_PORT --member INSTANCE:~nec:HTTP_PORT
       --shared-administration

Review the initial private local chain1337 two-ship profile offline. The foundation
file binds the genesis and nine qualified Azimuth contract addresses/code hashes.
It contains public metadata only; do not pass keys, a chain database or a pier.
No runner connection, reservation, key generation, chain or ship operation occurs.
Omit --runner, --socket and --expected-runner-id. HTTP ports are requested runner
loopback reservations; their availability and runtime publication are not checked.

The private-local-keyed-v1 plan is distinct from fake-project plans. It cannot be
passed to project-register. Use private-chain-register for reviewed reservations
on a capable pinned runner. Private-chain runtime lifecycle is not implemented yet.
The source qualification harness provides the isolated runtime experiment.
Members share administrative loopback access. Initial keyed shutdown requires an
acknowledged hood Lens exit and observed clean process exit; it has no forced-stop
fallback. Each member will require a fresh disposable private-chain network key.

Example:
  urbit-collector private-chain-plan --label local-apps --foundation /abs/foundation.json \
    --member dev-zod:~zod:18381 --member dev-nec:~nec:18382 --shared-administration
`
	flags := flag.NewFlagSet("private-chain-plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var req privatechain.Request
	var file string
	flags.StringVar(&req.Label, "label", "", "development project display label")
	flags.StringVar(&file, "foundation", "", "absolute public foundation metadata file")
	flags.BoolVar(&req.ConfirmSharedAdministration, "shared-administration", false, "acknowledge shared administrative loopback access")
	flags.Func("member", "repeat INSTANCE:~SHIP:HTTP_PORT", func(value string) error {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return errors.New("member must be INSTANCE:~SHIP:HTTP_PORT")
		}
		port, err := strconv.ParseUint(parts[2], 10, 16)
		if err != nil {
			return errors.New("member port must be an unsigned16-bit number")
		}
		req.Members = append(req.Members, privatechain.MemberRequest{InstanceID: parts[0], Ship: parts[1], HTTPPort: uint16(port)})
		return nil
	})
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if _, err = io.WriteString(stdout, usage); err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if flags.NArg() != 0 {
		return fail(errors.New("private-chain-plan accepts named flags only; use --help"), 2)
	}
	if selected {
		return fail(errors.New("private-chain-plan is offline; omit --runner, --socket and --expected-runner-id"), 2)
	}
	f, err := readPrivateFoundation(file)
	if err != nil {
		return fail(err, 2)
	}
	req.Foundation = f
	plan, err := privatechain.Build(req, privatechain.Reservations{})
	if err != nil {
		return fail(err, 2)
	}
	digest, err := plan.Digest()
	if err != nil {
		return fail(err, 1)
	}
	result := privateChainPlanView{SchemaVersion: 1, Plan: plan, ReviewDigest: digest, ReservationCheck: "not-performed", RuntimeAuthorized: false, Notice: "Offline private-chain review. No resources reserved or started. Runner private-chain lifecycle is not implemented; project-register accepts fake plans only."}
	if err = newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
