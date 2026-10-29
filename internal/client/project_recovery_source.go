package client

import (
	"context"
	"encoding/hex"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"slices"
)

var ErrProjectRecoverySourceInvalid = errors.New("invalid saved project recovery source")

func (c *Client) ProjectRecoverySource(ctx context.Context, expected Identity, id string) (protocol.ProjectRecoverySource, error) {
	var zero protocol.ProjectRecoverySource
	if !protocol.ValidID(id) {
		return zero, ErrProjectRecoverySourceInvalid
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return zero, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return zero, err
	}
	if !slices.Contains(hello.Features, protocol.ProjectRecoverySourceFeature) {
		return zero, errors.New("runner does not advertise " + protocol.ProjectRecoverySourceFeature)
	}
	var out protocol.ProjectRecoverySource
	if err = c.do(ctx, http.MethodGet, protocol.ProjectRecoverySourcePath(id), nil, &out, true); err != nil {
		return zero, err
	}
	if err = expected.Check(out.RunnerID, out.Mode); err != nil {
		return zero, err
	}
	if !validProjectRecoverySource(out, id) {
		return zero, ErrProjectRecoverySourceInvalid
	}
	return out, nil
}
func validProjectRecoverySource(r protocol.ProjectRecoverySource, id string) bool {
	if r.SchemaVersion != 1 || r.Project.ID != id || r.Basis != "stored-recovery-source" || r.RuntimeObservation != "not-performed" || r.ActionAuthority != "not-granted" || len(r.HistoryDigest) != 64 {
		return false
	}
	if _, err := hex.DecodeString(r.HistoryDigest); err != nil {
		return false
	}
	inv := protocol.ProjectInventory{SchemaVersion: 1, RunnerID: r.RunnerID, Mode: r.Mode, ReadAt: r.ReadAt, Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{r.Project}}
	if !validProjectInventory(inv) || len(r.Members) != len(r.Project.Members) {
		return false
	}
	used := map[string]bool{}
	for i, m := range r.Members {
		if m.InstanceID != r.Project.Members[i].InstanceID || m.ContainerGeneration == 0 {
			return false
		}
		for _, ref := range []string{m.CreatorOperationID, m.OriginalReadinessOperationID} {
			if !protocol.ValidID(ref) || used[ref] {
				return false
			}
			used[ref] = true
		}
		if m.CurrentReadinessOperationID != "" {
			if !protocol.ValidID(m.CurrentReadinessOperationID) || used[m.CurrentReadinessOperationID] {
				return false
			}
			used[m.CurrentReadinessOperationID] = true
		}
	}
	return true
}
