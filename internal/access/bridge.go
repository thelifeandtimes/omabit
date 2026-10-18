package access

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Bridge terminates the SSH byte stream and constructs new local requests. Actor
// headers from the client are never forwarded. The grant is selected by sshd's
// forced command and independently revalidated by the managed runner.
// Blocking streams must implement io.Closer with Close interrupting pending I/O.
// The SSH session owns both streams; cancellation closes both, including stdout.
func Bridge(ctx context.Context, input io.Reader, output io.Writer, socket, keys, id string) error {
	return bridge(ctx, input, output, socket, keys, id, 5*time.Second)
}

type bridgeIdentity struct {
	runner, mode string
	bound        bool
}

func (pin *bridgeIdentity) check(hello protocol.Hello) error {
	if !protocol.ValidID(hello.RunnerID) || hello.Mode == "" || pin.bound && (pin.runner != hello.RunnerID || pin.mode != hello.Mode) {
		return errors.New("local runner identity or mode changed; request was not forwarded")
	}
	pin.runner, pin.mode, pin.bound = hello.RunnerID, hello.Mode, true
	return nil
}

func bridge(ctx context.Context, input io.Reader, output io.Writer, socket, keys, id string, timeout time.Duration) error {
	interrupt := func() {
		if c, ok := input.(io.Closer); ok {
			c.Close()
		}
		if c, ok := output.(io.Closer); ok {
			c.Close()
		}
	}
	stop := context.AfterFunc(ctx, interrupt)
	defer stop()
	var pin bridgeIdentity
	// Each request, including headers and body, is bounded. The same buffered
	// reader is retained so HTTP pipelining cannot lose or reinterpret bytes.
	limited := &io.LimitedReader{R: input, N: 16384}
	reader := bufio.NewReader(limited)
	for ctx.Err() == nil {
		limited.N = 16384
		// Healthy watchers may remain idle indefinitely. Start the progress bound
		// only once framing begins, including bytes buffered by the previous RPC.
		_, err := reader.Peek(1)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return errors.New("invalid SSH RPC request")
		}
		close, err := bridgeRequest(ctx, reader, limited, output, socket, keys, id, &pin, timeout, interrupt)
		if err != nil || close {
			return err
		}
	}
	return ctx.Err()
}

