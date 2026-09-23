//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var errHostNotReady = errors.New("host-loopback HTTP endpoint is not ready")

// A credential-free GET to the exact loopback endpoint derived from inspected
// Docker bindings. Never honor proxies, redirects, cookies, or arbitrary URLs.
func probeHostHTTP(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("invalid host-loopback endpoint")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1024 || port > 65535 || endpoint != "http://"+net.JoinHostPort("127.0.0.1", strconv.Itoa(port))+"/~/host" {
		return errors.New("host HTTP probe requires the exact unprivileged loopback endpoint")
	}
	transport := &http.Transport{
		Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		DisableCompression: true, ResponseHeaderTimeout: 3 * time.Second,
		MaxResponseHeaderBytes: 8192, DisableKeepAlives: true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport, Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return errors.New("cannot create host HTTP probe")
	}
	req.Header.Set("Accept", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return errHostNotReady
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode >= 500 {
		return errHostNotReady
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("public host endpoint did not return HTTP 200; redirects and authentication are not attempted")
	}
	if resp.Header.Get("Content-Encoding") != "" && resp.Header.Get("Content-Encoding") != "identity" {
		return errors.New("unexpected encoding on public host response")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 129))
	if err != nil {
		return errHostNotReady
	}
	if len(body) > 128 || strings.TrimSpace(string(body)) != "~zod" {
		return errors.New("public host identity did not match the disposable fake zod")
	}
	return nil
}

func (q *qualifier) readyHostHTTP(ctx context.Context, index int) error {
	ctx, cancel := context.WithTimeout(ctx, q.d.times.ready)
	defer cancel()
	for ctx.Err() == nil {
		c, err := q.inspect(ctx, index)
		if err != nil {
			return err
		}
		if !*c.State.Running || c.State.Paused || c.State.Restarting || *c.State.OOMKilled || c.State.Dead {
			return errors.New("Vere exited or became unhealthy before host HTTP readiness")
		}
		if _, err := q.inspectNetwork(ctx); err != nil {
			return err
		}
		q.r.Containers[index].AliveObserved = true
		endpoint, err := q.publishedEndpoint(c, q.r.Containers[index].Kind)
		if err != nil && !errors.Is(err, errBindingPending) {
			return err
		}
		if err == nil {
			q.r.Containers[index].HostEndpoint = endpoint
			if err := q.save(); err != nil {
				return err
			}
			err = q.d.hostProbe(ctx, endpoint)
			if err == nil {
				// The process and its inspected binding must still be the same
				// after the request. Endpoint evidence does not grant owner login.
				after, inspectErr := q.inspect(ctx, index)
				if inspectErr != nil {
					return inspectErr
				}
				if !*after.State.Running || after.State.Paused || after.State.Restarting || *after.State.OOMKilled || after.State.Dead {
					return errors.New("Vere exited during the host HTTP check")
				}
				again, bindingErr := q.publishedEndpoint(after, q.r.Containers[index].Kind)
				if bindingErr != nil || again != endpoint {
					return errors.New("host HTTP binding changed during the probe")
				}
				if _, err := q.inspectNetwork(ctx); err != nil {
					return err
				}
				q.r.Containers[index].HostHTTPVerified = true
				return q.save()
			}
			if !errors.Is(err, errHostNotReady) {
				// Probe errors must not expose an untrusted HTTP body or headers.
				return errors.New("host HTTP identity or response policy check failed")
			}
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	return errors.New("host-loopback HTTP readiness deadline or cancellation")
}
