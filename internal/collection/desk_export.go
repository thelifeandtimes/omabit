package collection

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ExportDeskSource requires the complete observation returned by InspectDesk;
// the runner independently rejects changed runtime, pier or Clay bindings.
func (r *Registry) ExportDeskSource(ctx context.Context, name, runner string, observation protocol.DeskObservation, paths []string) (protocol.DeskSourceExport, error) {
	var out protocol.DeskSourceExport
	request := protocol.DeskSourceExportRequest{Selection: observation.Selection, Baseline: observation.Observation, Paths: paths}
	if err := request.Validate(); err != nil {
		return out, err
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	if err := entry.Identity().Check(observation.RunnerID, observation.Mode); err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ExportDeskSource(ctx, entry.Identity(), request)
}
