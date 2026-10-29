//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// ProjectPreparationSelection must come from an accepted, revision-bound runner
// job. It is not RPC input or permission to adopt existing directories/resources.
type ProjectPreparationSelection = projectSelection

// OpenProjectPreparation verifies the private engine and pinned local image,
// then allocates only a fresh private metadata workspace. It creates no engine
// resource or pier. The caller must hold the runner mutation/management fence,
// retain its accepted operation on failure, and pass the result to the durable
// controller. Existing workspaces are retained for explicit recovery, not reused.
func OpenProjectPreparation(ctx context.Context, s ProjectPreparationSelection) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
	return openProjectPreparation(ctx, s, defaults())
}
func openProjectPreparation(ctx context.Context, s projectSelection, d dependencies) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
	fail := func(err error) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
		return projectnetwork.Preparation{}, nil, err
	}
	// Validate all path/label/port inputs before filesystem or engine access.
	if _, err := newProjectResourceContract(s, "pending-engine-validation", candidate.ManifestDigest); err != nil {
		return fail(err)
	}
	if s.Generation != 1 {
		return fail(errors.New("initial project preparation requires its first recorded generation"))
	}
	q := &qualifier{d: d, phase: "project-preparation"}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
	if err := q.verifyEngine(ctx); err != nil {
		return fail(err)
	}
	if !hostcompat.ValidEngineIdentity(q.engineEvidence.ID) {
		return fail(errors.New("project preparation requires an identified private engine"))
	}
	for _, path := range q.storagePaths() {
		if err := q.owned(path, os.ModeDir, path == q.base); err != nil {
			return fail(err)
		}
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	dir := filepath.Join(q.base, "project-resources-"+s.ProjectID)
	if err := os.Mkdir(dir, 0700); err != nil {
		return fail(errors.New("project metadata workspace must be fresh; retain existing evidence for recovery"))
	}
	q.r.Directory = dir
	q.config = filepath.Join(dir, "docker-config")
	if err := os.Mkdir(q.config, 0700); err != nil {
		return fail(err)
	}
	// Require the already installed pinned image. Dependency installation/pulling
	// is a separate user-visible flow, not an incidental preparation side effect.
	if err := q.inspectImage(ctx); err != nil {
		return fail(err)
	}
	contract, err := newProjectResourceContract(s, q.engineEvidence.ID, q.r.ImageID)
	if err != nil {
		return fail(err)
	}
	engine := &projectDockerEngine{q: q, contract: contract}
	if err = engine.VerifyEngine(ctx, contract.Preparation.EngineID); err != nil {
		return fail(err)
	}
	return contract.Preparation, engine, nil
}
