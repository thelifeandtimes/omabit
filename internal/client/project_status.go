package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrProjectStatusUnsupported = errors.New("runner does not advertise project-saved-status-v1; saved operation status is unavailable")
var ErrProjectStatusInvalid = errors.New("invalid saved project status")

func (c *Client) ProjectStatus(ctx context.Context, expected Identity, project string) (protocol.ProjectStatus, error) {
	var result protocol.ProjectStatus
	if !protocol.ValidID(project) {
		return result, ErrProjectStatusInvalid
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return result, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return result, err
	}
	if !slices.Contains(hello.Features, protocol.ProjectStatusFeature) {
		return result, ErrProjectStatusUnsupported
	}
	if err = c.do(ctx, http.MethodGet, protocol.ProjectStatusPath(project), nil, &result, true); err != nil {
		return protocol.ProjectStatus{}, err
	}
	if err = expected.Check(result.RunnerID, result.Mode); err != nil {
		return protocol.ProjectStatus{}, err
	}
	if !validProjectStatus(result, project) {
		return protocol.ProjectStatus{}, ErrProjectStatusInvalid
	}
	return result, nil
}

func validProjectStatus(r protocol.ProjectStatus, project string) bool {
	if r.SchemaVersion != 1 || r.Project.ID != project || r.Basis != "stored-operation-history" || r.RuntimeObservation != "not-performed" || len(r.Operations) == 0 || len(r.Operations) > protocol.ProjectStatusOperationLimit || (r.HasOlderOperations && len(r.Operations) != protocol.ProjectStatusOperationLimit) {
		return false
	}
	inventory := protocol.ProjectInventory{SchemaVersion: 1, RunnerID: r.RunnerID, Mode: r.Mode, ReadAt: r.ReadAt, Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{r.Project}}
	if !validProjectInventory(inventory) {
		return false
	}
	ids := map[string]bool{}
	for _, op := range r.Operations {
		if ids[op.ID] || !protocol.ValidProjectOperationSummary(op, r.Project, r.ReadAt) {
			return false
		}
		ids[op.ID] = true
	}
	return true
}
