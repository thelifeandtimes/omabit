//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func RunRecordingService(context.Context, string, string) error {
	return errors.New("recording worker requires linux/amd64")
}
