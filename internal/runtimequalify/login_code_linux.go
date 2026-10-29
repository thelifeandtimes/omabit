//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ReadLoginCode discovers the selected pier's private Eyre endpoint and checks
// its identity before asking for a code. No conn.sock, proxy or redirect is used.
func ReadLoginCode(ctx context.Context, state ManagedState, expectedContainer string) (string, error) {
	return readLoginCode(ctx, state, expectedContainer, defaults())
}

func readLoginCode(ctx context.Context, state ManagedState, expectedContainer string, d dependencies) (string, error) {
	unavailable := errors.New("login code unavailable for the selected registered runtime")
	if !state.Prepared || !state.Initialized || state.UncertainMaintenance() || state.UncertainProvisioning() || !containerIDPattern.MatchString(expectedContainer) {
		return "", codeReadFailure(ctx, "selection")
	}
	ctx, cancel := context.WithTimeout(ctx, protocol.LoginCodeReadTimeout)
	defer cancel()
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return "", codeReadFailure(ctx, "session")
	}
	if err := q.verifyEngine(ctx); err != nil {
		return "", codeReadFailure(ctx, "engine")
	}
	index := -1
	for i, record := range q.r.Containers {
		if record.Removed {
			continue
		}
		if index >= 0 || record.ID != expectedContainer || record.CreateUncertain || !isNetworkedKind(record.Kind) {
			return "", codeReadFailure(ctx, "container")
		}
		index = i
	}
	if index < 0 {
		return "", codeReadFailure(ctx, "container")
	}
	c, err := q.inspect(ctx, index)
	if err != nil || !*c.State.Running || c.State.Paused || c.State.Restarting || c.State.Dead {
		return "", codeReadFailure(ctx, "container")
	}
	ports, err := q.readHTTPPorts()
	if err != nil {
		return "", codeReadFailure(ctx, "port-inventory")
	}
	port, err := privateHTTPPort(ports, 8080)
	if err != nil {
		return "", codeReadFailure(ctx, "port-inventory")
	}
	endpoint := "http://127.0.0.1:" + strconv.Itoa(int(port))
	read := func(formula string) (string, error) {
		// Formula is selected internally, never caller input. The URL can only
		// name the validated unprivileged loopback port, without a path/query.
		payload := `{"source":{"dojo":"` + formula + `"},"sink":{"stdout":null}}`
		out, err := q.control(ctx, index, 3*time.Second, "exec", "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--show-error", "--fail", "--max-time", "2", "--max-filesize", "1024", "--header", "Content-Type: application/json", "--data-binary", payload, endpoint)
		var value string
		if err != nil || len(out) > 1024 || json.Unmarshal([]byte(out), &value) != nil {
			return "", unavailable
		}
		return strings.TrimSpace(value), nil
	}
	ship := "~zod" // Historical fake records omit the explicit fake descriptor.
	if state.Fake != nil {
		ship = state.Fake.Ship
	} else if state.Comet != nil {
		ship = state.Comet.Ship
	}
	identity, err := read("our")
	if err != nil || identity != ship {
		return "", codeReadFailure(ctx, "identity-before")
	}
	if current, err := q.readHTTPPorts(); err != nil || current != ports {
		return "", codeReadFailure(ctx, "inventory-before-code")
	}
	code, err := read("+code")
	if err != nil || !protocol.ValidLoginCode(code) {
		return "", codeReadFailure(ctx, "read-code")
	}
	// Discard a response if the pier inventory or endpoint identity changed
	// during retrieval. A lost/replaced runtime must never become another ship.
	if current, err := q.readHTTPPorts(); err != nil || current != ports {
		return "", codeReadFailure(ctx, "inventory-after-code")
	}
	identity, err = read("our")
	if err != nil || identity != ship {
		return "", codeReadFailure(ctx, "identity-after")
	}
	return code, nil
}
