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

func OpenManagedHost() (*ManagedHost, error) {
	return nil, errors.New("managed runner requires linux/amd64")
}

func ApplyManaged(context.Context, string, ManagedState, func(ManagedState) error) (ManagedState, ManagedObservation, error) {
	return ManagedState{}, ManagedObservation{}, errors.New("managed runner requires linux/amd64")
}

func ObserveManaged(context.Context, ManagedState) (ManagedState, ManagedObservation, error) {
	return ManagedState{}, ManagedObservation{}, errors.New("managed runner requires linux/amd64")
}
