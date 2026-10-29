package projectnetwork

import (
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// MemberContext selects retained ownership for a future fresh observation. It is
// neither that observation nor permission to change a runtime. In particular,
// the original readiness keeps its historical binding after a peer replacement.
// Its persisted use requires the versioned current-readiness journal.
type MemberContext struct {
	Member  MemberGeneration   `json:"member"`
	Binding RuntimeBinding     `json:"binding"`
	Storage pierstore.Location `json:"storage"`
	// Initial launch/readiness receipts did not retain a process start time.
	// Zero means unknown, never proof of uninterrupted execution since launch.
	KnownStartedAt time.Time `json:"known_started_at"`
	// Fresh observations must follow all creation/readiness evidence used here.
	NotBefore time.Time `json:"not_before"`
}

// SelectMemberContext combines the initialized pier identity from original
// readiness with the exact latest creator from the complete causal history.
// Callers must load validated journals and reject unresolved operations before
// using this selection to collect a new observation. No discovery or adoption is
// performed, and no old readiness or Stop binding is rewritten.
func SelectMemberContext(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness) (MemberContext, error) {
	current, err := ResolveMemberGenerations(plan, launches, resumes)
	if err != nil {
		return MemberContext{}, err
	}
	return selectMemberContextFromValidatedGenerations(plan, launches, resumes, original, current)
}

// The caller just resolved this exact history or a prefix captured during that
// resolution. Keep this helper private: it still validates original pier proof,
// but must not repeat the completed-history traversal for each member.
func selectMemberContextFromValidatedGenerations(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness, current MemberGenerationSet) (MemberContext, error) {
	var result MemberContext
	if original.Validate(plan, launches) != nil {
		return result, errors.New("current context requires valid original readiness")
	}
	for _, member := range current.Members {
		if member.InstanceID != original.InstanceID {
			continue
		}
		if member.Storage.Path != original.Storage.Path || member.Storage.MountIdentity != original.Storage.MountIdentity {
			return result, errors.New("current creator changed original pier ownership")
		}
		result = MemberContext{Member: member, Binding: current.Binding, Storage: original.Storage, NotBefore: original.CapturedAt}
		result.Binding.Members = slices.Clone(current.Binding.Members)
		for _, resume := range resumes {
			if resume.After.CapturedAt.After(result.NotBefore) {
				result.NotBefore = resume.After.CapturedAt
			}
		}
		switch member.Mode {
		case "boot":
			if member.Generation != 1 || member.OperationID != original.LaunchOperationID {
				return MemberContext{}, errors.New("initial creator differs from original readiness")
			}
		case "resume":
			if member.Storage != original.Storage {
				return MemberContext{}, errors.New("replacement changed initialized pier identity")
			}
			for _, resume := range resumes {
				if resume.OperationID == member.OperationID && resume.After != nil {
					result.KnownStartedAt = resume.After.StartedAt
				}
			}
			if result.KnownStartedAt.IsZero() {
				return MemberContext{}, errors.New("replacement process evidence missing")
			}
		default:
			return MemberContext{}, errors.New("unknown member creator")
		}
		return result, nil
	}
	return result, errors.New("member absent from retained current context")
}
