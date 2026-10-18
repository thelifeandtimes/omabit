package client

import "context"

type creationReplayKey struct{}

// WithCreationReplayOnly preserves historical request bodies/fingerprints. The
// caller must first verify the server's creation-replay-only capability.
func WithCreationReplayOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, creationReplayKey{}, true)
}

// IsCreationReplayOnly lets provisioning callers compute the original content
// descriptor without uploading artifacts for a lookup-only creation request.
func IsCreationReplayOnly(ctx context.Context) bool {
	replay, _ := ctx.Value(creationReplayKey{}).(bool)
	return replay
}
