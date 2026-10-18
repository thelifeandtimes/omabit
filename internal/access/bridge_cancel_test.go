package access

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type bridgeWriteNotice struct {
	*io.PipeWriter
	entered chan struct{}
	once    sync.Once
}

func (w *bridgeWriteNotice) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	return w.PipeWriter.Write(p)
}

func TestBridgeCancellationInterruptsInputAndOutput(t *testing.T) {
	for _, phase := range []string{"idle", "header", "body", "pill", "handoff", "output", "denied-output", "handoff-output"} {
		t.Run(phase, func(t *testing.T) {
			socket, keys, grant := bridgeGrantFixture(t)
			var requests atomic.Int32
			bridgeRunnerFixture(t, socket, grant, "runner", "development-managed", &requests)
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			defer writer.Close()
			var output io.Writer = &bytes.Buffer{}
			var notice *bridgeWriteNotice
			if strings.HasSuffix(phase, "output") {
				read, write := io.Pipe()
				defer read.Close()
				defer write.Close()
				notice = &bridgeWriteNotice{PipeWriter: write, entered: make(chan struct{})}
				output = notice
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Bridge(ctx, input, output, socket, keys, grant.Fingerprint) }()
			partial := ""
			switch phase {
			case "header":
				partial = "GET /hello HTTP/1.1\r\nHost:"
			case "body":
				partial = "POST /v1/instances HTTP/1.1\r\nHost: runner\r\nContent-Length: 2\r\n\r\n{"
			case "pill":
				partial = "POST " + protocol.PillUploadPath + " HTTP/1.1\r\nHost: runner\r\nContent-Length: 10000\r\nContent-Type: application/octet-stream\r\n" + protocol.PillHashHeader + ": " + strings.Repeat("a", 64) + "\r\n\r\nx"
			case "handoff":
				partial = "PUT /v1/handoffs/" + strings.Repeat("a", 32) + "/data HTTP/1.1\r\nHost: runner\r\nContent-Length: 10000\r\nContent-Type: application/octet-stream\r\n\r\nx"
			case "output":
				partial = "GET /hello HTTP/1.1\r\nHost: runner\r\n\r\n"
			case "denied-output":
				partial = "GET /forbidden HTTP/1.1\r\nHost: runner\r\n\r\n"
			case "handoff-output":
				partial = "GET /v1/handoffs/" + strings.Repeat("a", 32) + "/data HTTP/1.1\r\nHost: runner\r\n\r\n"
			}
			if _, err := io.WriteString(writer, partial); err != nil {
				t.Fatal(err)
			}
			if notice != nil {
				select {
				case <-notice.entered:
				case <-time.After(time.Second):
					t.Fatal("never reached output")
				}
			} else {
				// The peer remains open. Give the bridge time to enter the read.
				select {
				case err := <-done:
					t.Fatal("did not block in fixture", err)
				case <-time.After(20 * time.Millisecond):
				}
			}
			cancel()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("canceled exchange succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("cancellation did not release I/O")
			}
		})
	}
}

func TestBridgeProgressDeadlinePreservesIdleWatcher(t *testing.T) {
	socket, keys, grant := bridgeGrantFixture(t)
	var requests atomic.Int32
	bridgeRunnerFixture(t, socket, grant, "runner", "development-managed", &requests)
	stream, peer := net.Pipe()
	defer peer.Close()
	defer stream.Close()
	done := make(chan error, 1)
	go func() {
		done <- bridge(context.Background(), stream, stream, socket, keys, grant.Fingerprint, 100*time.Millisecond)
	}()
	reader := bufio.NewReader(peer)
	for range 2 {
		select {
		case err := <-done:
			t.Fatal("healthy idle watcher was closed", err)
		case <-time.After(250 * time.Millisecond):
		}
		if _, err := io.WriteString(peer, "GET /hello HTTP/1.1\r\nHost: runner\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
		readBridgeResponse(t, reader, "GET")
	}
	if _, err := io.WriteString(peer, "GET /hello HTTP/1.1\r\nHost:"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("partial frame succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("partial framing was not bounded")
	}
}

func TestBridgePeerEOF(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	done := make(chan error, 1)
	go func() { done <- Bridge(context.Background(), input, io.Discard, "", "", "") }()
	writer.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("EOF did not release bridge")
	}
}
