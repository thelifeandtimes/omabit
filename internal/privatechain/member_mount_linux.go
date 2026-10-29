//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// MemberMountVerifier holds only the selected member's existing parent mount.
// It does not examine peer piers, chain/key files or engine state. It grants no
// creation, adoption, repair, sync or launch authority. The caller supplies the
// retained workspace and holds the runner lifecycle fence. Like the workspace
// verifier, this is not a security boundary against another process of its UID.
type MemberMountVerifier struct {
	mu        sync.Mutex
	directory string
	names     []string
	members   []string
	expected  []pierstore.DirectoryIdentity
	held      []*os.File
}

func OpenMemberMountVerifier(ctx context.Context, directory string, p Plan, w Workspace, instance string) (*MemberMountVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if w.Validate(p) != nil || w.Phase != "observed" || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errWorkspaceTree
	}
	index := -1
	members := make([]string, len(w.Mounts))
	for i, m := range w.Mounts {
		members[i] = m.InstanceID
		if m.InstanceID == instance {
			index = i
		}
	}
	if index < 0 {
		return nil, errWorkspaceTree
	}
	v := &MemberMountVerifier{directory: directory, names: []string{w.Binding.ProjectID, "piers", instance}, members: members, expected: []pierstore.DirectoryIdentity{w.Parent, w.Root, w.Observation.PiersRoot, w.Observation.Mounts[index]}}
	held, err := v.openChain(ctx)
	if err != nil {
		return nil, err
	}
	v.held = held
	if err = v.Verify(ctx); err != nil {
		v.Close()
		return nil, err
	}
	return v, nil
}

func closeMemberMountFiles(files []*os.File) error {
	var errs []error
	for i := len(files) - 1; i >= 0; i-- {
		errs = append(errs, files[i].Close())
	}
	return errors.Join(errs...)
}

func (v *MemberMountVerifier) openChain(ctx context.Context) (files []*os.File, err error) {
	defer func() {
		if err != nil {
			err = errors.Join(err, closeMemberMountFiles(files))
			files = nil
		}
	}()
	parent, err := workspaceParent(v.directory, v.expected[0])
	if err != nil {
		return nil, err
	}
	files = append(files, parent)
	for i, name := range v.names {
		if err = ctx.Err(); err != nil {
			return files, err
		}
		child, e := workspaceOpen(files[len(files)-1], name)
		if e != nil {
			return files, e
		}
		files = append(files, child)
		id, e := workspaceDirectoryID(child)
		if e != nil || !v.expected[i+1].SameObject(id) {
			return files, errWorkspaceTree
		}
	}
	// The project membership is immutable, but contents within a peer's mount are
	// deliberately not inspected by this member-local handle.
	if workspaceContents(files[2], v.members) != nil {
		return files, errWorkspaceTree
	}
	return files, ctx.Err()
}

func (v *MemberMountVerifier) verifyLocked(ctx context.Context, empty bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(v.held) != 4 {
		return errWorkspaceTree
	}
	current, err := v.openChain(ctx)
	if err != nil {
		return err
	}
	defer closeMemberMountFiles(current)
	for i, f := range v.held {
		before, err := workspaceDirectoryID(f)
		if err != nil || !v.expected[i].SameObject(before) {
			return errWorkspaceTree
		}
		named, err := workspaceDirectoryID(current[i])
		if err != nil || before != named {
			return errWorkspaceTree
		}
	}
	if empty && workspaceContents(v.held[3], nil) != nil {
		return errWorkspaceTree
	}
	// Reopen after content inspection so a rename during that check cannot supply
	// a successful result for an already replaced mount.
	after, err := v.openChain(ctx)
	if err != nil {
		return err
	}
	defer closeMemberMountFiles(after)
	for i, f := range v.held {
		held, e := workspaceDirectoryID(f)
		named, e2 := workspaceDirectoryID(after[i])
		if e != nil || e2 != nil || held != named || !v.expected[i].SameObject(held) {
			return errWorkspaceTree
		}
	}
	if empty && workspaceContents(after[3], nil) != nil {
		return errWorkspaceTree
	}
	return ctx.Err()
}

// Verify checks only the retained mount chain, irrespective of pier contents.
// A caller reopening a ship must additionally verify its recorded pier/.urb
// objects, stopped-state metadata and engine ownership before dispatch.
func (v *MemberMountVerifier) Verify(ctx context.Context) error {
	if v == nil {
		return errWorkspaceTree
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.verifyLocked(ctx, false)
}

// VerifyEmpty is a read-only precondition for a separate first-boot operation.
// Even a partial child causes refusal and remains in place for review.
func (v *MemberMountVerifier) VerifyEmpty(ctx context.Context) error {
	if v == nil {
		return errWorkspaceTree
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.verifyLocked(ctx, true)
}

func (v *MemberMountVerifier) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	files := slices.Clone(v.held)
	v.held = nil
	return closeMemberMountFiles(files)
}
