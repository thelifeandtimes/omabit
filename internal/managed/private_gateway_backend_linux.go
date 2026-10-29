//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Internal queues retain their accepted receipt and common work/update fence.
// These methods do not add a public route or startup dispatch.
func (realBackend) InspectPrivateGatewayInputs(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, manifest string) (string, privatechain.RuntimeInputInspection, error) {
	return runtimequalify.InspectPrivateGatewayInputs(ctx, directory, p, w, manifest)
}
func (realBackend) OpenPrivateGateway(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (privateGatewayDriver, error) {
	d, err := runtimequalify.OpenPrivateGateway(ctx, directory, p, w, g)
	if err != nil {
		return nil, err
	}
	return d, nil
}
func (realBackend) OpenPrivateGatewayObserver(ctx context.Context, directory string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, operation string) (privateGatewayObserver, error) {
	o, err := runtimequalify.OpenPrivateGatewayObserver(ctx, directory, p, w, g, operation)
	if err != nil {
		return nil, err
	}
	return o, nil
}

var _ privateGatewayBackend = realBackend{}
var _ privateGatewayObservationBackend = realBackend{}
