//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ReadLoginCode uses Eyre's container-private loopback endpoint. The conn.sock
// control plane stays disabled. The formula, URL and curl options are fixed.
func ReadLoginCode(ctx context.Context, state ManagedState, expectedContainer string) (string, error) {
	return readLoginCode(ctx, state, expectedContainer, defaults())
}

func readLoginCode(ctx context.Context, state ManagedState, expectedContainer string, d dependencies) (string, error) {
	unavailable := errors.New("login code unavailable for the selected registered runtime")
	if !state.Prepared || !state.Initialized || state.UncertainMaintenance() || state.UncertainProvisioning() || !containerIDPattern.MatchString(expectedContainer) {
		return "", unavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return "", unavailable
	}
	if err := q.verifyEngine(ctx); err != nil {
		return "", unavailable
	}
	index := -1
	for i, record := range q.r.Containers {
		if record.Removed {
			continue
		}
		if index >= 0 || record.ID != expectedContainer || record.CreateUncertain || !isNetworkedKind(record.Kind) {
			return "", unavailable
		}
		index = i
	}
	if index < 0 {
		return "", unavailable
	}
	c, err := q.inspect(ctx, index)
	if err != nil || !*c.State.Running || c.State.Paused || c.State.Restarting || c.State.Dead {
		return "", unavailable
	}
	out, err := q.control(ctx, index, 3*time.Second, "exec", "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--show-error", "--fail", "--max-time", "2", "--max-filesize", "1024", "--header", "Content-Type: application/json", "--data-binary", `{"source":{"dojo":"+code"},"sink":{"stdout":null}}`, "http://127.0.0.1:12321")
	if err != nil || len(out) > 1024 {
		return "", unavailable
	}
	var code string
	if json.Unmarshal([]byte(out), &code) != nil {
		return "", unavailable
	}
	code = strings.TrimSpace(code)
	if !protocol.ValidLoginCode(code) {
		return "", unavailable
	}
	return code, nil
}
