//go:build linux || darwin

package checkplan

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func privateReport(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && (info.Mode().Perm() == 0600 || info.Mode().Perm() == 0400) && info.Mode()&(os.ModeSetuid|os.ModeSetgid) == 0 && stat.Uid == uint32(os.Geteuid()) && stat.Nlink == 1 && info.Size() > 0 && info.Size() <= MaxReportBytes
}

func readReportFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, ErrReport
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil || physical != path {
		return nil, ErrReport
	}
	before, err := os.Lstat(path)
	if err != nil || !privateReport(before) {
		return nil, ErrReport
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrReport
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !privateReport(opened) || !os.SameFile(before, opened) {
		return nil, ErrReport
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxReportBytes+1))
	after, e1 := f.Stat()
	bound, e2 := os.Lstat(path)
	if err != nil || e1 != nil || e2 != nil || !privateReport(after) || !privateReport(bound) || !os.SameFile(opened, after) || !os.SameFile(after, bound) || opened.Size() != after.Size() || int64(len(raw)) != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, ErrReport
	}
	return raw, nil
}
