//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func statsJSON(id, usage, limit, pids string) string {
	return fmt.Sprintf(`{"id":%q,"read":"2026-09-25T18:00:00Z","memory_stats":{"usage":%s,"limit":%s},"pids_stats":{"current":%s}}`, id, usage, limit, pids)
}

func TestResourceParsingUsesExactUnitsIdentityAndUnknowns(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, current := range []string{"0", "67108864", "4294967297"} {
		m, err := parseResources(statsJSON(id, current, "4294967296", "12"), id)
		if err != nil || m.MemoryLimitBytes != 4<<30 {
			t.Fatalf("valid counters rejected: %+v %v", m, err)
		}
	}
	m, err := parseResources(statsJSON(id, "67108864", "4294967296", "12"), id)
	if err != nil || m.MemoryCurrentBytes != 64<<20 || m.PIDsCurrent != 12 || m.ObservedAt.Format(time.RFC3339) != "2026-09-25T18:00:00Z" {
		t.Fatalf("counter units changed: %+v %v", m, err)
	}
	for _, out := range []string{
		"", `{}`, `{"id":"` + id + `","read":"0001-01-01T00:00:00Z"}`,
		statsJSON("wrong", "0", "4294967296", "12"), statsJSON(id, "null", "4294967296", "12"),
		statsJSON(id, "0", "null", "12"), statsJSON(id, "0", "4294967296", "null"),
		statsJSON(id, "0", "0", "12"), statsJSON(id, "-1", "4294967296", "12"),
		statsJSON(id, "1.5", "4294967296", "12"), statsJSON(id, "18446744073709551616", "4294967296", "12"),
		statsJSON(id, "1", "4294967296", "12") + `{}`, strings.Repeat("1", statsLimit+1),
	} {
		if m, err := parseResources(out, id); err == nil || m != nil {
			t.Fatalf("invalid counters became a sample: %+v %v", m, err)
		}
	}
}

func TestManagedResourcesUseOwnedContainerAndDoNotChangeHealth(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now().UTC()
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || obs.Health != "healthy" || obs.RecoveryRequired || obs.Resources == nil {
		t.Fatalf("observation failed: %+v %v", obs, err)
	}
	m := obs.Resources
	if m.ContainerID != state.Runtime.Containers[len(state.Runtime.Containers)-1].ID || m.ObservedAt.Before(began) || m.MemoryCurrentBytes != 64<<20 || m.MemoryLimitBytes != 4<<30 || m.PIDsCurrent != 12 {
		t.Fatalf("unbound or malformed sample: %+v", m)
	}
	// The stats adapter authenticates the exact owned ID. No guest command or
	// control-socket access is allowed by the fake executor.
	if f.events[len(f.events)-1] != "resources" {
		t.Fatalf("health checks did not precede optional telemetry: %v", f.events)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	before := len(f.events)
	_, obs, err = observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "stopped" || obs.Resources != nil {
		t.Fatalf("stopped runtime retained live sample: %+v %v", obs, err)
	}
	for _, event := range f.events[before:] {
		if event == "resources" {
			t.Fatal("executed resource probe in stopped runtime")
		}
	}
}

func TestOptionalCPUParsingPreservesMemoryAndExactCounters(t *testing.T) {
	id := strings.Repeat("a", 64)
	base := statsJSON(id, "0", "4294967296", "12")
	for _, cpu := range []string{`null`, `{}`, `{"cpu_usage":{"total_usage":-1}}`, `{"cpu_usage":{"total_usage":0},"system_cpu_usage":10,"online_cpus":0}`, `{"cpu_usage":{"total_usage":1.2},"system_cpu_usage":10,"online_cpus":8}`} {
		r, err := parseResources(base[:len(base)-1]+`,"cpu_stats":`+cpu+`}`, id)
		if err != nil || r == nil || r.CPU != nil || r.MemoryCurrentBytes != 0 {
			t.Fatalf("invalid optional CPU damaged memory: %+v %v", r, err)
		}
	}
	r, err := parseResources(base[:len(base)-1]+`,"cpu_stats":{"cpu_usage":{"total_usage":9223372036854775809},"system_cpu_usage":18446744073709551615,"online_cpus":8}}`, id)
	if err != nil || r.CPU == nil || r.CPU.TotalNS != 9223372036854775809 || r.CPU.SystemNS != ^uint64(0) || r.CPU.OnlineCPUs != 8 || r.CPU.LimitCores != 2 {
		t.Fatalf("CPU counters lost exact units: %+v %v", r, err)
	}
}

func TestResourceReadFailureIsNotRuntimeFailure(t *testing.T) {
	for _, fault := range []string{"missing counters", "request error", "deadline"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			f.d.stats = func(ctx context.Context, _, _ string) (string, error) {
				deadline, bounded := ctx.Deadline()
				if !bounded || time.Until(deadline) > 3*time.Second {
					t.Fatal("unbounded resource probe")
				}
				switch fault {
				case "missing counters":
					return "", nil
				case "deadline":
					<-ctx.Done()
					return "", ctx.Err()
				default:
					return "PRIVATE-OUTPUT", errors.New("PRIVATE-ERROR")
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, obs, err := observeManaged(ctx, state, f.d)
			if err != nil || obs.Resources != nil || obs.State != "running" || obs.Health != "healthy" || obs.RecoveryRequired {
				t.Fatalf("resource failure poisoned runtime health: %+v %v", obs, err)
			}
		})
	}
}

func TestPrivateStatsTransportPolicies(t *testing.T) {
	id := strings.Repeat("a", 64)
	for _, fault := range []string{"", "redirect", "status", "oversize", "deadline"} {
		t.Run("response-"+fault, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "stats.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
			t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.RequestURI != "/v1.47/containers/"+id+"/stats?stream=false&one-shot=true" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Accept-Encoding") != "" {
					t.Errorf("unsafe stats request: %s %s %+v", r.Method, r.RequestURI, r.Header)
				}
				switch fault {
				case "redirect":
					w.Header().Set("Location", "http://untrusted.invalid/redirect")
					w.WriteHeader(302)
				case "status":
					w.WriteHeader(500)
					fmt.Fprint(w, "PRIVATE-ERROR")
				case "oversize":
					fmt.Fprint(w, strings.Repeat("x", statsLimit+1))
				case "deadline":
					<-r.Context().Done()
				default:
					fmt.Fprint(w, statsJSON(id, "67108864", "4294967296", "12"))
				}
			})
			server := &http.Server{Handler: handler}
			done := make(chan error, 1)
			go func() { done <- server.Serve(listener) }()
			defer func() { server.Close(); <-done }()
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			out, err := readDockerStats(ctx, socket, id)
			if fault == "" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := parseResources(out, id); err != nil {
					t.Fatal(err)
				}
			} else if err == nil || out != "" || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatalf("transport failure not bounded/redacted: %d bytes, %v", len(out), err)
			}
		})
	}
}
