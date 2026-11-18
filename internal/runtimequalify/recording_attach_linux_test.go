//go:build linux && amd64

package runtimequalify

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

const recordingUpgrade = "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: tcp\r\nContent-Type: application/vnd.docker.multiplexed-stream\r\n\r\n"

// An inert private Unix HTTP peer, never an engine, container or runtime fixture.
func recordingAttachPeer(t *testing.T, response string, payload []byte) (string, *atomic.Int32, <-chan error) {
	t.Helper()
	dir, err := os.MkdirTemp("", "omabit-recording-attach-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "peer.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	requests := &atomic.Int32{}
	closed := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			closed <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		reader := bufio.NewReader(conn)
		req, err := http.ReadRequest(reader)
		if err != nil {
			closed <- err
			return
		}
		requests.Add(1)
		if req.Method != http.MethodPost || req.URL.Path != "/v"+hostcompat.APIVersion+"/containers/"+strings.Repeat("a", 64)+"/attach" || req.URL.RawQuery != "logs=false&stderr=true&stdin=false&stdout=true&stream=true" || req.Header.Get("Upgrade") != "tcp" || !headerToken(req.Header.Values("Connection"), "upgrade") || req.ContentLength != 0 {
			t.Error("unexpected transport or lifecycle request", req.Method, req.URL.String())
		}
		if _, err = io.WriteString(conn, response); err == nil {
			_, err = conn.Write(payload)
		}
		if err != nil {
			closed <- err
			return
		}
		_, err = reader.ReadByte()
		if err == nil {
			t.Error("attachment sent unexpected input after its upgrade")
		}
		closed <- err
	}()
	return socket, requests, closed
}

func TestRecordingAttachmentPreservesMultiplexedBytesAndOnlyDetaches(t *testing.T) {
	payload := append(bytes.Repeat([]byte("trace\n"), 350), bytes.Repeat([]byte("x"), 12288)...)
	payload = append(payload, []byte("\r\x1b[2K\x00\xff 0wdiagnostic.atom Cookie: original-secret")...)
	packet := make([]byte, 8)
	packet[0] = 2
	binary.BigEndian.PutUint32(packet[4:], uint32(len(payload)))
	packet = append(packet, payload...)
	socket, requests, closed := recordingAttachPeer(t, recordingUpgrade, packet)
	attachment, err := attachDockerRecording(t.Context(), socket, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer attachment.Close()
	if _, writable := attachment.(io.Writer); writable {
		t.Fatal("recording attachment exposed stdin/console writes")
	}
	got := make([]byte, len(packet))
	if _, err = io.ReadFull(attachment, got); err != nil || !bytes.Equal(got, packet) {
		t.Fatal("upgrade lost buffered output", err)
	}
	var original []byte
	stats, err := recording.CopyDockerStream(t.Context(), bytes.NewReader(got), func(kind recording.Kind, data []byte) error {
		if kind != recording.Stderr {
			t.Fatal("stream identity changed")
		}
		original = append(original, data...)
		return nil
	})
	if err != nil || stats.OriginalBytes != uint64(len(payload)) || !bytes.Equal(original, payload) {
		t.Fatal("transport changed original bytes or invented framing", err)
	}
	_ = attachment.Close()
	select {
	case err := <-closed:
		if err == nil || requests.Load() != 1 {
			t.Fatal("detach sent another request or input")
		}
	case <-time.After(time.Second):
		t.Fatal("detach did not close its own connection")
	}
}

func TestRecordingAttachmentCancellationInterruptsBlockedRead(t *testing.T) {
	socket, requests, closed := recordingAttachPeer(t, recordingUpgrade, nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	attachment, err := attachDockerRecording(ctx, socket, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer attachment.Close()
	read := make(chan error, 1)
	go func() { var one [1]byte; _, err := attachment.Read(one[:]); read <- err }()
	cancel()
	select {
	case err := <-read:
		if err == nil {
			t.Fatal("cancel fabricated output")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel left a blocked original-stream reader")
	}
	select {
	case <-closed:
		if requests.Load() != 1 {
			t.Fatal("cancel sent a signal or lifecycle request")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not detach its connection")
	}
}

func TestRecordingAttachmentRefusesTTYAcknowledgementHeadersAndRedirects(t *testing.T) {
	for _, response := range []string{
		strings.Replace(recordingUpgrade, "101 Switching Protocols", "200 OK", 1),
		strings.Replace(recordingUpgrade, "multiplexed-stream", "raw-stream", 1),
		strings.Replace(recordingUpgrade, "Upgrade: tcp", "Upgrade: websocket", 1),
		strings.Replace(recordingUpgrade, "Connection: Upgrade", "Connection: close", 1),
		"HTTP/1.1 302 Found\r\nLocation: http://must-not-connect.invalid/stop\r\nContent-Length: 0\r\n\r\n",
		"HTTP/1.1 500 Failed\r\nContent-Length: 0\r\n\r\n",
		strings.Replace(recordingUpgrade, "\r\n\r\n", "\r\nPrivate: "+strings.Repeat("s", 16384)+"\r\n\r\n", 1),
	} {
		socket, requests, _ := recordingAttachPeer(t, response, nil)
		attachment, err := attachDockerRecording(t.Context(), socket, strings.Repeat("a", 64))
		if !errors.Is(err, errRecordingAttachment) || attachment != nil || requests.Load() != 1 || strings.Contains(err.Error(), "must-not-connect") || strings.Contains(err.Error(), "ssss") {
			t.Fatal("unsupported/hostile transport was accepted or reflected", err)
		}
	}
}

func TestRecordingAttachmentRequiresOwnedSocketAndFullContainerBeforeConnect(t *testing.T) {
	socket, requests, _ := recordingAttachPeer(t, recordingUpgrade, nil)
	for _, id := range []string{"short", "../other", strings.Repeat("A", 64), strings.Repeat("a", 64) + "?stdin=true"} {
		if r, err := attachDockerRecording(t.Context(), socket, id); !errors.Is(err, errRecordingAttachment) || r != nil {
			t.Fatal("unbounded/unpinned selection accepted")
		}
	}
	if err := os.Chmod(socket, 0666); err != nil {
		t.Fatal(err)
	}
	if r, err := attachDockerRecording(t.Context(), socket, strings.Repeat("a", 64)); !errors.Is(err, errRecordingAttachment) || r != nil {
		t.Fatal("public socket accepted")
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(filepath.Dir(socket), "alias.sock")
	if err := os.Symlink(socket, alias); err != nil {
		t.Fatal(err)
	}
	if r, err := attachDockerRecording(t.Context(), alias, strings.Repeat("a", 64)); !errors.Is(err, errRecordingAttachment) || r != nil || requests.Load() != 0 {
		t.Fatal("invalid ownership/layout submitted a request")
	}
}
