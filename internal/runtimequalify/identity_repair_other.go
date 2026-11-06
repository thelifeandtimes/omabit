//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
)

func RepairBootIdentity(context.Context, ManagedState, string, string, string, *BootIdentityRepair, func(BootIdentityRepair) error) (ManagedState, error) {
	return ManagedState{}, errors.New("identity repair requires Linux/amd64")
}
