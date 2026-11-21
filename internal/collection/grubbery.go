package collection

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) GrubberyAppOperation(ctx context.Context, name, runner, id string) (protocol.GrubberyAppOperation, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.GrubberyAppOperation(ctx, entry.Identity(), id)
}

func (r *Registry) ReviewGrubberyAppSource(ctx context.Context, name, runner string, q protocol.GrubberyAppReviewRequest) (protocol.GrubberyAppReview, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyAppReview{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ReviewGrubberyAppSource(ctx, entry.Identity(), q)
}
func (r *Registry) BeginGrubberyAppSource(ctx context.Context, name, runner string, q protocol.GrubberyAppBeginRequest) (protocol.GrubberyAppOperation, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.BeginGrubberyAppSource(ctx, entry.Identity(), q)
}
func (r *Registry) GrubberyAppSourcePhase(ctx context.Context, name, runner, id, action string, q protocol.GrubberyAppPhaseRequest) (protocol.GrubberyAppOperation, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.GrubberyAppSourcePhase(ctx, entry.Identity(), id, action, q)
}
