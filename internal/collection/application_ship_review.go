package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

// ApplicationShipApprovalView omits the potentially large audit history, but
// is read from the same validated, locked journal. It contains no credentials.
type ApplicationShipApprovalView struct {
	Version  int                       `json:"version"`
	Revision uint64                    `json:"revision"`
	Entries  []ApplicationShipApproval `json:"entries"`
}

func (s ApplicationShipApprovals) View() ApplicationShipApprovalView {
	return ApplicationShipApprovalView{Version: s.Version, Revision: s.Revision, Entries: s.Entries}
}

type ApplicationShipReview struct {
	Version   int                         `json:"version"`
	Routes    protocol.ShipRoutes         `json:"routes"`
	Approvals ApplicationShipApprovalView `json:"approvals"`
}

func (r *Registry) ReviewApplicationShip(ctx context.Context, name, runner, instance string) (ApplicationShipReview, error) {
	routes, err := r.ShipRoutes(ctx, name, runner, instance)
	if err != nil {
		return ApplicationShipReview{}, err
	}
	state, err := r.ApplicationShipApprovals(ctx)
	if err != nil {
		return ApplicationShipReview{}, err
	}
	return ApplicationShipReview{Version: 1, Routes: routes, Approvals: state.View()}, nil
}

type ApplicationShipApprovalRequest struct {
	Version          int                          `json:"version"`
	ExpectedRevision uint64                       `json:"expected_revision"`
	Binding          shiproute.Binding            `json:"binding"`
	Target           ApplicationTarget            `json:"target"`
	Allowed          *bool                        `json:"allowed"`
	Selection        *protocol.ShipRouteSelection `json:"selection,omitempty"`
}

func (u ApplicationShipApprovalRequest) Validate() error {
	if u.Version != 1 || u.ExpectedRevision > apppolicy.MaxRevision || u.Binding.Validate() != nil || !u.Target.Valid() || u.Allowed == nil {
		return ErrApplicationConflict
	}
	if *u.Allowed && u.Selection == nil || u.Selection != nil && (u.Selection.Validate() != nil || u.Selection.Target != u.Binding.Target) {
		return ErrApplicationConflict
	}
	return nil
}

func DecodeApplicationShipApprovalRequest(data []byte) (ApplicationShipApprovalRequest, error) {
	var u ApplicationShipApprovalRequest
	if len(data) > 16<<10 || !uniqueApplicationJSON(data) {
		return u, ErrApplicationConflict
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&u) != nil || d.Decode(new(any)) != io.EOF || u.Validate() != nil {
		return ApplicationShipApprovalRequest{}, ErrApplicationConflict
	}
	return u, nil
}

// ConfigureApplicationShipApproval is a trusted user-choice boundary. The
// caller resolves and compares the installed helper before allowing it. Grants
// revalidate the selected runtime and pier. Revocation needs neither a running
// ship nor an installed helper, so broken/uninstalled integrations stay revocable.
func (r *Registry) ConfigureApplicationShipApproval(ctx context.Context, name, runner string, u ApplicationShipApprovalRequest) (ApplicationShipApprovalView, error) {
	if u.Validate() != nil {
		return ApplicationShipApprovalView{}, ErrApplicationConflict
	}
	if *u.Allowed {
		if runner != u.Binding.RunnerID {
			return ApplicationShipApprovalView{}, ErrApplicationConflict
		}
		current, err := r.ShipRoutes(ctx, name, runner, u.Binding.Target.InstanceID)
		if err != nil {
			return ApplicationShipApprovalView{}, err
		}
		if current.Record.Binding != u.Binding || current.Selection != *u.Selection {
			return ApplicationShipApprovalView{}, ErrApplicationConflict
		}
	} else {
		state, err := r.ApplicationShipApprovals(ctx)
		if err != nil {
			return ApplicationShipApprovalView{}, err
		}
		found := false
		for _, a := range state.Entries {
			if a.Binding == u.Binding && a.Target == u.Target {
				found = true
			}
		}
		if !found || state.Revision != u.ExpectedRevision {
			return ApplicationShipApprovalView{}, ErrApplicationConflict
		}
	}
	state, err := r.SetApplicationShipApproval(ctx, u.ExpectedRevision, u.Binding, u.Target, *u.Allowed)
	return state.View(), err
}
