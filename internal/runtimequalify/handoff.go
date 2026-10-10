package runtimequalify

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func HandoffArchive(base string, h *protocol.Handoff) string {
	return filepath.Join(base, "handoffs", h.Direction+"-"+h.ID+".tar")
}

func SyncHandoffDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func VerifyHandoffArchive(path string, h *protocol.Handoff) bool {
	f, err := OpenHandoffArchive(path)
	if err != nil {
		return false
	}
	defer f.Close()
	digest := sha256.New()
	n, err := io.Copy(digest, io.LimitReader(f, protocol.MaxHandoffBytes+1))
	return err == nil && n == h.Bytes && fmt.Sprintf("%x", digest.Sum(nil)) == h.SHA256
}

// Handoff streams are larger than boot pills and have their own ownership and
// framing contract. Never use the boot-pill reader's smaller bound here.
func OpenHandoffArchive(path string) (*os.File, error) {
	if err := bootpill.PrivateDirectory(filepath.Dir(path)); err != nil {
		return nil, errors.New("handoff artifact directory is not private")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("handoff artifact unavailable")
	}
	i, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.Mode().IsRegular() || i.Mode().Perm() != 0400 || s.Uid != uint32(os.Geteuid()) || s.Nlink != 1 || i.Size() <= 0 || i.Size() > protocol.MaxHandoffBytes {
		f.Close()
		return nil, errors.New("handoff artifact must be a private owned regular file within the transfer limit")
	}
	return f, nil
}
