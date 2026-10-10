//go:build linux && amd64

package runtimequalify

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (q *qualifier) keyPath() string { return filepath.Join(q.base, "keys", q.r.RunID+".key") }
func (q *qualifier) verifyBootKey() error {
	bad := errors.New("staged boot key is unavailable, changed or has unsafe permissions")
	if q.d.nativeShip == "" || len(q.d.keyDigest) != 64 {
		return bad
	}
	if err := q.owned(filepath.Dir(q.keyPath()), os.ModeDir, true); err != nil {
		return bad
	}
	if err := q.owned(q.keyPath(), 0, true); err != nil {
		return bad
	}
	f, err := os.OpenFile(q.keyPath(), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return bad
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil {
		return bad
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	if !ok || st.Nlink != 1 {
		return bad
	}
	b, err := io.ReadAll(io.LimitReader(f, protocol.MaxBootKeyBytes+1))
	if err != nil || !protocol.ValidBootKey(string(b)) || fmt.Sprintf("%x", sha256.Sum256(b)) != q.d.keyDigest {
		return bad
	}
	return nil
}
