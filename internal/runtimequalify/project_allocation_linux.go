//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// This observes filesystem metadata for current saved member contracts. The
// manager may derive the selection from complete original journals without
// persisting an expansion; Binding.LedgerKind distinguishes that source. It never opens an
// engine/process transport, dispatches a lifecycle action or creates a path.
// Canonical project/ledger selection and before/after store bracketing belong to
// the authenticated manager; a matching observation grants no action authority.
func ReadExpandedProjectAllocation(ctx context.Context, state ProjectExpansion, binding projectusage.Binding) (projectusage.Observation, error) {
	return readExpandedProjectAllocation(ctx, state, binding, defaults())
}

func projectAllocationRoots(ctx context.Context, state ProjectExpansion, binding projectusage.Binding, d dependencies) ([]projectusage.Root, error) {
	if ctx.Err() != nil || binding.Validate() != nil || state.Version != 1 || state.Anchor.Validate(state.Project.Plan) != nil || state.Anchor.Phase != "observed" || len(state.Members) == 0 || len(state.Members) > 32 || state.Project.ProjectID != binding.ProjectID || state.Project.RunnerID != binding.RunnerID || state.Project.Revision != binding.Revision || state.Project.Generation != binding.Generation {
		return nil, errors.New("project allocation requires an exact saved current expansion")
	}
	digest, err := state.Project.Plan.Digest()
	if err != nil || digest != binding.PlanDigest {
		return nil, errors.New("project allocation plan changed")
	}
	// Resolve only the two pinned native image identities against the immutable
	// creation contract. No Docker discovery or current-health inference occurs.
	var anchor *projectAnchorEngine
	for _, image := range []string{candidate.ConfigDigest, candidate.ManifestDigest} {
		contract, err := newProjectResourceContract(state.Project, state.Anchor.Prepared.EngineID, image)
		if err != nil {
			continue
		}
		expected := contract.Preparation
		expected.Sequence = 6
		expected.Bridge.Phase, expected.Bridge.ID = "observed", state.Anchor.Prepared.Bridge.ID
		expected.Anchor.Phase, expected.Anchor.ID = "observed", state.Anchor.Prepared.Anchor.ID
		if reflect.DeepEqual(expected, state.Anchor.Prepared) {
			anchor = &projectAnchorEngine{base: &projectDockerEngine{q: &qualifier{d: d}, contract: contract}, prepared: state.Anchor.Prepared}
			break
		}
	}
	if anchor == nil {
		return nil, errors.New("project allocation creation contract changed")
	}
	seen := map[string]bool{}
	roots := []projectusage.Root{}
	base := filepath.Join(d.home, ".local/share/urbit-runner/managed")
	for index, member := range state.Members {
		if seen[member.Member.InstanceID] || member.Creator.InstanceID != member.Member.InstanceID || !projectRuntimeID.MatchString(member.Creator.RuntimeID) || member.Creator.Storage.MountIdentity.Inode == 0 {
			return nil, errors.New("project allocation lacks recorded member storage")
		}
		seen[member.Member.InstanceID] = true
		contract, err := expansionContract(anchor, state, member)
		if err != nil || member.Creator.Container.ContractDigest != "" && contract.digest != member.Creator.Container.ContractDigest {
			return nil, errors.New("project allocation member creation contract changed")
		}
		selected := member.Creator.Storage
		if member.Initialized.Identity.Inode != 0 {
			if member.Initialized.Path != selected.Path || !member.Initialized.MountIdentity.SameObject(selected.MountIdentity) {
				return nil, errors.New("project allocation initialized storage changed")
			}
			if _, err := pierstore.Check(d.home, member.Initialized); err != nil {
				return nil, err
			}
			selected = member.Initialized
		}
		path, err := pierstore.CheckMount(d.home, selected)
		if err != nil {
			return nil, err
		}
		roots = append(roots, projectusage.Root{ID: fmt.Sprintf("mount-%d", index), Kind: "member-mount", InstanceID: member.Member.InstanceID, RuntimeID: member.Creator.RuntimeID, Path: path, Expected: selected.MountIdentity})
		roots = append(roots, projectusage.Root{ID: fmt.Sprintf("runtime-%d", index), Kind: "runtime-cache", InstanceID: member.Member.InstanceID, RuntimeID: member.Creator.RuntimeID, Path: filepath.Join(base, member.Creator.RuntimeID)})
	}
	roots = append(roots, projectusage.Root{ID: "project-cache", Kind: "project-cache", Path: filepath.Join(base, "project-resources-"+state.Project.ProjectID)})
	return roots, nil
}

func readExpandedProjectAllocation(ctx context.Context, state ProjectExpansion, binding projectusage.Binding, d dependencies) (projectusage.Observation, error) {
	roots, err := projectAllocationRoots(ctx, state, binding, d)
	if err != nil {
		return projectusage.Observation{}, err
	}
	out, err := projectusage.Measure(ctx, binding, roots, projectusage.MaxEntries)
	if err != nil {
		return projectusage.Observation{}, err
	}
	// Revalidate saved mount/pier objects even after an unknown scan. A missing
	// or replaced enrolled path is never empty, disposable or permission to reset.
	after, err := projectAllocationRoots(ctx, state, binding, d)
	if err != nil || !reflect.DeepEqual(roots, after) {
		return projectusage.Observation{}, errors.New("project allocation storage changed during observation")
	}
	return out, nil
}
