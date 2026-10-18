// Package client implements runner RPC over a private Unix socket.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

const maxResponseBytes = 1 << 20

type Client struct {
	httpClient *http.Client
	cancel     context.CancelFunc
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
	return newClient(socketPath, nil)
}

func newClient(socketPath string, identity *Identity) *Client {
	dialer := &net.Dialer{}
	return newTransport(func(ctx context.Context) (net.Conn, error) {
		return dialer.DialContext(ctx, "unix", socketPath)
	}, identity)
}

func newTransport(dial func(context.Context) (net.Conn, error), identity *Identity) *Client {
	lifetime, cancel := context.WithCancel(context.Background())
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			// net/http can detach a dial from its initiating request. Closing this
			// client must also cancel any unfinished identity handshake.
			ctx, stop := context.WithCancel(ctx)
			stopLifetime := context.AfterFunc(lifetime, stop)
			defer stopLifetime()
			defer stop()
			conn, err := dial(ctx)
			if err != nil {
				return conn, err
			}
			if replay, _ := ctx.Value(creationReplayKey{}).(bool); identity != nil || replay {
				return verifyConnectionRequirements(ctx, conn, identity)
			}
			return conn, nil
		},
	}
	return &Client{cancel: cancel, httpClient: &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

// Close releases idle connections and pending dials. Construct a new client to
// reconnect after closing; the closed client's lifetime must not be reused.
func (c *Client) Close() {
	c.cancel()
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

func (c *Client) SuggestHTTPPort(ctx context.Context) (protocol.PortSuggestion, error) {
	var result protocol.PortSuggestion
	err := c.do(ctx, http.MethodGet, "/v1/port-suggestion", nil, &result, true)
	return result, err
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

func (c *Client) EnrollPier(ctx context.Context, req protocol.EnrollPierRequest) (protocol.Operation, error) {
	var result protocol.Operation
	if err := c.do(ctx, http.MethodPost, "/v1/piers", req, &result, true); err != nil {
		return result, err
	}
	want := "enroll-pier"
	if req.Relink {
		want = "relink-pier"
	}
	if result.InstanceID != req.ID || result.RequestID != req.RequestID || result.Kind != want {
		return protocol.Operation{}, errors.New("pier enrollment response did not match selection")
	}
	return result, nil
}

func (c *Client) CreateShip(ctx context.Context, req protocol.CreateShipRequest) (protocol.Operation, error) {
	var op protocol.Operation
	if err := c.do(ctx, http.MethodPost, "/v1/ships", req, &op, true); err != nil {
		return op, err
	}
	if op.InstanceID != req.ID || op.RequestID != req.RequestID || op.Kind != "register-ship" {
		return protocol.Operation{}, errors.New("key registration response did not match selection")
	}
	return op, nil
}

func (c *Client) CreateComet(ctx context.Context, req protocol.CreateCometRequest) (protocol.Operation, error) {
	if err := validateID(req.ID); err != nil {
		return protocol.Operation{}, err
	}
	var result protocol.Operation
	if err := c.do(ctx, http.MethodPost, "/v1/comets", req, &result, true); err != nil {
		return protocol.Operation{}, err
	}
	if result.InstanceID != req.ID || result.RequestID != req.RequestID || result.Kind != "register-comet" {
		return protocol.Operation{}, errors.New("comet registration response did not match selection")
	}
	return result, nil
}

func (c *Client) ImportSources(ctx context.Context, runner string) (protocol.ImportCatalogue, error) {
	var result protocol.ImportCatalogue
	if err := c.do(ctx, http.MethodGet, "/v1/imports", nil, &result, true); err != nil {
		return result, err
	}
	if result.RunnerID != runner || result.Sources == nil || len(result.Sources) > 256 {
		return protocol.ImportCatalogue{}, errors.New("invalid import catalogue identity or size")
	}
	return result, nil
}

func (c *Client) ImportFake(ctx context.Context, req protocol.ImportFakeRequest) (protocol.Operation, error) {
	var result protocol.Operation
	if err := validateID(req.ID); err != nil {
		return result, err
	}
	if err := c.do(ctx, http.MethodPost, "/v1/imports", req, &result, true); err != nil {
		return result, err
	}
	if result.InstanceID != req.ID || result.RequestID != req.RequestID || result.Kind != "import-fake" {
		return protocol.Operation{}, errors.New("import response does not match the submitted request")
	}
	return result, nil
}

func (c *Client) LoginCode(ctx context.Context, runner, id string, req protocol.LoginCodeRequest) (protocol.LoginCode, error) {
	var result protocol.LoginCode
	if validateID(id) != nil || req.ExpectedRevision == 0 {
		return result, errors.New("login-code selection is invalid")
	}
	if err := c.do(ctx, http.MethodPost, "/v1/instances/"+id+"/login-code", req, &result, true); err != nil {
		// Even malformed or hostile secret responses must not enter diagnostics.
		return protocol.LoginCode{}, errors.New("login-code request failed; verify operator access and the selected live runtime")
	}
	if result.RunnerID != runner || result.InstanceID != id || result.ConfigRevision != req.ExpectedRevision || result.ContainerID != req.ContainerID || !protocol.ValidLoginCode(result.Code) {
		return protocol.LoginCode{}, errors.New("login-code response did not match the selected runtime")
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

func (c *Client) Routine(ctx context.Context, runner, id string) (protocol.RoutineResponse, error) {
	var result protocol.RoutineResponse
	if err := validateID(id); err != nil {
		return result, err
	}
	if err := c.do(ctx, http.MethodGet, "/v1/instances/"+id+"/routine", nil, &result, true); err != nil {
		return result, err
	}
	if result.RunnerID != runner || result.InstanceID != id {
		return protocol.RoutineResponse{}, errors.New("routine response selection mismatch")
	}
	return result, nil
}

func (c *Client) PutRoutine(ctx context.Context, id string, req routines.Update) (routines.Policy, error) {
	var result routines.Policy
	if err := validateID(id); err != nil {
		return result, err
	}
	if err := c.do(ctx, http.MethodPut, "/v1/instances/"+id+"/routine", req, &result, true); err != nil {
		return result, err
	}
	if result.InstanceID != id || result.InstanceRevision != req.ExpectedInstanceRevision || result.Revision != req.ExpectedRevision+1 {
		return routines.Policy{}, errors.New("routine update response selection mismatch")
	}
	return result, nil
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
	if replay, _ := ctx.Value(creationReplayKey{}).(bool); replay && method == http.MethodPost && (path == "/v1/instances" || path == "/v1/comets" || path == "/v1/ships") {
		req.Header.Set(protocol.CreationReplayHeader, "true")
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
