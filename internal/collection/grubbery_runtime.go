package collection

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) GrubberyRuntimeRecovery(ctx context.Context, name, runner, original, audit string) (protocol.GrubberyRuntimeRecoveryStatus, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.GrubberyRuntimeRecovery(ctx, entry.Identity(), original, audit)
}
func (r *Registry) ReviewGrubberyRuntime(ctx context.Context, name, runner string, q protocol.GrubberyRuntimeReviewRequest) (protocol.GrubberyRuntimeReview, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyRuntimeReview{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ReviewGrubberyRuntime(ctx, entry.Identity(), q)
}
func (r *Registry) ApplyGrubberyRuntime(ctx context.Context, name, runner string, q protocol.GrubberyRuntimeApplyRequest) (protocol.GrubberyRuntimeRecoveryStatus, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ApplyGrubberyRuntime(ctx, entry.Identity(), q)
}
