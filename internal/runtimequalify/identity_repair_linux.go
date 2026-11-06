//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

func RepairBootIdentity(ctx context.Context, s ManagedState, expected, ship, id string, saved *BootIdentityRepair, persist func(BootIdentityRepair) error) (ManagedState, error) {
	return repairBootIdentity(ctx, s, expected, ship, id, saved, persist, defaults(), removedBootContainer)
}

func repairBootIdentity(ctx context.Context, s ManagedState, expected, ship, id string, saved *BootIdentityRepair, persist func(BootIdentityRepair) error, d dependencies, absent func(context.Context, string, string, string) error) (ManagedState, error) {
	ctx, cancel := context.WithTimeout(ctx, d.times.stop+time.Minute)
	defer cancel()
	fail := func(err error) (ManagedState, error) { return ManagedState{}, err }
	if persist == nil {
		return fail(errors.New("durable identity repair journal required"))
	}
	if err := BootIdentityRepairEligible(s, expected, ship, id); err != nil {
		return fail(err)
	}
	// Keep the caller's original state and retained journal immutable on failure.
	raw, err := json.Marshal(s)
	if err != nil {
		return fail(err)
	}
	s = ManagedState{}
	if err = json.Unmarshal(raw, &s); err != nil {
		return fail(err)
	}
	j := BootIdentityRepair{Version: 1, ExpectedShip: expected, Ship: ship, ContainerID: id, State: s}
	if saved != nil {
		raw, err := json.Marshal(saved)
		if err != nil {
			return fail(err)
		}
		j = BootIdentityRepair{}
		if err = json.Unmarshal(raw, &j); err != nil {
			return fail(err)
		}
		if j.Version != 1 || j.ExpectedShip != expected || j.Ship != ship || j.ContainerID != id || j.State.ID != s.ID || j.VerifiedAt.IsZero() || j.VerifiedAt.Before(*s.Comet.StartedAt) ||
			(j.Stage != "verified" && j.Stage != "removing" && j.Stage != "removed") {
			return fail(errors.New("invalid or mismatched identity repair journal"))
		}
		s = j.State
	}
	if err := BootIdentityRepairEligible(s, expected, ship, id); err != nil {
		return fail(err)
	}
	q, err := managedSession(&s, nil, d)
	if err != nil {
		return fail(err)
	}
	if err = q.verifyEngineAccess(ctx, true); err != nil {
		return fail(err)
	}
	if err = q.pier(); err != nil {
		return fail(err)
	}
	i := -1
	for n, c := range q.r.Containers {
		if c.ID == id {
			i = n
		}
	}
	flush := func() error { s.Runtime = q.r; j.State = s; return persist(j) }
	q.d.persist = func(r Result) error { q.r = r; return flush() }
	if j.Stage == "removed" || j.Stage == "removing" {
		last := q.r.Containers[i].LastState
		if last == nil || last.Running || last.Paused || last.Restarting || last.Dead || last.OOMKilled || last.ExitCode != 0 {
			return fail(errors.New("repair removal lacks durable clean terminal evidence"))
		}
	}
	if j.Stage == "removed" {
		if err = absent(ctx, q.socket, id, q.r.Containers[i].Name); err != nil {
			return fail(err)
		}
	} else {
		c, inspectErr := q.inspect(ctx, i)
		if inspectErr != nil {
			// A recorded clean stop and removal intent are required in addition to
			// explicit daemon 404s for BOTH immutable ID and exact original name.
			if j.Stage != "removing" {
				return fail(inspectErr)
			}
			if err = absent(ctx, q.socket, id, q.r.Containers[i].Name); err != nil {
				return fail(errors.Join(inspectErr, err))
			}
		} else {
			if err = q.containment(c, "comet-boot"); err != nil {
				return fail(err)
			}
			if _, err = q.inspectNetwork(ctx); err != nil {
				return fail(err)
			}
			started, parseErr := time.Parse(time.RFC3339Nano, c.State.StartedAt)
			if parseErr != nil || started.IsZero() || saved != nil && c.State.StartedAt != j.ContainerStartedAt {
				return fail(errors.New("boot process generation changed or unavailable"))
			}
			if active(c) {
				if j.Stage == "removing" || c.State.Paused || c.State.Restarting || c.State.Dead || *c.State.OOMKilled {
					return fail(errors.New("boot process is not eligible for graceful repair"))
				}
				endpoint, err := q.publishedEndpoint(c, "comet-boot")
				if err != nil {
					return fail(err)
				}
				actual, err := q.d.hostIdentity(ctx, endpoint)
				if err != nil || actual != ship {
					return fail(errors.New("public identity does not match confirmed correction"))
				}
				if _, err = q.verifiedKeyedControl(ctx, i, ship); err != nil {
					return fail(err)
				}
				if saved == nil {
					j.VerifiedAt, j.ContainerStartedAt, j.Stage = time.Now().UTC(), c.State.StartedAt, "verified"
					if err = flush(); err != nil {
						return fail(err)
					}
				}
				if err = q.stopManagedRuntimeIdentity(ctx, i, ship); err != nil {
					return fail(err)
				}
				c, err = q.inspect(ctx, i)
				if err != nil {
					return fail(err)
				}
			} else if saved == nil {
				return fail(errors.New("initial correction requires live public and private identity evidence"))
			}
			if c.State.StartedAt != j.ContainerStartedAt || active(c) || c.State.Dead || *c.State.OOMKilled || *c.State.ExitCode != 0 {
				return fail(errors.New("repair requires a clean exit of the verified process"))
			}
			if err = q.pier(); err != nil {
				return fail(err)
			}
			j.Stage = "removing"
			if err = flush(); err != nil {
				return fail(err)
			}
			if err = q.remove(ctx, i); err != nil {
				return fail(err)
			}
		}
		q.r.Containers[i].Removed = true
		j.Stage = "removed"
		if err = flush(); err != nil {
			return fail(err)
		}
	}
	if err = q.pier(); err != nil {
		return fail(err)
	}
	s.Runtime = q.r
	comet := *s.Comet
	s.Comet = &comet
	s.NativeShip, s.Comet.Ship = ship, ship
	s.Comet.Uncertain, s.Initialized = false, true
	s.Comet.FinishedAt = &j.VerifiedAt
	s.Runtime.Stage = "stopped"
	return s, nil
}

// A failed inspect, empty filtered inventory, or daemon outage is NOT absence.
func removedBootContainer(ctx context.Context, socket, id, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	transport := &http.Transport{DisableKeepAlives: true, MaxResponseHeaderBytes: 8192, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, ref := range []string{id, name} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/v"+hostcompat.APIVersion+"/containers/"+url.PathEscape(ref)+"/json", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return errors.New("cannot confirm removed boot container")
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			return errors.New("boot container ID/name not proven absent")
		}
	}
	return nil
}
