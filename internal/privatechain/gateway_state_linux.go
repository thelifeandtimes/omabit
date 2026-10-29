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
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

// GatewayStateObservation is metadata only. It says nothing about durability,
// database validity, live keys or the engine that uses this workspace.
type GatewayStateObservation struct {
	Root, Chain, Secrets pierstore.DirectoryIdentity
	Mounts               []pierstore.DirectoryIdentity
	Private              bool
}

// GatewayStateVerifier observes initialized workspace metadata. The initial
// constructor requires empty mounts; the member-aware constructor binds retained
// pier identities and leaves their payload opaque. Neither reads regular files, syncs,
// create, repair, initialize or adopt another workspace. The caller must hold
// the runner lifecycle fence and bind separate live engine/RPC observations.
// Like other workspace helpers, this is not a boundary against the same UID.
type GatewayStateVerifier struct {
	filesystem  *WorkspaceFilesystem
	workspace   Workspace
	initial     *GatewayStateObservation
	memberPiers map[string]*MemberPierVerifier
}

func OpenGatewayStateVerifier(ctx context.Context, directory string, p Plan, w Workspace, g Gateway) (*GatewayStateVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g.Validate(p, w) != nil || (g.Phase != "start-dispatched" && g.Phase != "started") || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errWorkspaceTree
	}
	return openGatewayStateVerifier(ctx, directory, p, w, nil, nil)
}
func (v *GatewayStateVerifier) Close() error {
	// Closing the shared parent first serializes against Verify and prevents a
	// later scan from reaching the held member handles. Their own Close is safe
	// under repeated/concurrent calls; the membership map never changes.
	err := v.filesystem.Close()
	for _, member := range v.memberPiers {
		err = errors.Join(err, member.Close())
	}
	return err
}

type gatewayStateEntry struct {
	Device, Inode, Size      uint64
	Mode                     uint16
	UID, Links               uint32
	Birth, Modified, Changed unix.StatxTimestamp
}

