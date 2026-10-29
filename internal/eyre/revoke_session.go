package eyre

import (
	"context"
	"errors"
	"time"
)

// RevokeStoredSession is only for cleanup of an exact retained credential. It
// permits sending a locally expired token to its verified origin for logout;
// normal application requests still enforce expiry. Rejection is separate from
// observed revocation. An unavailable/wrong-identity origin is never success.
// The supplied byte buffer is cleared, as in RestoreSession.
func RevokeStoredSession(ctx context.Context, origin, ship string, data []byte) (string, error) {
	defer clear(data)
	c, err := New(origin, ship)
	if err != nil {
		return "", err
	}
	defer c.Close()
	if err := c.restoreSession(data, true); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.Probe(ctx); err != nil {
		return "", err
	}
	resp, err := c.requestWithSession(ctx, "POST", "/~/logout", nil, true, nil, true)
	if errors.Is(err, ErrAuth) {
		return "rejected", nil
	}
	if err != nil {
		return "", err
	}
	body, err := readResponse(resp)
	clear(body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 303 {
		return "", ErrUncertain
	}
	return "revoked", nil
}
