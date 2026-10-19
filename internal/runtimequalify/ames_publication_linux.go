//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// ValidateAmesPublication is a read-only stopped-boundary check. It neither
// starts a runtime nor changes a pier, its enrollment, or the Docker network.
func ValidateAmesPublication(ctx context.Context, state ManagedState) error {
	return validateAmesPublication(ctx, state, defaults())
}

func validateAmesPublication(ctx context.Context, state ManagedState, d dependencies) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := AmesPublicationState(state, 1024); err != nil {
		return err
	}
	// Authenticate the legacy contract before constructing the successor. A
	// foreign run directory/image/pill must not be relabeled into valid state.
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return err
	}
	if q.r.Platform != candidate.Platform || !recordedEngineEvidenceValid(q.r) ||
		q.r.ImageID != "" && (!acceptedImageID(q.r.ImageID) || q.r.RepoDigest != candidate.Image && q.r.RepoDigest != strings.TrimPrefix(candidate.Image, "docker.io/")) ||
		q.r.ImageID == "" && q.r.RepoDigest != "" {
		return errors.New("legacy runtime platform or image provenance requires review")
	}
	checkPier := func() error {
		path, err := pierstore.Check(d.home, *state.Storage)
		if err != nil {
			return err
		}
		if pierstore.Busy(path) {
			return errors.New("registered pier has a Vere lock; stop its owner before publishing Ames")
		}
		return pierstore.CheckStateFiles(ctx, path)
	}
	if err := checkPier(); err != nil {
		return err
	}
	if err := q.verifyEngine(ctx); err != nil {
		return err
	}
	// Removed receipts alone do not allow publication over a daemon resource
	// that has since reappeared. Absence is used only after every create/removal
	// outcome was already resolved in the journal, never to clear uncertainty.
	out, err := q.docker(ctx, q.d.times.command, "container", "ls", "--all", "--filter", "label="+q.resourceLabel()+"=true", "--filter", "label="+q.resourceRunLabel()+"="+q.r.RunID, "--format", "{{.Names}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return errors.New("managed containers still exist; reconcile Stop before publishing Ames")
	}
	if n := q.r.Network; n != nil && !n.Removed {
		actual, err := q.inspectNetwork(ctx)
		if err != nil {
			return err
		}
		if len(actual.Containers) != 0 {
			return errors.New("managed network still has attached containers")
		}
	}
	if err := checkPier(); err != nil {
		return err
	}
	return ctx.Err()
}
