package service

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
)

type GuidanceInput struct {
	Path string `json:"path"`
}

func (s *Service) GuidanceIndex() (guidance.Index, bool) {
	if s.options.Guidance == nil {
		return guidance.Index{}, false
	}
	return s.options.Guidance.Index(), true
}
func (s *Service) registerGuidance() error {
	missing := func() error {
		return &refusal{"guidance_unavailable", "No verified skill bundle is configured.", "Configure a matching bundle and its verified SKILLS.json SHA256 at startup; reading guidance does not install it."}
	}
	if err := bind(s, description("guidance.list.v1", "List exact release-bound skill resource paths and hashes; no skill installation.", []string{}, []string{"offline-read"}, []string{}, "local-user", "startup-pinned skill manifest and matching BUILD provenance", "read-only"), func(_ context.Context, s *Service, _ Empty) (guidance.Index, error) {
		index, ok := s.GuidanceIndex()
		if !ok {
			return guidance.Index{}, missing()
		}
		return index, nil
	}); err != nil {
		return err
	}
	return bind(s, description("guidance.read.v1", "Read one bounded verified skill document; grants no authority and installs nothing.", []string{}, []string{"local-metadata-read"}, []string{}, "local-user", "exact resource path within startup-pinned manifest; bytes rehashed on every read", "read-only; changed bytes refuse"), func(_ context.Context, s *Service, in GuidanceInput) (guidance.Document, error) {
		if s.options.Guidance == nil {
			return guidance.Document{}, missing()
		}
		doc, err := s.options.Guidance.Read(in.Path)
		if err != nil {
			return guidance.Document{}, &refusal{"guidance_changed_or_unlisted", "Skill resource is unlisted, unavailable or differs from the verified bytes.", "Inspect the matching verified bundle without silently accepting changed guidance."}
		}
		return doc, nil
	})
}
