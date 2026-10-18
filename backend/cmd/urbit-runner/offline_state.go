package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
)

// Private subprocess entrypoint for the authorized host installer. The helper
// runs as the existing owner and inherits the parent's runtime flock on fd 3.
// Tokens travel on stdin, never process arguments or environment variables.
func runOfflineState(ctx context.Context, input io.Reader, stdout, stderr io.Writer) int {
	fail := func(err error) int { fmt.Fprintln(stderr, err); return 1 }
	if os.Geteuid() == 0 {
		return fail(errors.New("offline SQLite helper must run as the existing non-root state owner"))
	}
	var req managed.OfflineUpdateRequest
	d := json.NewDecoder(io.LimitReader(input, 8193))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || d.Decode(new(any)) != io.EOF {
		return fail(errors.New("invalid bounded offline helper request"))
	}
	f := os.NewFile(3, "managed-ownership-lock")
	if f == nil {
		return fail(errors.New("missing inherited ownership lock"))
	}
	defer f.Close()
	lock, err := managed.AdoptOfflineStateLock(req.Directory, uint32(os.Geteuid()), f)
	if err != nil {
		return fail(err)
	}
	var state managed.OfflineUpdateState
	switch req.Action {
	case "inspect":
		state, err = lock.Inspect(ctx)
	case "inspect-continuation":
		state, err = lock.InspectContinuation(ctx, req.RunnerID, req.Token, req.Schema)
	case "fence":
		state, err = lock.Fence(ctx, req.RunnerID, req.Token)
	default:
		err = errors.New("unknown offline helper action")
	}
	if err != nil {
		return fail(err)
	}
	if err := json.NewEncoder(stdout).Encode(state); err != nil {
		return fail(err)
	}
	return 0
}
