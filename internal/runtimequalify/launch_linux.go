//go:build linux && amd64

package runtimequalify

import (
	"fmt"
	"strings"
)

func (q *qualifier) admitPorts(kind string) error {
	if q.d.portAvailable == nil || !isNetworkedKind(kind) {
		return nil
	}
	if q.d.hostPort != 0 && !q.d.portAvailable("tcp", q.d.hostPort) {
		return fmt.Errorf("host HTTP port %d is already in use; choose another HTTP port", q.d.hostPort)
	}
	if q.publishAmes(kind) && !q.d.portAvailable("udp", q.d.amesPort) {
		return fmt.Errorf("host Ames UDP port %d is already in use; choose another Ames port", q.d.amesPort)
	}
	return nil
}

// Return actionable categories without reflecting daemon-supplied command lines,
// paths, credentials, or unbounded OCI error payloads into the operation journal.
func launchReason(raw string) string {
	if raw == "" {
		return ""
	}
	text := strings.ToLower(raw)
	switch {
	case strings.Contains(text, "address already in use"), strings.Contains(text, "port is already allocated"):
		return "Host port is already in use. Check the configured HTTP/Ames ports and choose an unused port."
	case strings.Contains(text, "permission denied"), strings.Contains(text, "operation not permitted"):
		return "Container launch was denied by host permissions or namespace policy. Review rootless Docker and mount permissions."
	case strings.Contains(text, "no space left on device"):
		return "The host ran out of disk space or inodes while launching the container."
	case strings.Contains(text, "cannot allocate memory"), strings.Contains(text, "out of memory"):
		return "The host could not allocate memory for the container."
	case strings.Contains(text, "read-only file system"):
		return "A required runtime filesystem is read-only."
	case strings.Contains(text, "no such file or directory"):
		return "A required runtime executable, mount source or host resource is missing."
	default:
		return "Docker could not launch the container. Inspect its host-side state for the underlying OCI/network error."
	}
}
