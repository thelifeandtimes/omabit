//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func OpenProjectDesk(ctx context.Context, s ProjectCurrentReadinessSelection, instance, containerID string) (*DeskRuntime, error) {
	e, before, err := projectFleetSnapshot(ctx, s)
	if err != nil {
		return nil, err
	}
	return deskFromProject(ctx, e, before, instance, containerID)
}
func OpenExpandedProjectDesk(ctx context.Context, s ProjectExpansion, instance, containerID string) (*DeskRuntime, error) {
	e, before, err := expansionSnapshot(ctx, s)
	if err != nil {
		return nil, err
	}
	return deskFromProject(ctx, e, before, instance, containerID)
}
func deskFromProject(ctx context.Context, e *projectCurrentReadinessEngine, before projectMemberSnapshot, instance, containerID string) (*DeskRuntime, error) {
	if !containerIDPattern.MatchString(containerID) {
		return nil, errors.New("project desk container is unresolved")
	}
	var selected *projectObservedMember
	for i := range e.source.members {
		m := &e.source.members[i]
		if m.id == containerID && m.contract.selection.InstanceID == instance {
			selected = m
		}
	}
	c, ok := before.Containers[containerID]
	if selected == nil || !ok || !deskRunning(c) {
		return nil, errors.New("selected project desk member is not running")
	}
	var memberPort, hostPort uint16
	var ship string
	for _, m := range e.source.anchor.base.contract.Selection.Plan.Members {
		if m.InstanceID == instance {
			memberPort = m.InternalHTTPPort
			hostPort = m.HTTPPort
			ship = m.Ship
		}
	}
	if m := selected.contract.selection.AddedMember; m != nil {
		memberPort = m.InternalHTTPPort
		hostPort = m.HTTPPort
		ship = m.Ship
	}
	if !protocol.ValidFakeShip(ship) {
		return nil, errors.New("project desk ship identity is invalid")
	}
	q := e.source.anchor.base.q
	path, err := pierstore.Absolute(q.d.home, selected.contract.selection.Storage.Path)
	if err != nil {
		return nil, err
	}
	ports, err := readHTTPPortsAt(path)
	if err != nil {
		return nil, err
	}
	port, err := privateHTTPPort(ports, memberPort)
	if err != nil {
		return nil, err
	}
	s := &DeskRuntime{ship: ship, pier: path, endpoint: fmt.Sprintf("http://127.0.0.1:%d", port), httpEndpoint: fmt.Sprintf("http://127.0.0.1:%d", memberPort), containerID: containerID, q: q}
	s.hostOrigin = func(ctx context.Context) (string, error) {
		if err := s.check(ctx); err != nil {
			return "", err
		}
		if hostPort < 1024 {
			return "", errors.New("project host route unavailable")
		}
		return fmt.Sprintf("http://127.0.0.1:%d", hostPort), nil
	}
	s.check = func(ctx context.Context) error {
		after, err := projectFleetInspect(ctx, e)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(before.Containers, after.Containers) {
			return errors.New("project changed during desk operation")
		}
		current, err := readHTTPPortsAt(path)
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
