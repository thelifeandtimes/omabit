package main

import (
	"context"
	"errors"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func validCreationLoom(loom uint) bool {
	return loom <= protocol.MaxLoom && protocol.ValidRecordedLoom(uint8(loom))
}

func creationContext(ctx context.Context, loom uint, hello protocol.Hello) (context.Context, error) {
	if loom >= protocol.MinLoom {
		return ctx, nil
	}
	if !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
		return ctx, errors.New("new --loom selections must be in 30..34; update this runner for creation-replay-only capability to retry an exact historical small-loom request")
	}
	return client.WithCreationReplayOnly(ctx), nil
}
