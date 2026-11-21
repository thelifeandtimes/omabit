//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

var errRecordingAttachment = errors.New("private non-TTY recording attachment unavailable; no runtime effect was submitted")

// attachDockerRecording is an unwired transport primitive. Before calling, a
// trusted runtime factory must verify the exact daemon/container/launch contract
// and TTY=false. It must revalidate that ownership and persist attachment/start
// admission before allowing Start. A socket peer UID is not daemon identity or
// ownership of a ship. This function grants no lifecycle or recording authority.
//
// The lifetime belongs to an independent recorder, never a UI/SSH/Start request.
// Only the response-header handshake is timed out. Close/cancel detaches this
// read-only connection and sends no signal, stdin, Start, Stop or replay request.
// An acknowledged attachment is not proof of process start, exit or coverage.
func attachDockerRecording(ctx context.Context, socket, id string) (io.ReadCloser, error) {
	if !containerIDPattern.MatchString(id) || !filepath.IsAbs(socket) || filepath.Clean(socket) != socket || len(socket) > 107 {
		return nil, errRecordingAttachment
	}
	before, err := recordingSocket(socket)
	if err != nil {
		return nil, errRecordingAttachment
	}
	lifetime, cancel := context.WithCancel(ctx)
	transport := &http.Transport{
		Proxy: nil, DisableKeepAlives: true, DisableCompression: true,
		MaxResponseHeaderBytes: 8192, ResponseHeaderTimeout: 5 * time.Second,
		DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
			conn, e := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(dialCtx, "unix", socket)
			if e != nil {
				return nil, errRecordingAttachment
			}
			after, statErr := recordingSocket(socket)
			if statErr != nil || !os.SameFile(before, after) || !recordingSocketPeer(conn) {
				conn.Close()
				return nil, errRecordingAttachment
			}
			// Upgraded HTTP bodies need an independent cancellation hook. Merely
			// cancelling a decoder's context cannot interrupt a blocked Read.
			stop := context.AfterFunc(lifetime, func() { conn.Close() })
			return &recordingBoundConnection{Conn: conn, stop: stop}, nil
		},
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(lifetime, http.MethodPost, "http://docker/v"+hostcompat.APIVersion+"/containers/"+id+"/attach?logs=false&stderr=true&stdin=false&stdout=true&stream=true", nil)
	if err != nil {
		cancel()
		transport.CloseIdleConnections()
		return nil, errRecordingAttachment
	}
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "tcp")
	response, err := client.Do(request)
	if err != nil {
		cancel()
		transport.CloseIdleConnections()
		return nil, errRecordingAttachment
	}
	// API1.47 distinguishes multiplexed non-TTY output from raw TTY output.
	// Refuse an unacknowledged upgrade or a different transport; never guess.
	if response.StatusCode != http.StatusSwitchingProtocols || !strings.EqualFold(response.Header.Get("Upgrade"), "tcp") || !headerToken(response.Header.Values("Connection"), "upgrade") || response.Header.Get("Content-Type") != "application/vnd.docker.multiplexed-stream" {
		response.Body.Close()
		cancel()
		transport.CloseIdleConnections()
		return nil, errRecordingAttachment
	}
	return &recordingAttachment{reader: response.Body, cancel: cancel, transport: transport}, nil
}

func headerToken(values []string, token string) bool {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func recordingSocket(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errRecordingAttachment
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0007 != 0 || st.Uid != uint32(os.Geteuid()) {
		return nil, errRecordingAttachment
	}
	return info, nil
}

func recordingSocketPeer(conn net.Conn) bool {
	unix, ok := conn.(*net.UnixConn)
	if !ok {
		return false
	}
	raw, err := unix.SyscallConn()
	if err != nil {
		return false
	}
	var peer *syscall.Ucred
	var peerErr error
	err = raw.Control(func(fd uintptr) {
		peer, peerErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	return err == nil && peerErr == nil && peer != nil && peer.Pid > 0 && peer.Uid == uint32(os.Geteuid())
}

type recordingBoundConnection struct {
	net.Conn
	stop func() bool
}

func (c *recordingBoundConnection) Close() error {
	c.stop()
	return c.Conn.Close()
}

// Deliberately expose no Write method even though HTTP's upgraded body is
// bidirectional. Consumers receive output; they cannot use it as a console.
type recordingAttachment struct {
	reader    io.ReadCloser
	cancel    context.CancelFunc
	transport *http.Transport
	once      sync.Once
	closeErr  error
}

func (a *recordingAttachment) Read(data []byte) (int, error) { return a.reader.Read(data) }
func (a *recordingAttachment) Close() error {
	a.once.Do(func() {
		a.closeErr = a.reader.Close()
		a.cancel()
		a.transport.CloseIdleConnections()
	})
	return a.closeErr
}