func gatewayStateStat(fd int, name string, flags int) (gatewayStateEntry, error) {
	var st unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if unix.Statx(fd, name, flags, mask, &st) != nil || st.Mask&uint32(mask) != uint32(mask) || st.Uid != uint32(os.Geteuid()) {
		return gatewayStateEntry{}, errWorkspaceTree
	}
	if st.Mode != unix.S_IFDIR|0700 && (st.Mode != unix.S_IFREG|0600 || st.Nlink != 1) {
		return gatewayStateEntry{}, errWorkspaceTree
	}
	return gatewayStateEntry{Device: unix.Mkdev(st.Dev_major, st.Dev_minor), Inode: st.Ino, Size: st.Size, Mode: st.Mode, UID: st.Uid, Links: st.Nlink, Birth: st.Btime, Modified: st.Mtime, Changed: st.Ctime}, nil
}
func (s gatewayStateEntry) directoryID() (pierstore.DirectoryIdentity, error) {
	id := pierstore.DirectoryIdentity{Device: s.Device, Inode: s.Inode, Birth: fmt.Sprintf("%d:%d", s.Birth.Sec, s.Birth.Nsec)}
	if s.Mode != unix.S_IFDIR|0700 || !workspaceIdentity(id) {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	return id, nil
}
func gatewayStateNames(f *os.File) ([]string, error) {
	copy, err := workspaceOpen(f, ".")
	if err != nil {
		return nil, err
	}
	defer copy.Close()
	names, err := copy.Readdirnames(40001)
	if err != nil && err != io.EOF || len(names) > 40000 {
		return nil, errWorkspaceTree
	}
	slices.Sort(names)
	return names, nil
}

func (v *GatewayStateVerifier) scan(ctx context.Context) (GatewayStateObservation, map[string]gatewayStateEntry, error) {
	d := v.filesystem
	bad := func() (GatewayStateObservation, map[string]gatewayStateEntry, error) {
		return GatewayStateObservation{}, nil, errWorkspaceTree
	}
	if ctx.Err() != nil {
		return GatewayStateObservation{}, nil, ctx.Err()
	}
	if d.parentCurrent() != nil {
		return bad()
	}
	root, err := d.root(v.workspace)
	if err != nil {
		return bad()
	}
	defer root.Close()
	entries := map[string]gatewayStateEntry{}
	var total uint64
	var walk func(*os.File, string, int) error
	walk = func(dir *os.File, rel string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth > 32 || len(entries) >= 40000 {
			return errWorkspaceTree
		}
		before, err := gatewayStateStat(int(dir.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
		if err != nil || before.Device != v.workspace.Root.Device {
			return errWorkspaceTree
		}
		entries[rel] = before
		if member := v.memberPiers[rel]; member != nil {
			// A live pier is opaque here. Verify its exact retained pier/.urb
			// objects and containing layout without walking database contents.
			if err := member.Verify(ctx); err != nil {
				return errWorkspaceTree
			}
			after, err := gatewayStateStat(int(dir.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
			if err != nil || after != before {
				return errWorkspaceTree
			}
			return nil
		}
		names, err := gatewayStateNames(dir)
		if err != nil {
			return err
		}
		for _, name := range names {
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(entries) >= 40000 {
				return errWorkspaceTree
			}
			fd, err := unix.Openat2(int(dir.Fd()), name, &unix.OpenHow{Flags: unix.O_PATH | unix.O_NOFOLLOW | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
			if err != nil {
				return errWorkspaceTree
			}
			entry, readErr := gatewayStateStat(fd, "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
			if readErr != nil || entry.Device != before.Device {
				unix.Close(fd)
				return errWorkspaceTree
			}
			childRel := name
			if rel != "." {
				childRel = rel + "/" + name
			}
			if entry.Mode == unix.S_IFDIR|0700 {
				child, e := workspaceOpen(dir, name)
				if e != nil {
					unix.Close(fd)
					return e
				}
				e = walk(child, childRel, depth+1)
				child.Close()
				if e != nil || entries[childRel] != entry {
					unix.Close(fd)
					return errWorkspaceTree
				}
			} else {
				if entry.Size > 1<<30 || total > 1<<30-entry.Size {
					unix.Close(fd)
					return errWorkspaceTree
				}
				total += entry.Size
				entries[childRel] = entry
			}
			held, e1 := gatewayStateStat(fd, "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
			unix.Close(fd)
			named, e2 := gatewayStateStat(int(dir.Fd()), name, unix.AT_SYMLINK_NOFOLLOW|unix.AT_NO_AUTOMOUNT)
			if e1 != nil || e2 != nil || held != entry || named != entry {
				return errWorkspaceTree
			}
		}
		afterNames, err := gatewayStateNames(dir)
		if err != nil || !slices.Equal(names, afterNames) {
			return errWorkspaceTree
		}
		after, err := gatewayStateStat(int(dir.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
		if err != nil || after != before {
			return errWorkspaceTree
		}
		return nil
	}
	if err = walk(root, ".", 0); err != nil {
		return GatewayStateObservation{}, nil, err
	}
	if workspaceContents(root, []string{"home", "chain", "secrets", "piers", "marker", "guest-namespaces.json", "gateway-ready.json"}) != nil {
		return bad()
	}
	for _, name := range []string{"marker", "guest-namespaces.json", "gateway-ready.json"} {
		s := entries[name]
		if s.Mode != unix.S_IFREG|0600 || s.Size == 0 || s.Size > 65536 {
			return bad()
		}
	}
	ids := map[string]pierstore.DirectoryIdentity{}
	for _, name := range []string{".", "home", "chain", "secrets", "piers"} {
		id, e := entries[name].directoryID()
		if e != nil {
			return bad()
		}
		ids[name] = id
	}
	if !v.workspace.Root.SameObject(ids["."]) || !v.workspace.Observation.PiersRoot.SameObject(ids["piers"]) {
		return bad()
	}
	o := GatewayStateObservation{Root: ids["."], Chain: ids["chain"], Secrets: ids["secrets"], Private: true}
	wantPiers := map[string]bool{"piers": true}
	for i, m := range v.workspace.Mounts {
		rel := "piers/" + m.InstanceID
		wantPiers[rel] = true
		id, e := entries[rel].directoryID()
		if e != nil || !v.workspace.Observation.Mounts[i].SameObject(id) {
			return bad()
		}
		o.Mounts = append(o.Mounts, id)
	}
	secrets := map[string]bool{"secrets": true, "secrets/0.generation-output": true, "secrets/1.generation-output": true, "secrets/zod.key": true, "secrets/nec.key": true}
	chainData := false
	for rel, entry := range entries {
		if strings.HasPrefix(rel, "chain/") && entry.Mode == unix.S_IFREG|0600 && entry.Size > 0 {
			chainData = true
		}
		if strings.HasPrefix(rel, "piers/") && !wantPiers[rel] {
			return bad()
		}
		if strings.HasPrefix(rel, "secrets/") && (!secrets[rel] || entry.Mode != unix.S_IFREG|0600 || entry.Size == 0 || entry.Size > 1<<20) {
			return bad()
		}
	}
	if !chainData {
		return bad()
	}
	for rel := range secrets {
		if _, ok := entries[rel]; !ok {
			return bad()
		}
	}
	current, err := d.root(v.workspace)
	if err != nil {
		return bad()
	}
	current.Close()
	if d.parentCurrent() != nil {
		return bad()
	}
	return o, entries, ctx.Err()
}

func (v *GatewayStateVerifier) Verify(ctx context.Context) (GatewayStateObservation, error) {
	d := v.filesystem
	d.mu.Lock()
	defer d.mu.Unlock()
	before, a, err := v.scan(ctx)
	if err != nil {
		return GatewayStateObservation{}, err
	}
	after, b, err := v.scan(ctx)
	if err != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(a, b) || (v.initial != nil && !reflect.DeepEqual(*v.initial, after)) {
		return GatewayStateObservation{}, errWorkspaceTree
	}
	return after, ctx.Err()
}
