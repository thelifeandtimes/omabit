//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// DeskRuntime binds typed desk calls to one verified process and its private
// Lens port. The runner must hold its lifecycle/policy guard for each effect
// and persist dispatch before calling a mutation method. This type grants no
// policy permission, retries no requests, and never cancels shared Lens work.
type DeskRuntime struct {
	ship, pier, endpoint, containerID string
	q                                 *qualifier
	check                             func(context.Context) error
}

func OpenManagedDesk(ctx context.Context, state ManagedState, expectedContainer string) (*DeskRuntime, error) {
	return openManagedDesk(ctx, state, expectedContainer, defaults())
}
func openManagedDesk(ctx context.Context, state ManagedState, expectedContainer string, d dependencies) (*DeskRuntime, error) {
	if !state.Prepared || !state.Initialized || state.UncertainMaintenance() || state.UncertainProvisioning() || !containerIDPattern.MatchString(expectedContainer) {
		return nil, errors.New("desk runtime selection is unresolved")
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return nil, err
	}
	if err = q.verifyEngine(ctx); err != nil {
		return nil, err
	}
	index := -1
	for i, c := range q.r.Containers {
		if c.Removed {
			continue
		}
		if index >= 0 || c.ID != expectedContainer || c.CreateUncertain || !isNetworkedKind(c.Kind) {
			return nil, errors.New("desk runtime container changed")
		}
		index = i
	}
	if index < 0 {
		return nil, errors.New("desk runtime container is absent")
	}
	selected, err := q.inspect(ctx, index)
	if err != nil {
		return nil, err
	}
	if !deskRunning(selected) {
		return nil, errors.New("desk runtime is not running")
	}
	ports, err := q.readHTTPPorts()
	if err != nil {
		return nil, err
	}
	port, err := privateHTTPPort(ports, 8080)
	if err != nil {
		return nil, err
	}
	ship := "~zod"
	if state.Fake != nil {
		ship = state.Fake.Ship
	} else if state.Comet != nil {
		ship = state.Comet.Ship
	}
	s := &DeskRuntime{ship: ship, pier: q.pierPath(), endpoint: fmt.Sprintf("http://127.0.0.1:%d", port), containerID: expectedContainer, q: q}
	s.check = func(ctx context.Context) error {
		c, err := q.inspect(ctx, index)
		if err != nil {
			return err
		}
		if !deskRunning(c) || !reflect.DeepEqual(c.State, selected.State) {
			return errors.New("desk runtime process changed")
		}
		if _, err = q.inspectNetwork(ctx); err != nil {
			return err
		}
		if err = q.containment(c, q.r.Containers[index].Kind); err != nil {
			return err
		}
		current, err := q.readHTTPPorts()
		if err != nil || current != ports {
			return errHTTPPorts
		}
		return nil
	}
	if err = s.check(ctx); err != nil {
		return nil, err
	}
	return s, nil
}
func deskRunning(c container) bool {
	return c.State != nil && c.State.Running != nil && *c.State.Running && !c.State.Paused && !c.State.Restarting && !c.State.Dead
}
func (s *DeskRuntime) Ship() string     { return s.ship }
func (s *DeskRuntime) PierPath() string { return s.pier }

