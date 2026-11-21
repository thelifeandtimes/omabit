//go:build !linux || !amd64

package runtimequalify

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func InstallRuntimeProfile(context.Context, string, string) (candidate.Profile, error) {
	return candidate.Profile{}, errors.New("runtime profiles require linux/amd64")
}
func VerifyRuntimeProfile(context.Context, string) error {
	return errors.New("runtime profiles require linux/amd64")
}
func InspectExternalPierWithProfile(context.Context, string, string) (pierstore.ExternalSource, error) {
	return pierstore.ExternalSource{}, errors.New("runtime profiles require linux/amd64")
}
func InspectPierIdentityWithProfile(context.Context, string, string) (protocol.PierIdentity, error) {
	return protocol.PierIdentity{}, errors.New("runtime profiles require linux/amd64")
}
