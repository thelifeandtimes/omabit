package managed

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const hostReportTimeout = 2 * time.Second
const hostEngineMaxAge = 5 * time.Second

// Native metadata and running engine evidence have independent caches. A live
// refresh never downloads packages, executes tools, or enumerates containers.
type hostCompatibilityCache struct {
	live    hostEvidenceCache
	native  hostEvidenceCache
	inspect func() (hostcompat.Report, error)
	owner   func() (*ownerhost.Profile, error)
	probe   func(context.Context, string, string) (hostcompat.Engine, error)
}

type hostEvidence struct {
	report             hostcompat.Report
	metadataObservedAt time.Time
	expires            time.Time
}

type hostEvidenceFlight struct {
	done  chan struct{}
	value hostEvidence
	err   error
}

// At most one worker per cache may run. Request cancellation stops that waiter;
// a shared refresh has its own deadline and the server's lifetime. InspectCurrent
// has no context API: if a filesystem read stalls, its single native worker stays
// occupied, but timed-out polls cannot spawn more inspectors or hold the journal.
type hostEvidenceCache struct {
	mu      sync.Mutex
	pending *hostEvidenceFlight
	value   hostEvidence
	err     error
	expires time.Time
}

func (c *hostEvidenceCache) get(ctx, lifetime context.Context, load func(context.Context) (hostEvidence, error)) (hostEvidence, error) {
	// Bound the wait even if a filesystem/NSS call cannot honor the worker's
	// context. Its occupied single-flight slot still prevents replacement workers.
	ctx, cancel := context.WithTimeout(ctx, hostReportTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return hostEvidence{}, err
	}
	if err := lifetime.Err(); err != nil {
		return hostEvidence{}, err
	}
	c.mu.Lock()
	if time.Now().Before(c.expires) {
		value, err := c.value, c.err
		c.mu.Unlock()
		return value, err
	}
	if c.pending == nil {
		flight := &hostEvidenceFlight{done: make(chan struct{})}
		c.pending = flight
		go func() {
			work, cancel := context.WithTimeout(lifetime, hostReportTimeout)
			defer cancel()
			value, err := load(work)
			if work.Err() != nil {
				err = work.Err()
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			c.value, c.err, c.expires = value, err, value.expires
			if err != nil {
				c.value = hostEvidence{}
				c.expires = time.Now().Add(hostEngineMaxAge)
			}
			flight.value, flight.err = c.value, c.err
			close(flight.done)
			c.pending = nil
		}()
	}
	ready := c.pending
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return hostEvidence{}, ctx.Err()
	case <-lifetime.Done():
		return hostEvidence{}, lifetime.Err()
	case <-ready.done:
		return ready.value, ready.err
	}
}

func (c *hostCompatibilityCache) get(ctx, lifetime context.Context, runnerID, mode string) (protocol.HostCompatibilityReport, error) {
	value, err := c.live.get(ctx, lifetime, func(work context.Context) (hostEvidence, error) {
		metadata, err := c.native.get(work, lifetime, func(context.Context) (hostEvidence, error) {
			inspect := c.inspect
			if inspect == nil {
				inspect = func() (hostcompat.Report, error) { return hostcompat.InspectCurrent(), nil }
			}
			report, err := inspect()
			return hostEvidence{report: report, metadataObservedAt: report.ObservedAt, expires: report.ObservedAt.Add(protocol.HostCompatibilityMaxAge)}, err
		})
		if err != nil {
			return hostEvidence{}, err
		}
		report := c.observeEngine(work, metadata.report)
		report.ObservedAt = time.Now().UTC()
		expires := report.ObservedAt.Add(hostEngineMaxAge)
		if metadata.expires.Before(expires) {
			expires = metadata.expires
		}
		return hostEvidence{report: report, metadataObservedAt: metadata.metadataObservedAt, expires: expires}, nil
	})
	if err != nil {
		return protocol.HostCompatibilityReport{}, err
	}
	bound := protocol.HostCompatibilityReport{
		SchemaVersion: protocol.HostCompatibilitySchema, ProtocolVersion: protocol.Version,
		RunnerID: runnerID, Mode: mode, Features: []string{protocol.HostCompatibilityFeature},
		// Evaluate uses the private data root before the public projection removes it.
		Report: protocol.PublicHostCompatibility(value.report), ExpiresAt: value.expires,
		MetadataObservedAt: &value.metadataObservedAt,
	}
	if !protocol.ValidHostCompatibility(bound) {
		return protocol.HostCompatibilityReport{}, errors.New("invalid host compatibility evidence")
	}
	return bound, nil
}

func (c *hostCompatibilityCache) observeEngine(ctx context.Context, metadata hostcompat.Report) hostcompat.Report {
	report := metadata
	report.Engine = nil
	report.Checks = append([]hostcompat.Check(nil), metadata.Checks...)
	owner := c.owner
	if owner == nil {
		owner = ownerhost.Current
	}
	p, err := owner()
	if err != nil || p == nil || p.Validate() != nil || p.UID != uint32(os.Geteuid()) {
		report = hostcompat.Evaluate(report, "")
		report.Checks = append(report.Checks, hostcompat.Check{ID: "host.owner", Status: "unknown", Detail: "Running engine evidence requires the installed non-root owner and a trusted owner profile."})
		return report
	}
	socket := filepath.Join("/run/user", strconv.FormatUint(uint64(p.UID), 10), rootless.SocketName)
	if run := ownerhost.RuntimeDirectory(); run != "" {
		socket = filepath.Join(run, rootless.SocketName)
	}
	dataRoot := filepath.Join(p.Home, ".local/share/urbit-runner/docker")
	probe := c.probe
	if probe == nil {
		probe = hostcompat.ProbeEngine
	}
	probeCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	engine, err := probe(probeCtx, socket, dataRoot)
	if err != nil {
		report = hostcompat.Evaluate(report, dataRoot)
		report.Checks = append(report.Checks, hostcompat.Check{ID: "host.engine", Status: "unknown", Detail: "Private owner engine is unavailable, timed out, or failed identity/API/rootless/data-root checks."})
		return report
	}
	report.Engine = &engine
	return hostcompat.Evaluate(report, dataRoot)
}

func (r *Server) serveHostCompatibility(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet || req.URL.RawQuery != "" {
		respond(w, http.StatusBadRequest, protocol.ErrorResponse{Error: protocol.APIError{Code: "host_report_read_only", Message: "host compatibility accepts GET without parameters"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), hostReportTimeout+250*time.Millisecond)
	defer cancel()
	value, err := r.hostCompatibility.get(ctx, r.ctx, r.s.runnerID, r.s.mode)
	if err != nil {
		// This is host evidence failure, never a state-store failure. In particular,
		// do not pass arbitrary provider errors to respondError, which fences writes.
		problem := protocol.APIError{Code: "host_report_unavailable", Message: "host compatibility evidence is unavailable; retry independently of ship observation"}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			problem.Code = "host_report_timeout"
		}
		respond(w, http.StatusServiceUnavailable, protocol.ErrorResponse{Error: problem})
		return
	}
	respond(w, http.StatusOK, value)
}
