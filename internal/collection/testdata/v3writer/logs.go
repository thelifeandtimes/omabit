package collection

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// LogUpdate is one bounded watch record, not an append-only log history. Errors
// invalidate the last tail. A zero retry delay on an error requires user review.
type LogUpdate struct {
	RunnerID       string                `json:"runner_id"`
	InstanceID     string                `json:"instance_id"`
	ConfigRevision uint64                `json:"config_revision"`
	Logs           *protocol.RuntimeLogs `json:"logs,omitempty"`
	Error          *protocol.APIError    `json:"error,omitempty"`
	RetryAfterMS   int64                 `json:"retry_after_ms,omitempty"`
}

// WatchLogs keeps one private pinned client for the lifetime of a selection.
// Every reconnect re-verifies identity; every read still checks operator access
// and the server's revision/runtime-generation fences. No server stream or SSH
// multiplexing is needed. Entries are fixed until an explicit watcher restart.
func WatchLogs(ctx context.Context, entry Connection, id string, revision uint64, interval time.Duration, emit func(LogUpdate) error) error {
	if !protocol.ValidID(entry.RunnerID) || !protocol.ValidID(id) || revision == 0 || interval < time.Second {
		return errors.New("log watch requires a pinned selection and an interval of at least 1s")
	}
	var c *client.Client
	defer func() {
		if c != nil {
			c.Close()
		}
	}()
	backoff := 3 * time.Second
	for ctx.Err() == nil {
		if c == nil {
			c = entry.Client(true)
		}
		logs, err := c.RuntimeLogs(ctx, entry.RunnerID, id, revision)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		update := LogUpdate{RunnerID: entry.RunnerID, InstanceID: id, ConfigRevision: revision}
		delay := interval
		terminal := false
		if err == nil {
			update.Logs = &logs
			backoff = 3 * time.Second
		} else {
			problem, retry, reconnect := logProblem(err)
			update.Error = &problem
			terminal = !retry
			if reconnect {
				c.Close()
				c = nil
			}
			if retry {
				backoff = min(30*time.Second, backoff*2)
				// Equal jitter: first retry in 3–6s, then 6–12s, 12–24s,
				// and 15–30s. Success resets it; UI respawns must not.
				delay = backoff/2 + time.Duration(rand.Int64N(int64(backoff/2)))
				update.RetryAfterMS = delay.Milliseconds()
			}
		}
		if err := emit(update); err != nil {
			return err
		}
		if terminal {
			return errors.New(update.Error.Message)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ctx.Err()
}

// Never copy arbitrary remote diagnostics (potentially another ship's data)
// into a watch error. Permanent identity/access/selection failures stop polling.
func logProblem(err error) (protocol.APIError, bool, bool) {
	problem := protocol.APIError{Code: "logs_invalid_response", Message: "Invalid runtime log response; review the selected runner before retrying."}
	if errors.Is(err, client.ErrIdentityMismatch) {
		return protocol.APIError{Code: "runner_identity_mismatch", Message: client.ErrIdentityMismatch.Error()}, false, true
	}
	var remote *client.RemoteError
	if errors.As(err, &remote) {
		switch {
		case remote.Status == 401 || remote.Status == 403:
			problem = protocol.APIError{Code: "logs_access_denied", Message: "Runtime log access was denied; review operator access before retrying."}
		case remote.Status == 409 && remote.Problem.Code == "runtime_changed":
			return protocol.APIError{Code: "runtime_changed", Message: "Runtime changed during the read; waiting for a fresh tail."}, true, false
		case remote.Status == 409:
			problem = protocol.APIError{Code: "revision_conflict", Message: "The selected configuration changed; select a fresh revision before reading logs."}
		case remote.Status == 404:
			problem = protocol.APIError{Code: "logs_not_found", Message: "The selected ship is no longer available."}
		case remote.Status == 408 || remote.Status == 429 || remote.Status == 500 || remote.Status == 502 || remote.Status == 503 || remote.Status == 504:
			return protocol.APIError{Code: "logs_unavailable", Message: "Runtime logs are temporarily unavailable; retrying with backoff."}, true, false
		default:
			problem = protocol.APIError{Code: "logs_request_rejected", Message: "Runner rejected the log request; review its capabilities before retrying."}
		}
		return problem, false, true
	}
	var network net.Error
	if errors.As(err, &network) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return protocol.APIError{Code: "connection_unavailable", Message: "Log connection unavailable; reconnecting with backoff."}, true, true
	}
	return problem, false, true
}
