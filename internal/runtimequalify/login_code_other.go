//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func ReadLoginCode(context.Context, ManagedState, string) (string, error) {
	return "", errors.New("login-code runtime access requires the qualified Linux/AMD64 deployment")
}
