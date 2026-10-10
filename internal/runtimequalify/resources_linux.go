//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

const statsLimit = 256 << 10

func (q *qualifier) resources(ctx context.Context, index int) (*ManagedResources, error) {
	if !q.d.managed || !isNetworkedKind(q.r.Containers[index].Kind) {
		return nil, errors.New("resource observation requires a managed runtime container")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c, err := q.inspect(ctx, index)
	if err != nil {
		return nil, err
	}
	if err := q.containment(c, q.r.Containers[index].Kind); err != nil {
		return nil, err
	}
	if !*c.State.Running || c.State.Paused || c.State.Restarting || c.State.Dead {
		return nil, errors.New("resource observation requires a running container")
	}
	if err := q.boundary(); err != nil {
		return nil, err
	}
	out, err := q.d.stats(ctx, q.socket, c.ID)
	if err != nil || ctx.Err() != nil {
		return nil, errors.New("container resource read unavailable")
	}
	return parseResources(out, c.ID)
}

// Read only the authenticated private daemon, independent of Docker contexts,
// proxy environment, credential configuration, redirects, and guest processes.
func readDockerStats(ctx context.Context, socket, id string) (string, error) {
	if !containerIDPattern.MatchString(id) {
		return "", errors.New("resource request requires a full container ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	transport := &http.Transport{DisableKeepAlives: true, DisableCompression: true, MaxResponseHeaderBytes: 8192,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/v1.47/containers/"+id+"/stats?stream=false&one-shot=true", nil)
	if err != nil {
		return "", errors.New("cannot prepare resource request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("private Docker stats request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("private Docker stats response unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, statsLimit+1))
	if err != nil || len(data) > statsLimit {
		return "", errors.New("private Docker stats response incomplete or oversized")
	}
	return string(data), nil
}

func parseResources(out, id string) (*ManagedResources, error) {
	invalid := errors.New("invalid or unsupported container resource observation")
	var wire struct {
		ID     string    `json:"id"`
		Read   time.Time `json:"read"`
		Memory struct {
			Usage *uint64 `json:"usage"`
			Limit *uint64 `json:"limit"`
		} `json:"memory_stats"`
		PIDs struct {
			Current *uint64 `json:"current"`
		} `json:"pids_stats"`
		CPU json.RawMessage `json:"cpu_stats"`
	}
	if len(out) > statsLimit || json.Unmarshal([]byte(out), &wire) != nil || wire.ID != id || !containerIDPattern.MatchString(id) || wire.Read.IsZero() || wire.Memory.Usage == nil || wire.Memory.Limit == nil || *wire.Memory.Limit != uint64(memoryLimit) || wire.PIDs.Current == nil {
		return nil, invalid
	}
	// Preserve the daemon's sample time; reading cached/old data cannot refresh it.
	// Optional CPU evidence must not invalidate otherwise usable memory readings.
	result := &ManagedResources{ContainerID: wire.ID, ObservedAt: wire.Read.UTC(), MemoryCurrentBytes: *wire.Memory.Usage, MemoryLimitBytes: *wire.Memory.Limit, PIDsCurrent: *wire.PIDs.Current}
	var cpu struct {
		Usage struct {
			Total *uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		System *uint64 `json:"system_cpu_usage"`
		Online *uint32 `json:"online_cpus"`
	}
	if json.Unmarshal(wire.CPU, &cpu) == nil && cpu.Usage.Total != nil && cpu.System != nil && cpu.Online != nil && *cpu.Online > 0 && *cpu.Online <= 4096 {
		// resources() verifies the pinned two-CPU cgroup in containment() first.
		result.CPU = &ManagedCPU{TotalNS: *cpu.Usage.Total, SystemNS: *cpu.System, OnlineCPUs: *cpu.Online, LimitCores: 2}
	}
	return result, nil
}
