package service

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type ProjectAllocationInput struct {
	ProjectInput
	ExpectedRevision   uint64 `json:"expected_revision"`
	ReviewDigest       string `json:"review_digest"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (in ProjectAllocationInput) request() projectusage.Request {
	return projectusage.Request{RunnerID: in.RunnerID, ProjectID: in.ProjectID, ExpectedRevision: in.ExpectedRevision, ReviewDigest: in.ReviewDigest, IncludePrivateData: in.IncludePrivateData}
}
func (in ProjectAllocationInput) Validate() error {
	if in.Target.Validate() != nil || in.request().Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerProjectAllocation() error {
	return selected(s, description("project.allocation.v1", "Read operator-private bounded allocation for the selected current expanded-member mounts and private caches. Requires include_private_data=true and exact saved revision/plan. Logical/inode-block bytes, root identities, scope exclusions and unknown reasons remain separate from a complete project budget. Server-held monotonic samples may report net growth; this is not event-growth evidence or permission to stop, prune or resume. No engine/process probe or background guard.", nil, []string{"local-metadata-read", "saved-runner-read", "live-probe", "private-runtime-read"}, []string{protocol.ProjectAllocationFeature}, "operator", "exact enrolled runner/project/revision/plan digest; response binds current ledger/generation and declared root objects; no input paths", "one metadata read; missing/changed/unsafe roots remain unknown, never zero or disposable; unknown observations clear sampled growth"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in ProjectAllocationInput) (projectusage.Observation, error) {
		c := e.Client(true)
		defer c.Close()
		out, err := c.ProjectAllocation(ctx, in.request())
		if err == nil {
			err = e.Identity().Check(out.Binding.RunnerID, out.Binding.Mode)
		}
		if err != nil {
			return projectusage.Observation{}, err
		}
		return out, nil
	})
}
