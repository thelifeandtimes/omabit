package eyre

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// GrubberyResponse contains private native data. Body is deliberately omitted
// from JSON: a facade must explicitly select bounded, sanitized observations.
// An HTTP acknowledgement alone proves neither ownership nor loaded app health.
type GrubberyResponse struct {
	SessionCleanup string `json:"session_cleanup,omitempty"`
	Attempted      bool   `json:"attempted"`
	Status         int    `json:"status"`
	ContentType    string `json:"content_type"`
	Body           []byte `json:"-"`
	BodySHA256     string `json:"body_sha256"`
}

// GrubberyStep constructs one exact original phase from validated actual bytes.
// Caller owns role/policy/runtime/ownership admission and durable dispatch before
// entry. There is no generic URL/header/method, retry or effect permission here.
func (c *Client) GrubberyStep(ctx context.Context, p grubbery.Plan, stepID string, files []deskwork.SourceFile) (GrubberyResponse, error) {
	if err := grubbery.CheckNativeSource(p.Source); err != nil {
		return GrubberyResponse{}, err
	}
	request, err := p.Request(stepID, files)
	if err != nil {
		return GrubberyResponse{}, err
	}
	defer clear(request.Body)
	return c.grubberyRequest(ctx, request)
}

// GrubberyRead accepts only constructor-produced bound reads. It requires no
// source cache and performs no app write.404 is an explicit absence observation,
// never proof that an existing or uncertain namespace is safe to adopt.
func (c *Client) GrubberyRead(ctx context.Context, read grubbery.ReadRequest) (GrubberyResponse, error) {
	request, err := read.Request()
	if err != nil {
		return GrubberyResponse{}, err
	}
	return c.grubberyRequest(ctx, request)
}

func (c *Client) grubberyRequest(ctx context.Context, q grubbery.Request) (GrubberyResponse, error) {
	var out GrubberyResponse
	if c == nil {
		return out, ErrAuth
	}
	if err := c.Check(ctx); err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	uri := c.origin + q.Path
	if q.Query != "" {
		uri += "?" + q.Query
	}
	req, err := http.NewRequestWithContext(ctx, q.Method, uri, bytes.NewReader(q.Body))
	if err != nil {
		return out, ErrProtocol
	}
	req.GetBody = nil // mutations are never replayable by net/http
	req.AddCookie(&http.Cookie{Name: c.session.Name, Value: c.session.Value})
	if q.ContentType != "" {
		req.Header.Set("Content-Type", q.ContentType)
	}
	out.Attempted = true
	response, err := c.http.Do(req)
	if err != nil {
		if q.Method == "GET" {
			return out, ErrTransport
		}
		return out, ErrUncertain
	}
	defer response.Body.Close()
	out.Status = response.StatusCode
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil || len(body) > maxBody {
		clear(body)
		if q.Method == "GET" {
			return out, ErrProtocol
		}
		return out, ErrUncertain
	}
	sum := sha256.Sum256(body)
	out.BodySHA256 = hex.EncodeToString(sum[:])
	out.Body = body
	content := response.Header.Get("Content-Type")
	if len(content) <= 256 && !strings.ContainsAny(content, "\r\n\x00") {
		out.ContentType = content
	}
	if out.Status >= 300 && out.Status < 400 {
		return out, ErrRedirect
	}
	if q.Method == "GET" {
		if out.Status == 401 || out.Status == 403 {
			return out, ErrAuth
		}
		if out.Status == 200 || out.Status == 404 {
			return out, nil
		}
		return out, ErrProtocol
	}
	// These exact pinned API statuses are emitted before its selected mutation.
	// Unknown transport, body truncation and5xx can never become refusal evidence.
	switch out.Status {
	case 400, 401, 403, 404, 409:
		return out, grubbery.ErrNativeRejected
	}
	if q.Method == "PUT" && out.Status == 201 && string(body) == "Created" || q.Method == "POST" && out.Status == 200 && string(body) == "OK" {
		return out, nil
	}
	return out, ErrUncertain
}
