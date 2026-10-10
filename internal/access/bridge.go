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
func Bridge(ctx context.Context, input io.Reader, output io.Writer, socket, keys, id string) error {
	// Each request, including headers and body, is bounded. The same buffered
	// reader is retained so HTTP pipelining cannot lose or reinterpret bytes.
	limited := &io.LimitedReader{R: input, N: 16384}
	reader := bufio.NewReader(limited)
	for ctx.Err() == nil {
		limited.N = 16384
		req, err := http.ReadRequest(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return errors.New("invalid SSH RPC request")
		}
		upload := req.Method == http.MethodPost && req.URL.Path == protocol.PillUploadPath
		if req.URL.IsAbs() || req.URL.RawQuery != "" || req.Header.Get("Origin") != "" {
			return errors.New("invalid SSH RPC request")
		}
		grant, err := Load(keys, id)
		if err != nil || !grant.Allows(req.Method, req.URL.Path) {
			data, _ := json.Marshal(protocol.ErrorResponse{Error: protocol.APIError{Code: "ssh_access_denied", Message: "SSH grant does not permit this request"}})
			resp := &http.Response{StatusCode: 403, ProtoMajor: 1, ProtoMinor: 1, Header: http.Header{protocol.Header: []string{"1"}, "Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Close: true}
			return resp.Write(output)
		}
		if protocol.HandoffDataPath(req.URL.Path) {
			if req.Method == "PUT" {
				if req.ContentLength <= 0 || req.ContentLength > protocol.MaxHandoffBytes || req.Header.Get("Content-Type") != "application/octet-stream" {
					return errors.New("invalid handoff framing")
				}
				limited.N = req.ContentLength
			} else if req.Method != "GET" || req.ContentLength != 0 {
				return errors.New("invalid handoff method/body")
			}
			return forwardHandoff(ctx, req, input, output, socket, grant)
		}
		var body io.Reader
		if upload {
			meta := protocol.BootPill{SHA256: req.Header.Get(protocol.PillHashHeader), Bytes: req.ContentLength}
			if !protocol.ValidBootPill(&meta) || req.Header.Get("Content-Type") != "application/octet-stream" {
				return errors.New("invalid pill upload framing")
			}
			limited.N = req.ContentLength
			body = io.LimitReader(req.Body, req.ContentLength)
		} else {
			data, err := io.ReadAll(io.LimitReader(req.Body, 4097))
			if err != nil || len(data) > 4096 {
				return errors.New("invalid SSH RPC body")
			}
			req.Body.Close()
			body = bytes.NewReader(data)
		}
		local, err := http.NewRequestWithContext(ctx, req.Method, "http://runner"+req.URL.EscapedPath(), body)
		if err != nil {
			return errors.New("invalid SSH RPC path")
		}
		local.Header.Set("Content-Type", req.Header.Get("Content-Type"))
		local.Header.Set(protocol.Header, req.Header.Get(protocol.Header))
		local.Header.Set("Accept", "application/json")
		local.Header.Set(Header, id)
		if upload {
			local.ContentLength = req.ContentLength
			local.Header.Set(protocol.PillHashHeader, req.Header.Get(protocol.PillHashHeader))
		}
		local.GetBody = nil // a bridge never retries a mutation either
		requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		var stop func() bool
		if upload {
			cancel()
			requestCtx, cancel = context.WithTimeout(ctx, 10*time.Minute)
			stop = context.AfterFunc(requestCtx, func() {
				if c, ok := input.(io.Closer); ok {
					c.Close()
				}
			})
		}
		resp, data, err := forward(requestCtx, local, socket, grant)
		if stop != nil {
			stop()
		}
		cancel()
		if err != nil {
			return err
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
			return err
		}
		if req.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
			return nil
		}
	}
	return ctx.Err()
}

// Authenticate delegation on the SAME local connection that will receive the
// request. An older runner ignores the grant header; checking its response after
// a mutation would already be too late. There are no transport retries.
func forward(ctx context.Context, req *http.Request, socket string, grant Grant) (*http.Response, []byte, error) {
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
