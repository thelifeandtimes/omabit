//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// A Stop request permits graceful shutdown, never escalation to SIGKILL.
// Fresh keyed Vere 4.6 processes need the qualified hood exit path because
// their initial process can ignore SIGTERM. Resume processes use SIGTERM.
func (q *qualifier) stopManagedRuntime(ctx context.Context, index int) error {
	ctx, cancel := context.WithTimeout(ctx, q.d.times.stop)
	defer cancel()
	c, err := q.inspect(ctx, index)
	if err != nil {
		return err
	}
	if !active(c) {
		return nil
	}
	if q.r.Containers[index].Kind == "comet-boot" && q.d.nativeShip != "" {
		ports, err := q.readHTTPPorts()
		if err != nil {
			return err
		}
		port, err := privateHTTPPort(ports, 8080)
		if err != nil {
			return err
		}
		endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
		call := func(payload string) (string, error) {
			return q.control(ctx, index, 5*time.Second, "exec", "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--fail", "--max-time", "4", "--max-filesize", "1024", "--header", "Content-Type: application/json", "--data-binary", payload, endpoint)
		}
		out, err := call(`{"source":{"dojo":"our"},"sink":{"stdout":null}}`)
		var ship string
		if err != nil || json.Unmarshal([]byte(out), &ship) != nil || strings.TrimSpace(ship) != q.d.nativeShip {
			return errors.New("graceful shutdown could not verify private Eyre identity; runtime retained")
		}
		if current, err := q.readHTTPPorts(); err != nil || current != ports {
			return errors.New("private Eyre changed before shutdown")
		}
		// The ship may close HTTP before sending the acknowledgement. Observe the
		// exact container to resolve exit; never repeat the poke after a lost reply.
		_, _ = call(`{"source":{"dojo":"+hood/exit"},"sink":{"app":"hood"}}`)
	} else {
		if _, err = q.control(ctx, index, q.d.times.command, "kill", "--signal", "SIGTERM"); err != nil {
			return err
		}
	}
	for {
		c, err = q.inspect(ctx, index)
		if err != nil {
			return err
		}
		if !active(c) {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("graceful shutdown timed out; runtime retained without force-kill")
		case <-time.After(q.d.times.poll):
		}
	}
}
