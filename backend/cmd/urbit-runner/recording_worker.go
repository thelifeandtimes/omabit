package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const recordingWorkerUsage = `Usage: urbit-runner recording-worker --job ID --sha256 DIGEST

Internal entry for the exact main process of a private Type=notify service.
Reads one existing immutable job below the installed owner's selected state
root. No roots, commands or native inputs are accepted. Refuses ordinary shell
invocation. Normal managed launch does not submit recording jobs yet.
`

func runRecordingWorker(ctx context.Context, args []string, stdout, stderr io.Writer, execute func(context.Context, string, string) error) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, recordingWorkerUsage)
		return 0
	}
	// Exact form also rejects duplicated flags, alternate spellings and trailing
	// positional inputs. Neither argument is a filesystem path or command.
	if len(args) != 4 || args[0] != "--job" || args[2] != "--sha256" || !recordingWorkerHex(args[1], 32) || !recordingWorkerHex(args[3], 64) {
		fmt.Fprint(stderr, recordingWorkerUsage)
		return 2
	}
	if execute == nil || execute(ctx, args[1], args[3]) != nil {
		fmt.Fprintln(stderr, "Recording worker did not finish its verified capture; preserve the private job and session for inspection.")
		return 1
	}
	return 0
}

func recordingWorkerHex(value string, size int) bool {
	if len(value) != size || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
