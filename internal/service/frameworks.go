package service

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

type FrameworkInput struct {
	Name string `json:"name" jsonschema:"Compiled framework definition: grubbery-7f7701d-v1; native workflow remains pending"`
}
type FrameworkSourceInput struct {
	FrameworkInput
	Source SourceSelection `json:"source"`
}

func (in FrameworkSourceInput) Validate() error {
	if in.Source.Validate() != nil || in.Name != appdev.GrubberyFramework {
		return invalid()
	}
	return nil
}

type FrameworkSourceCheck struct {
	Version       int               `json:"version"`
	Name          string            `json:"name"`
	Commit        string            `json:"commit"`
	Source        SourceSelection   `json:"source"`
	Manifest      deskwork.Manifest `json:"manifest"`
	Verified      bool              `json:"verified"`
	Qualification string            `json:"qualification"`
}

func (s *Service) registerFrameworks() error {
	return errors.Join(
		bind(s, description("app.framework.v1", "Read a frozen framework source manifest, upstream pin, license and explicit qualification boundary; downloads and executes nothing.", nil, []string{"offline-read"}, nil, "local-user", "compiled named framework only; source equality is separate from live readiness", "pure read"), func(_ context.Context, _ *Service, in FrameworkInput) (appdev.Framework, error) {
			out, err := appdev.ReadFramework(in.Name)
			if err != nil {
				return appdev.Framework{}, invalid()
			}
			return out, nil
		}),
		bind(s, description("app.verify-framework-source.v1", "Compare a complete ordinary source subtree to the frozen framework desk; never stages, commits, installs or grants policy.", nil, []string{"offline-read", "local-source-read"}, nil, "local-user", "startup-approved source-root handle and relative subtree; exact complete byte manifest", "pure observation; repeat before planning a transfer; staging independently binds and rechecks source bytes"), func(_ context.Context, service *Service, in FrameworkSourceInput) (FrameworkSourceCheck, error) {
			var out FrameworkSourceCheck
			framework, err := appdev.ReadFramework(in.Name)
			if err != nil {
				return out, invalid()
			}
			root, err := service.source(in.Source)
			if err != nil {
				return out, err
			}
			defer root.Close()
			manifest, err := deskwork.SnapshotRoot(root)
			if err != nil {
				return out, &refusal{"framework_source_unavailable", "The framework source could not be observed safely.", "Review the approved subtree; only ordinary bounded files and directories are accepted."}
			}
			if framework.VerifySource(manifest) != nil {
				return out, &refusal{"framework_source_mismatch", "The source differs from the complete frozen framework desk.", "Prepare the exact pinned desk subtree outside live mounts, then repeat verification before staging."}
			}
			return FrameworkSourceCheck{Version: 1, Name: framework.Name, Commit: framework.Commit, Source: in.Source, Manifest: manifest, Verified: true, Qualification: framework.Qualification}, nil
		}),
	)
}
