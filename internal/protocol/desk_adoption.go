package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"time"
)

const DeskAdoptionFeature = "desk-source-adoption-v1"
const DeskAdoptionReviewPath = "/v1/desk-adoption-review"
const DeskAdoptionPath = "/v1/desk-adoptions"

type DeskAdoptionReview struct {
	Version        int                   `json:"version"`
	RunnerID       string                `json:"runner_id"`
	Mode           string                `json:"mode"`
	Request        DeskDeploymentRequest `json:"request"`
	Observation    deskwork.Observation  `json:"observation"`
	PolicyRevision uint64                `json:"policy_revision"`
	SourceName     string                `json:"source_name"`
	Continuity     string                `json:"continuity"`
	SourceDigest   string                `json:"source_digest"`
	PlanDigest     string                `json:"plan_digest"`
	Paths          []string              `json:"paths"`
	PathCount      int                   `json:"path_count"`
	NewCount       int                   `json:"new_count"`
	RefreshCount   int                   `json:"refresh_count"`
	RetainedCount  int                   `json:"retained_count"`
	Conflicts      []string              `json:"conflicts"`
	ConflictCount  int                   `json:"conflict_count"`
	ReadAt         time.Time             `json:"read_at"`
}

func NewDeskAdoptionReview(v DeskDeploymentReview, p deskwork.AdoptionPlan) DeskAdoptionReview {
	out := DeskAdoptionReview{Version: 1, RunnerID: v.RunnerID, Mode: v.Mode, Request: v.Request, Observation: v.Observation, PolicyRevision: v.PolicyRevision, SourceName: v.SourceName, Continuity: v.Continuity, SourceDigest: p.Source.Digest, PlanDigest: p.Digest, Paths: []string{}, PathCount: len(p.Source.Files), NewCount: p.NewCount, RefreshCount: p.RefreshCount, Conflicts: []string{}, ConflictCount: len(p.Conflicts), ReadAt: v.ReadAt}
	selected := p.Source.Index()
	for _, f := range p.Previous.Raw.Files {
		if _, ok := selected[f.Path]; !ok {
			out.RetainedCount++
		}
	}
	for _, f := range p.Source.Files {
		if len(out.Paths) == MaxSourceDetails {
			break
		}
		out.Paths = append(out.Paths, f.Path)
	}
	out.Conflicts = append(out.Conflicts, p.Conflicts[:min(len(p.Conflicts), MaxSourceDetails)]...)
	return out
}
func (v DeskAdoptionReview) Validate() error {
	if v.Version != 1 || !ValidID(v.RunnerID) || !ValidID(v.Mode) || v.Request.Validate() != nil || v.Observation.Validate() != nil || !v.Observation.Exists || v.PolicyRevision > apppolicy.MaxRevision || !deskwork.ValidSourceName(v.SourceName) || !ValidDeskOperationID(v.Continuity) || !ValidDeskOperationID(v.SourceDigest) || !ValidDeskOperationID(v.PlanDigest) || v.ReadAt.IsZero() {
		return errors.New("invalid adoption review")
	}
	if v.PathCount < 1 || v.PathCount > deskwork.MaxFiles || v.NewCount < 0 || v.RefreshCount < 0 || v.ConflictCount < 0 || v.NewCount+v.RefreshCount+v.ConflictCount > v.PathCount || v.RetainedCount < 0 || v.RetainedCount+v.PathCount > deskwork.MaxFiles {
		return errors.New("invalid adoption counts")
	}
	if v.Paths == nil || v.Conflicts == nil || len(v.Paths) != min(v.PathCount, MaxSourceDetails) || len(v.Conflicts) != min(v.ConflictCount, MaxSourceDetails) {
		return errors.New("invalid adoption details")
	}
	for _, paths := range [][]string{v.Paths, v.Conflicts} {
		for i, p := range paths {
			if !deskwork.ValidPath(p) || i > 0 && paths[i-1] >= p {
				return errors.New("invalid adoption path")
			}
		}
	}
	return nil
}
