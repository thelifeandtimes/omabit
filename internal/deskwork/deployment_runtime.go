package deskwork

import (
	"context"
	"errors"
	"time"
)

// DeploymentReader exposes only typed reads for the selected runtime. Effect
// dispatch belongs to the runner's separately journaled, permission-checked path.
type DeploymentReader interface {
	Observe(context.Context, string) (Observation, error)
	ClayFiles(context.Context, string, uint64) (Manifest, error)
	NormalizeFiles(context.Context, string, uint64, []SourceFile) (Manifest, error)
}
type DeploymentObservation struct {
	Observation Observation    `json:"observation"`
	Plan        DeploymentPlan `json:"plan"`
	ReadAt      time.Time      `json:"read_at"`
}

func InspectDeployment(ctx context.Context, runtime DeploymentReader, desk, directory string, expected Manifest, previous Ownership) (DeploymentObservation, error) {
	var out DeploymentObservation
	if runtime == nil || !deskName.MatchString(desk) || expected.Validate() != nil || previous.Validate() != nil {
		return out, errors.New("invalid deployment selection")
	}
	before, err := runtime.Observe(ctx, desk)
	if err != nil || before.Validate() != nil || !before.Exists {
		return out, errors.New("deployment desk unavailable")
	}
	clay, err := runtime.ClayFiles(ctx, desk, before.Revision)
	if err != nil || !canonicalManifest(clay) {
		return out, errors.New("deployment Clay inventory unavailable")
	}
	current := clay.Index()
	paths := []string{}
	for _, f := range expected.Files {
		if _, exists := current[f.Path]; exists {
			paths = append(paths, f.Path)
		}
	}
	raw, known, err := SnapshotCanonicalPaths(ctx, directory, paths, func(ctx context.Context, files []SourceFile) (Manifest, error) {
		return runtime.NormalizeFiles(ctx, desk, before.Revision, files)
	})
	if err != nil || raw.Digest != expected.Digest {
		return out, errors.New("staged source changed or mark conversion unavailable")
	}
	after, err := runtime.Observe(ctx, desk)
	if err != nil || after != before {
		return out, errors.New("desk changed during deployment review")
	}
	plan, err := PlanDeployment(previous, clay, raw, known)
	if err != nil {
		return out, err
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	return DeploymentObservation{Observation: before, Plan: plan, ReadAt: time.Now().UTC()}, nil
}

// ObserveDeploymentResult is also used for read-only recovery after a lost
// acknowledgement. It does not replay writes and never infers app readiness.
func ObserveDeploymentResult(ctx context.Context, runtime DeploymentReader, desk, directory string, baseline Observation, plan DeploymentPlan) (Observation, Ownership, error) {
	fail := func() (Observation, Ownership, error) {
		return Observation{}, Ownership{}, errors.New("deployment result remains unverified")
	}
	if runtime == nil || !deskName.MatchString(desk) || baseline.Validate() != nil || !baseline.Exists || plan.Validate() != nil || len(plan.Conflicts) != 0 {
		return fail()
	}
	after, err := runtime.Observe(ctx, desk)
	if err != nil || after.Validate() != nil || !after.Exists || after.Revision < baseline.Revision || len(plan.Changes) > 0 && after.Revision == baseline.Revision || after.Publication != baseline.Publication {
		return fail()
	}
	clay, err := runtime.ClayFiles(ctx, desk, after.Revision)
	if err != nil {
		return fail()
	}
	raw, canonical, err := SnapshotCanonical(ctx, directory, func(ctx context.Context, files []SourceFile) (Manifest, error) {
		return runtime.NormalizeFiles(ctx, desk, after.Revision, files)
	})
	if err != nil || raw.Digest != plan.Source.Digest {
		return fail()
	}
	ownership, err := VerifyDeployment(plan, clay, canonical)
	if err != nil {
		return fail()
	}
	final, err := runtime.Observe(ctx, desk)
	if err != nil || final != after || ctx.Err() != nil {
		return fail()
	}
	return after, ownership, nil
}
