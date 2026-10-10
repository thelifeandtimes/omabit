//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func ReadManagedLogs(context.Context, ManagedState) (protocol.RuntimeLogs, error) {
	return protocol.RuntimeLogs{}, errors.New("runtime logs require Linux/AMD64")
}
