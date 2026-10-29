//go:build linux

package privatechain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

var errMemberKey = errors.New("selected private member boot key cannot be verified; retain state for review")

// MemberKeyVerifier holds one existing bootstrap file and its selected directory
// chain. The caller must supply an independently accepted whole-file SHA256;
// neither a saved gateway receipt nor a caller-chosen hash is launch authority.
// This verifies bytes/metadata, not their cryptographic correspondence to the
// chain's public keys. That check and fresh gateway admission belong to the driver.
// It never stages, rotates, repairs or returns a key. It is not wallet custody
// or protection from other code running as the same Unix user.
type MemberKeyVerifier struct {
	mu                                    sync.Mutex
	directory, project, leaf, expectedSHA string
	expectedDirs                          []pierstore.DirectoryIdentity
	held                                  []*os.File
	initial                               gatewayStateEntry
}

func OpenMemberKeyVerifier(ctx context.Context, directory string, p Plan, w Workspace, g Gateway, instance, expectedSHA string) (*MemberKeyVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g.Validate(p, w) != nil || g.Phase != "ready" || !hash(expectedSHA, false) || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errMemberKey
	}
	leaf := ""
	for _, m := range p.Members {
		if m.InstanceID == instance {
			leaf = strings.TrimPrefix(m.Ship, "~") + ".key"
		}
	}
	if leaf == "" {
		return nil, errMemberKey
	}
	v := &MemberKeyVerifier{directory: directory, project: w.Binding.ProjectID, leaf: leaf, expectedSHA: expectedSHA, expectedDirs: []pierstore.DirectoryIdentity{w.Parent, w.Root, g.Observation.Secrets}}
	files, entry, err := v.openSelected(ctx)
	if err != nil {
		return nil, err
	}
	v.held, v.initial = files, entry
	if err = v.Verify(ctx); err != nil {
		v.Close()
		return nil, err
	}
	return v, nil
}

func (v *MemberKeyVerifier) openSelected(ctx context.Context) (files []*os.File, entry gatewayStateEntry, err error) {
	defer func() {
		if err != nil {
			_ = closeMemberMountFiles(files)
			files = nil
			entry = gatewayStateEntry{}
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = errMemberKey
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	var parent *os.File
	parent, err = workspaceParent(v.directory, v.expectedDirs[0])
	if err != nil {
		return
	}
	files = append(files, parent)
	for i, name := range []string{v.project, "secrets"} {
		var child *os.File
		child, err = workspaceOpen(files[len(files)-1], name)
		if err != nil {
			return
		}
		files = append(files, child)
		var id pierstore.DirectoryIdentity
		id, err = workspaceDirectoryID(child)
		if err != nil {
			return
		}
		if !v.expectedDirs[i+1].SameObject(id) {
			err = errMemberKey
			return
		}
	}
	var fd int
	fd, err = unix.Openat2(int(files[2].Fd()), v.leaf, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
	if err != nil {
		return
	}
	files = append(files, os.NewFile(uintptr(fd), "selected member boot key"))
	entry, err = gatewayStateStat(fd, "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		return
	}
	if entry.Mode != unix.S_IFREG|0600 || entry.Size == 0 || entry.Size > protocol.MaxBootKeyBytes+1 {
		err = errMemberKey
		return
	}
	err = ctx.Err()
	return
}

func (v *MemberKeyVerifier) compareSelected(ctx context.Context) error {
	files, entry, err := v.openSelected(ctx)
	if err != nil {
		return err
	}
	defer closeMemberMountFiles(files)
	if entry != v.initial {
		return errMemberKey
	}
	for i := range v.expectedDirs {
		held, e := workspaceDirectoryID(v.held[i])
		named, e2 := workspaceDirectoryID(files[i])
		if e != nil || e2 != nil || held != named || !v.expectedDirs[i].SameObject(held) {
			return errMemberKey
		}
	}
	held, err := gatewayStateStat(int(v.held[3].Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW)
	if err != nil || held != v.initial {
		return errMemberKey
	}
	return ctx.Err()
}

func (v *MemberKeyVerifier) Verify(ctx context.Context) error {
	if v == nil {
		return errMemberKey
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(v.held) != 4 {
		return errMemberKey
	}
	if err := v.compareSelected(ctx); err != nil {
		return err
	}
	raw, err := io.ReadAll(io.NewSectionReader(v.held[3], 0, protocol.MaxBootKeyBytes+2))
	defer clear(raw)
	if err != nil || uint64(len(raw)) != v.initial.Size {
		return errMemberKey
	}
	// The private initializer writes exactly one LF after the key; the pinned
	// hash covers that byte too. Other whitespace is never normalized or repaired.
	key := strings.TrimSuffix(string(raw), "\n")
	digest := sha256.Sum256(raw)
	if !protocol.ValidBootKey(key) || hex.EncodeToString(digest[:]) != v.expectedSHA {
		return errMemberKey
	}
	if err = v.compareSelected(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

func (v *MemberKeyVerifier) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	files := v.held
	v.held = nil
	v.initial = gatewayStateEntry{}
	v.expectedSHA = ""
	return closeMemberMountFiles(files)
}
