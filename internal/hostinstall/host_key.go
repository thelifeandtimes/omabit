package hostinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
)

// Generate secrets in a fresh private root-owned staging directory. The private
// identity is published complete and without replacement before its optional
// public companion. An interrupted companion publication never rotates identity.
func (i *installer) createHostKey(ctx context.Context) error {
	if err := i.absent(hostKeyPath); err != nil {
		return err
	}
	if err := i.absent(hostKeyPath + ".pub"); err != nil {
		return err
	}
	if err := i.parents(ctx, hostKeyPath, true); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(filepath.Dir(i.path(hostKeyPath)), ".omabit-host-key-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir) // only this invocation's new private staging directory
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = errors.Join(i.chown(f), f.Chmod(0700), f.Close())
	if err != nil {
		return err
	}
	path := filepath.Join(filepath.Dir(hostKeyPath), filepath.Base(dir), "key")
	if _, err := i.command(ctx, "/usr/bin/ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", path); err != nil {
		return err
	}
	key, err := i.readInstalled(ctx, path, 16384)
	if err != nil || len(key) == 0 {
		return errors.New("staged SSH host key is unavailable")
	}
	pub, err := i.command(ctx, "/usr/bin/ssh-keygen", "-y", "-f", path)
	if err != nil {
		return err
	}
	if _, err := access.AuthorizedKey(pub, "operator"); err != nil {
		return err
	}
	if err := i.installFile(ctx, hostKeyPath, key, 0600, false); err != nil {
		return err
	}
	return i.installFile(ctx, hostKeyPath+".pub", pub, 0644, false)
}

func (i *installer) completeHostPublicKey(ctx context.Context, public []byte) error {
	// A crash between the two publications may leave only the complete private
	// key. Derive its companion without generating/rotating any secret. Preserve
	// existing public files, whose administrator-selected comments may differ.
	if _, err := i.lstat(i.path(hostKeyPath + ".pub")); errors.Is(err, os.ErrNotExist) {
		return i.installFile(ctx, hostKeyPath+".pub", public, 0644, false)
	} else {
		return err
	}
}
