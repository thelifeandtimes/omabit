//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func Run(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunHTTP(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunMeld(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunEpochs(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunPrune(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunPrefix(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunImport(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunComet(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunCometMaintenance(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunCometEpochs(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func RunCometPrefix(context.Context, func(string)) (Result, error) {
	return Result{}, errors.New("qualification requires linux/amd64")
}

func ListImportSources(context.Context) ([]protocol.ImportSource, error) {
	return nil, errors.New("import sources require linux/amd64")
}
func ListPierSources(context.Context) ([]pierstore.Candidate, error) {
	return nil, errors.New("pier discovery requires linux/amd64")
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
