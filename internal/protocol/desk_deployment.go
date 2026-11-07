package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"sort"
	"time"
)

const DeskCommitFeature = "desk-commit-v1"
const DeskDeploymentReviewPath = "/v1/desk-deployment-review"
const DeskCommitPath = "/v1/desk-commits"
const DeskDeploymentTimeout = 4 * time.Minute

type DeskDeploymentRequest struct {
	Selection        DeskSelection `json:"selection"`
	StageID          string        `json:"stage_id"`
	SettingsRevision uint64        `json:"settings_revision"`
}

func (q DeskDeploymentRequest) Validate() error {
	if q.Selection.Validate() != nil || !ValidDeskOperationID(q.StageID) || q.SettingsRevision > apppolicy.MaxRevision {
		return errors.New("invalid staged deployment selection")
	}
	return nil
}

type DeskDeploymentReview struct {
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
	Changes        []SourceDifference    `json:"changes"`
	ChangeCount    int                   `json:"change_count"`
	Conflicts      []string              `json:"conflicts"`
	ConflictCount  int                   `json:"conflict_count"`
	PreservedCount int                   `json:"preserved_count"`
	ManagedCount   int                   `json:"managed_count"`
	ReadAt         time.Time             `json:"read_at"`
}

func NewDeskDeploymentReview(runner, mode string, q DeskDeploymentRequest, source, continuity string, policy uint64, s deskwork.DeploymentObservation) DeskDeploymentReview {
	p := s.Plan
	out := DeskDeploymentReview{Version: 1, RunnerID: runner, Mode: mode, Request: q, Observation: s.Observation, PolicyRevision: policy, SourceName: source, Continuity: continuity, SourceDigest: p.Source.Digest, PlanDigest: p.Digest, Changes: []SourceDifference{}, ChangeCount: len(p.Changes), Conflicts: []string{}, ConflictCount: len(p.Conflicts), PreservedCount: len(p.Preserved), ManagedCount: len(p.Managed), ReadAt: s.ReadAt}
	for _, c := range p.Changes {
		if len(out.Changes) == MaxSourceDetails {
			break
		}
		out.Changes = append(out.Changes, SourceDifference{Path: c.Path, Kind: c.Kind})
	}
	for _, path := range p.Conflicts {
		if len(out.Conflicts) == MaxSourceDetails {
			break
		}
		out.Conflicts = append(out.Conflicts, path)
	}
	return out
}
func (s DeskDeploymentReview) Validate() error {
	if s.Version != 1 || !ValidID(s.RunnerID) || !ValidID(s.Mode) || s.Request.Validate() != nil || s.Observation.Validate() != nil || !s.Observation.Exists || s.PolicyRevision > apppolicy.MaxRevision || !deskwork.ValidSourceName(s.SourceName) || !ValidDeskOperationID(s.Continuity) || !ValidDeskOperationID(s.SourceDigest) || !ValidDeskOperationID(s.PlanDigest) || s.ReadAt.IsZero() {
		return errors.New("invalid deployment review")
	}
	if s.ConflictCount < 0 || s.ConflictCount > 2*deskwork.MaxFiles {
		return errors.New("invalid deployment conflict count")
	}
	for _, n := range []int{s.ChangeCount, s.PreservedCount, s.ManagedCount} {
		if n < 0 || n > deskwork.MaxFiles {
			return errors.New("invalid deployment review counts")
		}
	}
	if s.Changes == nil || s.Conflicts == nil || len(s.Changes) != min(s.ChangeCount, MaxSourceDetails) || len(s.Conflicts) != min(s.ConflictCount, MaxSourceDetails) {
		return errors.New("invalid deployment review detail bound")
	}
	for i, c := range s.Changes {
		if !deskwork.ValidPath(c.Path) || c.Kind != "add" && c.Kind != "replace" && c.Kind != "delete" || i > 0 && s.Changes[i-1].Path >= c.Path {
			return errors.New("invalid reviewed change")
		}
	}
	if !sort.StringsAreSorted(s.Conflicts) {
		return errors.New("invalid conflict ordering")
	}
	for i, p := range s.Conflicts {
		if !deskwork.ValidPath(p) || i > 0 && s.Conflicts[i-1] == p {
			return errors.New("invalid reviewed conflict")
		}
	}
	return nil
}
