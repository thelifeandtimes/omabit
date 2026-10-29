//go:build linux

package privatechain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

var errWorkspaceTree = errors.New("private workspace directory ownership, identity or contents could not be verified")

// WorkspaceFilesystem operates only under an explicitly selected, already-owned
// private-projects directory. It has no default host paths, engine access, keys,
// or executable inputs. The caller holds the common runner mutation/update fence
// and supplies a durable journal. This is not a boundary against the same UID.
type WorkspaceFilesystem struct {
	mu                             sync.Mutex
	directory                      string
	parent                         *os.File
	parentIdentity                 pierstore.DirectoryIdentity
	plan                           Plan
	initial                        Workspace
	rootAttempted, layoutAttempted bool
	created                        pierstore.DirectoryIdentity
	syncFile                       func(*os.File) error
}

// OpenWorkspaceFilesystem never creates the shared parent or adopts a project.
// The parent's identity must come from the runner's retained storage selection.
// Reopening an interrupted layout gives observation access only.
func OpenWorkspaceFilesystem(ctx context.Context, directory string, identity pierstore.DirectoryIdentity, plan Plan, w Workspace) (*WorkspaceFilesystem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if w.Validate(plan) != nil || !workspaceIdentity(identity) || identity != w.Parent || (w.Phase != "planned" && w.Phase != "layout-dispatched" && w.Phase != "layout-created") || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errWorkspaceTree
	}
	f, err := workspaceParent(directory, identity)
	if err != nil {
		return nil, err
	}
	plan.Members = slices.Clone(plan.Members)
	plan.Foundation.Contracts = slices.Clone(plan.Foundation.Contracts)
	return &WorkspaceFilesystem{directory: directory, parent: f, parentIdentity: identity, plan: plan, initial: w.Clone(), syncFile: (*os.File).Sync}, nil
}

func (d *WorkspaceFilesystem) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.parent == nil {
		return nil
	}
	err := d.parent.Close()
	d.parent = nil
	return err
}

