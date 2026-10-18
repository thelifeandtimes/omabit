// Package bootpill stages user-selected inert boot artifacts. Execution belongs
// to the pinned runtime, never to this transfer helper.
package bootpill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func CachePath(base string, p protocol.BootPill) string {
	return filepath.Join(base, "boot-pills", p.SHA256+".pill")
}
func PrivateDirectory(dir string) error {
	i, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.IsDir() || i.Mode().Perm() != 0700 || s.Uid != uint32(os.Geteuid()) {
		return errors.New("boot artifact directory must be private and owned")
	}
	return nil
}
func Open(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("pill file unavailable")
	}
	i, err := f.Stat()
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > protocol.MaxBootPillBytes {
		f.Close()
		return nil, errors.New("pill must be a nonempty regular file within 4 GB")
	}
	return f, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func Stage(ctx context.Context, dir string, input io.Reader, expected *protocol.BootPill) (path string, meta protocol.BootPill, err error) {
	if expected != nil && !protocol.ValidBootPill(expected) {
		return "", meta, errors.New("invalid expected boot pill descriptor")
	}
	if err = PrivateDirectory(dir); err != nil {
		return
	}
	f, err := os.CreateTemp(dir, ".pill-*")
	if err != nil {
		return "", meta, err
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(f.Name())
		}
	}()
	h := sha256.New()
	limit := protocol.MaxBootPillBytes
	if expected != nil {
		limit = expected.Bytes
	}
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(contextReader{ctx, input}, limit+1))
	if err != nil || n <= 0 || n > protocol.MaxBootPillBytes {
		return "", meta, errors.New("pill transfer failed or exceeds 4 GB")
	}
	meta = protocol.BootPill{SHA256: hex.EncodeToString(h.Sum(nil)), Bytes: n}
	if expected != nil && meta != *expected {
		return "", meta, errors.New("pill content differs from its selected hash or size")
	}
	if err = f.Chmod(0400); err != nil {
		return "", meta, err
	}
	if err = f.Sync(); err != nil {
		return "", meta, err
	}
	return f.Name(), meta, nil
}

func Publish(ctx context.Context, input io.Reader, expected protocol.BootPill, destination string) error {
	if !protocol.ValidBootPill(&expected) {
		return errors.New("invalid boot pill descriptor")
	}
	stage, _, err := Stage(ctx, filepath.Dir(destination), input, &expected)
	if err != nil {
		return err
	}
	defer os.Remove(stage)
	if err := os.Link(stage, destination); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func Verify(ctx context.Context, path string, expected protocol.BootPill) error {
	f, err := Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(contextReader{ctx, f}, expected.Bytes+1))
	if err != nil || n != expected.Bytes || hex.EncodeToString(h.Sum(nil)) != expected.SHA256 {
		return errors.New("staged pill content changed")
	}
	return nil
}
