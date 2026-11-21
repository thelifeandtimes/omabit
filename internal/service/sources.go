package service

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// SourceRoot is a user-selected directory handle, never a model-supplied host
// path. OpenSourceRoot is called by the CLI startup adapter. The caller closes it
// after all service calls finish. Renaming/replacing the original path cannot
// silently rebind an existing server to a different directory.
type SourceRoot struct {
	Name string
	root *os.Root
}

func OpenSourceRoot(name, directory string) (*SourceRoot, error) {
	if !protocol.ValidID(name) || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, errors.New("source root requires a name and an absolute ordinary directory")
	}
	before, err := os.Lstat(directory)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("source root must be an ordinary directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, errors.New("cannot open approved source root")
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		root.Close()
		return nil, errors.New("source root changed during selection")
	}
	return &SourceRoot{Name: name, root: root}, nil
}

func (r *SourceRoot) Close() error { return r.root.Close() }

type SourceSelection struct {
	Root      string `json:"root" jsonschema:"Name of a directory explicitly allowed at process startup; no host path"`
	Directory string `json:"directory" jsonschema:"Ordinary relative source subtree, or .; no parent traversal, absolute path or links"`
}

func (selection SourceSelection) Validate() error {
	if !protocol.ValidID(selection.Root) || !fs.ValidPath(selection.Directory) || strings.Contains(selection.Directory, "\\") || len(selection.Directory) > 1024 || len(strings.Split(selection.Directory, "/")) > 16 {
		return invalid()
	}
	return nil
}

func (s *Service) source(selection SourceSelection) (*os.Root, error) {
	if selection.Validate() != nil {
		return nil, invalid()
	}
	var bound *SourceRoot
	for _, root := range s.options.Sources {
		if root.Name == selection.Root {
			bound = root
			break
		}
	}
	if bound == nil {
		return nil, &refusal{"source_not_allowed", "The source root was not allowed at process startup.", "Select only a named, explicitly approved source root."}
	}
	// Check every selected path component. OpenRoot also confines races to the
	// approved root; identity comparison refuses replacement during this read.
	parts := strings.Split(selection.Directory, "/")
	for i := range parts {
		info, err := bound.root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, &refusal{"source_unavailable", "The selected source is not an ordinary approved directory.", "Review the source subtree; links and parent traversal are refused."}
		}
	}
	before, err := bound.root.Lstat(selection.Directory)
	if err != nil {
		return nil, invalid()
	}
	root, err := bound.root.OpenRoot(selection.Directory)
	if err != nil {
		return nil, invalid()
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		root.Close()
		return nil, invalid()
	}
	return root, nil
}

type SourceStageInput struct {
	DeskInput
	Source SourceSelection `json:"source"`
}
type SourceStagePlan struct {
	Version int                      `json:"version"`
	Source  SourceSelection          `json:"source"`
	Plan    collection.DeskStagePlan `json:"plan"`
}
type SourceStageSubmit struct {
	Target
	Plan SourceStagePlan `json:"plan"`
}

func (input SourceStageInput) Validate() error {
	if input.DeskInput.Validate() != nil || input.Source.Validate() != nil {
		return invalid()
	}
	return nil
}

func (input SourceStageSubmit) Validate() error {
	if input.Target.Validate() != nil || input.Plan.Version != 1 || input.Plan.Source.Validate() != nil || input.Plan.Plan.Validate() != nil || input.Plan.Plan.RunnerID != input.RunnerID {
		return invalid()
	}
	return nil
}

func (s *Service) registerSources() error {
	bindings := "explicit runner/instance/desk plus process-startup source-root handle and relative subtree; exact source digest/byte bounds/settings revision/stage ID"
	plan := description("desk.plan-stage.v1", "Prepare a bounded staging plan from an explicitly allowed local source subtree. Reads files; sends no source bytes and does not commit or mount.", []string{"omabit desk-plan-stage"}, []string{"local-metadata-read", "local-source-read", "live-probe"}, []string{protocol.DeskStageFeature, protocol.DeskSettingsFeature}, "authenticated", bindings, "save the exact returned source selection and plan before transfer")
	stage := description("desk.stage.v1", "Transfer the exact reviewed source into the runner cache; does not commit to Clay or create a Unix mount.", []string{"omabit desk-stage"}, []string{"local-metadata-read", "local-source-read", "live-probe", "journal-write", "runner-cache-write"}, []string{protocol.DeskStageFeature}, "operator", bindings, "preserve stage ID and digest after lost response; read desk.stage-status.v1 before further effects; unchanged exact replay verifies cached bytes")
	return errors.Join(
		selected(s, plan, func(ctx context.Context, registry *collection.Registry, _ collection.Connection, input SourceStageInput) (SourceStagePlan, error) {
			root, err := s.source(input.Source)
			if err != nil {
				return SourceStagePlan{}, err
			}
			defer root.Close()
			p, err := registry.PlanDeskStageRoot(ctx, input.Name, input.RunnerID, input.InstanceID, input.Desk, root)
			if err != nil {
				return SourceStagePlan{}, err
			}
			return SourceStagePlan{Version: 1, Source: input.Source, Plan: p}, nil
		}),
		selected(s, stage, func(ctx context.Context, registry *collection.Registry, _ collection.Connection, input SourceStageSubmit) (protocol.DeskStage, error) {
			if input.Plan.Version != 1 || input.Plan.Plan.Validate() != nil {
				return protocol.DeskStage{}, invalid()
			}
			root, err := s.source(input.Plan.Source)
			if err != nil {
				return protocol.DeskStage{}, err
			}
			defer root.Close()
			return registry.StageDeskRoot(ctx, input.Name, input.RunnerID, input.Plan.Plan, root)
		}),
	)
}
