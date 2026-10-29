//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func PreviewImportRestore(context.Context, ManagedState) (ImportRestoreReview, error) {
	return ImportRestoreReview{}, errors.New("import restoration requires linux/amd64")
}
