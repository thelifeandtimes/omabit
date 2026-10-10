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
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// One authenticated bulk exchange per SSH connection. The payload is streamed,
// never buffered in the bridge or interpreted as additional RPC requests.
func forwardHandoff(ctx context.Context, req *http.Request, input io.Reader, output io.Writer, socket string, grant Grant) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	if err != nil {
		return errors.New("local runner unavailable")
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() {
		conn.Close()
		if c, ok := input.(io.Closer); ok {
			c.Close()
		}
	})
	defer stop()
	deadline, _ := ctx.Deadline()
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	helloReq, _ := http.NewRequest("GET", "http://runner/hello", nil)
	helloReq.Header.Set(Header, grant.Fingerprint)
	if err = helloReq.Write(conn); err != nil {
		return errors.New("local handoff handshake failed")
	}
	response, err := http.ReadResponse(reader, helloReq)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	response.Body.Close()
	var hello protocol.Hello
	if err != nil || len(data) > 1<<20 || response.StatusCode != 200 || response.Close || json.Unmarshal(data, &hello) != nil || hello.ProtocolVersion != protocol.Version || hello.Actor != "ssh-key:"+grant.Fingerprint || hello.AccessRole != grant.Role || !slices.Contains(hello.Features, "host-handoff") || !slices.Contains(hello.Features, "ssh-key-grants") {
		return errors.New("runner does not authenticate handoff delegation")
	}
	local, err := http.NewRequestWithContext(ctx, req.Method, "http://runner"+req.URL.EscapedPath(), req.Body)
	if err != nil {
		return err
	}
	local.GetBody = nil
	local.Close = true
	local.ContentLength = req.ContentLength
	local.Header.Set(Header, grant.Fingerprint)
	local.Header.Set(protocol.Header, req.Header.Get(protocol.Header))
	local.Header.Set("Content-Type", req.Header.Get("Content-Type"))
	local.Header.Set(protocol.HandoffHashHeader, req.Header.Get(protocol.HandoffHashHeader))
	if err = local.Write(conn); err != nil {
		return errors.New("handoff stream interrupted; source remains fenced")
	}
	response, err = http.ReadResponse(reader, local)
	if err != nil {
		return errors.New("handoff response unavailable; inspect transfer status")
	}
	defer response.Body.Close()
	header := http.Header{protocol.Header: []string{response.Header.Get(protocol.Header)}, "Cache-Control": []string{"no-store"}, "Content-Type": []string{"application/json"}}
	if req.Method == "GET" && response.StatusCode == 200 {
		if response.ContentLength <= 0 || response.ContentLength > protocol.MaxHandoffBytes || response.Header.Get("Content-Type") != "application/octet-stream" {
			return errors.New("invalid handoff response framing")
		}
		header.Set("Content-Type", "application/octet-stream")
		header.Set(protocol.HandoffHashHeader, response.Header.Get(protocol.HandoffHashHeader))
	} else {
		data, err = io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		if err != nil || len(data) > 1<<20 {
			return errors.New("invalid handoff acknowledgement")
		}
		response.Body = io.NopCloser(bytes.NewReader(data))
		response.ContentLength = int64(len(data))
		response.TransferEncoding = nil
	}
	response.Header = header
	response.Close = true
	return response.Write(output)
}
