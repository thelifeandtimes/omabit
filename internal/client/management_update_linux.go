package client

import (
	"context"
	"errors"
	"net"
	"net/http"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

// NewLocalManagement pins every connection to the service MainPID and owner UID.
// It uses no SSH attribution and must be invoked by the authorized local root
// installer. PID/executable/invocation evidence is separately checked by it.
func NewLocalManagement(socket string, uid uint32, pid int) *Client {
	return newTransport(func(ctx context.Context) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
		if err != nil {
			return nil, err
		}
		u, ok := c.(*net.UnixConn)
		if !ok {
			c.Close()
			return nil, errors.New("management requires a Unix peer")
		}
		raw, err := u.SyscallConn()
		if err != nil {
			c.Close()
			return nil, err
		}
		var peer *unix.Ucred
		var peerErr error
		err = raw.Control(func(fd uintptr) { peer, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
		if err != nil || peerErr != nil || peer == nil || peer.Uid != uid || int(peer.Pid) != pid || pid <= 0 {
			c.Close()
			return nil, errors.New("management socket peer does not match service owner/MainPID")
		}
		return c, nil
	}, nil)
}

func (c *Client) ManagementUpdate(ctx context.Context, request *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
	var status protocol.ManagementUpdateStatus
	method := http.MethodGet
	var payload any
	if request != nil {
		method, payload = http.MethodPost, request
	}
	err := c.do(ctx, method, protocol.ManagementUpdatePath, payload, &status, true)
	return status, err
}
