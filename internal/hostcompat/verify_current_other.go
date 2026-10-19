//go:build !linux

package hostcompat

import (
	"context"
	"errors"
)

func VerifyCurrent(ctx context.Context) (Report, error) {
	return InspectCurrent(), errors.New("native host authentication requires Linux")
}

func VerifyCurrentWithOptions(ctx context.Context, options VerificationOptions) (Report, error) {
	return VerifyCurrent(ctx)
}
