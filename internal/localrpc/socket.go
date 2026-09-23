// Package localrpc provides the private Unix listener for local runner RPC.
package localrpc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// Listen requires an existing private directory owned by the current user. It
// never removes an existing path or changes permissions on a caller's directory.
func Listen(path string) (*net.UnixListener, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("socket path must be absolute")
	}
	if path != filepath.Clean(path) {
		return nil, fmt.Errorf("socket path must be clean (no repeated separators, . or .. components)")
	}
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil {
		return nil, fmt.Errorf("inspect socket directory: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("socket parent must be a private directory, not a symlink (use mode 0700): %s", parent)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, fmt.Errorf("socket directory must be owned by the current user")
	}
	if _, err := os.Lstat(path); err == nil {
		return nil, fmt.Errorf("socket path already exists; refusing to replace it: %s", path)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect socket path: %w", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("listen on private socket: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("restrict socket permissions: %w", err)
	}
	return listener, nil
}
