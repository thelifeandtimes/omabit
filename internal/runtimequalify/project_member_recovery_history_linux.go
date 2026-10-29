//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Reconstruct concrete current creators from validated retained history. Opening
// this source neither observes a ship nor dispatches a lifecycle operation. The
// ordinary initial path remains byte-for-byte contract compatible.
func openProjectRecoveryMemberGenerations(ctx context.Context, project ProjectPreparationSelection, h projectnetwork.AnchorRecoveryHistory, current projectnetwork.MemberGenerationSet, d dependencies) (*projectInitialReadinessEngine, error) {
	if len(h.Epochs) == 0 {
		return openProjectMemberGenerations(ctx, project, h.Launches, current, d)
	}
	raw, err := openInitialProjectReadiness(ctx, ProjectInitialReadinessSelection{Project: project, Launches: h.Launches}, d)
	if err != nil {
		return nil, err
	}
	source := raw.(*projectInitialReadinessEngine)
	source.members = nil
	source.binding = current.Binding
	i := len(h.Epochs) - 1
	epoch := h.Epochs[i]
	prior := h.Clone()
	prior.Epochs = prior.Epochs[:i]
	prior.Resumes = epoch.Resumes
	prior.Current = epoch.Current
	prior = prior.Clone()
	for _, m := range current.Members {
		path, err := pierstore.Absolute(d.home, m.Storage.Path)
		if err != nil {
			return nil, err
		}
		var c projectMemberContract
		switch m.Mode {
		case "recovery":
			found := false
			for _, recorded := range epoch.Readiness.Replacement.Members {
				if recorded.Creation.InstanceID == m.InstanceID && recorded.Creation.OperationID == m.OperationID {
					found = true
				}
			}
			if !found {
				return nil, errors.New("current recovery creator is not in the last boundary")
			}
			c, err = newRecoveryProjectMemberContract(project, prior, epoch.Readiness.Replacement.Retirement, m.InstanceID, m.OperationID, d.home, filepath.Dir(path), source.anchor.base.contract.ImageID)
		case "resume":
			found := false
			for _, resume := range h.Resumes {
				if resume.OperationID != m.OperationID {
					continue
				}
				if found {
					return nil, errors.New("duplicate ordinary successor creator")
				}
				found = true
				prefix, er := projectnetwork.AnchorRecoveryHistoryAtBinding(h, resume.Retirement.Exit.Stop.Binding)
				if er != nil {
					return nil, er
				}
				c, err = newRecoveredResumeProjectMemberContract(project, prefix, resume.Retirement, m.OperationID, d.home, filepath.Dir(path), source.anchor.base.contract.ImageID)
			}
			if !found {
				return nil, errors.New("current ordinary creator lacks its post-recovery receipt")
			}
		default:
			return nil, errors.New("unsupported current creation after recovery")
		}
		if err != nil || c.digest != m.Container.ContractDigest || c.selection.ContainerGeneration != m.Generation || c.selection.Storage != m.Storage {
			return nil, errors.New("current post-recovery contract differs from retained creator")
		}
		source.members = append(source.members, projectObservedMember{contract: c, id: m.Container.ID})
	}
	if err = source.parents(ctx); err != nil {
		return nil, err
	}
	return source, nil
}

func retiredRecoveryHistoryIDs(h projectnetwork.AnchorRecoveryHistory, current projectnetwork.MemberGenerationSet) []string {
	if len(h.Epochs) == 0 {
		return nil
	}
	ids := map[string]bool{}
	for _, l := range h.Launches {
		ids[l.Container.ID] = true
	}
	for _, r := range h.Resumes {
		ids[r.Container.ID] = true
	}
	for _, e := range h.Epochs {
		for _, r := range e.Resumes {
			ids[r.Container.ID] = true
		}
		for _, m := range e.Readiness.Replacement.Members {
			ids[m.Container.ID] = true
		}
	}
	for _, m := range current.Members {
		delete(ids, m.Container.ID)
	}
	out := []string{}
	for id := range ids {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
