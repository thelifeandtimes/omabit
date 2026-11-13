package shiproute

import (
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

// LocalTailnet returns only the current collector's network scope. A remote
// runner's discovery cannot establish the collector's network membership. This
// read never examines Serve configuration or changes the daemon.
func LocalTailnet(ctx context.Context) string {
	binary, err := exec.LookPath("tailscale")
	if err != nil || !filepath.IsAbs(binary) {
		return ""
	}
	return localTailnet(ctx, func(ctx context.Context, args ...string) ([]byte, error) {
		return readTailscale(ctx, binary, args...)
	})
}

func localTailnet(ctx context.Context, read tailscaleRead) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if read == nil || ctx.Err() != nil {
		return ""
	}
	data, err := read(ctx, "status", "--json", "--peers=false")
	if err != nil || ctx.Err() != nil {
		return ""
	}
	status, err := ParseTailnetStatus(data)
	if err != nil {
		return ""
	}
	return status.Suffix
}
