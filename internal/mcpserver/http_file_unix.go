//go:build linux || darwin

package mcpserver

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func httpPrivateNode(n os.FileInfo) bool {
	s, ok := n.Sys().(*syscall.Stat_t)
	return ok && n.Mode().IsRegular() && n.Mode().Perm() == 0600 && n.Mode()&(os.ModeSetuid|os.ModeSetgid) == 0 && s.Uid == uint32(os.Geteuid()) && s.Nlink == 1 && n.Size() > 0 && n.Size() <= 512
}
func readHTTPPrivateFile(path string) ([]byte, os.FileInfo, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, nil, errors.New("private credential refused")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return nil, nil, errors.New("private credential refused")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	node, err := f.Stat()
	if err != nil || !httpPrivateNode(node) {
		return nil, nil, errors.New("private credential refused")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 513))
	after, e := os.Lstat(path)
	if err != nil || e != nil || !httpPrivateNode(after) || !os.SameFile(node, after) || int64(len(raw)) != node.Size() {
		return nil, nil, errors.New("private credential refused")
	}
	return raw, node, nil
}
func writeHTTPPrivateFile(path string, raw []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	parent, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}
