package eyre

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrStoredSession = errors.New("saved application session is invalid; preserve it for recovery")

// StoredSessionHealth checks one retained origin-bound session without changing
// it. The caller must first check the current grant, ship binding and network
// eligibility. A health result is neither permission to issue a session nor
// evidence that an old session was revoked. The input buffer is always cleared.
// Malformed responses/state and identity changes return fixed errors; they are
// never treated as a healthy session or proof of remote revocation.
func StoredSessionHealth(ctx context.Context, origin, ship string, data []byte) (string, error) {
	defer clear(data)
	c, err := New(origin, ship)
	if err != nil || c.origin != origin {
		return "", ErrStoredSession
	}
	defer c.Close()
	if !healthSessionJSON(data) {
		return "", ErrStoredSession
	}
	// Validate the full session even when expired. Expiry alone permits an
	// observation, never transmission of the expired token.
	if err = c.restoreSession(data, true); err != nil {
		return "", ErrStoredSession
	}
	if !time.Now().Before(c.session.Expires) {
		return "expired", nil
	}
	ctx, stop := context.WithTimeout(ctx, 6*time.Second)
	defer stop()
	identity, err := c.healthIdentity(ctx, false)
	if err != nil {
		if errors.Is(err, ErrTransport) || errors.Is(err, ErrAuth) {
			return "unreachable", nil
		}
		return "", err
	}
	if identity != ship {
		return "", ErrIdentity // Never send a cookie to the wrong ship.
	}
	identity, err = c.healthIdentity(ctx, true)
	if errors.Is(err, ErrAuth) {
		return "rejected", nil
	}
	if errors.Is(err, ErrTransport) {
		return "unreachable", nil
	}
	if err != nil {
		return "", err
	}
	if identity != ship {
		return "", ErrIdentity
	}
	return "healthy", nil
}

// ValidateStoredSession validates a retained exact-origin credential without
// network IO, including expired credentials still needed for cleanup. It clears
// the input. Missing/invalid state cannot be treated as permission to reconnect.
func ValidateStoredSession(origin, ship string, data []byte) error {
	defer clear(data)
	c, err := New(origin, ship)
	if err != nil || c.origin != origin {
		return ErrStoredSession
	}
	defer c.Close()
	if !healthSessionJSON(data) || c.restoreSession(data, true) != nil {
		return ErrStoredSession
	}
	return nil
}

func (c *Client) healthIdentity(ctx context.Context, auth bool) (string, error) {
	parent := ctx
	path := "/~/host"
	if auth {
		path = "/~/name"
	}
	ctx, stop := context.WithTimeout(ctx, 3*time.Second)
	defer stop()
	response, err := c.request(ctx, http.MethodGet, path, nil, auth, nil)
	if err != nil {
		if parent.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			return "", ErrTransport
		}
		return "", err
	}
	defer response.Body.Close()
	// A gateway or temporarily unavailable service is an unavailable route,
	// not malformed identity evidence. This permits fresh route selection;
	// it never proves this retained session was revoked.
	if response.StatusCode == http.StatusBadGateway || response.StatusCode == http.StatusServiceUnavailable || response.StatusCode == http.StatusGatewayTimeout {
		return "", ErrTransport
	}
	if response.StatusCode != http.StatusOK {
		return "", ErrProtocol
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 257))
	defer clear(data)
	if err != nil || len(data) > 256 {
		return "", ErrProtocol
	}
	identity := strings.TrimSpace(string(data))
	if !ValidShip(identity) {
		return "", ErrProtocol
	}
	return identity, nil
}

func healthSessionJSON(data []byte) bool {
	if len(data) == 0 || len(data) > 16384 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return false
	}
	fields := map[string]bool{"origin": false, "ship": false, "cookie_name": false, "cookie_value": false, "expires": false}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		seen, known := fields[key]
		if err != nil || !ok || !known || seen {
			return false
		}
		var value string
		if d.Decode(&value) != nil || value == "" {
			return false
		}
		fields[key] = true
	}
	if token, err := d.Token(); err != nil || token != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return false
	}
	for _, seen := range fields {
		if !seen {
			return false
		}
	}
	return true
}
