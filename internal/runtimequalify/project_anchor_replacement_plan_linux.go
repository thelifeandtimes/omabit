//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorReplacementPlanSelection struct {
	Source      ProjectAnchorRecoveryReviewSelection
	Retirement  projectnetwork.AnchorRetirement
	OperationID string
	Members     []projectnetwork.RecoveryJournalMemberID
}

// PlanProjectAnchorReplacement builds immutable contracts from the verified
// existing piers after all old containers were retired. It performs no Docker
// mutation; the owning journal must commit its result before opening a driver.
func PlanProjectAnchorReplacement(ctx context.Context, s ProjectAnchorReplacementPlanSelection) (projectnetwork.AnchorReplacement, error) {
	return planProjectAnchorReplacement(ctx, s, defaults())
}
func planProjectAnchorReplacement(ctx context.Context, s ProjectAnchorReplacementPlanSelection, d dependencies) (projectnetwork.AnchorReplacement, error) {
	h := anchorDrainHistory(s.Source).Clone()
	retired := s.Retirement.Clone()
	if retired.Validate(h) != nil || !retired.Complete() || len(s.Members) != len(retired.Members) {
		return projectnetwork.AnchorReplacement{}, errors.New("replacement planning requires complete retirement and reserved member identities")
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, s.Source, d)
	if err != nil {
		return projectnetwork.AnchorReplacement{}, err
	}
	review := raw.(*projectAnchorRecoveryReviewEngine)
	var members []projectnetwork.RecoveredMember
	for i, id := range s.Members {
		if id.InstanceID != retired.Members[i].InstanceID {
			return projectnetwork.AnchorReplacement{}, errors.New("replacement planning changed ordered member selection")
		}
		creation, err := projectnetwork.SelectRecoveryMemberCreation(h, retired, id.InstanceID, id.OperationID)
		if err != nil {
			return projectnetwork.AnchorReplacement{}, err
		}
		path, err := pierstore.Absolute(d.home, creation.Storage.Path)
		if err != nil {
			return projectnetwork.AnchorReplacement{}, err
		}
		c, err := newRecoveryProjectMemberContract(s.Source.Project, h, retired, id.InstanceID, id.OperationID, d.home, filepath.Dir(path), review.source.anchor.base.contract.ImageID)
		if err != nil {
			return projectnetwork.AnchorReplacement{}, err
		}
		members = append(members, projectnetwork.RecoveredMember{Creation: creation, Container: projectnetwork.ResourceIdentity{ContractDigest: c.digest}, Phase: "planned"})
	}
	if err = ctx.Err(); err != nil {
		return projectnetwork.AnchorReplacement{}, err
	}
	return projectnetwork.NewAnchorReplacement(h, retired, s.OperationID, members)
}
