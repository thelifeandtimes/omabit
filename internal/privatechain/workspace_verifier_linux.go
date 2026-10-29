//go:build linux

package privatechain

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
)

// WorkspaceVerifier holds observation access to an already completed workspace.
// It exposes no creation, sync, adoption or repair operation. VerifyEmpty is a
// fresh precondition for a separate journalled effect, not permission to start.
type WorkspaceVerifier struct {
	filesystem *WorkspaceFilesystem
	workspace  Workspace
}

func OpenWorkspaceVerifier(ctx context.Context, directory string, p Plan, w Workspace) (*WorkspaceVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if w.Validate(p) != nil || w.Phase != "observed" || !filepath.IsAbs(directory) ||
		filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errWorkspaceTree
	}
	parent, err := workspaceParent(directory, w.Parent)
	if err != nil {
		return nil, err
	}
	p.Members = slices.Clone(p.Members)
	p.Foundation.Contracts = slices.Clone(p.Foundation.Contracts)
	fs := &WorkspaceFilesystem{directory: directory, parent: parent, parentIdentity: w.Parent, plan: p, initial: w.Clone()}
	v := &WorkspaceVerifier{filesystem: fs, workspace: w.Clone()}
	if err = v.VerifyEmpty(ctx); err != nil {
		fs.Close()
		return nil, err
	}
	return v, nil
}

func (v *WorkspaceVerifier) Close() error { return v.filesystem.Close() }

func (v *WorkspaceVerifier) VerifyEmpty(ctx context.Context) error {
	d := v.filesystem
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	// scan(false) only opens/reads. It does not perform the original layout's
	// durability step again or represent a new sync as having taken place.
	before, err := d.scan(ctx, v.workspace, false)
	if err != nil {
		return err
	}
	after, err := d.scan(ctx, v.workspace, false)
	if err != nil || !reflect.DeepEqual(before, after) || !v.sameObjects(after) {
		return errWorkspaceTree
	}
	if err = d.parentCurrent(); err != nil {
		return err
	}
	return ctx.Err()
}

func (v *WorkspaceVerifier) sameObjects(o WorkspaceObservation) bool {
	expected := v.workspace.Observation
	if expected == nil || !o.Complete || !o.Private || !o.Empty ||
		!expected.Parent.SameObject(o.Parent) || !expected.Root.SameObject(o.Root) ||
		!expected.PiersRoot.SameObject(o.PiersRoot) || len(expected.Mounts) != len(o.Mounts) {
		return false
	}
	for n, m := range expected.Mounts {
		if !m.SameObject(o.Mounts[n]) {
			return false
		}
	}
	return true
}
