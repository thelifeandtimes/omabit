//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func (realBackend) PrepareProjectDependencies(ctx context.Context) error {
	return runtimequalify.PrepareProjectDependencies(ctx)
}
