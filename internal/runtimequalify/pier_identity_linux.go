//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piermeta"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func InspectPierIdentity(ctx context.Context, path string) (protocol.PierIdentity, error) {
	var out protocol.PierIdentity
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if pierstore.Busy(path) {
		return out, errors.New("resolve the stopped pier's Vere lock before identification")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return out, err
	}
	defer root.Close()
	f, err := root.OpenFile(".urb/log/data.mdb", os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return out, errors.New("Vere public identity metadata is unavailable")
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return out, errors.New("invalid identity metadata file")
	}
	identity, err := piermeta.Read(f, before.Size())
	if err != nil {
		return out, err
	}
	after, err := f.Stat()
	if err != nil || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || pierstore.Busy(path) {
		return out, errors.New("pier changed during identification")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := inspectExternalEpochs(ctx, path); err != nil {
		return out, err
	}
	return protocol.PierIdentity{Ship: identity.Ship, Network: identity.Network, Kind: identity.Kind}, nil
}

func inspectExternalEpochs(ctx context.Context, path string) error {
	if err := pierstore.CheckStateFiles(ctx, path); err != nil {
		return err
	}
	q := &qualifier{d: defaults(), uid: os.Geteuid(), gid: os.Getegid()}
	_, err := q.inspectEpochInventoryAt(ctx, path, true)
	return err
}

// Inspect an automatic destination without creating directories. Existing
// exclusive parents may be reused only for this same reviewed source pier.
func ImportDestinationAvailable(relative, source string) (bool, error) {
	return importDestinationAvailable(defaults().home, relative, source)
}
func importDestinationAvailable(home, relative, source string) (bool, error) {
	target, err := pierstore.Absolute(home, relative)
	if err != nil {
		return false, err
	}
	if filepath.Dir(source) == filepath.Dir(target) {
		mount, err := pierstore.IdentifyMount(filepath.Dir(source))
		if err != nil {
			return false, err
		}
		_, rel, err := pierstore.Resolve(home, relative)
		if err != nil {
			return false, err
		}
		id, _ := pierstore.Split(relative)
		loc := pierstore.Location{Path: pierstore.Qualify(id, filepath.Join(filepath.Dir(rel), filepath.Base(source))), MountIdentity: mount}
		_, err = pierstore.CheckMount(home, loc)
		return err == nil, nil // a non-exclusive parent is occupied; choose another
	}
	_, err = os.Lstat(filepath.Dir(target))
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	return false, err
}
