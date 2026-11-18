//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordingSupervisorNotificationContainsOnlyFixedReadiness(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "omabit-recording-notify-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err = os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "notify.sock")
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err = notifyRecordingSupervisor(t.Context(), socket, strings.Repeat("a", 32)); err != nil {
		t.Fatal(err)
	}
	if err = listener.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 1024)
	n, _, err := listener.ReadFromUnix(data)
	if err != nil || string(data[:n]) != "READY=1\n" {
		t.Fatal("notification leaked or changed payload", err)
	}
}

func TestRecordingSupervisorNotificationRefusesUnsupervisedMalformedAndCanceled(t *testing.T) {
	for _, scenario := range []string{"missing-invocation", "invalid-invocation", "relative", "traversal", "newline", "oversize", "missing-socket", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			socket, invocation := "/tmp/omabit-recording-notify-unavailable.sock", strings.Repeat("a", 32)
			switch scenario {
			case "missing-invocation":
				invocation = ""
			case "invalid-invocation":
				invocation = "PRIVATE-DIAGNOSTIC"
			case "relative":
				socket = "relative.sock"
			case "traversal":
				socket = "/tmp/../tmp/notify.sock"
			case "newline":
				socket = "/tmp/notify\n.sock"
			case "oversize":
				socket = "/" + strings.Repeat("s", 108)
			case "canceled":
				cancel()
			}
			if err := notifyRecordingSupervisor(ctx, socket, invocation); !errors.Is(err, ErrRecordingWorker) {
				t.Fatal("invalid handoff accepted", err)
			}
		})
	}
}
