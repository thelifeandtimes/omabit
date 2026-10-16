package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Identity is an enrollment pin, not a substitute for transport authentication.
type Identity struct {
	RunnerID string
	Mode     string
}

var ErrIdentityMismatch = errors.New("runner identity or mode differs from enrollment; review the connection before re-enrolling")

// NewPinned verifies every newly dialed connection before handing it to HTTP.
// Checking only the CLI's first Hello would allow a replacement runner to receive
// mutations after a reconnect. The existing no-mutation-retry policy still applies.
func NewPinned(socketPath string, identity Identity) *Client {
	return newClient(socketPath, &identity)
}

func (i Identity) Check(id, mode string) error {
	if id != i.RunnerID || mode != i.Mode {
		return ErrIdentityMismatch
	}
	return nil
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

func verifyConnection(ctx context.Context, conn net.Conn, identity Identity) (_ net.Conn, err error) {
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer func() {
		stop()
		if err != nil {
			conn.Close()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, "http://runner/hello", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("verify runner connection: %w", err)
	}
	// Limit headers as well as the body on this pre-transport handshake.
	limited := &io.LimitedReader{R: conn, N: maxResponseBytes + 8192}
	reader := bufio.NewReader(limited)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, fmt.Errorf("read runner identity: %w", err)
	}
	// Close the connection on failures before closing a possibly undrained body.
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		conn.Close()
		resp.Body.Close()
		return nil, &RemoteError{Status: resp.StatusCode, Problem: protocol.APIError{Code: "runner_access_denied", Message: "runner identity handshake denied access"}}
	}
	if readErr != nil || len(data) > maxResponseBytes || resp.StatusCode != http.StatusOK || resp.Close {
		conn.Close()
		resp.Body.Close()
		return nil, errors.New("runner identity handshake failed or exceeded its limit")
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}
	var hello protocol.Hello
	if err := json.Unmarshal(data, &hello); err != nil {
		return nil, errors.New("invalid runner identity response")
	}
	if hello.ProtocolVersion != protocol.Version {
		return nil, errors.New("incompatible runner protocol during identity verification")
	}
	if err := identity.Check(hello.RunnerID, hello.Mode); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return nil, err
	}
	// Keep any buffered bytes, but remove the handshake byte limit for normal RPC.
	limited.N = 1<<63 - 1
	return &bufferedConn{Conn: conn, reader: reader}, nil
}
