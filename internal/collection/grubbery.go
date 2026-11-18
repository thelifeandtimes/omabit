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
