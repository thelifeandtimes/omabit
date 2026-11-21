package runtimequalify

import "context"

// ManagedRecordingLifecycle is trusted local manager code, never RPC input.
// Preparation runs after durable creation and before runtime Start. Its returned
// fresh fence runs again after final native checks, immediately before dispatch.
// BeforeRemove preserves recorder terminal evidence before daemon-side evidence
// disappears. Callbacks receive independent snapshots and may not control the
// container. Normal managed launch does not select this lifecycle yet.
type ManagedRecordingLifecycle struct {
	BeforeStart  func(context.Context, ManagedState, string) (func(context.Context) error, error)
	BeforeRemove func(context.Context, ManagedState, string) error
}
