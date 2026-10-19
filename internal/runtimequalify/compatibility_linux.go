//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func (q *qualifier) verifyEngine(ctx context.Context) error {
	return q.verifyEngineAccess(ctx, false)
}

// Control-only admission is reserved for observation and explicit Stop. The
// caller still authenticates every resource's identity and containment before
// using it; engine compatibility alone never authorizes a container operation.
func (q *qualifier) verifyEngineAccess(ctx context.Context, controlOnly bool) error {
	q.engineEvidence = hostcompat.Engine{}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrHostCompatibility, err)
	}
	if err := q.boundary(); err != nil {
		return fmt.Errorf("%w: private engine ownership boundary unavailable", ErrHostCompatibility)
	}
	// The former CLI info call checked this workspace even when there were no
	// containers to inspect. Retain that boundary for prepared sessions; fresh
	// provisioning admission intentionally precedes creation of a workspace.
	if q.r.Directory != "" {
		for _, path := range append(q.storagePaths(), q.r.Directory, q.config) {
			if err := q.owned(path, os.ModeDir, path == q.r.Directory || path == q.config); err != nil {
				return err
			}
		}
	}
	ctx, cancel := context.WithTimeout(ctx, q.d.times.command)
	defer cancel()
	expectedRoot := filepath.Join(q.d.home, ".local/share/urbit-runner/docker")
	probe := q.d.ProbeEngine
	if probe == nil {
		probe = hostcompat.ProbeEngine
	}
	engine, err := probe(ctx, q.socket, expectedRoot)
	if err != nil || ctx.Err() != nil {
		return compatibilityEvidenceError(ctx, err, "private engine API evidence unavailable")
	}
	if err := hostcompat.CheckEngine(engine, expectedRoot, controlOnly); err != nil {
		// Do not reflect daemon text, executable paths or raw transport errors.
		return fmt.Errorf("%w: engine API, version or rootless configuration does not satisfy policy", ErrHostCompatibility)
	}
	if !controlOnly {
		if ctx.Err() != nil {
			return compatibilityEvidenceError(ctx, nil, "host package evidence deadline exceeded")
		}
		inspect := q.d.InspectHost
		if inspect == nil {
			inspect = hostcompat.InspectCurrent
		}
		report := inspect()
		if ctx.Err() != nil {
			return compatibilityEvidenceError(ctx, nil, "host package evidence deadline exceeded")
		}
		report.Engine = &engine
		report = hostcompat.Evaluate(report, expectedRoot)
		if !report.CanMutate {
			return fmt.Errorf("%w: host package provenance or runtime policy does not permit new work", ErrHostCompatibility)
		}
	}
	q.engineEvidence = engine
	q.restrictedEngine = controlOnly && hostcompat.EngineVersionStatus(engine.Version) != "compatible"
	return nil
}

// A canceled evidence read is an admission failure, never proof of pier damage.
// Preserve cancellation identity without reflecting arbitrary provider errors.
func compatibilityEvidenceError(ctx context.Context, cause error, detail string) error {
	err := fmt.Errorf("%w: %s", ErrHostCompatibility, detail)
	if contextErr := ctx.Err(); contextErr != nil {
		return errors.Join(err, contextErr)
	}
	if errors.Is(cause, context.Canceled) {
		return errors.Join(err, context.Canceled)
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return errors.Join(err, context.DeadlineExceeded)
	}
	return err
}
