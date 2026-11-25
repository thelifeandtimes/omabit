package runtimecandidate

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"

type Profile = runtimeprofile.Profile

const (
	ProfilesFeature    = runtimeprofile.ProfilesFeature
	OfficialProfile    = runtimeprofile.OfficialProfile
	JSONFixProfile     = runtimeprofile.JSONFixProfile
	JSONFixRequirement = runtimeprofile.JSONFixRequirement
)

func Resolve(id string) (Profile, error)        { return runtimeprofile.Resolve(id) }
func Catalogue() []Profile                      { return runtimeprofile.Catalogue() }
func Channels() map[string]string               { return runtimeprofile.Channels() }
func CompatibleSuccessor(from, to string) error { return runtimeprofile.CompatibleSuccessor(from, to) }
