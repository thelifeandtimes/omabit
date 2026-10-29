package eyre

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

var ErrIndependentSessionNotIssued = errors.New("independent session login was not attempted")

var ErrIndependentSessionUnavailable = errors.New("ship does not support assisted session creation; use the plugin's URL and +code login")

// NewIndependentSession creates a separate ordinary owner session after the
// caller has obtained application-connection approval. It does not create a
// desk-scoped capability or provide an approval interface.
//
// The fixed base/code thread supplies the login code transiently; neither the
// code nor the parent session is handed to a plugin. The returned client's
// session may be passed privately to that plugin's credential adapter, after
// which it connects directly without this client or the collector running.
// The caller owns handoff, persistence and cleanup of the new session.
func (c *Client) NewIndependentSession(ctx context.Context) (result *Client, err error) {
	attemptedLogin := false
	defer func() {
		if err != nil && !attemptedLogin {
			err = errors.Join(ErrIndependentSessionNotIssued, err)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := c.Check(ctx); err != nil {
		return nil, err
	}
	resp, err := c.request(ctx, "POST", "/spider/base/json/code/ship", []byte("null"), true, http.Header{
		"Content-Type": {"application/json"}, "Accept": {"application/json"},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 405 || resp.StatusCode == 501 {
		return nil, ErrIndependentSessionUnavailable
	}
	media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode != 200 || media != "application/json" {
		return nil, ErrProtocol
	}
	// This response contains a credential. Bound it tightly and never include
	// its body (including a Spider error stack) in an error or public result.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 257))
	defer clear(data)
	if err != nil || len(data) > 256 {
		return nil, ErrProtocol
	}
	var code string
	if json.Unmarshal(data, &code) != nil {
		return nil, ErrProtocol
	}
	code = strings.TrimPrefix(code, "~") // %ship's JSON mark includes the sigil.
	if !codePattern.MatchString(code) {
		return nil, ErrProtocol
	}
	child, err := New(c.origin, c.ship)
	if err != nil {
		return nil, err
	}
	// Login deliberately sends no parent cookie. Eyre closes the supplied
	// session when replacing it, so reusing an authenticated jar would log the
	// original client out. Login clears its temporary code/body byte buffers.
	attemptedLogin = true
	if err := child.Login(ctx, []byte(code)); err != nil {
		child.Close()
		return nil, err
	}
	if child.session.Value == c.session.Value {
		// Never revoke a reused token: it is still the parent's session.
		child.Close()
		return nil, ErrProtocol
	}
	return child, nil
}
