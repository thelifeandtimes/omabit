//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func ValidateAmesPublication(context.Context, ManagedState) error {
	return errors.New("Ames publication requires linux/amd64")
}
