package main

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Unlike os.Pipe, these descriptors start as inherited stdio does: blocking and
// not registered with Go's poller. No sshd, service, or live runner is involved.
func inheritedSSHPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	var fds [2]int
	if err := unix.Pipe(fds[:]); err != nil {
		t.Fatal(err)
	}
	r, w := os.NewFile(uintptr(fds[0]), "ssh-in"), os.NewFile(uintptr(fds[1]), "ssh-out")
	t.Cleanup(func() { r.Close(); w.Close() })
	return r, w
}

func TestSSHBridgeCancelsInheritedDescriptors(t *testing.T) {
	for _, phase := range []string{"input", "output"} {
		t.Run(phase, func(t *testing.T) {
			input, peer := inheritedSSHPipe(t)
			var output io.Writer = io.Discard
			if phase == "output" {
				_, writer := inheritedSSHPipe(t)
				fd := int(writer.Fd())
				if err := unix.SetNonblock(fd, true); err != nil {
					t.Fatal(err)
				}
				// Fill the output pipe without consuming it; even a denial response
				// must be interruptible when the SSH peer stops reading.
				for {
					_, err := unix.Write(fd, make([]byte, 4096))
					if errors.Is(err, unix.EAGAIN) {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := unix.SetNonblock(fd, false); err != nil {
					t.Fatal(err)
				}
				output = writer
				if _, err := io.WriteString(peer, "GET /hello HTTP/1.1\r\nHost: runner\r\n\r\n"); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- sshBridge(ctx, input, output, "", "", "") }()
			select {
			case err := <-done:
				t.Fatal("fixture did not block", err)
			case <-time.After(30 * time.Millisecond):
			}
			cancel()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("canceled SSH exchange succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("inherited SSH descriptor still blocked after cancellation")
			}
		})
	}
}
