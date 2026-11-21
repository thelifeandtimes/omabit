package protocol

import candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"

const RuntimeProfilesFeature = candidate.ProfilesFeature

type RuntimeProfilePreflight struct {
	RunnerID  string            `json:"runner_id"`
	Profile   candidate.Profile `json:"profile"`
	Available bool              `json:"available"`
}
