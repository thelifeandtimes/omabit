// Package client implements runner RPC over a private Unix socket.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const maxResponseBytes = 1 << 20

type Client struct {
	httpClient *http.Client
}

// RemoteError is a structured error returned by the runner.
type RemoteError struct {
	Status  int
	Problem protocol.APIError
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("runner returned HTTP %d (%s): %s", e.Status, e.Problem.Code, e.Problem.Message)
}

func New(socketPath string) *Client {
	dialer := &net.Dialer{}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{httpClient: &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

func (c *Client) Hello(ctx context.Context) (protocol.Hello, error) {
	var result protocol.Hello
	if err := c.do(ctx, http.MethodGet, "/hello", nil, &result, false); err != nil {
		return protocol.Hello{}, err
	}
	if result.ProtocolVersion != protocol.Version {
		return protocol.Hello{}, fmt.Errorf("incompatible runner protocol version %d; want %d", result.ProtocolVersion, protocol.Version)
	}
	return result, nil
}

func (c *Client) Snapshot(ctx context.Context) (protocol.Snapshot, error) {
	var result protocol.Snapshot
	if err := c.do(ctx, http.MethodGet, "/v1/snapshot", nil, &result, true); err != nil {
		return protocol.Snapshot{}, err
	}
	return result, nil
}

func (c *Client) Configure(ctx context.Context, id string, req protocol.ConfigureRequest) (protocol.Instance, error) {
	if err := validateID(id); err != nil {
		return protocol.Instance{}, err
	}
	var result protocol.Instance
	if err := c.do(ctx, http.MethodPut, "/v1/instances/"+id+"/configuration", req, &result, true); err != nil {
		return protocol.Instance{}, err
	}
	return result, nil
}

func (c *Client) Operate(ctx context.Context, id string, req protocol.OperationRequest) (protocol.Operation, error) {
	if err := validateID(id); err != nil {
		return protocol.Operation{}, err
	}
	var result protocol.Operation
	if err := c.do(ctx, http.MethodPost, "/v1/instances/"+id+"/operations", req, &result, true); err != nil {
		return protocol.Operation{}, err
	}
	return result, nil
}

func (c *Client) CreateFake(ctx context.Context, req protocol.CreateFakeRequest) (protocol.Operation, error) {
	if err := validateID(req.ID); err != nil {
		return protocol.Operation{}, err
	}
	var result protocol.Operation
	if err := c.do(ctx, http.MethodPost, "/v1/instances", req, &result, true); err != nil {
		return protocol.Operation{}, err
	}
	return result, nil
}

func validateID(id string) error {
	if len(id) < 1 || len(id) > 64 {
		return fmt.Errorf("invalid instance ID: must contain 1 to 64 ASCII letters, digits, underscores, or hyphens")
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-') {
			return fmt.Errorf("invalid instance ID: must contain 1 to 64 ASCII letters, digits, underscores, or hyphens")
		}
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, payload, result any, versioned bool) (err error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode runner request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://runner"+path, body)
	if err != nil {
		return fmt.Errorf("create runner request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if versioned {
		req.Header.Set(protocol.Header, strconv.Itoa(protocol.Version))
	}
	if method == http.MethodPut || method == http.MethodPost {
		// Even a failed write on a reused connection must not replay a mutation.
		req.GetBody = nil
		defer func() {
			if err != nil {
				if _, remote := err.(*RemoteError); !remote {
					err = fmt.Errorf("%s %s outcome uncertain; the runner may have accepted the request: %w", method, path, err)
				}
			}
		}()
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send runner request: %w", err)
	}
	defer resp.Body.Close()
	if versioned && resp.Header.Get(protocol.Header) != strconv.Itoa(protocol.Version) {
		return fmt.Errorf("incompatible runner protocol header %q; want %d", resp.Header.Get(protocol.Header), protocol.Version)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read runner response (HTTP %d): %w", resp.StatusCode, err)
	}
	if len(data) > maxResponseBytes {
		return fmt.Errorf("runner response (HTTP %d) exceeds %d-byte limit", resp.StatusCode, maxResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var response protocol.ErrorResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return fmt.Errorf("decode runner error response (HTTP %d): %w", resp.StatusCode, err)
		}
		return &RemoteError{Status: resp.StatusCode, Problem: response.Error}
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("decode runner response: %w", err)
	}
	return nil
}
