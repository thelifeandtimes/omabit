//go:build !linux

package managed

import (
	"context"
	"net"
)

func PeerContext(ctx context.Context, _ net.Conn) context.Context { return ctx }
