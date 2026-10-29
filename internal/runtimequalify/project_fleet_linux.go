//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Project fleet observations reuse retained creation contracts. They are read-only
// and ephemeral: polling does not append a journal or confer lifecycle authority.
func ReadProjectFleet(ctx context.Context, s ProjectCurrentReadinessSelection) ([]protocol.Instance, error) {
	e, snap, err := projectFleetSnapshotWithDependencies(ctx, s, fleetObservationDependencies(defaults()))
	if err != nil {
		return nil, err
	}
	return projectFleetRows(ctx, s, e, snap)
}

func projectFleetRows(ctx context.Context, s ProjectCurrentReadinessSelection, e *projectCurrentReadinessEngine, snap projectMemberSnapshot) ([]protocol.Instance, error) {
	rows := []protocol.Instance{}
	for _, m := range s.Project.Plan.Members {
		var containerID string
		for _, c := range e.source.members {
			if c.contract.selection.InstanceID == m.InstanceID {
				containerID = c.id
			}
		}
		c, ok := snap.Containers[containerID]
		if !ok {
			rows = append(rows, protocol.Instance{ID: m.InstanceID, Label: m.InstanceID, ProjectID: s.Project.ProjectID, ProjectLabel: s.Project.Plan.Label, Ship: m.Ship, Mode: "fake", ConfigRevision: s.Project.Revision, Config: protocol.Configuration{HTTPPort: m.HTTPPort, Loom: 31}, ObservedState: "unknown", Health: "unknown"})
			continue
		}
		at := snap.Observation.ObservedAt
		row := protocol.Instance{ID: m.InstanceID, Label: m.InstanceID, ProjectID: s.Project.ProjectID, ProjectLabel: s.Project.Plan.Label, Ship: m.Ship, Mode: "fake", ConfigRevision: s.Project.Revision, AppliedRevision: s.Project.Revision, Config: protocol.Configuration{HTTPPort: m.HTTPPort, Loom: 31}, ObservedState: "stopped", DesiredState: "stopped", Health: "unknown", ObservedAt: &at, Endpoint: fmt.Sprintf("http://127.0.0.1:%d", m.HTTPPort), Resources: &protocol.ResourceMetrics{Status: "unavailable", Source: "project-runtime", ContainerID: containerID, ObservedAt: &at}}
		if *c.State.Running {
			row.ObservedState, row.DesiredState = "running", "running"
		}

		for _, known := range e.source.members {
			if known.id == containerID {
				row.PierPath = known.contract.selection.Storage.Path
				row.PierName = strings.TrimPrefix(m.Ship, "~")
			}
		}
		rows = append(rows, row)
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	for index := range rows {
		if rows[index].Resources == nil {
			continue
		}
		c := snap.Containers[rows[index].Resources.ContainerID]
		wg.Add(1)
		go func(index int, c container) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				return
			}
			row := &rows[index]
			if *c.State.Running {
				identity, err := e.source.anchor.base.q.d.hostIdentity(ctx, row.Endpoint+"/~/host")
				if err == nil && identity == row.Ship {
					row.Health = "healthy"
				}
			}
			row.Resources = projectResourceMetrics(ctx, e.source.anchor.base.q, c)
		}(index, c)
	}
	wg.Wait()
	// Discard identity probes if a process changed during the HTTP reads.
	after, err := projectFleetInspect(ctx, e)
	if err != nil || !reflect.DeepEqual(snap.Containers, after.Containers) {
		return nil, errors.New("project changed during fleet observation")
	}
	return rows, nil
}

func projectFleetSnapshot(ctx context.Context, s ProjectCurrentReadinessSelection) (*projectCurrentReadinessEngine, projectMemberSnapshot, error) {
	return projectFleetSnapshotWithDependencies(ctx, s, defaults())
}

func projectFleetSnapshotWithDependencies(ctx context.Context, s ProjectCurrentReadinessSelection, d dependencies) (*projectCurrentReadinessEngine, projectMemberSnapshot, error) {
	raw, err := openCurrentProjectReadiness(ctx, s, d)
	if err != nil {
		return nil, projectMemberSnapshot{}, err
	}
	e := raw.(*projectCurrentReadinessEngine)
	snap, err := projectFleetInspect(ctx, e)
	return e, snap, err
}
func projectFleetInspect(ctx context.Context, e *projectCurrentReadinessEngine) (projectMemberSnapshot, error) {
	// Displaying an exited owned container is safe without granting Stop/Resume
	// authority. Derive its process timestamp from inspection, then validate its
	// full creation contract and bracket the complete namespace inventory.
	ids := []string{}
	for _, m := range e.source.members {
		ids = append(ids, m.id)
	}
	actual := map[string]container{}
	var err error
	if len(ids) > 0 {
		actual, err = e.source.anchor.inspectProjectContainers(ctx, ids)
	}
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	for i, m := range e.source.members {
		c := actual[m.id]
		if c.State == nil || c.State.Running == nil {
			return projectMemberSnapshot{}, errUntrustedContainer
		}
		if !*c.State.Running && c.State.Status == "exited" {
			started, err := time.Parse(time.RFC3339Nano, c.State.StartedAt)
			if err != nil || started.IsZero() {
				return projectMemberSnapshot{}, errUntrustedContainer
			}
			e.source.members[i].allowExited = true
			e.source.members[i].processStartedAt = started
		}
	}
	if e.recovery != nil {
		return e.recovery.inspectMembers(ctx, e.source.members, nil)
	}
	return e.source.anchor.inspectProjectMemberSnapshot(ctx, e.source.members)
}

