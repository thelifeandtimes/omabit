package main

import (
	"context"
	"errors"
	"flag"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateChainInputsView = privatechain.RuntimeInputInspection

func runPrivateChainInputs(ctx context.Context, args []string, selected bool, format string, stdout io.Writer, fail func(error, int) int) int {
	const usage = `Usage: urbit-collector [--format json|text] private-chain-inputs
       --directory ABSOLUTE_SEALED_BUNDLE
       --manifest-sha256 REVIEWED_HASH --foundation-digest REVIEWED_FOUNDATION

Read and hash one local private-chain runtime input generation. Obtain the manifest
hash from a separate artifact review and the foundation digest from the reviewed
private-chain plan. A hash supplied by the same untrusted bundle is not independent
publisher authentication. This command never fetches, installs or executes inputs,
contacts a runner, opens a chain through a database library, or starts a ship.
Omit --runner, --socket and --expected-runner-id. No collector registry is opened.

The current format is private-chain-runtime-inputs-v1 for linux/amd64. It binds the
pinned Vere, pill, Node and dependency bytes, foundation and gateway sources. The
complete owner-only tree must already be sealed: directories 0500; files 0400 or
0500 according to the manifest. Links, mount traversal, unexpected entries, altered
bytes and unsupported versions refuse. Do not chmod existing projects or installed
files to make them pass; use the separately prepared inert bundle.

The current gateway sources are guest-only qualification candidates. Successful
inspection reports runtime_authorized=false. It is neither a live chain check nor
permission to initialize, reopen, register or install anything. Runner private-chain
runtime dispatch and supported delivery remain separate development gates.
`
	f := flag.NewFlagSet("private-chain-inputs", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	directory := f.String("directory", "", "absolute sealed input directory")
	manifest := f.String("manifest-sha256", "", "separately reviewed exact manifest hash")
	foundation := f.String("foundation-digest", "", "reviewed private-chain foundation digest")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if _, err = io.WriteString(stdout, usage); err != nil {
				return fail(err, 1)
			}
			return 0
		}
		return fail(err, 2)
	}
	if selected {
		return fail(errors.New("private-chain-inputs is local and offline; omit --runner, --socket and --expected-runner-id"), 2)
	}
	if f.NArg() != 0 || *directory == "" || *manifest == "" || *foundation == "" {
		return fail(errors.New("private-chain-inputs requires only --directory, --manifest-sha256 and --foundation-digest; use --help"), 2)
	}
	result, err := privatechain.InspectRuntimeInputs(ctx, *directory, *manifest, *foundation)
	if err != nil {
		return fail(err, 1)
	}
	if err = newOutput(stdout, format, false).Encode(result); err != nil {
		return fail(err, 1)
	}
	return 0
}
