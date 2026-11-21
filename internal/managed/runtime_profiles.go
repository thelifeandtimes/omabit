package managed

import candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"

func validateRuntimeProfile(id string) error {
	if _, err := candidate.Resolve(id); err != nil {
		return bad(err.Error())
	}
	return nil
}

func validateRuntimeChange(from, to string) error {
	if err := candidate.CompatibleSuccessor(from, to); err != nil {
		return conflict("runtime_incompatible", err.Error())
	}
	a, _ := candidate.Resolve(from)
	b, _ := candidate.Resolve(to)
	if a.ID != b.ID {
		return conflict("runtime_transition_required", "runtime selection is bound to this pier; a transition requires a separately reviewed checkpoint/replay plan. Do not re-enroll, edit epoch metadata or downgrade to bypass this refusal")
	}
	// Avoid changing the persisted encoding of an existing equivalent binding.
	if from != to {
		return conflict("runtime_binding_immutable", "keep the existing runtime selection encoding")
	}
	return nil
}
