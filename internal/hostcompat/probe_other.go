//go:build !linux

package hostcompat

import (
	"context"
	"errors"
)

func ProbeEngine(ctx context.Context, socket, expectedDataRoot string) (Engine, error) {
	return Engine{}, errors.New("private engine probing requires Linux")
}
