package client

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrHostCompatibilityUnsupported = errors.New("runner does not advertise host-compatibility-v1; update the selected runner for host reports")
var ErrHostCompatibilityUnavailable = errors.New("host compatibility report is unavailable; ship observation remains independent")

// HostCompatibility checks capability and binds the body to the selected identity.
// Enrolled callers must also use a pinned transport, including on reconnects.
// Errors deliberately never carry arbitrary remote bodies or tool diagnostics.
func (c *Client) HostCompatibility(ctx context.Context, expected Identity) (protocol.HostCompatibilityReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var result protocol.HostCompatibilityReport
	hello, err := c.Hello(ctx)
	if err != nil {
		return result, hostCompatibilityError(err)
	}
	if err := expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return result, err
	}
	if !slices.Contains(hello.Features, protocol.HostCompatibilityFeature) {
		return result, ErrHostCompatibilityUnsupported
	}
	if err := c.do(ctx, http.MethodGet, protocol.HostCompatibilityPath, nil, &result, true); err != nil {
		return protocol.HostCompatibilityReport{}, hostCompatibilityError(err)
	}
	if err := expected.Check(result.RunnerID, result.Mode); err != nil {
		return protocol.HostCompatibilityReport{}, err
	}
	if !protocol.ValidHostCompatibility(result) {
		return protocol.HostCompatibilityReport{}, ErrHostCompatibilityUnavailable
	}
	return result, nil
}

func hostCompatibilityError(err error) error {
	for _, known := range []error{ErrIdentityMismatch, context.DeadlineExceeded, context.Canceled} {
		if errors.Is(err, known) {
			return known
		}
	}
	var remote *RemoteError
	if errors.As(err, &remote) && remote.Status == http.StatusServiceUnavailable && remote.Problem.Code == "host_report_timeout" {
		return context.DeadlineExceeded
	}
	return ErrHostCompatibilityUnavailable
}
