package eyre

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"
)

var ErrNack = errors.New("the Gall agent rejected the poke")

type PokeResult struct {
	Acknowledged     bool
	CleanupConfirmed bool
}

// Poke submits once and waits for Gall's acknowledgement, not just HTTP acceptance.
// A missing acknowledgement is uncertain and never triggers automatic resubmission.
func (c *Client) Poke(ctx context.Context, app, mark string, payload json.RawMessage) (result PokeResult, err error) {
	if !termPattern.MatchString(app) || !termPattern.MatchString(mark) || len(payload) > maxBody/2 || !json.Valid(payload) {
		return result, errors.New("invalid app, mark, or bounded JSON poke")
	}
	if err := c.Check(ctx); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return result, err
	}
	channel := "/~/channel/omabit-" + hex.EncodeToString(nonce[:])
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		result.CleanupConfirmed = c.channelPut(cleanup, channel, []any{map[string]any{"action": "delete"}}) == nil
	}()
	if err := c.channelPut(ctx, channel, []any{map[string]any{"id": 1, "action": "poke", "ship": strings.TrimPrefix(c.ship, "~"), "app": app, "mark": mark, "json": payload}}); err != nil {
		return result, err
	}
	resp, err := c.request(ctx, "GET", channel, nil, true, http.Header{"Accept": []string{"text/event-stream"}})
	if err != nil {
		return result, ErrUncertain
	}
	defer resp.Body.Close()
	media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode != 200 || media != "text/event-stream" {
		return result, ErrUncertain
	}
	acked := errors.New("acknowledged")
	err = readEvents(resp.Body, func(id uint64, data []byte) error {
		var event wireEvent
		if json.Unmarshal(data, &event) != nil || event.ID != 1 || event.Response != "poke" {
			return ErrProtocol
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return ErrNack
		}
		if event.OK != "ok" {
			return ErrProtocol
		}
		result.Acknowledged = true
		// The acknowledgement has been received even if acknowledging the SSE
		// frame fails. Channel deletion below disposes of the retained frame.
		_ = c.channelPut(ctx, channel, []any{map[string]any{"action": "ack", "event-id": id}})
		return acked
	})
	if errors.Is(err, acked) {
		return result, nil
	}
	if errors.Is(err, ErrNack) {
		return result, ErrNack
	}
	return result, ErrUncertain
}
