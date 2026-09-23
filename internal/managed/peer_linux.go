//go:build linux

package managed

import (
	"context"
	"net"
	"syscall"
)

// PeerContext must be installed on http.Server.ConnContext. Identity comes from
// the kernel's accepted Unix connection, never a header or caller-supplied JSON.
func PeerContext(ctx context.Context, connection net.Conn) context.Context {
	c, ok := connection.(*net.UnixConn)
	if !ok {
		return ctx
	}
	raw, err := c.SyscallConn()
	if err != nil {
		return ctx
	}
	var peer *syscall.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		peer, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credErr != nil || peer == nil {
		return ctx
	}
	return context.WithValue(ctx, peerKey{}, peer.Uid)
}
