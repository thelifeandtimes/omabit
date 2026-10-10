//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func ReadManagedLogs(ctx context.Context, state ManagedState) (protocol.RuntimeLogs, error) {
	return readManagedLogs(ctx, state, defaults())
}

func readManagedLogs(ctx context.Context, state ManagedState, d dependencies) (protocol.RuntimeLogs, error) {
	if !state.Prepared {
		return protocol.RuntimeLogs{Source: "none", ObservedAt: time.Now().UTC(), Lines: []string{}}, nil
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	for i := len(q.r.Containers) - 1; i >= 0; i-- {
		c := q.r.Containers[i]
		if c.Kind != "probe" && !c.Removed {
			return q.containerLogs(ctx, i)
		}
	}
	if q.r.RetainedLogs != nil {
		out := *q.r.RetainedLogs
		out.Source = "retained"
		return out, nil
	}
	return protocol.RuntimeLogs{Source: "none", ObservedAt: time.Now().UTC(), Lines: []string{}}, nil
}

func (q *qualifier) containerLogs(ctx context.Context, index int) (protocol.RuntimeLogs, error) {
	if q.d.logs == nil {
		return protocol.RuntimeLogs{}, errors.New("runtime log reader unavailable")
	}
	c, err := q.inspect(ctx, index)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	if err = q.containment(c, q.r.Containers[index].Kind); err != nil {
		return protocol.RuntimeLogs{}, err
	}
	text, err := q.d.logs(ctx, q.socket, c.ID)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	lines, truncated := sanitizeLogs(text)
	return protocol.RuntimeLogs{ContainerID: c.ID, Kind: q.r.Containers[index].Kind, Source: "docker", ObservedAt: time.Now().UTC(), Lines: lines, Truncated: truncated}, nil
}

// Docker's non-TTY response multiplexes stdout and stderr in 8-byte frames.
// Bound both the transport and rendered tail; never invoke a shell or guest code.
func readDockerLogs(ctx context.Context, socket, id string) (string, error) {
	if !containerIDPattern.MatchString(id) {
		return "", errors.New("logs require a full container ID")
	}
	transport := &http.Transport{DisableKeepAlives: true, DisableCompression: true, MaxResponseHeaderBytes: 8192, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://docker/v1.47/containers/"+id+"/logs?stdout=true&stderr=true&timestamps=true&tail=200", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("private runtime logs unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("private runtime logs unavailable")
	}
	return decodeDockerLogs(io.LimitReader(resp.Body, outputLimit+1))
}

func decodeDockerLogs(reader io.Reader) (string, error) {
	var out strings.Builder
	consumed := 0
	for {
		var h [8]byte
		_, err := io.ReadFull(reader, h[:])
		if err == io.EOF {
			return out.String(), nil
		}
		if err != nil || (h[0] != 1 && h[0] != 2) || h[1] != 0 || h[2] != 0 || h[3] != 0 {
			return "", errors.New("invalid runtime log framing")
		}
		n := int64(binary.BigEndian.Uint32(h[4:]))
		if n > int64(outputLimit-consumed-8) {
			return "", errors.New("runtime log response exceeds bounded read")
		}
		consumed += int(n) + 8
		if _, err = io.CopyN(&out, reader, n); err != nil {
			return "", errors.New("incomplete runtime log frame")
		}
	}
}