func (s *DeskRuntime) post(ctx context.Context, body []byte) (string, error) {
	// Input is stdin, never argv or a file in the running pier. curl's config,
	// proxy, redirects and alternate destinations are all disabled/fixed.
	out, err := s.q.dockerInput(ctx, 35*time.Second, body, "container", "exec", "--interactive", s.containerID,
		"/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--fail",
		"--max-time", "30", "--max-filesize", "1048576", "--header", "Content-Type: application/json", "--data-binary", "@-", s.endpoint)
	if err != nil {
		return "", errors.New("desk transport unavailable; inspect before retrying an effect")
	}
	var value string
	if json.Unmarshal([]byte(out), &value) != nil || len(value) > 1<<20 {
		return "", errors.New("invalid desk transport response")
	}
	return value, nil
}
func (s *DeskRuntime) call(ctx context.Context, r deskwork.LensRequest) (string, error) {
	body, err := r.Body()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err = s.check(ctx); err != nil {
		return "", err
	}
	identity, err := s.post(ctx, []byte(`{"source":{"dojo":"our"},"sink":{"stdout":null}}`))
	if err != nil || strings.TrimSpace(identity) != s.ship {
		return "", errors.New("desk runtime identity unavailable or changed")
	}
	if err = s.check(ctx); err != nil {
		return "", err
	}
	result, err := s.post(ctx, body)
	if err != nil {
		return "", err
	}
	identity, err = s.post(ctx, []byte(`{"source":{"dojo":"our"},"sink":{"stdout":null}}`))
	if err != nil || strings.TrimSpace(identity) != s.ship {
		return "", errors.New("desk runtime identity changed after request")
	}
	if err = s.check(ctx); err != nil {
		return "", err
	}
	return result, nil
}
func (s *DeskRuntime) Observe(ctx context.Context, desk string) (deskwork.Observation, error) {
	var out deskwork.Observation
	r, err := deskwork.Inventory(s.ship, desk)
	if err != nil {
		return out, err
	}
	response, err := s.call(ctx, r)
	if err != nil {
		return out, err
	}
	body, _ := json.Marshal(response)
	if err = deskwork.DecodeJSONOutput(body, &out); err != nil {
		return out, errors.New("desk observation unavailable or unsupported")
	}
	return out, out.Validate()
}
func (s *DeskRuntime) effect(ctx context.Context, r deskwork.LensRequest, err error) error {
	if err != nil {
		return err
	}
	reply, err := s.call(ctx, r)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reply) != ">=" {
		return errors.New("desk effect was not acknowledged; inspect its outcome")
	}
	return nil
}
func (s *DeskRuntime) Create(ctx context.Context, desk string) error {
	r, e := deskwork.Create(s.ship, desk)
	return s.effect(ctx, r, e)
}
func (s *DeskRuntime) Mount(ctx context.Context, desk, alias string, revision uint64) error {
	r, e := deskwork.Mount(s.ship, desk, alias, revision)
	return s.effect(ctx, r, e)
}
func (s *DeskRuntime) Commit(ctx context.Context, alias string) error {
	r, e := deskwork.Commit(alias)
	return s.effect(ctx, r, e)
}
func (s *DeskRuntime) PublishSource(ctx context.Context, desk string) error {
	r, e := deskwork.PublishSource(desk)
	return s.effect(ctx, r, e)
}
func (s *DeskRuntime) Matches(ctx context.Context, desk string, revision uint64, name string, raw []byte) (bool, error) {
	r, e := deskwork.MarkEquivalent(s.ship, desk, revision, name, raw)
	if e != nil {
		return false, e
	}
	reply, e := s.call(ctx, r)
	if e != nil {
		return false, e
	}
	switch strings.TrimSpace(reply) {
	case "%.y":
		return true, nil
	case "%.n":
		return false, nil
	default:
		return false, errors.New("desk mark comparison unavailable or unsupported")
	}
}

func (s *DeskRuntime) ListDesks(ctx context.Context) ([]string, error) {
	request, err := deskwork.ListDesks(s.ship)
	if err != nil {
		return nil, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(reply)
	var names []string
	if err = deskwork.DecodeJSONOutput(body, &names); err != nil {
		return nil, errors.New("desk inventory unavailable or unsupported")
	}
	return deskwork.NormalizeDeskNames(names)
}
func (s *DeskRuntime) FilePaths(ctx context.Context, desk string, revision uint64) ([]string, error) {
	request, err := deskwork.FilePaths(s.ship, desk, revision)
	if err != nil {
		return nil, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(reply)
	var paths [][]string
	if err = deskwork.DecodeJSONOutput(body, &paths); err != nil {
		return nil, errors.New("desk file inventory unavailable or unsupported")
	}
	return deskwork.UnixPaths(paths)
}
func (s *DeskRuntime) AutocommitActive(ctx context.Context) (bool, error) {
	request, err := deskwork.AutocommitActive(s.ship)
	if err != nil {
		return false, err
	}
	reply, err := s.call(ctx, request)
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(reply) {
	case "%.y":
		return true, nil
	case "%.n":
		return false, nil
	default:
		return false, errors.New("autocommit state unavailable or unsupported")
	}
}