func workspaceDirectoryID(f *os.File) (pierstore.DirectoryIdentity, error) {
	var st unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if f == nil || unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW, mask, &st) != nil || st.Mask&uint32(mask) != uint32(mask) || st.Mode != unix.S_IFDIR|0700 || st.Uid != uint32(os.Geteuid()) {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	id := pierstore.DirectoryIdentity{Device: unix.Mkdev(st.Dev_major, st.Dev_minor), Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", st.Btime.Sec, st.Btime.Nsec)}
	if !workspaceIdentity(id) {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	return id, nil
}

func workspaceParent(directory string, expected pierstore.DirectoryIdentity) (*os.File, error) {
	real, err := filepath.EvalSymlinks(directory)
	if err != nil || real != directory {
		return nil, errWorkspaceTree
	}
	fd, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errWorkspaceTree
	}
	f := os.NewFile(uintptr(fd), "private workspace parent")
	id, err := workspaceDirectoryID(f)
	if err != nil || !expected.SameObject(id) {
		f.Close()
		return nil, errWorkspaceTree
	}
	return f, nil
}

func workspaceOpen(parent *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat2(int(parent.Fd()), name, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
	if err != nil {
		return nil, errWorkspaceTree
	}
	f := os.NewFile(uintptr(fd), "private workspace directory")
	if _, err = workspaceDirectoryID(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func workspaceContents(f *os.File, expected []string) error {
	copy, err := workspaceOpen(f, ".")
	if err != nil {
		return err
	}
	defer copy.Close()
	names, err := copy.Readdirnames(len(expected) + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return errWorkspaceTree
	}
	slices.Sort(names)
	want := slices.Clone(expected)
	slices.Sort(want)
	if !slices.Equal(names, want) {
		return errWorkspaceTree
	}
	return nil
}

func (d *WorkspaceFilesystem) parentCurrent() error {
	if d.parent == nil {
		return errWorkspaceTree
	}
	held, err := workspaceDirectoryID(d.parent)
	if err != nil || !d.parentIdentity.SameObject(held) {
		return errWorkspaceTree
	}
	f, err := workspaceParent(d.directory, d.parentIdentity)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}
func (d *WorkspaceFilesystem) selected(ctx context.Context, w Workspace, phases ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if w.Validate(d.plan) != nil || !slices.Contains(phases, w.Phase) {
		return errWorkspaceTree
	}
	a, b := w.Clone(), d.initial.Clone()
	a.Phase = "planned"
	b.Phase = "planned"
	a.Root = pierstore.DirectoryIdentity{}
	b.Root = pierstore.DirectoryIdentity{}
	a.Observation = nil
	b.Observation = nil
	if !reflect.DeepEqual(a, b) {
		return errWorkspaceTree
	}
	if d.initial.Root != (pierstore.DirectoryIdentity{}) && w.Root != d.initial.Root {
		return errWorkspaceTree
	}
	return d.parentCurrent()
}
func (d *WorkspaceFilesystem) root(w Workspace) (*os.File, error) {
	f, err := workspaceOpen(d.parent, w.Binding.ProjectID)
	if err != nil {
		return nil, err
	}
	id, err := workspaceDirectoryID(f)
	if err != nil || !w.Root.SameObject(id) {
		f.Close()
		return nil, errWorkspaceTree
	}
	return f, nil
}

func (d *WorkspaceFilesystem) CreateRoot(ctx context.Context, w Workspace) (pierstore.DirectoryIdentity, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.selected(ctx, w, "root-dispatched"); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if d.initial.Phase != "planned" || d.rootAttempted {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	d.rootAttempted = true
	if unix.Mkdirat(int(d.parent.Fd()), w.Binding.ProjectID, 0700) != nil {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	f, err := workspaceOpen(d.parent, w.Binding.ProjectID)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	defer f.Close()
	id, err := workspaceDirectoryID(f)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	d.created = id
	if d.syncFile(f) != nil || d.syncFile(d.parent) != nil || d.parentCurrent() != nil {
		return id, errWorkspaceTree
	}
	return id, ctx.Err()
}

func (d *WorkspaceFilesystem) InspectRoot(ctx context.Context, w Workspace) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.selected(ctx, w, "root-created"); err != nil {
		return err
	}
	if !d.rootAttempted || w.Root != d.created {
		return errWorkspaceTree
	}
	f, err := d.root(w)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = workspaceContents(f, nil); err != nil {
		return err
	}
	return ctx.Err()
}

func (d *WorkspaceFilesystem) CreateLayout(ctx context.Context, w Workspace) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.selected(ctx, w, "layout-dispatched"); err != nil {
		return err
	}
	if d.initial.Phase != "planned" || !d.rootAttempted || d.layoutAttempted || w.Root != d.created {
		return errWorkspaceTree
	}
	d.layoutAttempted = true
	r, err := d.root(w)
	if err != nil {
		return err
	}
	defer r.Close()
	if workspaceContents(r, nil) != nil || unix.Mkdirat(int(r.Fd()), "piers", 0700) != nil {
		return errWorkspaceTree
	}
	piers, err := workspaceOpen(r, "piers")
	if err != nil {
		return err
	}
	defer piers.Close()
	for _, m := range w.Mounts {
		if err = ctx.Err(); err != nil {
			return err
		}
		if d.parentCurrent() != nil {
			return errWorkspaceTree
		}
		current, e := d.root(w)
		if e != nil {
			return e
		}
		current.Close()
		currentPiers, e := workspaceOpen(r, "piers")
		if e != nil {
			return e
		}
		currentID, e := workspaceDirectoryID(currentPiers)
		currentPiers.Close()
		heldID, heldErr := workspaceDirectoryID(piers)
		if e != nil || heldErr != nil || !currentID.SameObject(heldID) {
			return errWorkspaceTree
		}
		if unix.Mkdirat(int(piers.Fd()), m.InstanceID, 0700) != nil {
			return errWorkspaceTree
		}
		leaf, e := workspaceOpen(piers, m.InstanceID)
		if e != nil {
			return e
		}
		e = d.syncFile(leaf)
		leaf.Close()
		if e != nil {
			return errWorkspaceTree
		}
	}
	if d.syncFile(piers) != nil || d.syncFile(r) != nil || d.syncFile(d.parent) != nil {
		return errWorkspaceTree
	}
	return ctx.Err()
}

func (d *WorkspaceFilesystem) scan(ctx context.Context, w Workspace, syncDirectories bool) (WorkspaceObservation, error) {
	bad := WorkspaceObservation{}
	if err := ctx.Err(); err != nil {
		return bad, err
	}
	if d.parentCurrent() != nil {
		return bad, errWorkspaceTree
	}
	r, err := d.root(w)
	if err != nil {
		return bad, err
	}
	defer r.Close()
	rootID, err := workspaceDirectoryID(r)
	if err != nil || workspaceContents(r, []string{"piers"}) != nil {
		return bad, errWorkspaceTree
	}
	piers, err := workspaceOpen(r, "piers")
	if err != nil {
		return bad, err
	}
	defer piers.Close()
	piersID, err := workspaceDirectoryID(piers)
	if err != nil {
		return bad, err
	}
	names := []string{}
	for _, m := range w.Mounts {
		names = append(names, m.InstanceID)
	}
	if workspaceContents(piers, names) != nil {
		return bad, errWorkspaceTree
	}
	parentID, err := workspaceDirectoryID(d.parent)
	if err != nil {
		return bad, err
	}
	o := WorkspaceObservation{Parent: parentID, Root: rootID, PiersRoot: piersID, Complete: true, Private: true, Empty: true, Synced: true}
	for _, m := range w.Mounts {
		if err = ctx.Err(); err != nil {
			return bad, err
		}
		leaf, e := workspaceOpen(piers, m.InstanceID)
		if e != nil {
			return bad, e
		}
		id, e := workspaceDirectoryID(leaf)
		if e == nil {
			e = workspaceContents(leaf, nil)
		}
		if e == nil && syncDirectories {
			e = d.syncFile(leaf)
		}
		leaf.Close()
		if e != nil {
			return bad, errWorkspaceTree
		}
		o.Mounts = append(o.Mounts, id)
	}
	if syncDirectories && (d.syncFile(piers) != nil || d.syncFile(r) != nil || d.syncFile(d.parent) != nil) {
		return bad, errWorkspaceTree
	}
	return o, ctx.Err()
}

// ObserveLayout cannot fill a partial tree or delete unexpected contents. It
// syncs the verified empty directories, then rechecks their exact identities and
// layout. This resolves a lost layout/sync acknowledgement without another mkdir.
func (d *WorkspaceFilesystem) ObserveLayout(ctx context.Context, w Workspace) (WorkspaceObservation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.selected(ctx, w, "layout-dispatched", "layout-created"); err != nil {
		return WorkspaceObservation{}, err
	}
	before, err := d.scan(ctx, w, true)
	if err != nil {
		return WorkspaceObservation{}, err
	}
	after, err := d.scan(ctx, w, false)
	if err != nil || !reflect.DeepEqual(before, after) {
		return WorkspaceObservation{}, errWorkspaceTree
	}
	if err = d.parentCurrent(); err != nil {
		return WorkspaceObservation{}, err
	}
	after.ObservedAt = time.Now()
	return after, ctx.Err()
}

var _ WorkspaceDriver = (*WorkspaceFilesystem)(nil)
