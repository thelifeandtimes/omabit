//go:build linux

package hostcompat

import (
	"io/fs"
	"syscall"
)

func trustedInfo(info fs.FileInfo) bool {
	if info == nil {
		return false
	}
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && s.Uid == 0 && (info.Mode()&fs.ModeSymlink != 0 || info.Mode().Perm()&0022 == 0)
}

func sameFileState(a, b fs.FileInfo) bool {
	if a == nil || b == nil || a.Size() != b.Size() || a.Mode() != b.Mode() || !a.ModTime().Equal(b.ModTime()) {
		return false
	}
	x, xok := a.Sys().(*syscall.Stat_t)
	y, yok := b.Sys().(*syscall.Stat_t)
	return xok && yok && x.Dev == y.Dev && x.Ino == y.Ino && x.Uid == y.Uid && x.Gid == y.Gid && x.Nlink == y.Nlink && x.Ctim == y.Ctim && x.Mtim == y.Mtim
}

func stampFor(info fs.FileInfo) (fileStamp, bool) {
	if info == nil {
		return fileStamp{}, false
	}
	s, ok := info.Sys().(*syscall.Stat_t)
	if !ok || s.Dev == 0 || s.Ino == 0 || s.Nlink == 0 || s.Ctim.Sec == 0 {
		return fileStamp{}, false
	}
	return fileStamp{uint64(s.Dev), uint64(s.Ino), uint64(s.Nlink), info.Size(), uint32(info.Mode()), s.Uid, s.Gid, int64(s.Mtim.Sec), int64(s.Mtim.Nsec), int64(s.Ctim.Sec), int64(s.Ctim.Nsec)}, true
}

func sameFileObject(a, b fs.FileInfo) bool {
	if a == nil || b == nil {
		return false
	}
	x, xok := a.Sys().(*syscall.Stat_t)
	y, yok := b.Sys().(*syscall.Stat_t)
	return xok && yok && x.Dev == y.Dev && x.Ino == y.Ino
}