// ReadProjectLoginCode verifies the same owned project/container before and after
// private Eyre access. Neither the code nor the response is persisted.
func ReadProjectLoginCode(ctx context.Context, s ProjectCurrentReadinessSelection, instance, containerID string) (string, error) {
	e, before, err := projectFleetSnapshot(ctx, s)
	if err != nil {
		return "", err
	}
	return projectCodeFromEngine(ctx, e, before, instance, containerID)
}

func projectCodeFromEngine(ctx context.Context, e *projectCurrentReadinessEngine, before projectMemberSnapshot, instance, containerID string) (string, error) {
	var selected *projectObservedMember
	for i := range e.source.members {
		m := &e.source.members[i]
		if m.contract.selection.InstanceID == instance && m.id == containerID {
			selected = m
		}
	}
	if selected == nil || !*before.Containers[containerID].State.Running {
		return "", errors.New("selected project member is not running")
	}
	var memberPort uint16
	var ship string
	for _, m := range e.source.anchor.base.contract.Selection.Plan.Members {
		if m.InstanceID == instance {
			memberPort = m.InternalHTTPPort
			ship = m.Ship
		}
	}
	if m := selected.contract.selection.AddedMember; m != nil {
		memberPort = m.InternalHTTPPort
		ship = m.Ship
	}
	path, err := pierstore.Absolute(e.source.anchor.base.q.d.home, selected.contract.selection.Storage.Path)
	if err != nil {
		return "", err
	}
	ports, err := readHTTPPortsAt(path)
	if err != nil {
		return "", err
	}
	port, err := privateHTTPPort(ports, memberPort)
	if err != nil {
		return "", err
	}
	q := e.source.anchor.base.q
	read := func(formula string) (string, error) {
		body := `{"source":{"dojo":"` + formula + `"},"sink":{"stdout":null}}`
		out, err := q.docker(ctx, 3*time.Second, "container", "exec", containerID, "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--fail", "--max-time", "2", "--max-filesize", "1024", "--header", "Content-Type: application/json", "--data-binary", body, fmt.Sprintf("http://127.0.0.1:%d", port))
		var value string
		if err != nil || len(out) > 1024 || json.Unmarshal([]byte(out), &value) != nil {
			return "", errors.New("project private Eyre read failed")
		}
		return strings.TrimSpace(value), nil
	}
	identity, err := read("our")
	if err != nil || identity != ship {
		return "", errors.New("project identity changed")
	}
	code, err := read("+code")
	if err != nil || !protocol.ValidLoginCode(code) {
		return "", errors.New("project code unavailable")
	}
	identity, err = read("our")
	if err != nil || identity != ship {
		return "", errors.New("project identity changed")
	}
	after, err := projectFleetInspect(ctx, e)
	if err != nil || !reflect.DeepEqual(before.Containers, after.Containers) {
		return "", errors.New("project changed during code retrieval")
	}
	current, err := readHTTPPortsAt(path)
	if err != nil || current != ports {
		return "", errHTTPPorts
	}
	return code, nil
}

func ReadProjectLogs(ctx context.Context, s ProjectCurrentReadinessSelection, id string) (protocol.RuntimeLogs, error) {
	e, before, err := projectFleetSnapshot(ctx, s)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	return projectLogsFromEngine(ctx, e, before, id)
}
func ReadExpandedProjectLogs(ctx context.Context, s ProjectExpansion, id string) (protocol.RuntimeLogs, error) {
	e, before, err := expansionSnapshot(ctx, s)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	return projectLogsFromEngine(ctx, e, before, id)
}
func projectLogsFromEngine(ctx context.Context, e *projectCurrentReadinessEngine, before projectMemberSnapshot, id string) (protocol.RuntimeLogs, error) {
	var cid string
	for _, m := range e.source.members {
		if m.contract.selection.InstanceID == id {
			cid = m.id
		}
	}
	if cid == "" {
		return protocol.RuntimeLogs{}, errors.New("project member unavailable")
	}
	q := e.source.anchor.base.q
	data, err := q.d.logs(ctx, q.socket, cid)
	if err != nil {
		return protocol.RuntimeLogs{}, err
	}
	after, err := projectFleetInspect(ctx, e)
	if err != nil || !reflect.DeepEqual(before.Containers, after.Containers) {
		return protocol.RuntimeLogs{}, errors.New("project runtime changed during log read")
	}
	lines, truncated := sanitizeLogs(data)
	return protocol.RuntimeLogs{ContainerID: cid, Kind: "run", Source: "docker", ObservedAt: time.Now().UTC(), Lines: lines, Truncated: truncated}, nil
}
