package eyre

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var ErrReconcile = errors.New("Eyre channel was lost; reconcile application state before starting a new subscription")
var ErrSubscription = errors.New("subscription was rejected or ended by the agent")
var ErrCleanup = errors.New("channel cleanup was not confirmed; the owned channel may remain until Eyre expires it")

type Event struct {
	Kind     string          `json:"kind"`
	EventID  uint64          `json:"event_id"`
	Response string          `json:"response,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type wireEvent struct {
	ID       uint64          `json:"id"`
	Response string          `json:"response"`
	OK       string          `json:"ok"`
	Error    json.RawMessage `json:"err"`
	JSON     json.RawMessage `json:"json"`
}

func (c *Client) channelPut(ctx context.Context, p string, actions any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	body, err := json.Marshal(actions)
	if err != nil {
		return ErrProtocol
	}
	resp, err := c.request(ctx, "PUT", p, body, true, http.Header{"Content-Type": []string{"application/json"}})
	if err != nil {
		return err
	}
	data, err := readResponse(resp)
	clear(data)
	if err != nil {
		return ErrUncertain
	}
	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("Eyre channel request returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Watch owns one channel/subscription. Callback success is the delivery boundary:
// only then do we advance Last-Event-ID and acknowledge. No application pokes are
// replayed. A lost channel requires application-specific snapshot reconciliation.
// maxFacts zero means unlimited. Cleanup uses a separate bounded context.
func (c *Client) Watch(ctx context.Context, app, p string, maxFacts uint, emit func(Event) error) (err error) {
	if !termPattern.MatchString(app) || !ValidPath(p) {
		return errors.New("invalid subscription app or path")
	}
	if err := c.Check(ctx); err != nil {
		return err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	channel := "/~/channel/omabit-" + hex.EncodeToString(nonce[:])
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := c.channelPut(cleanup, channel, []any{map[string]any{"action": "delete"}}); e != nil {
			if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				err = ErrCleanup
			}
		}
	}()
	if err := c.channelPut(ctx, channel, []any{map[string]any{"id": 1, "action": "subscribe", "ship": strings.TrimPrefix(c.ship, "~"), "app": app, "path": p}}); err != nil {
		return err
	}
	var last uint64
	var haveLast bool
	var facts uint
	attempts := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		streamCtx, cancel := context.WithCancel(ctx)
		headers := http.Header{"Accept": []string{"text/event-stream"}}
		if haveLast {
			headers.Set("Last-Event-ID", strconv.FormatUint(last, 10))
		}
		resp, err := c.request(streamCtx, "GET", channel, nil, true, headers)
		if err == nil {
			if resp.StatusCode == 404 {
				resp.Body.Close()
				cancel()
				return ErrReconcile
			}
			media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
			if resp.StatusCode != 200 || media != "text/event-stream" {
				resp.Body.Close()
				cancel()
				return fmt.Errorf("%w: expected HTTP 200 text/event-stream", ErrProtocol)
			}
			if err = emit(Event{Kind: "connected"}); err != nil {
				resp.Body.Close()
				cancel()
				return err
			}
			// Eyre sends a heartbeat every 20 seconds. Reconnect a stalled stream
			// rather than letting it block indefinitely after a laptop/network outage.
			var idle atomic.Bool
			timer := time.AfterFunc(45*time.Second, func() { idle.Store(true); cancel() })
			reader := &heartbeatReader{Reader: resp.Body, timer: timer}
			before := last
			hadBefore := haveLast
			err = readEvents(reader, func(id uint64, data []byte) error {
				if haveLast && id <= last {
					return nil
				}
				if haveLast && id-last != 1 {
					return ErrReconcile
				}
				var event wireEvent
				if json.Unmarshal(data, &event) != nil || event.ID != 1 {
					return fmt.Errorf("%w: invalid event JSON or subscription ID", ErrProtocol)
				}
				if len(event.Error) > 0 && string(event.Error) != "null" {
					return ErrSubscription
				}
				if event.Response == "quit" {
					return ErrSubscription
				}
				if event.Response != "subscribe" && event.Response != "diff" {
					return fmt.Errorf("%w: unexpected subscription response kind", ErrProtocol)
				}
				if event.Response == "subscribe" && event.OK != "ok" {
					return fmt.Errorf("%w: invalid subscribe acknowledgement", ErrProtocol)
				}
				if event.Response == "diff" && len(event.JSON) == 0 {
					return fmt.Errorf("%w: diff has no JSON payload", ErrProtocol)
				}
				if err := emit(Event{Kind: "event", EventID: id, Response: event.Response, Data: event.JSON}); err != nil {
					return &deliveryError{err}
				}
				last = id
				haveLast = true
				if event.Response == "diff" {
					facts++
				}
				if err := c.channelPut(ctx, channel, []any{map[string]any{"action": "ack", "event-id": last}}); err != nil {
					return err
				}
				if maxFacts > 0 && facts >= maxFacts {
					return errEnough
				}
				return nil
			})
			timer.Stop()
			resp.Body.Close()
			cancel()
			if errors.Is(err, errEnough) {
				return nil
			}
			var delivery *deliveryError
			if errors.As(err, &delivery) {
				return delivery.err
			}
			if errors.Is(err, ErrProtocol) || errors.Is(err, ErrSubscription) || errors.Is(err, ErrAuth) || errors.Is(err, ErrReconcile) {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if last > before || (!hadBefore && haveLast) {
				attempts = 0
			}
			if idle.Load() {
				err = ErrTransport
			}
		} else {
			cancel()
		}
		if errors.Is(err, ErrAuth) || errors.Is(err, ErrRedirect) {
			return err
		}
		if attempts >= 3 {
			return errors.New("Eyre stream reconnect attempts exhausted")
		}
		if err := emit(Event{Kind: "reconnecting", EventID: last}); err != nil {
			return err
		}
		delay := []time.Duration{250 * time.Millisecond, time.Second, 3 * time.Second}[attempts]
		attempts++
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

var errEnough = errors.New("enough facts")

type deliveryError struct{ err error }

func (e *deliveryError) Error() string { return "subscription consumer failed" }

type heartbeatReader struct {
	io.Reader
	timer *time.Timer
}

func (r *heartbeatReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.timer.Reset(45 * time.Second)
	}
	return n, err
}

// readEvents bounds both lines and complete events, ignores heartbeat comments,
// joins multiline data, and never delivers an incomplete final frame.
func readEvents(input io.Reader, emit func(uint64, []byte) error) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxBody+1)
	var data strings.Builder
	var id uint64
	var seenID bool
	frameBytes := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if data.Len() > 0 {
				if !seenID {
					return fmt.Errorf("%w: missing SSE event ID", ErrProtocol)
				}
				if err := emit(id, []byte(strings.TrimSuffix(data.String(), "\n"))); err != nil {
					return err
				}
			}
			data.Reset()
			seenID = false
			id = 0
			frameBytes = 0
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		frameBytes += len(line)
		if frameBytes > maxBody {
			return ErrProtocol
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			n, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ErrProtocol
			}
			id = n
			seenID = true
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
		case "event":
			if value != "" && value != "message" {
				return ErrProtocol
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}
