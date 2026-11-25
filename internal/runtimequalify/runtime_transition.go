package runtimequalify

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"time"
)

// SelectStoppedRuntime changes only stopped runtime intent. The caller binds
// the configuration revision and journals this state atomically. Each previous
// operation keeps its original launch records; no pier bytes are changed here.
func SelectStoppedRuntime(s ManagedState, id string, revision uint64, at time.Time) (ManagedState, error) {
	if s.UncertainMaintenance() || s.UncertainProvisioning() || s.StartupResume != nil || s.Handoff.Fenced() || len(s.RuntimeHistory) >= 256 || revision == 0 || at.IsZero() {
		return s, errors.New("runtime transition requires resolved stopped state")
	}
	for _, c := range s.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return s, errors.New("runtime transition requires removed containers")
		}
	}
	if err := runtimeprofile.CompatibleSuccessor(s.RuntimeProfile, id); err != nil {
		return s, err
	}
	from, err := runtimeprofile.Resolve(s.RuntimeProfile)
	if err != nil {
		return s, err
	}
	to, err := runtimeprofile.Resolve(id)
	if err != nil {
		return s, err
	}
	s.RuntimeHistory = append(append([]RuntimeTransition{}, s.RuntimeHistory...), RuntimeTransition{From: from, To: to, Revision: revision, ChangedAt: at})
	s.RuntimeProfile = id
	if s.Prepared {
		s.Runtime.Containers = nil
		s.Runtime.RuntimeProfile = id
		s.Runtime.Image = to.Image
		s.Runtime.VereVersion = to.VereVersion
		s.Runtime.ImageID = ""
		s.Runtime.RepoDigest = ""
	}
	return s, nil
}