func bridgeRequest(ctx context.Context, reader *bufio.Reader, limited *io.LimitedReader, output io.Writer, socket, keys, id string, pin *bridgeIdentity, timeout time.Duration, interrupt func()) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(timeout, cancel)
	stop := context.AfterFunc(ctx, interrupt)
	defer func() {
		timer.Stop()
		stop()
		cancel()
	}()
	req, err := http.ReadRequest(reader)
	if err != nil {
		return true, errors.New("invalid SSH RPC request")
	}
	upload := req.Method == http.MethodPost && req.URL.Path == protocol.PillUploadPath
	if req.URL.IsAbs() || req.URL.RawQuery != "" || req.Header.Get("Origin") != "" {
		return true, errors.New("invalid SSH RPC request")
	}
	replayOnly, err := protocol.CreationReplayOnly(req.Header)
	if err != nil {
		return true, err
	}
	if replayOnly && (req.Method != http.MethodPost || req.URL.Path != "/v1/instances" && req.URL.Path != "/v1/comets" && req.URL.Path != "/v1/ships") {
		return true, errors.New("creation-replay-only header requires a creation endpoint")
	}
	grant, err := Load(keys, id)
	if err != nil || !grant.Allows(req.Method, req.URL.Path) {
		data, _ := json.Marshal(protocol.ErrorResponse{Error: protocol.APIError{Code: "ssh_access_denied", Message: "SSH grant does not permit this request"}})
		resp := &http.Response{StatusCode: 403, ProtoMajor: 1, ProtoMinor: 1, Header: http.Header{protocol.Header: []string{"1"}, "Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Close: true}
		return true, resp.Write(output)
	}
	if protocol.HandoffDataPath(req.URL.Path) {
		if req.Method == "PUT" {
			if req.ContentLength <= 0 || req.ContentLength > protocol.MaxHandoffBytes || req.Header.Get("Content-Type") != "application/octet-stream" {
				return true, errors.New("invalid handoff framing")
			}
			limited.N = req.ContentLength
		} else if req.Method != "GET" || req.ContentLength != 0 {
			return true, errors.New("invalid handoff method/body")
		}
		timer.Reset(30 * time.Minute)
		return true, forwardHandoff(ctx, req, output, socket, grant, pin)
	}
	var body io.Reader
	if upload {
		meta := protocol.BootPill{SHA256: req.Header.Get(protocol.PillHashHeader), Bytes: req.ContentLength}
		if !protocol.ValidBootPill(&meta) || req.Header.Get("Content-Type") != "application/octet-stream" {
			return true, errors.New("invalid pill upload framing")
		}
		timer.Reset(10 * time.Minute)
		limited.N = req.ContentLength
		body = io.LimitReader(req.Body, req.ContentLength)
	} else {
		data, err := io.ReadAll(io.LimitReader(req.Body, 4097))
		if err != nil || len(data) > 4096 {
			return true, errors.New("invalid SSH RPC body")
		}
		req.Body.Close()
		body = bytes.NewReader(data)
	}
	local, err := http.NewRequestWithContext(ctx, req.Method, "http://runner"+req.URL.EscapedPath(), body)
	if err != nil {
		return true, errors.New("invalid SSH RPC path")
	}
	local.Header.Set("Content-Type", req.Header.Get("Content-Type"))
	local.Header.Set(protocol.Header, req.Header.Get(protocol.Header))
	local.Header.Set("Accept", "application/json")
	local.Header.Set(Header, id)
	if replayOnly {
		local.Header.Set(protocol.CreationReplayHeader, "true")
	}
	if upload {
		local.ContentLength = req.ContentLength
		local.Header.Set(protocol.PillHashHeader, req.Header.Get(protocol.PillHashHeader))
	}
	local.GetBody = nil // a bridge never retries a mutation either
	resp, data, err := forward(ctx, local, socket, grant, pin)
	if err != nil {
		return true, err
	}
	if upload {
		req.Body.Close()
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	resp.TransferEncoding = nil
	// Pass only protocol metadata, never local response cookies or credentials.
	resp.Header = http.Header{protocol.Header: []string{resp.Header.Get(protocol.Header)}, "Content-Type": []string{"application/json"}, "Cache-Control": []string{"no-store"}}
	resp.Request = &http.Request{Method: req.Method, URL: &url.URL{Path: req.URL.Path}}
	resp.Close = req.Close
	if err := resp.Write(output); err != nil {
		return true, err
	}
	if req.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
		return true, nil
	}
	return false, nil
}

// Authenticate delegation on the SAME local connection that will receive the
// request. An older runner ignores the grant header; checking its response after
// a mutation would already be too late. There are no transport retries.
func forward(ctx context.Context, req *http.Request, socket string, grant Grant, pin *bridgeIdentity) (*http.Response, []byte, error) {
	duration := 5 * time.Second
	if req.URL.Path == protocol.PillUploadPath {
		duration = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, nil, errors.New("local runner unavailable")
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, nil, err
	}
	limited := &io.LimitedReader{R: conn, N: (1 << 20) + 8192}
	reader := bufio.NewReader(limited)
	read := func(request *http.Request) (*http.Response, []byte, error) {
		limited.N = (1 << 20) + 8192
		resp, err := http.ReadResponse(reader, request)
		if err != nil {
			return nil, nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		if err != nil || len(body) > 1<<20 {
			conn.Close()
			resp.Body.Close()
			return nil, nil, errors.New("invalid runner response")
		}
		resp.Body.Close()
		return resp, body, nil
	}
	helloRequest, _ := http.NewRequest(http.MethodGet, "http://runner/hello", nil)
	helloRequest.Header.Set(Header, grant.Fingerprint)
	if err := helloRequest.Write(conn); err != nil {
		return nil, nil, errors.New("local runner handshake failed")
	}
	helloResponse, data, err := read(helloRequest)
	var hello protocol.Hello
	if err != nil || helloResponse.StatusCode != 200 || helloResponse.Close || json.Unmarshal(data, &hello) != nil || hello.ProtocolVersion != protocol.Version || hello.Actor != "ssh-key:"+grant.Fingerprint || hello.AccessRole != grant.Role || !slices.Contains(hello.Features, "ssh-key-grants") {
		return nil, nil, errors.New("local runner does not verify this SSH grant; request was not forwarded")
	}
	if err := pin.check(hello); err != nil {
		return nil, nil, err
	}
	replayOnly, err := protocol.CreationReplayOnly(req.Header)
	if err != nil {
		return nil, nil, err
	}
	if replayOnly && !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
		return nil, nil, errors.New("local runner no longer supports creation-replay-only; request was not forwarded")
	}
	req.Close = true
	if err := req.Write(conn); err != nil {
		return nil, nil, errors.New("local runner write failed; mutation outcome may be uncertain")
	}
	resp, data, err := read(req)
	if err != nil {
		return nil, nil, errors.New("local runner response failed; mutation outcome may be uncertain")
	}
	return resp, data, nil
}
