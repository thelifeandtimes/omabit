//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func Run(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunHTTP(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}
