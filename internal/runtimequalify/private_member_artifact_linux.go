//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"golang.org/x/sys/unix"
)

// Create exclusively beneath a held managed root. No path-based write follows
// a renamed root, and a partial artifact is retained for attributed recovery.
// The caller journals dispatch first and must never repeat this effect.
func preparePrivateMemberArtifact(ctx context.Context, c privateMemberContract, meta protocol.BootPill) (id pierstore.DirectoryIdentity, err error) {
	if ctx.Err() != nil {
		return id, ctx.Err()
	}
	canonical, e := newPrivateMemberContract(c.selection, c.root, c.imageID)
	if e != nil || c.digest != canonical.digest || c.pillPath != canonical.pillPath || c.selection.Mode != "boot" || !protocol.ValidBootPill(&meta) {
		return id, errPrivateMemberLaunch
	}
	real, e := filepath.EvalSymlinks(c.root)
	if e != nil || real != c.root {
		return id, errPrivateMemberLaunch
	}
	root, e := privateClientFile(c.root, true)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	defer root.Close()
	rootID, e := privatePillDirectory(root)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	current := func() bool {
		f, e := privateClientFile(c.root, true)
		if e != nil {
			return false
		}
		defer f.Close()
		got, e := privatePillDirectory(f)
		return e == nil && rootID.SameObject(got)
	}
	// Open the cache through the held root too. Copy-time verification is the
	// authority; a previously checked cache hash is never trusted here.
	cache, e := privatePillChild(root, "boot-pills", true)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	defer cache.Close()
	if !privateClientOwned(cache, true) {
		return id, errPrivateMemberLaunch
	}
	input, e := privatePillChild(cache, filepath.Base(bootpill.CachePath(c.root, meta)), false)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	defer input.Close()
	st, e := input.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() != meta.Bytes || !current() || ctx.Err() != nil {
		return id, errPrivateMemberLaunch
	}
	if unix.Mkdirat(int(root.Fd()), c.selection.RuntimeID, 0700) != nil {
		return id, errPrivateMemberLaunch
	}
	dir, e := privatePillChild(root, c.selection.RuntimeID, true)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	defer dir.Close()
	id, e = privatePillDirectory(dir)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	// Every error below returns the newly created directory's identity.
	if root.Sync() != nil || !current() || ctx.Err() != nil {
		return id, errPrivateMemberLaunch
	}
	fd, e := unix.Openat(int(dir.Fd()), candidate.PillName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		return id, errPrivateMemberLaunch
	}
	output := os.NewFile(uintptr(fd), "private member boot artifact")
	defer output.Close()
	h := sha256.New()
	n, e := io.Copy(io.MultiWriter(output, h), io.LimitReader(privatePillReader{ctx: ctx, source: input}, meta.Bytes+1))
	if e != nil || n != meta.Bytes || fmt.Sprintf("%x", h.Sum(nil)) != meta.SHA256 {
		return id, errPrivateMemberLaunch
	}
	if output.Chmod(0400) != nil || output.Sync() != nil || dir.Sync() != nil || !current() || ctx.Err() != nil {
		return id, errPrivateMemberLaunch
	}
	return id, nil
}

// Before Create, the artifact is not yet part of an owned member's container
// contract. Still exclude foreign readers/writers of it, including ancestor and
// inode aliases. Once created, only the already fully inspected member is exempt.
func checkPrivateMemberArtifactMounts(ctx context.Context, inventory func(context.Context) ([]projectContainerMounts, error), c privateMemberContract, members []privateGatewayMember) error {
	path := filepath.Dir(c.pillPath)
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return errPrivateMemberLaunch
	}
	owned, err := projectPathObjects(ctx, path)
	if err != nil {
		return errPrivateMemberLaunch
	}
	pill, err := projectPathObjects(ctx, c.pillPath)
	if err != nil {
		return errPrivateMemberLaunch
	}
	before, err := inventory(ctx)
	if err != nil {
		return errPrivateMemberLaunch
	}
	for _, row := range before {
		exempt := false
		for _, m := range members {
			if row.ID == m.ContainerID && reflect.DeepEqual(m.Contract, c) {
				exempt = true
			}
		}
		if exempt {
			continue
		}
		for _, m := range row.Mounts {
			if m.Type == "tmpfs" && m.Source == "" {
				continue
			}
			if m.Type != "bind" {
				return errPrivateMemberLaunch
			}
			source, err := projectPathObjects(ctx, m.Source)
			if err != nil || projectPathsOverlap(owned, source) || projectPathsOverlap(pill, source) {
				return errPrivateMemberLaunch
			}
		}
	}
	after, err := inventory(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		return errPrivateMemberLaunch
	}
	last, err := projectPathObjects(ctx, path)
	if err != nil || !sameProjectPathObjects(owned, last) || ctx.Err() != nil {
		return errPrivateMemberLaunch
	}
	lastPill, err := projectPathObjects(ctx, c.pillPath)
	if err != nil || !sameProjectPathObjects(pill, lastPill) {
		return errPrivateMemberLaunch
	}
	return nil
}
