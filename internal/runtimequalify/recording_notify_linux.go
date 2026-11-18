//go:build linux && amd64

package runtimequalify

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NotifyRecordingSupervisor is only for the main independently supervised
// worker, after its successful readiness fsync. It sends a fixed local protocol
// message, never raw output, arbitrary status, a lifecycle effect or credentials.
// Parent admission still verifies the exact unit/process/invocation and native
// launch fence. Sending a datagram alone is not proof of manager acceptance.
func NotifyRecordingSupervisor(ctx context.Context) error {
	return notifyRecordingSupervisor(ctx, os.Getenv("NOTIFY_SOCKET"), os.Getenv("INVOCATION_ID"))
}

func notifyRecordingSupervisor(ctx context.Context, socket, invocation string) error {
	if !recordingJobHex(invocation, 32) || len(socket) < 2 || len(socket) > 107 || strings.ContainsAny(socket, "\x00\r\n") || socket[0] != '@' && (!filepath.IsAbs(socket) || filepath.Clean(socket) != socket) {
		return ErrRecordingWorker
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "unixgram", socket)
	if err != nil {
		return ErrRecordingWorker
	}
	defer connection.Close()
	deadline, _ := ctx.Deadline()
	if connection.SetWriteDeadline(deadline) != nil {
		return ErrRecordingWorker
	}
	stop := context.AfterFunc(ctx, func() { connection.Close() })
	defer stop()
	const message = "READY=1\n"
	n, err := connection.Write([]byte(message))
	if err != nil || n != len(message) || ctx.Err() != nil {
		return ErrRecordingWorker
	}
	return nil
}
